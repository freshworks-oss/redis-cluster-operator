// Package integration contains Ginkgo integration tests (envtest) for heal and clustering flows.
// Exported helpers here are in non-test files so subpackages (e.g. test/integration/clustering)
// can import github.com/.../test/integration.
package integration

import (
	"context"
	"crypto/rand"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/metrics"
)

// K8sClient and Ctx are initialized by BeforeSuite in suite_test.go.
var (
	K8sClient     client.Client
	Ctx           context.Context
	MetricsClient metricsClientHolder
)

// metricsClientHolder carries a logger and metrics recorder for code paths that expect both.
type metricsClientHolder struct {
	logr.Logger
	metrics.Recorder
}

type noopMetricsRecorder struct{}

func (noopMetricsRecorder) SetClusterOK(namespace, name string, val float64) {}

func (noopMetricsRecorder) ReconcileClusterGauge(namespace, name, function string, val float64) {}

func (noopMetricsRecorder) RecordK8sOperation(namespace, kind, name, operation, status, err string) {}

func (noopMetricsRecorder) RecordRedisOperation(namespace, name, operation, ip, status, err string) {}

func (noopMetricsRecorder) ReconcileCluster(namespace, name, operation, function, status, err string) {
}

// RandomString returns a random alphanumeric string of length n (for unique namespaces, etc.).
func RandomString(n int) string {
	if n <= 0 {
		return ""
	}
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("integration.RandomString: crypto/rand.Read: %v", err))
	}
	for i := range b {
		b[i] = letters[int(buf[i])%len(letters)]
	}
	return string(b)
}

// CreateNamespace creates a Namespace object in the test cluster.
func CreateNamespace(name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	return K8sClient.Create(Ctx, ns)
}

// DeleteNamespace removes a Namespace from the test cluster.
func DeleteNamespace(name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	return K8sClient.Delete(Ctx, ns)
}

// CreateTestDRC returns a minimal valid DistributedRedisCluster for integration tests.
func CreateTestDRC(name, namespace string) *rediskunv1alpha1.DistributedRedisCluster {
	return &rediskunv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: rediskunv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			AdminSecret: &corev1.LocalObjectReference{
				Name: "test-admin-secret",
			},
			Image: "redis:7.0-alpine",
			Config: map[string]string{
				"maxmemory": "2gb",
			},
			ConfStorage: &rediskunv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  rediskunv1alpha1.Ephemeral,
				Class: "standard",
			},
			BackupStorage: &rediskunv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  rediskunv1alpha1.Ephemeral,
				Class: "standard",
			},
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}
}
