package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AnnotationScope annotation name for defining instance scope. Used for specifying cluster wide clusters.
	// A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide.
	AnnotationScope = "redis.kun/scope"
	//AnnotationClusterScoped annotation value for cluster wide clusters.
	AnnotationClusterScoped   = "cluster-scoped"
	AnnotationNamespaceScoped = "namespace-scoped"
)

var isClusterScoped = true
var namespaceList []string
var namespaceUnwatchList []string
var watchNamespaceEnvVar = "WATCH_NAMESPACE"
var unwatchNamespaceEnvVar = "UNWATCH_NAMESPACE"

func IsClusterScoped() bool {
	return isClusterScoped
}

func GetNamespacesList() ([]string, []string, error) {
	namespacesWatch, watchfound := os.LookupEnv(watchNamespaceEnvVar)
	namespacesunWatch, unwatchfound := os.LookupEnv(unwatchNamespaceEnvVar)

	var tempNamespaceList []string
	var tempNamespaceUnwatchList []string

	if !watchfound || !unwatchfound {
		return nil, nil, fmt.Errorf("%s & %s - must be set", watchNamespaceEnvVar, unwatchNamespaceEnvVar)
	}

	tempNamespaceList = strings.Split(namespacesWatch, ",")
	tempNamespaceUnwatchList = strings.Split(namespacesunWatch, ",")
	return tempNamespaceList, tempNamespaceUnwatchList, nil

}

func GenerateNamespaceList(log logr.Logger) error {
	var err error
	namespaceList, namespaceUnwatchList, err = GetNamespacesList()
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Watching namespaces: %v", namespaceList))
	log.Info(fmt.Sprintf("UnWatching namespaces: %v", namespaceUnwatchList))
	return nil
}

func GetNamespaceList() ([]string, []string) {
	return namespaceList, namespaceUnwatchList
}

func SetOperatorScope() {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found || ns == "" {
		isClusterScoped = true
	} else {
		isClusterScoped = false
	}
}

func ShoudManage(meta metav1.Object) bool {
	if isClusterScoped { //If cluster scoped, watch all namespaces except unwatch list
		for _, namespace := range namespaceUnwatchList {
			if meta.GetNamespace() == namespace {
				return false
			}
		}
		return true
	} else { //If namespace scoped, watch only namespaces in the watch list
		for _, namespace := range namespaceList {
			if meta.GetNamespace() == namespace {
				return true
			}
		}
		return false
	}
}
