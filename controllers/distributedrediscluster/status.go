package distributedrediscluster

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/clustering"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/configmaps"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/k8sutil"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

func SetClusterFailed(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusKO
	status.Reason = reason
}

func SetClusterOK(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusOK
	status.Reason = reason
}

func SetClusterNOK(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusNOK
	status.Reason = reason
}

func SetClusterRebalancing(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusRebalancing
	status.Reason = reason
}

func SetClusterScaling(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusScaling
	status.Reason = reason
}

func SetClusterUpdating(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusOnDelete
	status.Reason = reason
}

func SetSecretStatus(status *redisv1alpha1.DistributedRedisClusterStatus, secretstate string) {
	status.SecretStatus = secretstate
}

func SetHAStatus(status *redisv1alpha1.DistributedRedisClusterStatus, HAstate redisv1alpha1.HaStatus) {
	status.HAStatus = HAstate
}

func SetClusterResetPassword(status *redisv1alpha1.DistributedRedisClusterStatus, reason string) {
	status.Status = redisv1alpha1.ClusterStatusResetPassword
	status.Reason = reason
}

func CheackandUpdateClusterHealth(clusterStatus *redisv1alpha1.DistributedRedisClusterStatus, clusterInfos *redisutil.ClusterInfos, cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) bool {

	var failreason []string
	var foundReason bool = false

	if clusterInfos.Status == redisutil.ClusterInfosInconsistent {
		SetClusterNOK(clusterStatus, "Cluster view is inconsistent")
		return false
	}

	if clusterStatus.MinReplicationFactor < cluster.Spec.ClusterReplicas {
		SetClusterNOK(clusterStatus, "Cluster Replication Factor is not met")
		return false
	}

	if clusterInfos.ClusterState == "NOK" && len(clusterInfos.ClusterNokReason) > 0 {
		for key, value := range clusterInfos.ClusterNokReason {
			reqLogger.Info("newClusterInfos - cluster not healthy", "NodeIP", key, "Reason", value)
			if len(failreason) == 0 {
				failreason = append(failreason, value[strings.LastIndex(value, "-")+1:])
			} else {
				foundReason = false
				for _, reason := range failreason {
					if reason == value[strings.LastIndex(value, "-")+1:] {
						foundReason = true
					}
				}
				if !foundReason {
					failreason = append(failreason, value[strings.LastIndex(value, "-")+1:])
				}
			}
		}
		SetClusterNOK(clusterStatus, strings.Join(failreason, ","))
		return false
	}
	SetClusterOK(clusterStatus, "Cluster is healthy")
	return true
}

func CheckallZonesActive(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger, Client client.Client, metrics metrics.Recorder) (bool, int32, error) {

	var activeZones int32 = 0
	var activeZonenames []string = make([]string, 0, len(cluster.Spec.HaConfig.ZonesInfo))
	nodectrl := k8sutil.NewNodeController(Client, metrics)
	for zone, _ := range cluster.Spec.HaConfig.ZonesInfo {
		zoneactive, err := nodectrl.CheckZoneAvailable(zone)
		if err != nil {
			reqLogger.Error(err, "unable to check zone availability for", "zone", zone)
			return false, 0, err
		}
		if zoneactive {
			activeZones = activeZones + 1
			activeZonenames = append(activeZonenames, zone)
		}
	}
	if activeZones == int32(len(cluster.Spec.HaConfig.ZonesInfo)) {
		reqLogger.Info("CheckallZonesActive - All Zones Active", "Active Zones:", activeZonenames)
		return true, activeZones, nil
	} else {
		reqLogger.Info("CheckallZonesActive - Not all Zones Active", "Active Zones:", activeZonenames)
		return false, activeZones, nil
	}
}

func buildHAStatus(clusterInfos *redisutil.ClusterInfos, cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) (*clustering.RankingInfo, redisv1alpha1.HaStatus, redisutil.Nodes, error) {

	var haPlacedMasterCount int32 = 0
	var misplacedMasters redisutil.Nodes = make([]*redisutil.Node, 0)

	haRanking := clustering.CreateRankingFromClusterInfo(cluster.Spec.HaConfig, clusterInfos, cluster.Spec.MasterSize, cluster.Spec.ClusterReplicas, cluster.Name, reqLogger)
	haRanking.SetCurrentMastersZoneRank()

	zones := make([]string, 0, len(cluster.Spec.HaConfig.ZonesInfo))
	for zone := range cluster.Spec.HaConfig.ZonesInfo {
		zones = append(zones, zone)
	}
	sort.Strings(zones)

	masterNodes, err := clusterInfos.GetNodes().GetNodesByFunc(func(node *redisutil.Node) bool {
		return (node.Role == redisutil.RedisMasterRole) && (node.TotalSlots() > 0)
	})

	if err != nil || len(masterNodes) != int(cluster.Spec.MasterSize) {
		reqLogger.Error(err, "unable to retrieve sufficient redis node with the role master")
		return haRanking, redisv1alpha1.HaStatusFailed, nil, err
	}

	for _, node := range masterNodes {
		if clusterInfos.ProxyPodAz == "" {
			if node.ZoneDistributionRank >= clustering.HaplacedMasterRankThreshold {
				haPlacedMasterCount++
			} else {
				misplacedMasters = append(misplacedMasters, node)
			}
		} else {
			if node.Zonename == clusterInfos.ProxyPodAz {
				haPlacedMasterCount++
			} else {
				misplacedMasters = append(misplacedMasters, node)
			}
		}
	}

	if haPlacedMasterCount != cluster.Spec.MasterSize {
		err := fmt.Errorf("master placement not suitable for cluster ha")
		for _, node := range misplacedMasters {
			reqLogger.Info("buildHAStatus", "misplaced master", node.PodName, "misplaced zone", node.Zonename)
		}
		return haRanking, redisv1alpha1.HaStatusRedistribute, misplacedMasters, err
	}
	return haRanking, redisv1alpha1.HaStatusHealthy, nil, nil

}

