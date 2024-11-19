package metrics

import (
	"fmt"
	"log"
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
	RECONCILE                    = "RECONCILE"
	ENSURE                       = "ENSURE"
	CHECK_AND_HEAL               = "CHECK_AND_HEAL"
	SYNC_CLUSTER                 = "SYNC_CLUSTER"
	HA_HEAL                      = "HA_HEAL"
	NOT_APPLICABLE               = "NA"
	GET_CLUSTERADMIN_PASSWORD    = "GET_CLUSTERADMIN_PASSWORD"
	CREATE_ADMIN_CONN            = "CREATE_ADMIN_CONNECTION"
	FORGET_NODE                  = "FORGET NODE"
	NODE_NOT_FOUND               = "NODE_NOT_FOUND"
	UNUSED_GHOST_NODE            = "UNUSED/GHOST NODE"
	GEN_ACL_CONF                 = "GENERATE_ACL_CONF"
	SET_ACL                      = "SET_ACL"
	CLUSTER_NODES                = "CLUSTER_NODES"
	CLUSTER_MEET                 = "CLUSTER_MEET"
	CLUSTER_INFO                 = "CLUSTER_INFO"
	CLUSTER_FAILOVER             = "CLUSTER_FAILOVER"
	CLUSTER_REPLICATE            = "CLUSTER_REPLICATE"
	CLUSTER_FORGET               = "CLUSTER_FORGET"
	CLUSTER_SETSLOT              = "CLUSTER_SETSLOT"
	CLUSTER_SET_CONFIG_EPOCH     = "CLUSTER_SET_CONFIG_EPOCH"
	CLUSTER_GET_KEYS_INSLOT      = "CLUSTER_GET_KEYS_INSLOT"
	CLUSTER_RESET                = "CLUSTER_RESET"
	CLUSTER_RESET_SOFT           = "CLUSTER_RESET_SOFT"
	FLUSHALL                     = "FLUSHALL"
	REDIS_CONFIG_GET             = "CONFIG_GET"
	REDIS_CONFIG_SET             = "CONFIG_SET"
	MIGRATE                      = "MIGRATE"
	GET_CLUSTER_NODES_ALL        = "GET_CLUSTER_NODES_ALL"
	GET_CLUSTER_INFOS            = "GET_CLUSTER_INFOS"
	REQUEUE                      = "REQUEUE"
	HEAL                         = "HEAL"
	CLUSTER_JOIN_FAILURE         = "CLUSTER_JOIN_FAILURE"
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

// MetricsTracker handles thread-safe tracking of metric updates
type MetricsTracker struct {
	mu              sync.RWMutex
	resourceMetrics map[string]time.Time
	instanceMetrics map[string]time.Time
}

// NewMetricsTracker creates a new MetricsTracker
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		resourceMetrics: make(map[string]time.Time),
		instanceMetrics: make(map[string]time.Time),
	}
}

type Recorder interface {
	// ClusterOK metrics
	SetClusterOK(namespace string, name string, val float64)
	ReconcileClusterGauge(namespace string, name string, function string, val float64)
	RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string)
	RecordRedisOperation(namespace string, name string, operation string, ip string, status string, err string)
	ReconcileCluster(namespace string, name string, operation string, function string, status string, err string)
}

type recorder struct {
	metricsTracker        *MetricsTracker
	clusterOK             *prometheus.GaugeVec   // clusterOk is the status of a cluster
	reconcileClusterGauge *prometheus.GaugeVec   // reconcile cluster gauge
	k8sServiceOperations  *prometheus.CounterVec // number of operations performed on k8s
	recordRedisOperation  *prometheus.CounterVec // record redis operation
	reconcileCluster      *prometheus.CounterVec // reconcile cluster
}

func NewRecorder(namespace string, reg prometheus.Registerer) Recorder {
	metricsTracker := NewMetricsTracker()
	clusterOK := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: promControllerSubsystemRedis,
		Name:      "cluster_ok",
		Help:      "Number of clusters managed by the operator.",
	}, []string{"namespace", "name"})

	reconcileClusterGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: promControllerSubsystemRedis,
		Name:      "reconcile_cluster_gauge",
		Help:      "Gauge metrices for different reconcile functions.",
	}, []string{"namespace", "name", "function"})

	k8sServiceOperations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "k8s_operations_total",
			Help:      "number of operations performed on k8s",
		}, []string{"namespace", "kind", "name", "operation", "status", "err"})

	recordRedisOperation := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "redis_operation",
			Help:      "redis operation",
		}, []string{"namespace", "name", "operation", "ip", "status", "err"})

	reconcileCluster := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystemRedis,
			Name:      "reconcile_cluster",
			Help:      "reconcile cluster",
		}, []string{"namespace", "name", "operation", "function", "status", "err"})

	r := recorder{
		metricsTracker:        metricsTracker,
		clusterOK:             clusterOK,
		k8sServiceOperations:  k8sServiceOperations,
		recordRedisOperation:  recordRedisOperation,
		reconcileCluster:      reconcileCluster,
		reconcileClusterGauge: reconcileClusterGauge,
	}

	// Register metrics.
	reg.MustRegister(
		r.clusterOK,
		r.k8sServiceOperations,
		r.recordRedisOperation,
		r.reconcileCluster,
		r.reconcileClusterGauge,
	)
	// Start the metrics garbage collector
	go r.removeStaleMetrics()
	return r
}

