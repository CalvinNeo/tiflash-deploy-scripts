sudo apt update
sudo apt install lrzsz
sudo apt install mysql-client-core-8.0
sudo apt install build-essential
sudo apt install pkg-config
sudo apt install openssl
sudo apt install libssl-dev

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
. "$HOME/.cargo/env"
git clone https://github.com/CalvinNeo/pressure.git
pushd pressure/pressure/
cargo build --release
popd

mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"

mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"


./pressure/pressure/target/release/pressure free-issue --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/ --batch-size 10 --workers 10  --tasks 200
./pressure/pressure/target/release/pressure free-issue --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/ --batch-size 10 --workers 10  --tasks 400


mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER TABLE narrow.t SET TIFLASH REPLICA 1;"


mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tikv[narrow.t]) */ count(*) from narrow.t;"


mysql --host 172.31.7.1 --port 4000 -u root -e "select * from information_schema.tiflash_replica;"

./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-15k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a
./br-cse backup db --db=narrow --storage=s3://calvin-west-2/table/narrow-24k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a

./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-7.5k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-15k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true
./br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-24k --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 172.31.8.1:2379 --keyspace-name a --leader-download=true


mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER TABLE narrow.t SET TIFLASH REPLICA 2;"

# 改变测试环境
# 禁用 fap
ssh 172.31.9.1
ssh 172.31.9.2
sudo chmod 777 /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = false/enable-fast-add-peer = true/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml