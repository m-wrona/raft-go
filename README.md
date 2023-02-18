# raft-go

Sample `GRPC` service prepared for embedded devices & IoT where orchestration (like k8s) is not present.

Service is using `etcd` implementation of `Raft` algorithm (https://github.com/etcd-io/raft).

Prerequisites:

* Go 1.19
* GRPC for Go (check `make install`)
* [Goreman](https://github.com/mattn/goreman)

## Assumptions

**Scope:**

* embedded service composed of N nodes  

* each service is using `Raft` consensus algorithm to find the leader

* leader handles basic API operations using `GRPC`

**Goals:**

* check in details how `Raft` works

* check performance of such service

* check how resilient such service is and what's the impact of failures on service performance

## Design

Design is borrowed from [raft example](https://github.com/etcd-io/etcd/tree/main/contrib/raftexample) which is based on REST API

and fits GRPC needs perfectly too. 

The service consists of three components:
* a raft-backed key-value store, 
* a GRPC server 
* a raft consensus server based on etcd's raft implementation.

The `raft-backed key-value store` is a key-value map that holds all committed key-values.
The store bridges communication between the raft server and the GRPC server.
Key-value updates are issued through the store to the raft server.
The store updates its map once raft reports the updates are committed.

The `GRPC server` exposes the current raft consensus by accessing the raft-backed key-value store.

The `raft server` participates in consensus with its cluster peers.
When the GRPC server submits a proposal, the raft server transmits the proposal to its peers.
When raft reaches a consensus, the server publishes all committed updates over a commit channel.
In our case, this commit channel is consumed by the key-value store.

## Other docs

* [Consensus algorithms in theory & practice](https://raft.github.io/raft.pdf)

* [Raft visualisation](https://raft.github.io)

* [Raft visualisation step-by-step](http://thesecretlivesofdata.com/raft/)

* [Swarmkit - toolkit for service orchestration](https://github.com/moby/swarmkit)
