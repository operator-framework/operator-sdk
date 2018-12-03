package cache

import (
	"fmt"
	"time"

	toolscache "k8s.io/client-go/tools/cache"
)

type multiNamesapceIndexInformer struct {
	indexInformers map[string]toolscache.SharedIndexInformer
}

func (m *multiNamesapceIndexInformer) AddIndexers(indexers toolscache.Indexers) error {
	return nil
}

func (m *multiNamesapceIndexInformer) GetIndexer() toolscache.Indexer {
	return nil
}

func (m *multiNamesapceIndexInformer) AddEventHandler(handler toolscache.ResourceEventHandler) {
	for namespace, i := range m.indexInformers {
		fmt.Printf("\nadding event handler: %v for namespace: %v\n", i, namespace)
		i.AddEventHandler(handler)
	}
}

func (m *multiNamesapceIndexInformer) AddEventHandlerWithResyncPeriod(handler toolscache.ResourceEventHandler, resyncPeriod time.Duration) {
	for _, i := range m.indexInformers {
		i.AddEventHandlerWithResyncPeriod(handler, resyncPeriod)
	}
}

func (m *multiNamesapceIndexInformer) GetStore() toolscache.Store {
	return nil
}

func (m *multiNamesapceIndexInformer) GetController() toolscache.Controller {
	return nil

}

func (m *multiNamesapceIndexInformer) Run(stopCh <-chan struct{}) {
	for _, i := range m.indexInformers {
		i.Run(stopCh)
	}
}

func (m *multiNamesapceIndexInformer) HasSynced() bool {
	fmt.Printf("has synced")
	for _, i := range m.indexInformers {
		if synced := i.HasSynced(); !synced {
			fmt.Printf("has synced - %v", synced)
			return synced
		}
	}
	fmt.Printf("has synced- true")
	return true
}

func (m *multiNamesapceIndexInformer) LastSyncResourceVersion() string {
	return ""
}
