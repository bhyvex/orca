#!/bin/sh

cd /data
etcd -name etcd1 -listen-client-urls http://localhost:4001 -advertise-client-urls http://localhost:4001 -listen-peer-urls http://localhost:7001 -initial-advertise-peer-urls http://localhost:7001 -initial-cluster-token etcd-cluster-1 -initial-cluster 'etcd1=http://localhost:7001' -initial-cluster-state new &

/work/src/github.com/clusterit/orca/packaging/orcaman provider github $CLIENTID $CLIENTSECRET
/work/src/github.com/clusterit/orca/packaging/orcaman admins github:$USERID 
/work/src/github.com/clusterit/orca/packaging/orcaman serve 2>&1 >orcaman.logs &
sleep 3
/work/src/github.com/clusterit/orca/packaging/sshgw serve 2>&1 >sshgw.logs 

