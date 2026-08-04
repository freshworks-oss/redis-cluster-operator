package distributedrediscluster

import (
	"fmt"
	"testing"

	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// ghostInfos builds a ClusterInfos of numPeers peers; the first failVotes of them report node "g0"
// with a fail flag. If ghostHasSlots, g0 is reported owning a slot (which must exempt it).
func ghostInfos(numPeers, failVotes int, ghostHasSlots bool) *redisutil.ClusterInfos {
	ci := redisutil.NewClusterInfos()
	for i := 0; i < numPeers; i++ {
		peerIP := fmt.Sprintf("10.0.0.%d", i+1)
		peer := &redisutil.Node{ID: fmt.Sprintf("p%d", i), IP: peerIP, Port: "6379", Role: redisutil.RedisMasterRole, Slots: []redisutil.Slot{redisutil.Slot(i)}}
		ghost := &redisutil.Node{ID: "g0", IP: "10.0.9.9", Port: "6379"}
		if ghostHasSlots {
			ghost.Slots = []redisutil.Slot{100}
		}
		if i < failVotes {
			ghost.FailStatus = []string{redisutil.NodeStatusFail}
		}
		ci.Infos[peerIP+":6379"] = &redisutil.NodeInfos{Node: peer, Friends: redisutil.Nodes{ghost}}
	}
	return ci
}

func TestListGhostBootstrapNodes(t *testing.T) {
	cases := []struct {
		name      string
		peers     int
		votes     int
		slots     bool
		wantGhost bool
	}{
		{"3 peers, 1 vote is a minority -> not forgotten", 3, 1, false, false},
		{"3 peers, 2 votes is a majority -> forgotten", 3, 2, false, true},
		{"2 peers, 1 vote is not a majority -> not forgotten", 2, 1, false, false},
		{"2 peers, 2 votes is a majority -> forgotten", 2, 2, false, true},
		{"node still owning slots is never forgotten", 3, 3, true, false},
		{"empty infos -> nothing forgotten", 0, 0, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := listGhostBootstrapNodes(ghostInfos(tc.peers, tc.votes, tc.slots))
			if got["g0"] != tc.wantGhost {
				t.Fatalf("g0 forgotten=%v, want %v (peers=%d votes=%d)", got["g0"], tc.wantGhost, tc.peers, tc.votes)
			}
		})
	}
}
