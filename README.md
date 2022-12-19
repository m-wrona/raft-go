# raft-go

Experiments using consensus algorithms, focusing on Raft. 

## Assumptions

**Scope:**

* service composed of N nodes (set-up in `docker-compose` & `kubernetes`)

* each service is using `Raft` consensus algorithm to find the leader

* leader handles basic API operations using `GRPC`

**Goals:**

* check in details how `Raft` works

* check performance of such service

* check how resilient such service is and what's the impact of failures on service performance

## Raft

Sample is using `etcd` implementation of `Raft` algorithm:

* https://github.com/etcd-io/raft

* https://github.com/etcd-io/etcd/tree/main/contrib/raftexample


## Other docs

* [Consensus algorithms in theory & practice](https://raft.github.io/raft.pdf)

* [Raft visualisation](https://raft.github.io)

* [Raft visualisation step-by-step](http://thesecretlivesofdata.com/raft/)

* [Swarmkit - toolkit for service orchestration](https://github.com/moby/swarmkit)
