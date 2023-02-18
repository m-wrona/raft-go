package main

import (
	"fmt"
	"os"
	"testing"

	"go.etcd.io/etcd/raft/v3/raftpb"
)

func getSnapshotFn() (func() ([]byte, error), <-chan struct{}) {
	snapshotTriggeredC := make(chan struct{})
	return func() ([]byte, error) {
		snapshotTriggeredC <- struct{}{}
		return nil, nil
	}, snapshotTriggeredC
}

type cluster struct {
	peers              []string
	commitC            []<-chan *commit
	errorC             []<-chan error
	proposeC           []chan string
	confChangeC        []chan raftpb.ConfChange
	snapshotTriggeredC []<-chan struct{}
}

// newCluster creates a cluster of n nodes
func newCluster(n int, dirPath string) *cluster {
	peers := make([]string, n)
	for i := range peers {
		peers[i] = fmt.Sprintf("http://127.0.0.1:%d", 10000+i)
	}

	clus := &cluster{
		peers:              peers,
		commitC:            make([]<-chan *commit, len(peers)),
		errorC:             make([]<-chan error, len(peers)),
		proposeC:           make([]chan string, len(peers)),
		confChangeC:        make([]chan raftpb.ConfChange, len(peers)),
		snapshotTriggeredC: make([]<-chan struct{}, len(peers)),
	}

	for i := range clus.peers {
		os.RemoveAll(fmt.Sprintf("raftexample-%d", i+1))
		os.RemoveAll(fmt.Sprintf("raftexample-%d-snap", i+1))
		clus.proposeC[i] = make(chan string, 1)
		clus.confChangeC[i] = make(chan raftpb.ConfChange, 1)
		fn, snapshotTriggeredC := getSnapshotFn()
		clus.snapshotTriggeredC[i] = snapshotTriggeredC
		clus.commitC[i], clus.errorC[i], _ = newRaftNode(i+1, clus.peers, false, fn, clus.proposeC[i], clus.confChangeC[i], dirPath)
	}

	return clus
}

// Close closes all cluster nodes and returns an error if any failed.
func (clus *cluster) Close() (err error) {
	for i := range clus.peers {
		go func(i int) {
			for range clus.commitC[i] {
				// drain pending commits
			}
		}(i)
		close(clus.proposeC[i])
		// wait for channel to close
		if erri := <-clus.errorC[i]; erri != nil {
			err = erri
		}
		// clean intermediates
		os.RemoveAll(fmt.Sprintf("raftexample-%d", i+1))
		os.RemoveAll(fmt.Sprintf("raftexample-%d-snap", i+1))
	}
	return err
}

func (clus *cluster) closeNoErrors(t *testing.T) {
	t.Log("closing cluster...")
	if err := clus.Close(); err != nil {
		t.Fatal(err)
	}
	t.Log("closing cluster [done]")
}

// TestProposeOnCommit starts three nodes and feeds commits back into the proposal
// channel. The intent is to ensure blocking on a proposal won't block raft progress.
func Test_Raft_ProposeOnCommit(t *testing.T) {
	clus := newCluster(3, t.TempDir())
	defer clus.closeNoErrors(t)

	donec := make(chan struct{})
	for i := range clus.peers {
		// feedback for "n" committed entries, then update donec
		go func(pC chan<- string, cC <-chan *commit, eC <-chan error) {
			for n := 0; n < 100; n++ {
				c, ok := <-cC
				if !ok {
					pC = nil
				}
				select {
				case pC <- c.data[0]:
					continue
				case err := <-eC:
					t.Errorf("eC message (%v)", err)
				}
			}
			donec <- struct{}{}
			for range cC {
				// acknowledge the commits from other nodes so
				// raft continues to make progress
			}
		}(clus.proposeC[i], clus.commitC[i], clus.errorC[i])

		// one message feedback per node
		go func(i int) { clus.proposeC[i] <- "foo" }(i)
	}

	for range clus.peers {
		<-donec
	}
}

// TestCloseProposerBeforeReplay tests closing the producer before raft starts.
func Test_Raft_CloseProposerBeforeReplay(t *testing.T) {
	clus := newCluster(1, t.TempDir())
	// close before replay so raft never starts
	defer clus.closeNoErrors(t)
}

// TestCloseProposerInflight tests closing the producer while
// committed messages are being published to the client.
func Test_Raft_CloseProposerInflight(t *testing.T) {
	clus := newCluster(1, t.TempDir())
	defer clus.closeNoErrors(t)

	// some inflight ops
	go func() {
		clus.proposeC[0] <- "foo"
		clus.proposeC[0] <- "bar"
	}()

	// wait for one message
	if c, ok := <-clus.commitC[0]; !ok || c.data[0] != "foo" {
		t.Fatalf("Commit failed")
	}
}

// TestAddNewNode tests adding new node to the existing cluster.
func Test_Raft_AddNewNode(t *testing.T) {
	clus := newCluster(3, t.TempDir())
	defer clus.closeNoErrors(t)

	os.RemoveAll("raftexample-4")
	os.RemoveAll("raftexample-4-snap")
	defer func() {
		os.RemoveAll("raftexample-4")
		os.RemoveAll("raftexample-4-snap")
	}()

	newNodeURL := "http://127.0.0.1:10004"
	clus.confChangeC[0] <- raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  4,
		Context: []byte(newNodeURL),
	}

	proposeC := make(chan string)
	defer close(proposeC)

	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	newRaftNode(4, append(clus.peers, newNodeURL), true, nil, proposeC, confChangeC, t.TempDir())

	go func() {
		proposeC <- "foo"
	}()

	if c, ok := <-clus.commitC[0]; !ok || c.data[0] != "foo" {
		t.Fatalf("Commit failed")
	}
}

func Test_Raft_Snapshot(t *testing.T) {
	prevDefaultSnapshotCount := defaultSnapshotCount
	prevSnapshotCatchUpEntriesN := snapshotCatchUpEntriesN
	defaultSnapshotCount = 4
	snapshotCatchUpEntriesN = 4
	defer func() {
		defaultSnapshotCount = prevDefaultSnapshotCount
		snapshotCatchUpEntriesN = prevSnapshotCatchUpEntriesN
	}()

	clus := newCluster(3, t.TempDir())
	defer clus.closeNoErrors(t)

	go func() {
		clus.proposeC[0] <- "foo"
	}()

	c := <-clus.commitC[0]

	select {
	case <-clus.snapshotTriggeredC[0]:
		t.Fatalf("snapshot triggered before applying done")
	default:
	}
	close(c.applyDoneC)
	<-clus.snapshotTriggeredC[0]
}
