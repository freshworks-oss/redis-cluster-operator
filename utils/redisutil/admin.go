package redisutil

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
)

const (
	// defaultHashMaxSlots higher value of slot
	// as slots start at 0, total number of slots is defaultHashMaxSlots+1
	DefaultHashMaxSlots = 16383

	// ResetHard HARD mode for RESET command
	ResetHard = "HARD"
	// ResetSoft SOFT mode for RESET command
	ResetSoft = "SOFT"
	//Timeout in seconds waiting for role change after failover
	RoleChangeTimeout = 20
	//Interval in seconds to check for role change after failover
	RoleChangeInterval = 1
)

// Redis Status Redis server status
type RedisReplStatus string
type RedisFailover string

type ReplicationInfo struct {
	Role                    string
	MasterLinkStatus        string
	MasterHost              string
	LastIoSecondsAgo        int64
	SlaveReplicationOffset  int64
	MasterReplicationOffset int64
	MasterSyncInProgress    bool
}

const (
	RedisFailoverForce    RedisFailover = "FORCE"
	RedisFailoverTakeover RedisFailover = "TAKEOVER"
	RedisFailoverDefault  RedisFailover = ""
)

const (
	// RedisSyncing Redis server syncing in in progress with master
	RedisSyncing           RedisReplStatus = "master_sync_in_progress:1"
	RedisMasterSillPending RedisReplStatus = "master_host:127.0.0.1"
	RedisLinkUp            RedisReplStatus = "master_link_status:up"
)
const (
	ReplicaStatusUnknown   = "ReplicaStatusunknown"
	ReplicaStatusReady     = "ReplicaSatusready"
	ReplicaLinkDown        = "MasterLinkDown"
	ReplicaStatusSyncing   = "ReplicaSyncing"
	ReplicaStatusMasterNil = "ReplicaMasterNil"
	ClusterStatusMigrating = "SlotsMigrating"
)

const (
	clusterKnownNodesREString = "cluster_known_nodes:([0-9]+)"
	forgotnodeUnknownREString = "ERR Unknown node ([a-z0-9]+)"
	clusterStateREString      = "cluster_state:([a-z]+)"
)

var (
	clusterKnownNodesRE = regexp.MustCompile(clusterKnownNodesREString)
	forgetnodeUnknownRE = regexp.MustCompile(forgotnodeUnknownREString)
	clusterStateRE      = regexp.MustCompile(clusterStateREString)
)

// IAdmin redis cluster admin interface
type IAdmin interface {
	// Connections returns the connection map of all clients
	Connections() IAdminConnections
	// Close the admin connections
	Close(ctx context.Context)
	// GetClusterInfos get node infos for all nodes
	GetClusterInfos(ctx context.Context, namespace string, name string) (*ClusterInfos, error)
	// ClusterManagerNodeIsEmpty Checks whether the node is empty. Node is considered not-empty if it has
	// some key or if it already knows other nodes
	ClusterManagerNodeIsEmpty(ctx context.Context, namespace string, name string) (bool, error)
	// SetConfigEpoch Assign a different config epoch to each node
	SetConfigEpoch(ctx context.Context, namespace string, name string) error
	// SetConfigIfNeed set redis config
	SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace string, name string) error
	// GetAllConfig get redis config by CONFIG GET *
	GetAllConfig(ctx context.Context, c *redis.Client, addr string, namespace string, name string) (map[string]string, error)
	// AttachNodeToCluster command use to connect a Node to the cluster
	// the connection will be done on a random node part of the connection pool
	AttachNodeToCluster(ctx context.Context, addr string, namespace string, name string) error
	// AttachSlaveToMaster attach a slave to a master node
	AttachSlaveToMaster(ctx context.Context, slave *Node, masterID string, namespace string, name string) error
	// DetachSlave dettach a slave to its master
	DetachSlave(ctx context.Context, slave *Node, namespace string, name string) error
	// ForgetNode execute the Redis command to force the cluster to forgot the the Node
	ForgetNode(ctx context.Context, id string, namespace string, name string) error
	// SetSlots exec the redis command to set slots in a pipeline, provide
	// and empty nodeID if the set slots commands doesn't take a nodeID in parameter
	SetSlots(ctx context.Context, addr string, action string, slots []Slot, nodeID string, namespace string, name string) error
	// AddSlots exec the redis command to add slots in a pipeline
	AddSlots(ctx context.Context, addr string, slots []Slot) error
	// SetSlot use to set SETSLOT command on a slot
	SetSlot(ctx context.Context, addr string, action string, slot Slot, nodeID string, namespace string, name string) error
	// MigrateKeys from addr to destination node. returns number of slot migrated. If replace is true, replace key on busy error
	MigrateKeys(ctx context.Context, addr string, dest *Node, slots []Slot, batch, timeout int, replace bool, namespace string, name string) (int, error)
	// MigrateKeys use to migrate keys from slot to other slot. if replace is true, replace key on busy error
	// timeout is in milliseconds
	MigrateKeysInSlot(ctx context.Context, addr string, dest *Node, slot Slot, batch int, timeout int, replace bool, namespace string, name string) (int, error)
	// FlushAndReset reset the cluster configuration of the node, the node is flushed in the same pipe to ensure reset works
	FlushAndReset(ctx context.Context, addr string, mode string, namespace string, name string) error
	// GetHashMaxSlot get the max slot value
	GetHashMaxSlot() Slot
	// ResetPassword reset redis node masterauth and requirepass.
	ResetPassword(ctx context.Context, newPassword string, namespace string, name string) error

	FailoverSlaveToMaster(ctx context.Context, slave *Node, option RedisFailover, namespace string, name string) error

	GetReplicationStatus(ip string, ctx context.Context, namespace string, name string) (string, ReplicationInfo, error)

	GetInfoReplication(ipPort string, ctx context.Context, namespace string, name string) (ReplicationInfo, error)
}

