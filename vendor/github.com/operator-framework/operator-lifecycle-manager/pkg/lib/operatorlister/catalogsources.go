package operatorlister

import (
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	listers "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/listers/operators/v1alpha1"
)

type UnionCatalogSourceLister struct {
	catsrcListers map[string]listers.CatalogSourceLister
	catsrcLock    sync.RWMutex
}

// List lists all CatalogSources in the indexer.
func (ucl *UnionCatalogSourceLister) List(selector labels.Selector) (ret []*v1alpha1.CatalogSource, err error) {
	ucl.catsrcLock.RLock()
	defer ucl.catsrcLock.RUnlock()

	set := make(map[types.UID]*v1alpha1.CatalogSource)
	for _, cl := range ucl.catsrcListers {
		catsrcs, err := cl.List(selector)
		if err != nil {
			return nil, err
		}

		for _, catsrc := range catsrcs {
			set[catsrc.GetUID()] = catsrc
		}
	}

	for _, catsrc := range set {
		ret = append(ret, catsrc)
	}

	return
}

// CatalogSources returns an object that can list and get CatalogSources.
func (ucl *UnionCatalogSourceLister) CatalogSources(namespace string) listers.CatalogSourceNamespaceLister {
	ucl.catsrcLock.RLock()
	defer ucl.catsrcLock.RUnlock()

	// Check for specific namespace listers
	if cl, ok := ucl.catsrcListers[namespace]; ok {
		return cl.CatalogSources(namespace)
	}

	// Check for any namespace-all listers
	if cl, ok := ucl.catsrcListers[metav1.NamespaceAll]; ok {
		return cl.CatalogSources(namespace)
	}

	return &NullCatalogSourceNamespaceLister{}
}

func (ucl *UnionCatalogSourceLister) RegisterCatalogSourceLister(namespace string, lister listers.CatalogSourceLister) {
	ucl.catsrcLock.Lock()
	defer ucl.catsrcLock.Unlock()

	if ucl.catsrcListers == nil {
		ucl.catsrcListers = make(map[string]listers.CatalogSourceLister)
	}

	ucl.catsrcListers[namespace] = lister
}

func (l *operatorsV1alpha1Lister) RegisterCatalogSourceLister(namespace string, lister listers.CatalogSourceLister) {
	l.catalogSourceLister.RegisterCatalogSourceLister(namespace, lister)
}

func (l *operatorsV1alpha1Lister) CatalogSourceLister() listers.CatalogSourceLister {
	return l.catalogSourceLister
}

// NullCatalogSourceNamespaceLister is an implementation of a null CatalogSourceNamespaceLister. It is
// used to prevent nil pointers when no CatalogSourceNamespaceLister has been registered for a given
// namespace.
type NullCatalogSourceNamespaceLister struct {
	listers.CatalogSourceNamespaceLister
}

// List returns nil and an error explaining that this is a NullCatalogSourceNamespaceLister.
func (n *NullCatalogSourceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.CatalogSource, err error) {
	return nil, fmt.Errorf("cannot list CatalogSources with a NullCatalogSourceNamespaceLister")
}

// Get returns nil and an error explaining that this is a NullCatalogSourceNamespaceLister.
func (n *NullCatalogSourceNamespaceLister) Get(name string) (*v1alpha1.CatalogSource, error) {
	return nil, fmt.Errorf("cannot get CatalogSource with a NullCatalogSourceNamespaceLister")
}