func buildClusterStatus(clusterInfos *redisutil.ClusterInfos, pods []*corev1.Pod,
	cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) *redisv1alpha1.DistributedRedisClusterStatus {
	oldStatus := cluster.Status
	status := &redisv1alpha1.DistributedRedisClusterStatus{
		Status:       oldStatus.Status,
		Reason:       oldStatus.Reason,
		SecretStatus: oldStatus.SecretStatus,
		SecretsVer:   oldStatus.SecretsVer,
		Restore:      oldStatus.Restore,
		HAStatus:     oldStatus.HAStatus,
	}
	nbMaster := int32(0)
	nbSlaveByMaster := map[string]int{}

	for _, pod := range pods {
		redisNodes, err := clusterInfos.GetNodes().GetNodesByFunc(func(node *redisutil.Node) bool {
			return node.IP == pod.Status.PodIP
		})
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("unable to retrieve the associated redis node with the pod: %s, ip:%s", pod.Name, pod.Status.PodIP))
			continue
		}
		if len(redisNodes) == 1 {
			redisNode := redisNodes[0]

			newNode := redisv1alpha1.RedisClusterNode{
				PodName:  pod.Name,
				NodeName: pod.Spec.NodeName,
				IP:       pod.Status.PodIP,
				Slots:    []string{},
			}

			newNode.Zonename = redisNode.Zonename
			if len(pod.OwnerReferences) > 0 {
				if pod.OwnerReferences[0].Kind == "StatefulSet" {
					newNode.StatefulSet = pod.OwnerReferences[0].Name
				}
			}
			if redisutil.IsMasterWithSlot(redisNode) {
				if _, ok := nbSlaveByMaster[redisNode.ID]; !ok {
					nbSlaveByMaster[redisNode.ID] = 0
				}
				nbMaster++
			}

			newNode.ID = redisNode.ID
			newNode.Role = redisNode.GetRole()
			newNode.Port = redisNode.Port
			newNode.Slots = []string{}
			if redisutil.IsSlave(redisNode) && redisNode.MasterReferent != "" {
				nbSlaveByMaster[redisNode.MasterReferent] = nbSlaveByMaster[redisNode.MasterReferent] + 1
				newNode.MasterRef = redisNode.MasterReferent
			}
			if len(redisNode.Slots) > 0 {
				slots := redisutil.SlotRangesFromSlots(redisNode.Slots)
				for _, slot := range slots {
					newNode.Slots = append(newNode.Slots, slot.String())
				}
			}
			status.Nodes = append(status.Nodes, newNode)
		} else {
			err1 := fmt.Errorf("multiple nodes with same podip")
			reqLogger.Error(err1, fmt.Sprintf("multiple nodes in cluster info has same podip:%s", pod.Status.PodIP))
			continue
		}
	}
	status.NumberOfMaster = nbMaster

	minReplicationFactor := math.MaxInt32
	maxReplicationFactor := 0
	for _, counter := range nbSlaveByMaster {
		if counter > maxReplicationFactor {
			maxReplicationFactor = counter
		}
		if counter < minReplicationFactor {
			minReplicationFactor = counter
		}
	}
	if len(nbSlaveByMaster) == 0 {
		minReplicationFactor = 0
	}
	status.MaxReplicationFactor = int32(maxReplicationFactor)
	status.MinReplicationFactor = int32(minReplicationFactor)

	return status
}

func (r *DistributedRedisClusterReconciler) updateClusterIfNeed(cluster *redisv1alpha1.DistributedRedisCluster,
	newStatus *redisv1alpha1.DistributedRedisClusterStatus,
	reqLogger logr.Logger) {
	if compareStatus(&cluster.Status, newStatus, reqLogger) {
		reqLogger.WithValues("namespace", cluster.Namespace, "name", cluster.Name).
			V(3).Info("status changed")
		cluster.Status = *newStatus
		r.CrController.UpdateCRStatus(cluster)
	}
}