// AdminOptions optional options for redis admin
type AdminOptions struct {
	ConnectionTimeout  time.Duration
	ClientName         string
	RenameCommandsFile string
	Password           string
}

// Admin wraps redis cluster admin logic
type Admin struct {
	hashMaxSlots Slot
	cnx          IAdminConnections
	log          logr.Logger
	metrics      metrics.Recorder
}

// NewAdmin returns new AdminInterface instance
// at the same time it connects to all Redis Nodes thanks to the addrs list
func NewAdmin(addrs []string, options *AdminOptions, log logr.Logger, ctx context.Context, metrics metrics.Recorder) IAdmin {
	a := &Admin{
		hashMaxSlots: DefaultHashMaxSlots,
		log:          log.WithName("redis_util"),
		metrics:      metrics,
	}

	// perform initial connections
	a.cnx = NewAdminConnections(addrs, options, log, ctx)

	return a
}

// Connections returns the connection map of all clients
func (a *Admin) Connections() IAdminConnections {
	return a.cnx
}

// Close used to close all possible resources instanciate by the Admin
func (a *Admin) Close(ctx context.Context) {
	a.Connections().Reset(ctx)
}

// Metrics will be added to all the functions which return error
// GetClusterInfos return the Nodes infos for all nodes
func (a *Admin) GetClusterInfos(ctx context.Context, namespace string, name string) (*ClusterInfos, error) {
	infos := NewClusterInfos()
	clusterErr := NewClusterInfosError()

	for addr, c := range a.Connections().GetAll() {
		nodeinfos, err := a.getInfos(ctx, c, addr, namespace, name)
		if err != nil {
			a.log.WithValues("err", err).Info("get redis info failed")
			infos.Status = ClusterInfosPartial
			clusterErr.partial = true
			clusterErr.errs[addr] = err
			continue
		}
		if nodeinfos.Node != nil && nodeinfos.Node.IPPort() == addr {
			infos.Infos[addr] = nodeinfos

			//Check for the cluster state and update the cluster status
			cstatus, err := a.clusterGetState(ctx, c, addr, namespace, name)
			if err != nil {
				//a.log.WithValues("err", err).Info("get redis cluster status failed")
				if updateErr := CheckandUpdateClusterState(infos, ClusterStatusNOK, nodeinfos.Node.IP, "", cstatus); updateErr != nil {
					a.log.WithValues("err", updateErr).Info("update cluster state failed")
				}

			} else {
				if updateErr := CheckandUpdateClusterState(infos, ClusterStatusOK, "", "", cstatus); updateErr != nil {
					a.log.WithValues("err", updateErr).Info("update cluster state failed")
				}
			}

			//Check for the replication status and update the cluster status if its broken
			if nodeinfos.Node.Role == RedisSlaveRole {
				replicaStatus, replinfo, err := a.GetReplicationStatus(addr, ctx, namespace, name)
				if err != nil {
					//a.log.WithValues("err", err).Info("get redis slave replication status failed")
					if updateErr := CheckandUpdateClusterState(infos, ClusterStatusNOK, nodeinfos.Node.IP, nodeinfos.Node.IP, replicaStatus); updateErr != nil {
						a.log.WithValues("err", updateErr).Info("update cluster state failed")
					}
				} else {
					if updateErr := CheckandUpdateClusterState(infos, ClusterStatusOK, "", "", replicaStatus); updateErr != nil {
						a.log.WithValues("err", updateErr).Info("update cluster state failed")
					}
				}

				nodeinfos.Node.LastIOSecs = replinfo.LastIoSecondsAgo
				nodeinfos.Node.SlaveReplOffset = replinfo.SlaveReplicationOffset
				nodeinfos.Node.MasterReplOffset = replinfo.MasterReplicationOffset
				nodeinfos.Node.MasterLinkStatus = replinfo.MasterLinkStatus

			}
			//Check for Flags and if any failed update the cluster status
			if nodeinfos.Friends != nil {
				isFail := false
				for _, friend := range nodeinfos.Friends {
					if len(friend.FailStatus) > 0 {
						if updateErr := CheckandUpdateClusterState(infos, ClusterStatusNOK, nodeinfos.Node.IP, friend.IP, strings.Join(friend.FailStatus, ",")); updateErr != nil {
							a.log.WithValues("err", updateErr).Info("update cluster state failed")
						}
						isFail = true
					}
				}
				if !isFail {
					if updateErr := CheckandUpdateClusterState(infos, ClusterStatusOK, "", "", "NoFailedNodes"); updateErr != nil {
						a.log.WithValues("err", updateErr).Info("update cluster state failed")
					}
				}
			}
			//Check for Migrating Slots and upadte the cluster status
			if len(nodeinfos.Node.MigratingSlots) != 0 || len(nodeinfos.Node.ImportingSlots) != 0 {
				CheckandUpdateClusterState(infos, ClusterStatusNOK, nodeinfos.Node.IP, "", ClusterStatusMigrating)
			} else {
				CheckandUpdateClusterState(infos, ClusterStatusOK, "", "", "NoMigratingSlots")
			}
		} else {
			a.log.Info("bad node info retrieved from", "addr", addr)
		}

	}

	if len(clusterErr.errs) == 0 {
		clusterErr.inconsistent = !infos.ComputeStatus(a.log)
	}
	if infos.Status == ClusterInfosConsistent {
		return infos, nil
	}
	return infos, clusterErr
}

