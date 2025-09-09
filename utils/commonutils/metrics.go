// add required imports
package utils

import (
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"k8s.io/apimachinery/pkg/api/errors"
)

func RecordK8sMetrics(namespace string, kind string, object string, operation string, err error, metricsRecorder metrics.Recorder) {
	if err == nil {
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

func RecordRedisMetrics(namespace string, name string, operation string, ip string, err error, metricsRecorder metrics.Recorder) {
	if metricsRecorder == nil {
		return // Skip metrics recording if recorder is nil
	}
	if err == nil {
		metricsRecorder.RecordRedisOperation(namespace, name, operation, ip, metrics.SUCCESS, "nil")
	} else {
		metricsRecorder.RecordRedisOperation(namespace, name, operation, ip, metrics.FAIL, err.Error())
	}
}
