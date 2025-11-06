
export PATH=$PATH:/data1/calvin/bin
sh burn.sh

// 下面是 burn.sh 的内容
tiup cluster destroy calvin-cse-s3 -y
sudo rm -rf /data3/luorongzhen/s3_data/
kill `ps aux | grep "minio server /data3/luorongzhen/s3_data/" | grep -v grep | awk '{print $2}' `

tcc3

export PATH=$PATH:/data2/calvin_81/c
sudo mkdir -p /data3/luorongzhen/s3_data/
sudo chmod 777 /data3/luorongzhen/s3_data/
minio server /data3/luorongzhen/s3_data/ --console-address "0.0.0.0:11998" --address "0.0.0.0:11999" > /dev/null &
sleep 2
mc config host add localminio http://127.0.0.1:11999 minioadmin minioadmin
set -euxo pipefail
# set -ex
mc mb localminio/tiflash-cse-s3
mc mb localminio/tikv-cse-s3

tiup cluster deploy -y calvin-cse-s3 7.5.0 /data2/calvin_81/topos/topology.yaml.81.fap --skip-create-user --ignore-config-check

if [[ $MOD -eq "release" ]]
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tiflash-cse/rel/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/cloud-storage-engine/target/release/tikv.tar.gz --overwrite --offline -R tikv
else
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tiflash-cse/debug/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/cloud-storage-engine/target/debug/tikv.tar.gz --overwrite --offline -R tikv
fi

tiup cluster start calvin-cse-s3 -R pd
tiup cluster start calvin-cse-s3 -R tikv
tiup ctl:nightly pd -u http://127.0.0.1:11003 config set replication.max-replicas 1
# 如果 pd 配置文件中有 pre-alloc 提前分配租户，这里就不用再手动创建了
# curl -X POST http://localhost:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"a"}'

tiup cluster start calvin-cse-s3

# sh make_data.sh

# 下面两个数据集没啥用

/DATA/disk2/calvin_81/mess/bin/br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-700m --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 10.2.12.81:11003 --keyspace-name a --leader-download=true

/DATA/disk2/calvin_81/mess/bin/br-cse restore db --db=narrow --storage=s3://calvin-west-2/table/narrow-bigint-2.6b --s3.region=us-west-2 --send-credentials-to-tikv=false --check-requirements=false --pd 10.2.12.81:11003 --keyspace-name a --leader-download=true


mysql --host 10.2.12.81 --port 11005 -u root -e "select * from information_schema.tiflash_replica;"

mysql --host 10.2.12.81 --port 11005 -u root -e "select * from information_schema.tiflash_tables;"

mysql --host 10.2.12.81 --port 11005 -u root -e "select * from narrow.t;"

mysql --host 10.2.12.81 --port 11005 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"

mysql --host 10.2.12.81 --port 11005 -u root -e "alter database tpcc set tiflash replica 2;"
mysql --host 10.2.12.81 --port 11005 -u root -e "alter database tpcc set tiflash replica 1;"
mysql --host 10.2.12.81 --port 11005 -u root -e "alter database tpcc set tiflash replica 0;"


mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[customer]) */ count(*) from customer;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[customer]) */ count(*) from customer;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[district]) */ count(*) from district;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[district]) */ count(*) from district;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[history]) */ count(*) from history;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[history]) */ count(*) from history;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[item]) */ count(*) from item;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[item]) */ count(*) from item;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[new_order]) */ count(*) from new_order;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[new_order]) */ count(*) from new_order;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[order_line]) */ count(*) from order_line;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[order_line]) */ count(*) from order_line;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[orders]) */ count(*) from orders;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[orders]) */ count(*) from orders;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[stock]) */ count(*) from stock;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[stock]) */ count(*) from stock;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tikv[warehouse]) */ count(*) from warehouse;"
mysql --host 10.2.12.81 --port 11005 -u root --comments -e "use tpcc; select /*+ read_from_storage(tiflash[warehouse]) */ count(*) from warehouse;"

tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tiflash-cse/rel/install_tiflash/tiflash.tar.gz --overwrite -R tiflash
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tiflash-cse/dbg-install/install_tiflash/tiflash.tar.gz --overwrite -R tiflash
