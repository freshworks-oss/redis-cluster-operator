package e2e

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/sync/errgroup"
)

const defaultTimeOut = time.Second * 2

// GoRedis contains ClusterClient.
type GoRedis struct {
	client   *redis.ClusterClient
	password string
}

// NewGoRedis return a new ClusterClient.
func NewGoRedis(addr, password string) *GoRedis {
	return &GoRedis{
		client: redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{addr},
			Password: password,
			Protocol: 2, // Pin to RESP2 for Redis 7.2 compatibility
			//MaxRetries: 5,
			//
			//PoolSize:     3,
			//MinIdleConns: 1,
			//PoolTimeout:  defaultTimeOut,
			//IdleTimeout:  defaultTimeOut,
		}),
		password: password,
	}
}

// StuffingData filled with (round * n)'s key.
func (g *GoRedis) StuffingData(round, n int) error {
	ctx := context.Background()
	var group errgroup.Group
	for i := 0; i < round; i++ {
		group.Go(func() error {
			for j := 0; j < n; j++ {
				key := uuid.NewV4().String()
				if err := g.client.Set(ctx, key, key, 0).Err(); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}

// DBSize return DBsize of all master nodes.
func (g *GoRedis) DBSize() (int64, error) {
	ctx := context.Background()
	return g.client.DBSize(ctx).Result()
}

// Password return redis password.
func (g *GoRedis) Password() string {
	return g.password
}

// Close closes the cluster client.
func (g *GoRedis) Close() error {
	return g.client.Close()
}
