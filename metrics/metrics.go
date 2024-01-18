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
	GET_CLUSTERADMIN_PASSWORD    = "GET_CLUSTERADMIN_PASSWORD"
	CREATE_ADMIN_CONN            = "CREATE_ADMIN_CONNECTION"
	FORGET_NODE                  = "FORGET NODE"
	NODE_NOT_FOUND               = "NODE_NOT_FOUND"
	UNUSED_GHOST_NODE            = "UNUSED/GHOST NODE"
	GEN_ACL_CONF                 = "GENERATE_ACL_CONF"
	SET_ACL                      = "SET_ACL"
	CLUSTER_MEET                 = "CLUSTER_MEET"
	GET_CLUSTER_NODES_ALL        = "GET_CLUSTER_NODES_ALL"
	REQUEUE                      = "REQUEUE"
	HEAL                         = "HEAL"
	CLUSTER_JOIN_FAILURE         = "CLUSTER_JOIN_FAILURE"
	REDIS_CONFIG_SET             = "CONFIG_SET"
	CLUSTERING                   = "SYNC_CLUSTER"
	DISPATCH_MASTER              = "DISPATCH_MASTERS"
	PLACE_SLAVE                  = "PLACE_SLAVES"
	ATTACH_SLAVE_TO_MASTER       = "ATTACH_SLAVES_TO_MASTER"
	SLOT_ALLOCATION              = "SLOT_ALLOCATION"
	REBALANCE_CLUSTER            = "REBALANCE_CLUSTER"
	ALLOT_SLOTS_TO_NEW_MASTERS   = "ALLOT_SLOTS_TO_NEW_MASTERS"
	SCALE_DOWN                   = "SCALE_DOWN"
	CHECK_AND_UPDATE_REDIS_PODS  = "CHECK_AND_UPDATE_REDIS_PODS"
)

type Recorder interface {
	// ClusterOK metrics
	SetClusterOK(namespace string, name string, val float64)
	RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string)
	EnsureCluster(namespace string, name string, status string, operation string, err string)
	RecordRedisOperation(namespace string, name string, operation string, ip string, status string, err string)
}

type recorder struct {
	clusterOK            *prometheus.GaugeVec   // clusterOk is the status of a cluster
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
		k8sServiceOperations: k8sServiceOperations,
		ensureCluster:        ensureCluster,
		recordRedisOperation: recordRedisOperation,
	}

	// Register metrics.
	reg.MustRegister(
		r.clusterOK,
		r.k8sServiceOperations,
		r.ensureCluster,
		r.recordRedisOperation,
	)
	return r
}

func (r recorder) SetClusterOK(namespace string, name string, val float64) {
	r.clusterOK.WithLabelValues(namespace, name).Set(val)
}

func (r recorder) RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string) {
	r.k8sServiceOperations.WithLabelValues(namespace, kind, name, operation, status, err).Add(1)
}

func (r recorder) EnsureCluster(namespace string, name string, status string, operation string, err string) {
	r.ensureCluster.WithLabelValues(namespace, name, status, operation, err).Add(1)
}

func (r recorder) RecordRedisOperation(namespace string, name string, operation string, ip string, status string, err string) {
	r.recordRedisOperation.WithLabelValues(namespace, name, operation, ip, status, err).Add(1)
}
