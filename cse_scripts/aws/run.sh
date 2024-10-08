# terraform 准备
# 也就是使用
# https://github.com/CalvinNeo/terraform-cloud-native-tiflash/tree/serverless-dev

git checkout serverless-dev
terraform init
terraform apply -auto-approve

# 开发机

export U=ubuntu@54.212.203.10

scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.parallel.limiter.tar.gz $U:/home/ubuntu/tiflash.parallel.limiter.tar.gz
scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz $U:/home/ubuntu/tiflash.tar.gz

scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz $U:/home/ubuntu/tiflash.tar.gz
scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz $U:/home/ubuntu/pd.tar.gz
scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz $U:/home/ubuntu/tidb.tar.gz
scp calvin@10.2.12.81:/DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz $U:/home/ubuntu/tikv.tar.gz
scp calvin@10.2.12.81:/data1/calvin/bin/br-cse $U:/home/ubuntu/br-cse


scp $idc81:/DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz $U:/home/ubuntu/tiflash.tar.gz
scp $idc81:/DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz $U:/home/ubuntu/pd.tar.gz
scp $idc81:/DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz $U:/home/ubuntu/tidb.tar.gz
scp $idc81:/DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz $U:/home/ubuntu/tikv.tar.gz
scp $idc81:/data1/calvin/bin/br-cse $U:/home/ubuntu/br-cse


# 已经登录
scp /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz $U:/home/ubuntu/tiflash.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz $U:/home/ubuntu/pd.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz $U:/home/ubuntu/tidb.tar.gz
scp /DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz $U:/home/ubuntu/tikv.tar.gz
scp /data1/calvin/bin/br-cse $U:/home/ubuntu/br-cse


# 直接传到 S3
aws s3 cp /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz s3://yunyantest/calvin/fap/tiflash.tar.gz
aws s3 cp /DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz s3://yunyantest/calvin/fap/tidb.tar.gz
aws s3 cp /DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz s3://yunyantest/calvin/fap/pd.tar.gz
aws s3 cp /DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz s3://yunyantest/calvin/fap/tikv.tar.gz
aws s3 cp /data1/calvin/bin/br-cse s3://yunyantest/calvin/fap/br-cse

# 后续版本
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz s3://yunyantest/calvin/fap/tiflash14.tar.gz
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/tidb-cse/bin/tidb.tar.gz s3://yunyantest/calvin/fap/tidb2.tar.gz
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/pd-cse/bin/pd.tar.gz s3://yunyantest/calvin/fap/pd3.tar.gz
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/pd-cse/bin/pd-ctl s3://yunyantest/calvin/fap/pd-ctl2
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz s3://yunyantest/calvin/fap/tikv3.tar.gz
aws s3 cp /data3/calvin_81/disk1/tiflash/cse/cloud-storage-engine/target/release/tikv-worker s3://yunyantest/calvin/fap/tikv-worker


# 从 S3 下载
aws s3 cp s3://yunyantest/calvin/fap/tiflash.tar.gz tiflash.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/tidb.tar.gz tidb.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/pd.tar.gz pd.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/tikv.tar.gz tikv.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/br-cse br-cse
aws s3 cp s3://yunyantest/calvin/fap/tikv-worker tikv-worker
scp tikv-worker 172.31.6.1:~/tikv-worker

nohup ./tikv-worker -C tikv-worker.toml --pd-endpoints 172.31.8.1:2379 &

# 后续版本
rm tiflash.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/tiflash14.tar.gz tiflash.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/tidb2.tar.gz tidb.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/pd3.tar.gz pd.tar.gz
aws s3 cp s3://yunyantest/calvin/fap/pd-ctl2 pd-ctl
aws s3 cp s3://yunyantest/calvin/fap/tikv3.tar.gz tikv.tar.gz
chmod 777 pd-ctl
tiup cluster patch -y test tiflash.tar.gz --overwrite -R tiflash


# 中控机
# 需要先配置 ~/.ssh/authorized_keys