func (a *Admin) getInfos(ctx context.Context, c *redis.Client, addr string, namespace string, name string) (*NodeInfos, error) {
	resp := c.Do(ctx, "CLUSTER", "NODES")
	err := a.Connections().ValidateResp(ctx, resp, addr, "unable to retrieve node info")
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_NODES, addr, err, a.metrics)
	if err != nil {
		return nil, err
	}

	//var err error
	raw, _ := resp.Result()
	raw_str := fmt.Sprintf("%v", raw)
	nodeInfos := DecodeNodeInfos(&raw_str, addr, a.log)
	return nodeInfos, nil
}

// ClusterManagerNodeIsEmpty Checks whether the node is empty. Node is considered not-empty if it has
// some key or if it already knows other nodes
// TODO : This function is not used anywhere, need to remove it - cleanup
func (a *Admin) ClusterManagerNodeIsEmpty(ctx context.Context, namespace string, name string) (bool, error) {
	for addr, c := range a.Connections().GetAll() {
		knowNodes, err := a.clusterKnowNodes(ctx, c, addr, namespace, name)
		if err != nil {
			return false, err
		}
		if knowNodes != 1 {
			return false, nil
		}
	}
	return true, nil
}

// TODO : This function is not used anywhere, need to remove it - cleanup
func (a *Admin) clusterKnowNodes(ctx context.Context, client *redis.Client, addr string, namespace string, name string) (int, error) {
	resp := client.Do(ctx, "CLUSTER", "INFO")
	err := a.Connections().ValidateResp(ctx, resp, addr, "unable to retrieve cluster info")
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_INFO, addr, err, a.metrics)
	if err != nil {
		return 0, err
	}
	raw := resp.String()

	match := clusterKnownNodesRE.FindStringSubmatch(raw)
	if len(match) == 0 {
		return 0, fmt.Errorf("cluster_known_nodes regex not found")
	}
	return strconv.Atoi(match[1])
}

func CheckandUpdateClusterState(cinfos *ClusterInfos, cs clusterstate, fromip string, nip string, reason string) error {

	if (cs != ClusterStatusOK && cs != ClusterStatusNOK) || (cs == ClusterStatusNOK && reason == "") || (cs == ClusterStatusNOK && fromip == "") || cinfos == nil {
		err := fmt.Errorf("invalid Arguments for CheckandUpdateClusterState")
		return err

	}

	if cs == ClusterStatusNOK {
		cinfos.ClusterState = cs
		if nip == "" {
			cinfos.ClusterNokReason[fromip] = string(reason) + ": "
		} else {
			cinfos.ClusterNokReason[fromip] = nip + "-" + string(reason) + ": "
		}
	} else if cs == ClusterStatusOK && cinfos.ClusterState == ClusterStatusUnknown {
		cinfos.ClusterState = cs
	}
	return nil
}

