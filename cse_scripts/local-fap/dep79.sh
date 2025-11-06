
export PATH=$PATH:/data1/calvin/bin
sh burn.sh

export PATH=$PATH:/data1/calvin/bin
sudo mkdir -p /data3/luorongzhen/s3_data/
sudo chmod 777 /data3/luorongzhen/s3_data/
nohup minio server /data3/luorongzhen/s3_data/ --console-address "0.0.0.0:11998" --address "0.0.0.0:11999" > a.log &
sleep 2
/DATA/disk1/ra_common/minio/bin/mc config host add localminio http://127.0.0.1:11999 minioadmin minioadmin
# set -euxo pipefail
# set -ex
/DATA/disk1/ra_common/minio/bin/mc mb localminio/tiflash-cse-s3
/DATA/disk1/ra_common/minio/bin/mc mb localminio/tikv-cse-s3

sudo lsof -i:11998

tiup cluster deploy -y calvin-cse-s3 7.5.0 fap79.yaml --skip-create-user --ignore-config-check

if [[ $MOD -eq "release" ]]
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz --overwrite --offline -R tikv
else
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/tiflash-cse/build/debug/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz --overwrite --offline -R tikv
fi

tiup cluster start calvin-cse-s3 -R pd
tiup cluster start calvin-cse-s3 -R tikv
curl -H "Content-Type: application/json" -d '{"name":"b"}' http://10.2.12.79:11003/pd/api/v2/keyspaces
tiup ctl:nightly pd -u http://127.0.0.1:11003 config set replication.max-replicas 1
# 如果 pd 配置文件中有 pre-alloc 提前分配租户，这里就不用再手动创建了
# curl -X POST http://localhost:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"a"}'

tiup cluster start calvin-cse-s3

# sh make_data.sh
