package main

import (
	"context"
	"go.etcd.io/raft/v3"
	"time"
)

func main() {
	peers := []raft.Peer{
		{ID: 1, Context: nil},
		{ID: 2, Context: nil},
		{ID: 3, Context: nil},
	}
	nt := newRaftNetwork(1, 2, 3)

	nodes := make([]*node, 0)

	for i := 1; i <= len(peers); i++ {
		n := startNode(uint64(i), peers, nt.nodeNetwork(uint64(i)))
		nodes = append(nodes, n)
	}

	waitLeader(nodes)

	var messagesCount uint64 = 100
	for i := 0; i < int(messagesCount); i++ {
		nodes[0].Propose(context.TODO(), []byte("somedata"))
	}

	if !waitCommitConverge(nodes, messagesCount) {
		panic("commits failed to converge!")
	}

	for _, n := range nodes {
		n.stop()
	}
}

func waitLeader(ns []*node) int {
	var l map[uint64]struct{}
	var lindex int

	for {
		l = make(map[uint64]struct{})

		for i, n := range ns {
			lead := n.Status().SoftState.Lead
			if lead != 0 {
				l[lead] = struct{}{}
				if n.id == lead {
					lindex = i
				}
			}
		}

		if len(l) == 1 {
			return lindex
		}
	}
}

func waitCommitConverge(ns []*node, target uint64) bool {
	var c map[uint64]struct{}

	for i := 0; i < 50; i++ {
		c = make(map[uint64]struct{})
		var good int

		for _, n := range ns {
			commit := n.Node.Status().HardState.Commit
			c[commit] = struct{}{}
			if commit > target {
				good++
			}
		}

		if len(c) == 1 && good == len(ns) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}