func (a *Admin) clusterGetState(ctx context.Context, client *redis.Client, addr string, namespace string, name string) (string, error) {
	resp := client.Do(ctx, "CLUSTER", "INFO")
	err := a.Connections().ValidateResp(ctx, resp, addr, "unable to retrieve cluster info")
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_INFO, addr, err, a.metrics)
	if err != nil {
		return ClusterStateGetFailed, err
	}
	raw := resp.String()
	if raw != "" {
		match := clusterStateRE.FindStringSubmatch(raw)
		if len(match) == 0 {
			return ClusterStateUnknown, fmt.Errorf("cluster_state not found")
		} else {
			//a.log.Info("cluster state", "state", match[1])
			if match[1] == "fail" {
				return ClusterStateFail, fmt.Errorf("cluster_state is fail")
			} else {
				return ClusterStateOK, nil
			}
		}
	} else {
		return ClusterStateUnknown, fmt.Errorf("cluster_state not found")

	}
}

// FailoverSlaveToMaster manual failover of a slave node to master node
func (a *Admin) FailoverSlaveToMaster(ctx context.Context, slave *Node, option RedisFailover, namespace string, name string) error {
	c, err := a.Connections().Get(ctx, slave.IPPort())
	if err != nil {
		return err
	}
	var resp *redis.Cmd
	if option != RedisFailoverDefault {
		resp = c.Do(ctx, "CLUSTER", "FAILOVER", string(option))
	} else {
		resp = c.Do(ctx, "CLUSTER", "FAILOVER")
	}

	err1 := a.Connections().ValidateResp(ctx, resp, slave.IPPort(), "unable to run command FAILOVER")
	if err1 != nil {
		return err
	}
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_FAILOVER, slave.IPPort(), err1, a.metrics)

	result, err := resp.Result()
	if err != nil {
		return err
	}
	a.log.Info("FailoverSlaveToMaster", "slave", slave.IPPort(), "option", option, "resp", result)
	// Wait for the node to become a master or timeout.
	timeout := time.After(RoleChangeTimeout * time.Second)
	ticker := time.NewTicker(RoleChangeInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for node %s to become master", slave.IPPort())
		case <-ticker.C:

			nodeInfo, err := a.GetClusterNodeInfos(ctx, slave.IPPort(), namespace, name)
			if err != nil {
				return fmt.Errorf("error checking role of node %s: %w", slave.IPPort(), err)
			}
			if nodeInfo.Node.Role == RedisMasterRole && nodeInfo.Node.TotalSlots() > 0 {
				slave.SetRole(RedisMasterRole)
				slave.SetReferentMaster("")
				return nil
			} else {
				failoverErr := fmt.Errorf("FailoverSlaveToMaster failed for node %s", slave.IPPort())
				a.log.Error(failoverErr, "FailoverSlaveToMaster", "failed over node role", nodeInfo.Node.Role, "slots", nodeInfo.Node.TotalSlots())
				return failoverErr
			}
		}
	}
}

// AttachSlaveToMaster attach a slave to a master node
func (a *Admin) AttachSlaveToMaster(ctx context.Context, slave *Node, masterID string, namespace string, name string) error {
	c, err := a.Connections().Get(ctx, slave.IPPort())
	if err != nil {
		return err
	}

	resp := c.Do(ctx, "CLUSTER", "REPLICATE", masterID)
	err = a.Connections().ValidateResp(ctx, resp, slave.IPPort(), "unable to run command REPLICATE")
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_REPLICATE, slave.IPPort(), err, a.metrics)
	if err != nil {
		return err
	}
	if resp.Err() != nil {
		return err
	}
	slave.SetReferentMaster(masterID)
	slave.SetRole(RedisSlaveRole)

	return nil
}

// AddSlots use to ADDSLOT commands on several slots
func (a *Admin) AddSlots(ctx context.Context, addr string, slots []Slot) error {
	if len(slots) == 0 {
		return nil
	}
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return err
	}
	firstSlot := slots[0].ToInt()
	lastSlot := slots[len(slots)-1].ToInt()
	resp := c.ClusterAddSlotsRange(ctx, firstSlot, lastSlot)
	if resp.Err() != nil {
		return resp.Err()
	}
	return err
	// TODO - add validateresp to handle  *redis.StatusCmd, currently it handles only *redis.Cmd
	//return a.Connections().ValidateResp(ctx, resp, addr, "unable to run CLUSTER ADDSLOTS")
}