func (r recorder) SetClusterOK(namespace string, name string, val float64) {
	r.clusterOK.WithLabelValues(namespace, name).Set(val)
}

func (r recorder) ReconcileClusterGauge(namespace string, name string, function string, val float64) {
	r.reconcileClusterGauge.WithLabelValues(namespace, name, function).Set(val)
}

func (r recorder) RecordK8sOperation(namespace string, kind string, name string, operation string, status string, err string) {
	r.k8sServiceOperations.WithLabelValues(namespace, kind, name, operation, status, err).Add(1)
	r.metricsTracker.UpdateResourceMetric(namespace, kind, name)
}

func (r recorder) RecordRedisOperation(namespace string, name string, operation string, IP string, status string, err string) {
	r.recordRedisOperation.WithLabelValues(namespace, name, operation, IP, status, err).Add(1)
	r.metricsTracker.UpdateInstanceMetric(IP)
}

func (r recorder) ReconcileCluster(namespace string, name string, operation string, function string, status string, err string) {
	r.reconcileCluster.WithLabelValues(namespace, name, operation, function, status, err).Add(1)
}

// MetricsTracker methods
func (mt *MetricsTracker) UpdateResourceMetric(namespace, kind, name string) {
	key := fmt.Sprintf("%v/%v/%v", namespace, kind, name)
	mt.mu.Lock()
	mt.resourceMetrics[key] = time.Now()
	mt.mu.Unlock()
}

func (mt *MetricsTracker) UpdateInstanceMetric(IP string) {
	mt.mu.Lock()
	mt.instanceMetrics[IP] = time.Now()
	mt.mu.Unlock()
}

func (mt *MetricsTracker) GetStaleMetrics(gcInterval time.Duration) ([]prometheus.Labels, []prometheus.Labels, []prometheus.Labels) {
	var kubernetesResourceBasedLabels, customResourceBasedLabels, ipBasedLabels []prometheus.Labels
	staleTime := time.Now().Add(-gcInterval)

	mt.mu.Lock()
	defer mt.mu.Unlock()

	for key, lastUpdate := range mt.resourceMetrics {
		if lastUpdate.Before(staleTime) {
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
			delete(mt.resourceMetrics, key)
		}
	}

	for IP, lastUpdate := range mt.instanceMetrics {
		if lastUpdate.Before(staleTime) {
			ipBasedLabels = append(ipBasedLabels,
				prometheus.Labels{
					"IP": IP,
				},
			)
			delete(mt.instanceMetrics, IP)
		}
	}

	return kubernetesResourceBasedLabels, customResourceBasedLabels, ipBasedLabels
}

// Garbage collection
func (r *recorder) removeStaleMetrics() {
	// Runs every `metricsGCIntervalMinutes`. It keeps track of recently updated metrics
	// And every metric that was not updated after `metricsGCIntervalMinutes` gets deleted
	for {
		k, c, i := r.metricsTracker.GetStaleMetrics(metricsGCIntervalMinutes * time.Minute)
		metricsDeletedCount := 0

		for _, label := range k {
			metricsDeletedCount += r.k8sServiceOperations.DeletePartialMatch(label)
		}
		for _, label := range c {
			labelWithName := label
			labelWithName["name"] = labelWithName["resource"]
			metricsDeletedCount += r.clusterOK.DeletePartialMatch(label)
			metricsDeletedCount += r.reconcileClusterGauge.DeletePartialMatch(label)
			metricsDeletedCount += r.reconcileCluster.DeletePartialMatch(label)
		}
		for _, label := range i {
			metricsDeletedCount += r.recordRedisOperation.DeletePartialMatch(label)
		}
		log.Printf("Number of delted metrics %v", metricsDeletedCount)
		time.Sleep(metricsGCIntervalMinutes * time.Minute)
	}
}