tiup cluster destroy test -y

# 必须 patch，否则是 tikv 的代码

tiup cluster deploy -y test v8.0.0 topology.yaml --native-ssh=1 --ignore-config-check
tiup cluster patch -y test tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y test pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y test tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y test tikv.tar.gz --overwrite --offline -R tikv


tiup cluster start test -R pd
tiup cluster start test -R tikv
curl -H "Content-Type: application/json" -d '{"name":"b", "config":{"encryption":"{\"enabled\":true}"}}' http://172.31.8.1:2379/pd/api/v2/keyspaces
# ./pd-ctl -u 172.31.8.1:2379 keyspace create b --config encryption="{'enabled': true}"
tiup ctl:v7.5.0 pd -u http://172.31.8.1:2379 config set replication.max-replicas 1

tiup cluster start test -R tidb
tiup cluster start test 

curl -X POST http://172.31.8.1:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"a"}'


tiup cluster restart test -y

tiup cluster patch -y test tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y test tiflash.tar.gz --overwrite -R tiflash
tiup cluster patch -y test tiflash.parallel.500.tar.gz --overwrite -R tiflash


# 天生两副本
./br-cse restore db --db=chbenchmark --s3.region=ap-northeast-2 --storage "s3://yunyantest/chbenmark-1500" --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true

./br-cse restore db --db=rtdb --s3.region=ap-northeast-2 --storage "s3://yunyantest/luorongzhen/rtdb-100m-with-pk-info" --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true

./br-cse restore db --db=test --storage=s3://qa-workload-datasets/benchmark/tpch50 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true


mysql --host 172.31.7.1 --port 4000 -u root -e "alter database chbenchmark set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database chbenchmark set tiflash replica 2;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[chbenchmark.stock]) */ count(*) from chbenchmark.stock;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[chbenchmark.stock]) */ count(*) from chbenchmark.stock;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[chbenchmark.orders]) */ count(*) from chbenchmark.orders;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[chbenchmark.orders]) */ count(*) from chbenchmark.orders;"


mysql --host 172.31.7.1 --port 4000 -u root -e "alter database test set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database test set tiflash replica 2;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.customer]) */ count(*) from test.customer;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.customer]) */ count(*) from test.customer;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.lineitem]) */ count(*) from test.lineitem;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.lineitem]) */ count(*) from test.lineitem;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.nation]) */ count(*) from test.nation;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.nation]) */ count(*) from test.nation;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.orders]) */ count(*) from test.orders;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.orders]) */ count(*) from test.orders;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.part]) */ count(*) from test.part;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.part]) */ count(*) from test.part;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.partsupp]) */ count(*) from test.partsupp;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.partsupp]) */ count(*) from test.partsupp;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.region]) */ count(*) from test.region;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.region]) */ count(*) from test.region;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[test.supplier]) */ count(*) from test.supplier;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[test.supplier]) */ count(*) from test.supplier;"

mysql --host 172.31.7.1 --port 4000 -u root -e "alter database rtdb set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database tpcc set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "alter database test set tiflash replica 1;"

tiup cluster stop test -R tiflash -y
tiup cluster start test -R tiflash -y

tiup cluster restart test -R tiflash -y


# 改变测试环境
# 禁用 fap
ssh 172.31.9.1
ssh 172.31.9.2
sudo chmod 777 /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = false/enable-fast-add-peer = true/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml


# sum(tiflash_fap_task_result{k8s_cluster="$k8s_cluster", tidb_cluster="$tidb_cluster", instance=~"$instance"}) by (type)
# sum(tiflash_fap_task_state{k8s_cluster="$k8s_cluster", tidb_cluster="$tidb_cluster", instance=~"$instance"}) by (type)
# sum(tiflash_system_current_metric_RaftNumPrehandlingSubTasks{k8s_cluster="$k8s_cluster", tidb_cluster="$tidb_cluster", instance=~"$instance"}) by (instance)