// SetSlots use to set SETSLOT command on several slots
func (a *Admin) SetSlots(ctx context.Context, addr, action string, slots []Slot, nodeID string, namespace string, name string) error {
	if len(slots) == 0 {
		return nil
	}
	c, err := a.Connections().Get(ctx, addr)

	if err != nil {
		return err
	}
	pipeClient := c.Pipeline()

	for _, slot := range slots {
		slotint := slot.ToInt()
		args := []interface{}{"CLUSTER", "SETSLOT", slotint, action}
		if nodeID != "" {
			args = append(args, nodeID)
		}
		pipeClient.Do(ctx, args...)
	}

	out, err := pipeClient.Exec(ctx)
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_SETSLOT, addr, err, a.metrics)

	if err != nil {
		return err
	}

	if !a.Connections().ValidatePipeResp(ctx, out, addr, err, "Cannot SETSLOTS") {
		return fmt.Errorf("Error occured during CLUSTER SETSLOT %s", action)
	}
	// TODO : check need to clear pipeline
	//c.PipeClear()

	return nil
}

// SetSlot use to set SETSLOT command on a slot
func (a *Admin) SetSlot(ctx context.Context, addr, action string, slot Slot, nodeID string, namespace string, name string) error {
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return err
	}
	pipeClient := c.Pipeline()
	slotint := slot.ToInt()
	args := []interface{}{"CLUSTER", "SETSLOT", slotint, action}

	if nodeID != "" {
		args = append(args, nodeID)
	}
	pipeClient.Do(ctx, args...)
	out, err := pipeClient.Exec(ctx)
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_SETSLOT, addr, err, a.metrics)
	// TODO: Handle response

	if !a.Connections().ValidatePipeResp(ctx, out, addr, err, "Cannot SETSLOT") {
		return fmt.Errorf("Error occured during CLUSTER SETSLOT %s", action)
	}
	// TODO : check need to clear pipeline
	//c.PipeClear()

	return nil
}

// this is function is not used, can be removed - cleanup
func (a *Admin) SetConfigEpoch(ctx context.Context, namespace string, name string) error {
	configEpoch := 1
	for addr, c := range a.Connections().GetAll() {
		resp := c.Do(ctx, "CLUSTER", "SET-CONFIG-EPOCH", configEpoch)
		utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_SET_CONFIG_EPOCH, addr, resp.Err(), a.metrics)
		if err := a.Connections().ValidateResp(ctx, resp, addr, "unable to run command SET-CONFIG-EPOCH"); err != nil {
			return err
		}
		if resp.Err() != nil {
			return resp.Err()
		}
		configEpoch++
	}
	return nil
}

// AttachNodeToCluster command use to connect a Node to the cluster
func (a *Admin) AttachNodeToCluster(ctx context.Context, addr string, namespace string, name string) error {
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	all := a.Connections().GetAll()
	if len(all) == 0 {
		return fmt.Errorf("no connection for other redis-node found")
	}
	for cAddr, c := range a.Connections().GetAll() {
		if cAddr == addr {
			continue
		}
		a.log.V(3).Info("CLUSTER MEET", "from addr", cAddr, "to", addr)
		resp := c.Do(ctx, "CLUSTER", "MEET", ip, port)
		err = a.Connections().ValidateResp(ctx, resp, cAddr, "cannot attach node to cluster")
		utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_MEET, cAddr, err, a.metrics)
		if err != nil {
			return err
		}
		if resp.Err() != nil {
			return resp.Err()
		}
	}
	a.Connections().Add(ctx, addr)
	a.log.Info(fmt.Sprintf("node %s attached properly", addr))
	return nil
}

// GetAllConfig get redis config by CONFIG GET *
func (a *Admin) GetAllConfig(ctx context.Context, c *redis.Client, addr string, namespace string, name string) (map[string]string, error) {
	resp := c.Do(ctx, "CONFIG", "GET", "*")
	err := a.Connections().ValidateResp(ctx, resp, addr, "unable to retrieve config")
	utils.RecordRedisMetrics(namespace, name, metrics.REDIS_CONFIG_GET, addr, err, a.metrics)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}

	res, err := resp.Result()

	originalMap := res.(map[interface{}]interface{})
	convertedMap := make(map[string]string)

	for key, value := range originalMap {
		// Check if the key and value are of the expected types
		keyStr, keyIsString := key.(string)
		valueStr, valueIsString := value.(string)

		if keyIsString && valueIsString {
			convertedMap[keyStr] = valueStr
		}
	}
	return convertedMap, err
}

/*
var parseConfigMap = map[string]int8{
	"maxmemory":                  0,
	"proto-max-bulk-len":         0,
	"client-query-buffer-limit":  0,
	"repl-backlog-size":          0,
	"auto-aof-rewrite-min-size":  0,
	"active-defrag-ignore-bytes": 0,
	"hash-max-ziplist-entries":   0,
	"hash-max-ziplist-value":     0,
	"stream-node-max-bytes":      0,
	"set-max-intset-entries":     0,
	"zset-max-ziplist-entries":   0,
	"zset-max-ziplist-value":     0,
	"hll-sparse-max-bytes":       0,
	// TODO parse client-output-buffer-limit
	//"client-output-buffer-limit": 0,
}
*/

