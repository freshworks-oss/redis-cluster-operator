package clustering

import (
	"context"
	"fmt"

	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// AttachingSlavesToMaster used to attach slaves to there masters
func (c *Ctx) AttachingSlavesToMaster(admin redisutil.IAdmin, ctx context.Context, namespace string, name string) error {
	var globalErr error
	for masterID, slaves := range c.slavesByMaster {
		masterNode, err := c.cluster.GetNodeByID(masterID)
		if err != nil {
			c.log.Error(err, fmt.Sprintf("unable fo found the Cluster.Node with redis ID:%s", masterID))
			continue
		}
		for _, slave := range slaves {
			c.log.Info(fmt.Sprintf("attaching node %s to master %s", slave.ID, masterID))

			err := admin.AttachSlaveToMaster(ctx, slave, masterNode.ID, namespace, name)
			if err != nil {
				c.log.Error(err, fmt.Sprintf("attaching node %s to master %s", slave.ID, masterID))
				globalErr = err
			}
		}
	}
	return globalErr
}
