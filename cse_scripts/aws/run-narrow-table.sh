sudo apt remove needrestart
sudo apt update
sudo apt install lrzsz -y
sudo apt install mysql-client-core-8.0 -y
sudo apt install build-essential -y
sudo apt install pkg-config -y
sudo apt install openssl -y
sudo apt install libssl-dev -y

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
. "$HOME/.cargo/env"
git clone https://github.com/CalvinNeo/pressure.git
pushd pressure/pressure/
cargo build --release
popd

mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"

mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"

mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE TABLE narrow.t(id bigint NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 172.31.7.1 --port 4000 -u root -e "INSERT INTO narrow.t2 (SELECT * FROM narrow.t);"



./pressure/pressure/target/release/pressure free-issue --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/ --batch-size 10 --workers 10  --tasks 200
./pressure/pressure/target/release/pressure free-issue --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/ --batch-size 10 --workers 10  --tasks 400


mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER TABLE narrow.t SET TIFLASH REPLICA 1;"


mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[narrow.t]) */ count(*) from narrow.t;"


mysql --host 172.31.7.1 --port 4000 -u root -e "select * from information_schema.tiflash_replica;"

./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-700m --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-1.1b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-1.6b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-2.6b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-3b-rep0 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-4.2b-rep0 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a


./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-1.6b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# br 弄不动
./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-2.6b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-3b-rep0 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true


mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER TABLE narrow.t SET TIFLASH REPLICA 0;"


# # 叫 15k，实际上是 150m
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-15k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-24k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-58k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-58k-rep1 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-78k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
# ./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-100k-rep1 --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a


# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-7.5k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-15k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-24k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-58k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
# ./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-78k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true


mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER TABLE narrow.t SET TIFLASH REPLICA 2;"

# 改变测试环境
# 禁用 fap
ssh 172.31.9.1
ssh 172.31.9.2
sudo chmod 777 /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = false/enable-fast-add-peer = true/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml


ssh 172.31.9.1 "sudo sed -i 's/fap_task_timeout_seconds = .*/fap_task_timeout_seconds = 240/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"
ssh 172.31.9.2 "sudo sed -i 's/fap_task_timeout_seconds = .*/fap_task_timeout_seconds = 240/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"

ssh 172.31.9.1 "sudo sed -i 's/fap_task_timeout_seconds = 150/fap_task_timeout_seconds = 120/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"
ssh 172.31.9.2 "sudo sed -i 's/fap_task_timeout_seconds = 150/fap_task_timeout_seconds = 120/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"

ssh 172.31.9.1 "sudo sed -i -E 's/fap_handle_concurrency = .*/fap_handle_concurrency = 25/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"
ssh 172.31.9.2 "sudo sed -i -E 's/fap_handle_concurrency = .*/fap_handle_concurrency = 25/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"

ssh 172.31.9.1 "sudo sed -i -E 's/fap_handle_concurrency = .*/fap_handle_concurrency = 10/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"
ssh 172.31.9.2 "sudo sed -i -E 's/fap_handle_concurrency = .*/fap_handle_concurrency = 10/' /tidb-deploy/tiflash-9000/conf/tiflash.toml"


ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/conf/tiflash.toml"
ssh 172.31.9.2 "cat /tidb-deploy/tiflash-9000/conf/tiflash.toml"

ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"
ssh 172.31.9.2 "cat /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"

ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep FastAddPeer | tail -n 100"
ssh 172.31.9.2 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep FastAddPeer | tail -n 100"


ssh 172.31.9.2 "cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep remove_cached_region_info"

ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep Parallel"
ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep 'Transformed snapshot in SSTFile to DTFiles'"


ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep 'STFilesToBlockInputStream C' | grep 12:30"

cat /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml


vi /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml

```
[engine-store]
enable-fast-add-peer = true
```