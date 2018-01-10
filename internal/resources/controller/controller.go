package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

func indexByIDKeyFun() cache.Indexers {
	return cache.Indexers{
		"byID": func(obj interface{}) ([]string, error) {
			meta, err := meta.Accessor(obj)
			if err != nil {
				return []string{""}, fmt.Errorf("object has no meta: %v", err)
			}
			return []string{meta.GetName()}, nil
		},
	}
}
