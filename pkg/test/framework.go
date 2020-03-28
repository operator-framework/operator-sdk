// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"bytes"
	goctx "context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	log "github.com/sirupsen/logrus"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Global framework struct
	Global *Framework
)

type Framework struct {
	Client            *frameworkClient
	KubeConfig        *rest.Config
	KubeClient        kubernetes.Interface
	Scheme            *runtime.Scheme
	NamespacedManPath *string
	OperatorNamespace string
	WatchNamespace    string

	restMapper *restmapper.DeferredDiscoveryRESTMapper

	projectRoot        string
	globalManPath      string
	localOperatorArgs  string
	kubeconfigPath     string
	schemeMutex        sync.Mutex
	LocalOperator      bool
	skipCleanupOnError bool
}

type frameworkOpts struct {
	projectRoot        string
	kubeconfigPath     string
	globalManPath      string
	namespacedManPath  string
	localOperatorArgs  string
	isLocalOperator    bool
	skipCleanupOnError bool
}

const (
	ProjRootFlag           = "root"
	KubeConfigFlag         = "kubeconfig"
	NamespacedManPathFlag  = "namespacedMan"
	GlobalManPathFlag      = "globalMan"
	LocalOperatorFlag      = "localOperator"
	LocalOperatorArgs      = "localOperatorArgs"
	SkipCleanupOnErrorFlag = "skipCleanupOnError"

	TestOperatorNamespaceEnv = "TEST_OPERATOR_NAMESPACE"
	TestWatchNamespaceEnv    = "TEST_WATCH_NAMESPACE"
)

func (opts *frameworkOpts) addToFlagSet(flagset *flag.FlagSet) {
	flagset.StringVar(&opts.projectRoot, ProjRootFlag, "", "path to project root")
	flagset.StringVar(&opts.namespacedManPath, NamespacedManPathFlag, "", "path to rbac manifest")
	flagset.BoolVar(&opts.isLocalOperator, LocalOperatorFlag, false,
		"enable if operator is running locally (not in cluster)")
	flagset.StringVar(&opts.globalManPath, GlobalManPathFlag, "", "path to operator manifest")
	flagset.StringVar(&opts.localOperatorArgs, LocalOperatorArgs, "",
		"flags that the operator needs (while using --up-local). example: \"--flag1 value1 --flag2=value2\"")
	flagset.BoolVar(&opts.skipCleanupOnError, SkipCleanupOnErrorFlag, false,
		"If set as true, the cleanup function responsible to remove all artifacts "+
			"will be skipped if an error is faced.")
}

func newFramework(opts *frameworkOpts) (*Framework, error) {
	kubeconfig, kcNamespace, err := k8sInternal.GetKubeconfigAndNamespace(opts.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeconfig: %w", err)
	}

	operatorNamespace := kcNamespace
	ns, ok := os.LookupEnv(TestOperatorNamespaceEnv)
	if ok && ns != "" {
		operatorNamespace = ns
	}

	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeclient: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add cgo scheme to runtime scheme: %w", err)
	}
	if err := extscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add api extensions scheme to runtime scheme: %w", err)
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dynClient, err := dynclient.New(kubeconfig, dynclient.Options{Scheme: scheme, Mapper: restMapper})
	if err != nil {
		return nil, fmt.Errorf("failed to build the dynamic client: %w", err)
	}
	framework := &Framework{
		Client:            &frameworkClient{Client: dynClient},
		KubeConfig:        kubeconfig,
		KubeClient:        kubeclient,
		Scheme:            scheme,
		NamespacedManPath: &opts.namespacedManPath,
		OperatorNamespace: operatorNamespace,
		LocalOperator:     opts.isLocalOperator,

		projectRoot:        opts.projectRoot,
		globalManPath:      opts.globalManPath,
		localOperatorArgs:  opts.localOperatorArgs,
		kubeconfigPath:     opts.kubeconfigPath,
		restMapper:         restMapper,
		skipCleanupOnError: opts.skipCleanupOnError,
	}
	return framework, nil
}

type addToSchemeFunc func(*runtime.Scheme) error

// AddToFrameworkScheme allows users to add the scheme for their custom resources
// to the framework's scheme for use with the dynamic client. The user provides
// the addToScheme function (located in the register.go file of their operator
// project) and the List struct for their custom resource. For example, for a
// memcached operator, the list stuct may look like:
// &MemcachedList{}
// The List object is needed because the CRD has not always been fully registered
// by the time this function is called. If the CRD takes more than 5 seconds to
// become ready, this function throws an error
func AddToFrameworkScheme(addToScheme addToSchemeFunc, obj runtime.Object) error {
	return Global.addToScheme(addToScheme, obj)
}

