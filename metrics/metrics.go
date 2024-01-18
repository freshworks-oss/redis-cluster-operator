package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// variables for setting various indicator labels
const (
	promControllerSubsystemRedis = "rediscontroller"
	promControllerSubsystemUtil  = "utilcontroller"
	promController               = "maincontroller"
	K8S_FORBIDDEN_ERR            = "USER_FORBIDDEN_TO_PERFORM_ACTION"
	K8S_UNAUTH                   = "CLIENT_NOT_AUTHORISED"
	K8S_MISC                     = "MISC_ERROR_CHECK_LOGS"
	K8S_NOT_FOUND                = "RESOURCE_NOT_FOUND"
	SUCCESS                      = "SUCCESS"
	FAIL                         = "FAIL"
	NOT_APPLICABLE               = "NA"
	MISC                         = "MISC_ERROR"
)

type Recorder interface {
	// ClusterOK metrics
	SetClusterOK(namespace string, name string)
	SetControllerError(component string, name string)
	SetWebhookError(component string, name string)
	SetControllerSuccess(component string, name string)
	//Actual metrics starts from here
	RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string)
	EnsureCluster(namespace string, name string, status string, operation string, err string)
	RecordRedisOperation(namespace string, name string, operation string, ip string, status string, err string)
}

type recorder struct {
	clusterOK         *prometheus.GaugeVec // clusterOk is the status of a cluster
	contollerError    *prometheus.GaugeVec // Controller instance creation failure
	webhookError      *prometheus.GaugeVec // Webhook creation failure for rediscontroller
	controllerSuccess *prometheus.GaugeVec // test
	//Actual metrics starts from here
	k8sServiceOperations *prometheus.CounterVec // number of operations performed on k8s
	ensureCluster        *prometheus.CounterVec // status of ensure cluster
	recordRedisOperation *prometheus.CounterVec // record redis operation
}

func NewRecorder(namespace string, reg prometheus.Registerer) Recorder {
	clusterOK := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: promControllerSubsystemRedis,
		Name:      "cluster_ok",
		Help:      "Number of clusters managed by the operator.",
	}, []string{"namespace", "name"})

	contollerError := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: promController,
		Name:      "contoller_error",
		Help:      "Controller instance creation failure.",
	}, []string{"component", "name"})

	webhookError := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: promController,
		Name:      "webhook_error",
		Help:      "webhook instance creation failure.",
	}, []string{"component", "name"})

	controllerSuccess := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: promController,
		Name:      "contoller_success",
		Help:      "controller creation success.",
	}, []string{"component", "name"})

	k8sServiceOperations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "k8s_operations_total",
			Help:      "number of operations performed on k8s",
		}, []string{"namespace", "kind", "name", "operation", "status", "err"})

	ensureCluster := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "ensure_cluster",
			Help:      "ensure cluster status",
		}, []string{"namespace", "name", "status", "operation", "err"})
	recordRedisOperation := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "redis_operation",
			Help:      "redis operation",
		}, []string{"namespace", "name", "operation", "ip", "status", "err"})

	r := recorder{
		clusterOK:            clusterOK,
		contollerError:       contollerError,
		webhookError:         webhookError,
		controllerSuccess:    controllerSuccess,
		k8sServiceOperations: k8sServiceOperations,
		ensureCluster:        ensureCluster,
		recordRedisOperation: recordRedisOperation,
	}

	// Register metrics.
	reg.MustRegister(
		r.clusterOK,
		r.contollerError,
		r.webhookError,
		r.controllerSuccess,
		r.k8sServiceOperations,
		r.ensureCluster,
		r.recordRedisOperation,
	)
	return r
}

func (r recorder) SetClusterOK(namespace string, name string) {
	r.clusterOK.WithLabelValues(namespace, name).Set(1)
}

func (r recorder) SetControllerError(component string, name string) {
	r.contollerError.WithLabelValues(component, name).Set(1)
}

func (r recorder) SetWebhookError(component string, name string) {
	r.webhookError.WithLabelValues(component, name).Set(1)
}

func (r recorder) SetControllerSuccess(component string, name string) {
	r.controllerSuccess.WithLabelValues(component, name).Set(1)
}

func (r recorder) RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string) {
	r.k8sServiceOperations.WithLabelValues(namespace, kind, name, operation, status, err).Add(1)
}

func (r recorder) EnsureCluster(namespace string, name string, status string, operation string, err string) {
	r.ensureCluster.WithLabelValues(namespace, name, status, operation, err).Add(1)
}

func (r recorder) RecordRedisOperation(namespace string, name string, operation string, ip string, status string, err string) {
	r.ensureCluster.WithLabelValues(namespace, name, operation, ip, status, err).Add(1)
}
