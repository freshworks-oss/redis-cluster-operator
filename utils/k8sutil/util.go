package k8sutil

import (
	"context"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsRequestRetryable(err error) bool {
	return errors.IsServiceUnavailable(err) ||
		errors.IsTimeout(err) ||
		errors.IsServerTimeout(err) ||
		errors.IsTooManyRequests(err)
}

func RecordMetrics(namespace string, kind string, object string, operation string, err error, metricsRecorder metrics.Recorder) {
	if nil == err {
		metricsRecorder.RecordK8sOperation(namespace, kind, object, operation, metrics.SUCCESS, metrics.NOT_APPLICABLE)
	} else if errors.IsForbidden(err) {
		metricsRecorder.RecordK8sOperation(namespace, kind, object, operation, metrics.FAIL, metrics.K8S_FORBIDDEN_ERR)
	} else if errors.IsUnauthorized(err) {
		metricsRecorder.RecordK8sOperation(namespace, kind, object, operation, metrics.FAIL, metrics.K8S_UNAUTH)
	} else if errors.IsNotFound(err) {
		metricsRecorder.RecordK8sOperation(namespace, kind, object, operation, metrics.FAIL, metrics.K8S_NOT_FOUND)
	} else {
		metricsRecorder.RecordK8sOperation(namespace, kind, object, operation, metrics.FAIL, metrics.K8S_MISC)
	}
}

func CreateSecret(client client.Client, secret *corev1.Secret, logger logr.Logger) error {
	ctx := context.TODO()
	s := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}, s)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.WithValues("Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name).
				Info("creating a new secret")
			return client.Create(ctx, secret)
		}
	}
	return err
}