// SetConfigIfNeed set redis config
func (a *Admin) SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace string, name string) error {
	for addr, c := range a.Connections().GetAll() {
		oldConfig, err := a.GetAllConfig(ctx, c, addr, namespace, name)
		if err != nil {
			return err
		}

		for key, value := range newConfig {
			// Disabling byte conversion and sending only byte as value
			// var err error
			// if _, ok := parseConfigMap[key]; ok {
			// 	value, err = utils.ParseRedisMemConf(value)
			// 	if err != nil {
			// 		a.log.Error(err, "redis config format err", "key", key, "value", value)
			// 		continue
			// 	}
			// }
			//
			// TODO: Adding a hacky way to ignore loadmodule config. This is a temporary fix and need to be removed
			// load module can be set only during the start of the redis server
			if key == "loadmodule" {
				continue
			}
			if value != oldConfig[key] {
				a.log.V(3).Info("CONFIG SET", key, value)
				resp := c.Do(ctx, "CONFIG", "SET", key, value)
				err = a.Connections().ValidateResp(ctx, resp, addr, "unable to set config")
				utils.RecordRedisMetrics(namespace, name, metrics.REDIS_CONFIG_SET, addr, err, a.metrics)
				if err != nil {
					return err
				}
				if resp.Err() != nil {
					return resp.Err()
				}
			}
		}
	}
	return nil
}

// GetHashMaxSlot get the max slot value
func (a *Admin) GetHashMaxSlot() Slot {
	return a.hashMaxSlots
}

// MigrateKeys use to migrate keys from slots to other slots. if replace is true, replace key on busy error
// timeout is in milliseconds
func (a *Admin) MigrateKeys(ctx context.Context, addr string, dest *Node, slots []Slot, batch int, timeout int, replace bool, namespace string, name string) (int, error) {
	if len(slots) == 0 {
		return 0, nil
	}
	keyCount := 0
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return keyCount, err
	}
	timeoutStr := strconv.Itoa(timeout)
	batchStr := strconv.Itoa(batch)

	for _, slot := range slots {
		for {
			slotint := slot.ToInt()
			resp := c.Do(ctx, "CLUSTER", "GETKEYSINSLOT", slotint, batchStr)
			err := a.Connections().ValidateResp(ctx, resp, addr, "Unable to run command GETKEYSINSLOT")
			utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_GET_KEYS_INSLOT, addr, err, a.metrics)
			if err != nil {
				return keyCount, err
			}
			if resp.Err() != nil {
				return 0, resp.Err()
			}
			res, err := resp.Result()
			if err != nil {
				return keyCount, err
			}

			//TODO:  better way to convert this to list
			keySlice, ok := res.([]interface{})
			if !ok {
				return keyCount, fmt.Errorf("unexpected type for res: %T", res)
			}

			keys := make([]string, len(keySlice))

			for i := 0; i < len(keySlice); i++ {
				key, ok := keySlice[i].(string)
				if !ok {
					return keyCount, fmt.Errorf("unexpected type for key at index %d: %T", i, keySlice[i])
				}
				keys[i] = key
			}

			keyCount += len(keys)
			if len(keys) == 0 {
				break
			}

			args := a.migrateCmdArgs(dest, timeoutStr, replace, keys)
			resp = c.Do(ctx, "MIGRATE", args)
			err = resp.Err()
			utils.RecordRedisMetrics(namespace, name, metrics.MIGRATE, addr, err, a.metrics)
			if err != nil {
				return keyCount, err
			}
		}
	}

	return keyCount, nil
}

// MigrateKeys use to migrate keys from slot to other slot. if replace is true, replace key on busy error
// timeout is in milliseconds
func (a *Admin) MigrateKeysInSlot(ctx context.Context, addr string, dest *Node, slot Slot, batch int, timeout int, replace bool, namespace string, name string) (int, error) {
	keyCount := 0
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return keyCount, err
	}
	timeoutStr := strconv.Itoa(timeout)
	batchStr := strconv.Itoa(batch)

	slotint := slot.ToInt()

	for {
		resp := c.Do(ctx, "CLUSTER", "GETKEYSINSLOT", slotint, batchStr)
		err = a.Connections().ValidateResp(ctx, resp, addr, "Unable to run command GETKEYSINSLOT")
		utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_GET_KEYS_INSLOT, addr, err, a.metrics)
		if err != nil {
			return keyCount, err
		}
		if resp.Err() != nil {
			return 0, resp.Err()
		}
		res, err := resp.Result()
		if err != nil {
			return keyCount, err
		}

		// TODO: Better way to convert to list
		keySlice, ok := res.([]interface{})
		if !ok {
			return keyCount, fmt.Errorf("unexpected type for res: %T", res)
		}

		keys := make([]string, len(keySlice))

		for i := 0; i < len(keySlice); i++ {
			key, ok := keySlice[i].(string)
			if !ok {
				return keyCount, fmt.Errorf("unexpected type for key at index %d: %T", i, keySlice[i])
			}
			keys[i] = key
		}

		keyCount += len(keys)
		if len(keys) == 0 {
			break
		}

		args := a.migrateCmdArgs(dest, timeoutStr, replace, keys)

		resp = c.Do(ctx, "MIGRATE", args)
		err = a.Connections().ValidateResp(ctx, resp, addr, "Unable to run command MIGRATE")
		utils.RecordRedisMetrics(namespace, name, metrics.MIGRATE, addr, err, a.metrics)
		if err != nil {
			return keyCount, err
		}
		if resp.Err() != nil {
			return 0, resp.Err()
		}

	}

	return keyCount, nil
}

