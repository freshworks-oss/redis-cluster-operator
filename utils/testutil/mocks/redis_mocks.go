package mocks

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

// MockRedisAdmin provides a unified mock for all Redis admin operations
// Implements the actual redisutil.IAdmin interface
type MockRedisAdmin struct {
	mock.Mock
}

// Methods that match the actual redisutil.IAdmin interface
func (m *MockRedisAdmin) Connections() redisutil.IAdminConnections {
	args := m.Called()
	return args.Get(0).(redisutil.IAdminConnections)
}

func (m *MockRedisAdmin) GetHashMaxSlot() redisutil.Slot {
	args := m.Called()
	return redisutil.Slot(args.Int(0))
}

func (m *MockRedisAdmin) AddSlots(ctx context.Context, addr string, slots []redisutil.Slot) error {
	args := m.Called(ctx, addr, slots)
	return args.Error(0)
}

func (m *MockRedisAdmin) SetSlots(ctx context.Context, addr, state string, slots []redisutil.Slot, nodeID string, namespace, name string) error {
	args := m.Called(ctx, addr, state, slots, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) MigrateKeys(ctx context.Context, addr string, dest *redisutil.Node, slots []redisutil.Slot, batch int, timeout int, replace bool, namespace, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slots, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockRedisAdmin) ForgetNode(ctx context.Context, nodeID, namespace, name string) error {
	args := m.Called(ctx, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) FailoverSlaveToMaster(ctx context.Context, slave *redisutil.Node, takeoverType redisutil.RedisFailover, namespace, name string) error {
	args := m.Called(ctx, slave, takeoverType, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) AttachNodeToCluster(ctx context.Context, addr, namespace, name string) error {
	args := m.Called(ctx, addr, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) FlushAndReset(ctx context.Context, addr string, resetType string, namespace, name string) error {
	args := m.Called(ctx, addr, resetType, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) AttachSlaveToMaster(ctx context.Context, slave *redisutil.Node, masterID string, namespace, name string) error {
	args := m.Called(ctx, slave, masterID, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) GetClusterInfos(ctx context.Context, namespace, name string) (*redisutil.ClusterInfos, error) {
	args := m.Called(ctx, namespace, name)
	return args.Get(0).(*redisutil.ClusterInfos), args.Error(1)
}

func (m *MockRedisAdmin) GetClusterNodeInfos(ctx context.Context, addr, namespace, name string) (*redisutil.NodeInfos, error) {
	args := m.Called(ctx, addr, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redisutil.NodeInfos), args.Error(1)
}

func (m *MockRedisAdmin) ClusterManagerNodeIsEmpty(ctx context.Context, namespace, name string) (bool, error) {
	args := m.Called(ctx, namespace, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisAdmin) SetConfigEpoch(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace, name string) error {
	args := m.Called(ctx, newConfig, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) GetAllConfig(ctx context.Context, c *redis.Client, addr, namespace, name string) (map[string]string, error) {
	args := m.Called(ctx, c, addr, namespace, name)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockRedisAdmin) DetachSlave(ctx context.Context, slave *redisutil.Node, namespace, name string) error {
	args := m.Called(ctx, slave, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) SetSlot(ctx context.Context, addr, action string, slot redisutil.Slot, nodeID, namespace, name string) error {
	args := m.Called(ctx, addr, action, slot, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) MigrateKeysInSlot(ctx context.Context, addr string, dest *redisutil.Node, slot redisutil.Slot, batch, timeout int, replace bool, namespace, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slot, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockRedisAdmin) ResetPassword(ctx context.Context, newPassword, namespace, name string) error {
	args := m.Called(ctx, newPassword, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) GetReplicationStatus(ip string, ctx context.Context, namespace, name string) (string, redisutil.ReplicationInfo, error) {
	args := m.Called(ip, ctx, namespace, name)
	return args.String(0), args.Get(1).(redisutil.ReplicationInfo), args.Error(2)
}

func (m *MockRedisAdmin) GetInfoReplication(ipPort string, ctx context.Context, namespace, name string) (redisutil.ReplicationInfo, error) {
	args := m.Called(ipPort, ctx, namespace, name)
	return args.Get(0).(redisutil.ReplicationInfo), args.Error(1)
}

func (m *MockRedisAdmin) DrainClientsInOldMaster(ctx context.Context, node *redisutil.Node, namespace, name string) error {
	args := m.Called(ctx, node, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) MeetNodeFromAddr(ctx context.Context, fromAddr, targetHost, targetPort, namespace, name string) error {
	args := m.Called(ctx, fromAddr, targetHost, targetPort, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) SetConfigOnAddr(ctx context.Context, addr string, config map[string]string, namespace, name string) error {
	args := m.Called(ctx, addr, config, namespace, name)
	return args.Error(0)
}

func (m *MockRedisAdmin) Close(ctx context.Context) {
	m.Called(ctx)
}

// MockAdminConnections provides a unified mock for Redis connection management
type MockAdminConnections struct {
	mock.Mock
}

func (m *MockAdminConnections) Add(addr string, conn interface{}) {
	m.Called(addr, conn)
}

func (m *MockAdminConnections) Remove(addr string) {
	m.Called(addr)
}

func (m *MockAdminConnections) Get(addr string) (interface{}, error) {
	args := m.Called(addr)
	return args.Get(0), args.Error(1)
}

func (m *MockAdminConnections) GetRandom() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockAdminConnections) GetAll() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockAdminConnections) ReplaceAll(addrs []string) error {
	args := m.Called(addrs)
	return args.Error(0)
}

// MockRedisConnection provides a mock for a single Redis connection
type MockRedisConnection struct {
	mock.Mock
}

func (m *MockRedisConnection) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockRedisConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockStringCmd provides a mock for redis.StringCmd
type MockStringCmd struct {
	mock.Mock
}

func (m *MockStringCmd) Result() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// MockCmd provides a mock for redis.Cmd
type MockCmd struct {
	mock.Mock
}

func (m *MockCmd) Result() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}
