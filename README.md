# raft-fuzzing
Model coverage guided fuzzing of etcd raft

This is a forked repo for testing the method in the paper[Model-guided Fuzzing of Distributed Systems](https://arxiv.org/abs/2410.02307) by Ege Berkay Gulcan, Burcu Kulahcioglu Ozkan, Rupak Majumdar, Srinidhi Nagendra.

# How to use

## The Raft test in the original paper

1. Build and start the TLC server:

    https://github.com/weng-chenghui/tlc-server-docker

    Running by the script in that repository to specify the right model:
    ./run.sh ./example/tla-benchmarks/Raft/model /model/RAFT_3_3.tla

2. Then build and run the command for the original etcd-fuzzing test:

    make build && ./bin/etcd-fuzzer compare