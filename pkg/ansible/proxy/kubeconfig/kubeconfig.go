package kubeconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// kubectl, as of 1.10.5, only does basic auth if the username is present in
// the URL. The python client used by ansible, as of 6.0.0, only does basic
// auth if the username and password are provided under the "user" key within
// "users".
const kubeConfigTemplate = `---
apiVersion: v1
kind: Config
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: {{.ProxyURL}}
  name: proxy-server
contexts:
- context:
    cluster: proxy-server
    user: admin/proxy-server
  name: {{.Namespace}}/proxy-server
current-context: {{.Namespace}}/proxy-server
preferences: {}
users:
- name: admin/proxy-server
  user:
    username: {{.Username}}
    password: unused
`

// values holds the data used to render the template
type values struct {
	Username  string
	ProxyURL  string
	Namespace string
}

// Create renders a kubeconfig template and writes it to disk
func Create(ownerRef metav1.OwnerReference, proxyURL string, namespace string) (*os.File, error) {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	ownerRefJSON, err := json.Marshal(ownerRef)
	if err != nil {
		return nil, err
	}
	username := base64.URLEncoding.EncodeToString([]byte(ownerRefJSON))
	parsedURL.User = url.User(username)
	v := values{
		Username:  username,
		ProxyURL:  parsedURL.String(),
		Namespace: namespace,
	}

	var parsed bytes.Buffer

	t := template.Must(template.New("kubeconfig").Parse(kubeConfigTemplate))
	t.Execute(&parsed, v)

	file, err := ioutil.TempFile("", "kubeconfig")
	if err != nil {
		return nil, err
	}
	// multiple calls to close file will not hurt anything,
	// but we don't want to lose the error because we are
	// writing to the file, so we will call close twice.
	defer file.Close()

	if _, err := file.WriteString(parsed.String()); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return file, nil
}
