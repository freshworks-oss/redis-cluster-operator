package distributedrediscluster

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	config "github.com/dinesh-murugiah/rediscluster-operator/redisconfig"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/k8sutil"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

var (
	defaultLabels = map[string]string{
		redisv1alpha1.LabelManagedByKey: redisv1alpha1.OperatorName,
	}
)

func getLabels(cluster *redisv1alpha1.DistributedRedisCluster) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelClusterName: cluster.Name,
	}
	return utils.MergeLabels(defaultLabels, dynLabels, cluster.Labels)
}

// Helper functin to upadte the cluster info with additional information from other sources
func UpdateClusterInfos(clusterInfos *redisutil.ClusterInfos, pods []*corev1.Pod, Client client.Client, reqLogger logr.Logger, metrics metrics.Recorder) error {

	nodectrl := k8sutil.NewNodeController(Client, metrics)
	nodes := clusterInfos.GetNodes()
	for _, node := range nodes {
		if len(node.FailStatus) > 0 {
			reqLogger.Info("UpdateClusterInfos", "Skipping Node Update for Failed Node", node.ID, "FailStatus", node.FailStatus)
			continue
		}
		pod := getpodFromlist(pods, node.IP)
		if pod == nil {
			return fmt.Errorf("pod not found for node %s", node.IP)
		}
		node.NodeName = pod.Spec.NodeName
		if len(pod.OwnerReferences) > 0 {
			if pod.OwnerReferences[0].Kind == "StatefulSet" {
				node.StatefulSet = pod.OwnerReferences[0].Name
			}
		}
		node.PodName = pod.Name
		znode, err := nodectrl.GetNode(pod.Spec.NodeName)
		if err == nil {
			node.Zonename = nodectrl.GetZoneLabel(znode)
			//reqLogger.Info("UpdateClusterInfos", "Zone label found", node.Zonename)
		} else {
			reqLogger.Error(err, "GetNode Returned Error", "context", "UpdateClusterInfos", "action", "setting zonename to unknown")
			node.Zonename = "unknown"
		}
	}
	return nil
}

func getpodFromlist(pods []*corev1.Pod, nodeIP string) *corev1.Pod {
	for _, pod := range pods {
		if pod.Status.PodIP == nodeIP {
			return pod
		}
	}
	return nil

}

// newRedisAdmin builds and returns new redis.Admin from the list of pods
func newRedisAdmin(pods []*corev1.Pod, password string, cfg *config.Redis, reqLogger logr.Logger, ctx context.Context, metrics metrics.Recorder) (redisutil.IAdmin, error) {
	nodesAddrs := []string{}
	for _, pod := range pods {
		redisPort := redisutil.DefaultRedisPort
		for _, container := range pod.Spec.Containers {
			if container.Name == "redis" {
				for _, port := range container.Ports {
					if port.Name == "client" {
						redisPort = fmt.Sprintf("%d", port.ContainerPort)
					}
				}
			}
		}
		reqLogger.V(4).Info("append redis admin addr", "addr", pod.Status.PodIP, "port", redisPort)
		nodesAddrs = append(nodesAddrs, net.JoinHostPort(pod.Status.PodIP, redisPort))
	}
	adminConfig := redisutil.AdminOptions{
		ConnectionTimeout:  time.Duration(cfg.DialTimeout) * time.Millisecond,
		RenameCommandsFile: cfg.GetRenameCommandsFile(),
		Password:           password,
	}

	return redisutil.NewAdmin(nodesAddrs, &adminConfig, reqLogger, ctx, metrics), nil
}

func newRedisCluster(infos *redisutil.ClusterInfos, cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) (*redisutil.Cluster, redisutil.Nodes, error) {
	// now we can trigger the rebalance
	nodes := infos.GetNodes()

	// build redis cluster vision
	rCluster := &redisutil.Cluster{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
		Nodes:     make(map[string]*redisutil.Node),
		HaStatus:  cluster.Status.HAStatus,
	}

	for _, node := range nodes {
		rCluster.Nodes[node.ID] = node

		//reqLogger.Info("newRedisCluster", "STS", node.StatefulSet, "NodeName", node.NodeName, "Zone", node.Zonename)

		if cluster.Spec.HaConfig != nil && cluster.Spec.HaConfig.HaEnabled && (node.Zonename == "" || node.Zonename == "unknown") {
			err1 := fmt.Errorf("zone label not found in node %s", node.NodeName)
			reqLogger.Error(err1, "Unable to find Zone label", "err", err1)
			return nil, nil, err1
		}
	}
	return rCluster, nodes, nil
}

func clusterPods(pods []corev1.Pod) []*corev1.Pod {
	var podSlice []*corev1.Pod
	for _, pod := range pods {
		// Only work with running pods
		if pod.Status.Phase == corev1.PodRunning {
			podPointer := pod
			podSlice = append(podSlice, &podPointer)
		}
	}
	return podSlice
}

func proxyPods(syncContext *syncContext, r *DistributedRedisClusterReconciler, request ctrl.Request) error {
	if syncContext.isSingleAzDeployment { //move this to new function called proxyPods
		proxyPods, err := r.StatefulSetController.GetStatefulSetPodsByLabels(request.Namespace, getProxyLabels(request.Namespace))
		if err != nil {
			return err
		}
		syncContext.proxyPods = clusterPods(proxyPods.Items)
	}
	return nil
}