func (a *Admin) migrateCmdArgs(dest *Node, timeoutStr string, replace bool, keys []string) []string {
	args := []string{dest.IP, dest.Port, "", "0", timeoutStr}
	if password, ok := a.Connections().GetAUTH(); ok {
		args = append(args, "AUTH", password)
	}
	if replace {
		args = append(args, "REPLACE", "KEYS")
	} else {
		args = append(args, "KEYS")
	}
	args = append(args, keys...)
	return args
}

// ForgetNode used to force other redis cluster node to forget a specific node
func (a *Admin) ForgetNode(ctx context.Context, id string, namespace string, name string) error {
	infos, _ := a.GetClusterInfos(ctx, namespace, name)
	for nodeAddr, nodeinfos := range infos.Infos {
		if nodeinfos.Node.ID == id {
			continue
		}

		if IsSlave(nodeinfos.Node) && nodeinfos.Node.MasterReferent == id {
			if err := a.DetachSlave(ctx, nodeinfos.Node, namespace, name); err != nil {
				a.log.Error(err, "DetachSlave", "node", nodeAddr)
			}
			a.log.Info(fmt.Sprintf("detach slave id: %s of master: %s", nodeinfos.Node.ID, id))
		}

		c, err := a.Connections().Get(ctx, nodeAddr)
		if err != nil {
			a.log.Error(err, fmt.Sprintf("cannot force a forget on node %s, for node %s", nodeAddr, id))
			continue
		}

		a.log.Info("CLUSTER FORGET", "id", id, "from", nodeAddr)
		resp := c.Do(ctx, "CLUSTER", "FORGET", id)
		err = resp.Err()
		utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_FORGET, nodeAddr, err, a.metrics)
		a.Connections().ValidateResp(ctx, resp, nodeAddr, "Unable to execute FORGET command")
		if resp.Err() != nil {
			match := forgetnodeUnknownRE.FindStringSubmatch(resp.Err().Error())
			a.log.Info("CLUSTER FORGET", "Error", resp.Err().Error(), "from", nodeAddr, "match:", match)
			if len(match) == 0 {
				return resp.Err()
			}
		}
	}

	a.log.Info("Forget node done", "node", id)
	return nil
}

// DetachSlave use to detach a slave to a master
func (a *Admin) DetachSlave(ctx context.Context, slave *Node, namespace string, name string) error {
	c, err := a.Connections().Get(ctx, slave.IPPort())
	if err != nil {
		a.log.Error(err, fmt.Sprintf("unable to get the connection for slave ID:%s, addr:%s", slave.ID, slave.IPPort()))
		return err
	}

	resp := c.Do(ctx, "CLUSTER", "RESET", "SOFT")
	err = a.Connections().ValidateResp(ctx, resp, slave.IPPort(), "cannot detach node from cluster")
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_RESET_SOFT, slave.IPPort(), err, a.metrics)
	if err != nil {
		return err
	}
	if resp.Err() != nil {
		return resp.Err()
	}

	if err = a.AttachNodeToCluster(ctx, slave.IPPort(), namespace, name); err != nil {
		a.log.Error(err, fmt.Sprintf("[DetachSlave] unable to AttachNodeToCluster the Slave id: %s addr:%s", slave.ID, slave.IPPort()))
		return err
	}

	slave.SetReferentMaster("")
	slave.SetRole(RedisMasterRole)

	return nil
}

