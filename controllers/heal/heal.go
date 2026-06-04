package heal

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/freshworks/redis-cluster-operator/metrics"
	"github.com/freshworks/redis-cluster-operator/utils/k8sutil"
)

type CheckAndHeal struct {
	Logger        logr.Logger
	PodControl    k8sutil.IPodControl
	Pods          []*corev1.Pod
	DryRun        bool
	MetricsClient metrics.Recorder
}

// NewCheckAndHeal builds a CheckAndHeal wired with Kubernetes pod control (e.g. integration / envtest).
func NewCheckAndHeal(c client.Client, log logr.Logger, m metrics.Recorder) *CheckAndHeal {
	return &CheckAndHeal{
		Logger:        log,
		PodControl:    k8sutil.NewPodController(c, m),
		Pods:          nil,
		DryRun:        false,
		MetricsClient: m,
	}
}