func (f *Framework) addToScheme(addToScheme addToSchemeFunc, obj runtime.Object) error {
	f.schemeMutex.Lock()
	defer f.schemeMutex.Unlock()

	err := addToScheme(f.Scheme)
	if err != nil {
		return err
	}
	f.restMapper.Reset()
	dynClient, err := dynclient.New(f.KubeConfig, dynclient.Options{Scheme: f.Scheme, Mapper: f.restMapper})
	if err != nil {
		return fmt.Errorf("failed to initialize new dynamic client: %w", err)
	}
	err = wait.PollImmediate(time.Second, time.Second*10, func() (done bool, err error) {
		ns, ok := os.LookupEnv(TestOperatorNamespaceEnv)
		if ok && ns != "" {
			err = dynClient.List(goctx.TODO(), obj, dynclient.InNamespace(f.OperatorNamespace))
		} else {
			err = dynClient.List(goctx.TODO(), obj, dynclient.InNamespace("default"))
		}
		if err != nil {
			f.restMapper.Reset()
			return false, nil
		}
		f.Client = &frameworkClient{Client: dynClient}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to build the dynamic client: %w", err)
	}
	return nil
}

func (f *Framework) runM(m *testing.M) (int, error) {
	// setup context to use when setting up crd
	ctx := f.newContext(nil)
	defer ctx.Cleanup()

	// go test always runs from the test directory; change to project root
	err := os.Chdir(f.projectRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to change directory to project root: %w", err)
	}

	// create crd
	globalYAML, err := ioutil.ReadFile(f.globalManPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read global resource manifest: %w", err)
	}
	err = ctx.createFromYAML(globalYAML, true, &CleanupOptions{TestContext: ctx})
	if err != nil {
		return 0, fmt.Errorf("failed to create resource(s) in global resource manifest: %w", err)
	}

	if !f.LocalOperator {
		return m.Run(), nil
	}

	// start local operator before running tests
	outBuf := &bytes.Buffer{}
	localCmd, err := f.setupLocalCommand()
	if err != nil {
		return 0, fmt.Errorf("failed to setup local command: %w", err)
	}
	localCmd.Stdout = outBuf
	localCmd.Stderr = outBuf

	err = localCmd.Start()
	if err != nil {
		return 0, fmt.Errorf("failed to run operator locally: %w", err)
	}
	log.Info("Started local operator")

	// run the tests
	exitCode := m.Run()

	// kill the local operator and print its logs
	err = localCmd.Process.Kill()
	if err != nil {
		log.Warn("Failed to stop local operator process")
	}
	fmt.Printf("\n------ Local operator output ------\n%s\n", outBuf.String())
	return exitCode, nil
}

func (f *Framework) setupLocalCommand() (*exec.Cmd, error) {
	projectName := filepath.Base(projutil.MustGetwd())
	outputBinName := filepath.Join(scaffold.BuildBinDir, projectName+"-local")
	opts := projutil.GoCmdOptions{
		BinName:     outputBinName,
		PackagePath: filepath.Join(scaffold.ManagerDir, scaffold.CmdFile),
	}
	if err := projutil.GoBuild(opts); err != nil {
		return nil, fmt.Errorf("failed to build local operator binary: %w", err)
	}

	args := []string{}
	if f.localOperatorArgs != "" {
		args = append(args, strings.Split(f.localOperatorArgs, " ")...)
	}

	localCmd := exec.Command(outputBinName, args...)

	if f.kubeconfigPath != "" {
		localCmd.Env = append(os.Environ(), fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, f.kubeconfigPath))
	} else {
		// we can hardcode index 0 as that is the highest priority kubeconfig to be loaded and will always
		// be populated by NewDefaultClientConfigLoadingRules()
		localCmd.Env = append(os.Environ(), fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar,
			clientcmd.NewDefaultClientConfigLoadingRules().Precedence[0]))
	}
	watchNamespace := f.OperatorNamespace
	ns, ok := os.LookupEnv(TestWatchNamespaceEnv)
	if ok {
		watchNamespace = ns
	}
	localCmd.Env = append(localCmd.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, watchNamespace))
	return localCmd, nil
}