func checkProxyPodsInSameAz(syncContext *syncContext, r *DistributedRedisClusterReconciler) (string, error) {
	if len(syncContext.proxyPods) == 0 {
		return "", fmt.Errorf("proxy pods are empty")
	}

	nodectrl := k8sutil.NewNodeController(r.Client, r.MetricsClient)
	azSet := make(map[string][]string) // map of AZ to pod names

	// Collect AZs for all pods
	for _, pod := range syncContext.proxyPods {
		zoneNode, err := nodectrl.GetNode(pod.Spec.NodeName)
		if err != nil {
			return "", fmt.Errorf("unable to find node %s: %v", pod.Spec.NodeName, err)
		}

		az := nodectrl.GetZoneLabel(zoneNode)
		if az == "unknown" {
			return "", fmt.Errorf("waiting for AZ information for pod %s on node %s", pod.Name, pod.Spec.NodeName)
		}

		azSet[az] = append(azSet[az], pod.Name)
	}

	// If we have more than one AZ, throw an error
	if len(azSet) > 1 {
		return "", fmt.Errorf("proxy pods are spread across multiple availability zones, cannot proceed as this is singleAz deployment")
	}

	for az := range azSet {
		return az, nil
	}

	return "", nil
}

func findProxyAz(r *DistributedRedisClusterReconciler, syncCtx *syncContext) (string, error) {
	if syncCtx.isSingleAzDeployment {
		err, az := checkProxyPodsInSameAz(syncCtx, r)
		return err, az
	}
	return "", nil
}

func verifyIfSingleAzDeployment(instance *redisv1alpha1.DistributedRedisCluster) bool {
	if instance.Spec.HaConfig == nil {
		// HaConfig is not defined, which is normal for non-HA deployments
		return false
	}

	if instance.Spec.HaConfig.HaEnabled && instance.Spec.HaConfig.ZonesInfo != nil {
		// Check if value exists as value "RedisMasterSingleAz" in ZonesInfo.
		// If Yes, get the AZ information of the Proxy pods of this namespace.
		for _, value := range instance.Spec.HaConfig.ZonesInfo {
			if value == "RedisMasterSingleAz" {
				return true
			}
		}
	}
	return false
}

func needClusterOperation(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) bool {
	if utils.CompareIntValue("NumberOfMaster", &cluster.Status.NumberOfMaster, &cluster.Spec.MasterSize, reqLogger) {
		reqLogger.V(4).Info("needClusterOperation---NumberOfMaster")
		return true
	}

	if utils.CompareIntValue("MinReplicationFactor", &cluster.Status.MinReplicationFactor, &cluster.Spec.ClusterReplicas, reqLogger) {
		reqLogger.V(4).Info("needClusterOperation---MinReplicationFactor")
		return true
	}

	if utils.CompareIntValue("MaxReplicationFactor", &cluster.Status.MaxReplicationFactor, &cluster.Spec.ClusterReplicas, reqLogger) {
		reqLogger.V(4).Info("needClusterOperation---MaxReplicationFactor")
		return true
	}

	return false
}

type IWaitHandle interface {
	Name() string
	Tick() time.Duration
	Timeout() time.Duration
	Handler() error
}

// waiting will keep trying to handler.Handler() until either
// we get a result from handler.Handler() or the timeout expires
func waiting(handler IWaitHandle, reqLogger logr.Logger) error {
	timeout := time.After(handler.Timeout())
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return fmt.Errorf("%s timed out", handler.Name())
		// Got a tick, we should check on Handler()
		case <-tick.C:
			err := handler.Handler()
			if err == nil {
				return nil
			}
			reqLogger.V(4).Info(err.Error())
		}
	}
}

type waitPodTerminating struct {
	name                  string
	statefulSet           string
	timeout               time.Duration
	tick                  time.Duration
	statefulSetController k8sutil.IStatefulSetControl
	cluster               *redisv1alpha1.DistributedRedisCluster
}

func (w *waitPodTerminating) Name() string {
	return w.name
}

func (w *waitPodTerminating) Tick() time.Duration {
	return w.tick
}

func (w *waitPodTerminating) Timeout() time.Duration {
	return w.timeout
}

func (w *waitPodTerminating) Handler() error {
	labels := getLabels(w.cluster)
	labels[redisv1alpha1.StatefulSetLabel] = w.statefulSet
	podList, err := w.statefulSetController.GetStatefulSetPodsByLabels(w.cluster.Namespace, labels)
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return fmt.Errorf("[%s] pod still runing", pod.Name)
		}
	}
	return nil
}

type waitStatefulSetUpdating struct {
	name                  string
	timeout               time.Duration
	tick                  time.Duration
	statefulSetController k8sutil.IStatefulSetControl
	cluster               *redisv1alpha1.DistributedRedisCluster
}

func (w *waitStatefulSetUpdating) Name() string {
	return w.name
}

func (w *waitStatefulSetUpdating) Tick() time.Duration {
	return w.tick
}

func (w *waitStatefulSetUpdating) Timeout() time.Duration {
	return w.timeout
}

func (w *waitStatefulSetUpdating) Handler() error {
	labels := getLabels(w.cluster)
	stsList, err := w.statefulSetController.ListStatefulSetByLabels(w.cluster.Namespace, labels)
	if err != nil {
		return err
	}
	for _, sts := range stsList.Items {
		if sts.Status.ReadyReplicas != (w.cluster.Spec.ClusterReplicas + 1) {
			return nil
		}
	}
	return fmt.Errorf("statefulSet still not updated")
}