func compareStatus(old, new *redisv1alpha1.DistributedRedisClusterStatus, reqLogger logr.Logger) bool {
	if utils.CompareStringValue("ClusterStatus", string(old.Status), string(new.Status), reqLogger) {
		return true
	}

	if utils.CompareStringValue("HaStatus", string(old.HAStatus), string(new.HAStatus), reqLogger) {
		return true
	}

	if utils.CompareStringValue("SecretStatus", string(old.SecretStatus), string(new.SecretStatus), reqLogger) {
		return true
	}

	if old.SecretsVer == nil {
		return true
	} else if len(old.SecretsVer) == 0 {
		return true
	}

	if utils.CompareStringValue("ClusterStatusReason", old.Reason, new.Reason, reqLogger) {
		return true
	}

	if utils.CompareInt32("NumberOfMaster", old.NumberOfMaster, new.NumberOfMaster, reqLogger) {
		return true
	}

	if utils.CompareInt32("len(Nodes)", int32(len(old.Nodes)), int32(len(new.Nodes)), reqLogger) {
		return true
	}

	if utils.CompareStringValue("restoreSucceeded", string(old.Restore.Phase), string(new.Restore.Phase), reqLogger) {
		return true
	}

	for _, nodeA := range old.Nodes {
		found := false
		for _, nodeB := range new.Nodes {
			if nodeA.ID == nodeB.ID {
				found = true
				if compareNodes(&nodeA, &nodeB, reqLogger) {
					return true
				}
			}
		}
		if !found {
			return true
		}
	}

	return false
}

func compareNodes(nodeA, nodeB *redisv1alpha1.RedisClusterNode, reqLogger logr.Logger) bool {
	if utils.CompareStringValue("Node.IP", nodeA.IP, nodeB.IP, reqLogger) {
		return true
	}
	if utils.CompareStringValue("Node.MasterRef", nodeA.MasterRef, nodeB.MasterRef, reqLogger) {
		return true
	}
	if utils.CompareStringValue("Node.PodName", nodeA.PodName, nodeB.PodName, reqLogger) {
		return true
	}
	if utils.CompareStringValue("Node.Port", nodeA.Port, nodeB.Port, reqLogger) {
		return true
	}
	if utils.CompareStringValue("Node.Role", string(nodeA.Role), string(nodeB.Role), reqLogger) {
		return true
	}

	sizeSlotsA := 0
	sizeSlotsB := 0
	if nodeA.Slots != nil {
		sizeSlotsA = len(nodeA.Slots)
	}
	if nodeB.Slots != nil {
		sizeSlotsB = len(nodeB.Slots)
	}
	if sizeSlotsA != sizeSlotsB {
		reqLogger.V(4).Info(fmt.Sprintf("compare Node.Slote size: %d - %d", sizeSlotsA, sizeSlotsB))
		return true
	}

	if (sizeSlotsA != 0) && !reflect.DeepEqual(nodeA.Slots, nodeB.Slots) {
		reqLogger.V(4).Info(fmt.Sprintf("compare Node.Slote deepEqual: %v - %v", nodeA.Slots, nodeB.Slots))
		return true
	}

	return false
}

func UpdateHaStatus(instance *redisv1alpha1.DistributedRedisCluster, newStatus *redisv1alpha1.DistributedRedisClusterStatus, hastatus string, stsStatus redisv1alpha1.StsEnsureStatus) {

	if !instance.Spec.HaConfig.HaEnabled {
		SetHAStatus(newStatus, redisv1alpha1.HaStatusDisabled)
		return
	}
	if hastatus != "" {
		SetHAStatus(newStatus, redisv1alpha1.HaStatus(hastatus))
		return
	}
	if stsStatus != "" {
		if stsStatus == redisv1alpha1.STSCreated || stsStatus == redisv1alpha1.STSCreateUpadate {
			SetHAStatus(newStatus, redisv1alpha1.HaStatusCreating)
		} else if stsStatus == redisv1alpha1.STSUpdated {
			SetHAStatus(newStatus, redisv1alpha1.HaStatusUpdating)
		} else {
			if newStatus.HAStatus == "" {
				SetHAStatus(newStatus, redisv1alpha1.HaStatusCreating)
			}

		}
	}

}
func UpdateSecretsStatus(instance *redisv1alpha1.DistributedRedisCluster, newStatus *redisv1alpha1.DistributedRedisClusterStatus, client client.Client, reqLogger logr.Logger) error {
	if instance.Status.SecretStatus == "" {
		secretVersions, err := configmaps.Getsecretversions(reqLogger, client, instance)
		if err != nil {
			SetSecretStatus(newStatus, "aclconferror")
			return err
		} else {
			SetSecretStatus(newStatus, "aclconfdone")
			newStatus.SecretsVer = make(map[string]string)
			for key, value := range secretVersions {
				newStatus.SecretsVer[key] = value
			}
		}
	}
	return nil
}
