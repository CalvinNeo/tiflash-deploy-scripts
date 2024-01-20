
tiup cluster deploy -y test v6.6.0 topology.yaml --native-ssh=1 --ignore-config-check

scp /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz ubuntu@35.165.12.34:/home/ubuntu/tiflash.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz ubuntu@35.165.12.34:/home/ubuntu/pd.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz ubuntu@35.165.12.34:/home/ubuntu/tidb.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz ubuntu@35.165.12.34:/home/ubuntu/tikv.tar.gz
scp /data1/calvin/bin/br-cse ubuntu@35.165.12.34:/home/ubuntu/br-cse

tiup cluster destroy test -y

tiup cluster deploy -y test v6.6.0 topology.yaml --native-ssh=1 --ignore-config-check

tiup cluster patch -y test tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y test pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y test tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y test tikv.tar.gz --overwrite --offline -R tikv


tiup cluster start test -R pd
tiup cluster start test -R tikv
tiup ctl:v6.5.2 pd -u http://172.31.8.1:2379 config set replication.max-replicas 1

tiup cluster start test 

curl -X POST http://172.31.8.1:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"a"}'

./br-cse restore db --db=chbenchmark --s3.region=ap-northeast-2 --storage "s3://yunyantest/chbenmark-1500" --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true

./br-cse restore db --db=rtdb --s3.region=ap-northeast-2 --storage "s3://yunyantest/luorongzhen/rtdb-100m-with-pk-info" --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true

./br-cse restore db --db=test --storage=s3://qa-workload-datasets/benchmark/tpch50 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true


mysql --host 172.31.7.1 --port 4000 -u root -e "alter database chbenchmark set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database chbenchmark set tiflash replica 2;"


mysql --host 172.31.7.1 --port 4000 -u root -e "alter database rtdb set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database tpcc set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database test set tiflash replica 1;"

# sum(tiflash_fap_task_result{k8s_cluster="$k8s_cluster", tidb_cluster="$tidb_cluster", instance=~"$instance"}) by (type)