package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

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
	FATAL                        = "FATAL"
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
	GET_CLUSTER_INFOS            = "GET_CLUSTER_INFOS"
	REQUEUE                      = "REQUEUE"
	HEAL                         = "HEAL"
	CLUSTER_JOIN_FAILURE         = "CLUSTER_JOIN_FAILURE"
	REDIS_CONFIG_SET             = "CONFIG_SET"
	CLUSTERING                   = "SYNC_CLUSTER"
	SELFHEAL_CLUSTER             = "SELF_HEAL_CLUSTER"
	DISPATCH_MASTER              = "DISPATCH_MASTERS"
	PLACE_SLAVE                  = "PLACE_SLAVES"
	ATTACH_SLAVE_TO_MASTER       = "ATTACH_SLAVES_TO_MASTER"
	SLOT_ALLOCATION              = "SLOT_ALLOCATION"
	REBALANCE_CLUSTER            = "REBALANCE_CLUSTER"
	ALLOT_SLOTS_TO_NEW_MASTERS   = "ALLOT_SLOTS_TO_NEW_MASTERS"
	SCALE_DOWN                   = "SCALE_DOWN"
	CHECK_AND_UPDATE_REDIS_PODS  = "CHECK_AND_UPDATE_REDIS_PODS"
	metricsGCIntervalMinutes     = 5
)

var ( // used for grabage collection of metrics
	mutex                     sync.Mutex
	recorders                 = []recorder{}
	instanceMetricLastUpdated = map[string]time.Time{}
	resourceMetricLastUpdated = map[string]time.Time{}
)

func init() {
	go removeStaleMetrics()
}

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
	updateResourceMetricLastUpdatedTracker(namespace, kind, name)
}

func (r recorder) EnsureCluster(namespace string, name string, status string, operation string, err string) {
	r.ensureCluster.WithLabelValues(namespace, name, status, operation, err).Add(1)
}

func (r recorder) RecordRedisOperation(namespace string, name string, operation string, IP string, status string, err string) {
	r.recordRedisOperation.WithLabelValues(namespace, name, operation, IP, status, err).Add(1)
	updateInstanceMetricLastUpdatedTracker(IP)
}

func updateResourceMetricLastUpdatedTracker(namespace string, kind string, name string) {
	mutex.Lock()
	resourceMetricLastUpdated[fmt.Sprintf("%v/%v/%v", namespace, kind, name)] = time.Now()
	mutex.Unlock()
}

func updateInstanceMetricLastUpdatedTracker(IP string) {
	mutex.Lock()
	instanceMetricLastUpdated[IP] = time.Now()
	mutex.Unlock()
}

// Garbage collection
func removeStaleMetrics() {
	// Runs every `metricsGCIntervalMinutes`. It keeps track of recently updated metrics
	// And every metric that was not updated after `metricsGCIntervalMinutes` gets deleted
	for {
		metricsDeletedCount := 0
		kubernetesResourceBasedLabels, customResourceBasedLabels, ipBasedLabels := getLabelsOfStaleMetrics()
		for _, recorder := range recorders {
			for _, label := range kubernetesResourceBasedLabels {
				metricsDeletedCount += recorder.ensureCluster.DeletePartialMatch(label)
				metricsDeletedCount += recorder.k8sServiceOperations.DeletePartialMatch(label)
			}
			for _, label := range customResourceBasedLabels {
				labelWithName := label
				labelWithName["name"] = labelWithName["resource"]
				mutex.Lock()
				delete(labelWithName, "resource")
				mutex.Unlock()
				metricsDeletedCount += recorder.clusterOK.DeletePartialMatch(label)
			}
			for _, label := range ipBasedLabels {
				metricsDeletedCount += recorder.recordRedisOperation.DeletePartialMatch(label)
			}
		}
		time.Sleep(metricsGCIntervalMinutes * time.Minute)
	}
}

func getLabelsOfStaleMetrics() (kubernetesResourceBasedLabels []prometheus.Labels, customResourceBasedLabels []prometheus.Labels, ipBasedLabels []prometheus.Labels) {

	kubernetesResourceBasedLabels = []prometheus.Labels{}
	customResourceBasedLabels = []prometheus.Labels{}
	ipBasedLabels = []prometheus.Labels{}

	for key, value := range resourceMetricLastUpdated {
		// if the key is stale
		if value.Before(time.Now().Add(-metricsGCIntervalMinutes * time.Minute)) {
			// extract keys and create labels
			ids := strings.Split(key, "/")
			namespace := ids[0]
			kind := ids[1]
			resource := ids[2]
			kubernetesResourceBasedLabels = append(kubernetesResourceBasedLabels,
				prometheus.Labels{
					"namespace": namespace,
					"name":      resource,
					"kind":      kind,
				},
			)
			customResourceBasedLabels = append(customResourceBasedLabels,
				prometheus.Labels{
					"namespace": namespace,
					"resource":  resource,
				},
			)
			// once we have created labels out of the contents of the key,
			// its not longer required - since it is known to be stale. remove it from the tracker.
			mutex.Lock()
			delete(resourceMetricLastUpdated, key)
			mutex.Unlock()
		}
	}
	for IP, value := range instanceMetricLastUpdated {
		if value.Before(time.Now().Add(-metricsGCIntervalMinutes * time.Minute)) {
			ipBasedLabels = append(ipBasedLabels,
				prometheus.Labels{
					"IP": IP,
				},
			)
			// once we have created labels out of the contents of the key,
			// its not longer required - since it is known to be stale. remove it from the tracker.
			mutex.Lock()
			delete(instanceMetricLastUpdated, IP)
			mutex.Unlock()
		}

	}
	return kubernetesResourceBasedLabels, customResourceBasedLabels, ipBasedLabels
}
