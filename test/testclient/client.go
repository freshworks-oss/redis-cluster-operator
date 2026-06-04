package testclient

import (
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

func NewClient(config *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rediskunv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(config, httpClient)
	if err != nil {
		return nil, err
	}
	options := client.Options{
		Scheme: scheme,
		Mapper: mapper,
	}
	cli, err := client.New(config, options)
	if err != nil {
		return nil, err
	}
	return cli, nil
}