// FlushAndReset flush the cluster and reset the cluster configuration of the node. Commands are piped, to ensure no items arrived between flush and reset
func (a *Admin) FlushAndReset(ctx context.Context, addr string, mode string, namespace string, name string) error {
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return err
	}
	pipeClient := c.Pipeline()
	pipeClient.Do(ctx, "FLUSHALL")
	pipeClient.Do(ctx, "CLUSTER", "RESET", mode)

	out, err := pipeClient.Exec(ctx)
	utils.RecordRedisMetrics(namespace, name, metrics.FLUSHALL, addr, err, a.metrics)
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_RESET, addr, err, a.metrics)

	if !a.Connections().ValidatePipeResp(ctx, out, addr, err, "Cannot reset node") {
		return fmt.Errorf("cannot reset node %s", addr)
	}

	return nil
}

// this function is not used, can be removed - cleanup
// ResetPassword reset redis node masterauth and requirepass.
func (a *Admin) ResetPassword(ctx context.Context, newPassword string, namespace string, name string) error {
	all := a.Connections().GetAll()
	if len(all) == 0 {
		return fmt.Errorf("no connection for other redis-node found")
	}
	for addr, c := range a.Connections().GetAll() {
		a.log.Info("reset password", "addr", addr)
		resp := c.Do(ctx, "CONFIG", "SET", "masterauth", newPassword)
		err := a.Connections().ValidateResp(ctx, resp, addr, "cannot set new masterauth")
		utils.RecordRedisMetrics(namespace, name, metrics.REDIS_CONFIG_SET, addr, err, a.metrics)
		if err != nil {
			return err
		}
		if resp.Err() != nil {
			return resp.Err()
		}
		resp = c.Do(ctx, "CONFIG", "SET", "requirepass", newPassword)
		err = a.Connections().ValidateResp(ctx, resp, addr, "cannot set new requirepass")
		utils.RecordRedisMetrics(namespace, name, metrics.REDIS_CONFIG_SET, addr, err, a.metrics)
		if err != nil {
			return err
		}
		if resp.Err() != nil {
			return resp.Err()
		}
	}
	return nil
}

func (a *Admin) GetClusterNodeInfos(ctx context.Context, addr string, namespace string, name string) (*NodeInfos, error) {
	c, err := a.Connections().Get(ctx, addr)
	if err != nil {
		return nil, err
	}
	return a.getInfos(ctx, c, addr, namespace, name)
}

func (a *Admin) GetInfoReplication(ipPort string, ctx context.Context, namespace string, name string) (ReplicationInfo, error) {

	var replInfo ReplicationInfo = ReplicationInfo{}

	c, err := a.Connections().Get(ctx, ipPort)
	if err != nil {
		return replInfo, err
	}
	info, err := c.Info(context.TODO(), "replication").Result()
	utils.RecordRedisMetrics(namespace, name, metrics.CLUSTER_INFO, ipPort, err, a.metrics)
	if err != nil {
		return replInfo, err
	}

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		var err error
		switch key {
		case "role":
			replInfo.Role = value
		case "master_link_status":
			replInfo.MasterLinkStatus = value
		case "master_host":
			replInfo.MasterHost = value
		case "master_last_io_seconds_ago":
			replInfo.LastIoSecondsAgo, err = strconv.ParseInt(value, 10, 64)
		case "slave_repl_offset":
			replInfo.SlaveReplicationOffset, err = strconv.ParseInt(value, 10, 64)
		case "master_repl_offset":
			replInfo.MasterReplicationOffset, err = strconv.ParseInt(value, 10, 64)
		case "master_sync_in_progress":
			var val int
			val, err = strconv.Atoi(value)
			replInfo.MasterSyncInProgress = (val != 0)
		}

		if err != nil {
			return ReplicationInfo{}, fmt.Errorf("failed to parse value for key %s: %v", key, err)
		}
	}
	return replInfo, nil
}
func (a *Admin) GetReplicationStatus(ip string, ctx context.Context, namespace string, name string) (string, ReplicationInfo, error) {

	var replInfo ReplicationInfo = ReplicationInfo{}
	var status string = ReplicaStatusUnknown

	replInfo, err := a.GetInfoReplication(ip, ctx, namespace, name)
	if err != nil {
		a.log.Error(err, "GetInfoReplication failed")
		return status, replInfo, err
	}

	if replInfo.MasterLinkStatus == "up" && !replInfo.MasterSyncInProgress && replInfo.MasterHost != "127.0.0.1" {
		status = ReplicaStatusReady
		return status, replInfo, nil
	} else if replInfo.MasterHost == "127.0.0.1" {
		status = status + "-" + ReplicaStatusMasterNil
	} else if replInfo.MasterLinkStatus == "down" {
		status = status + "-" + ReplicaLinkDown
	} else if replInfo.MasterSyncInProgress {
		status = status + "-" + ReplicaStatusSyncing
	} else {
		status = status + "-" + ReplicaStatusUnknown
	}
	a.log.Info("Replica status NOK", "status", status)
	return status, replInfo, fmt.Errorf("replica status NOK")

}
