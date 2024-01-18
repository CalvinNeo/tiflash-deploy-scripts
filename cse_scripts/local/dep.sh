
export PATH=$PATH:/data1/calvin/bin

sh burn.sh

export PATH=$PATH:/data1/calvin/bin
mkdir /data3/luorongzhen/s3_data/


minio server /data3/luorongzhen/s3_data/ --console-address "0.0.0.0:11998" --address "0.0.0.0:11999" > /dev/nul &

mc config host add localminio http://127.0.0.1:11999 minioadmin minioadmin

set -euxo pipefail
# set -ex

mc mb localminio/tiflash-cse-s3
mc mb localminio/tikv-cse-s3

tiup cluster deploy -y calvin-cse-s3 v6.6.0 /DATA/disk1/calvin/tiflash/cse/topology.yaml --skip-create-user --native-ssh=1 --ignore-config-check

if [[ $MOD -eq "release" ]]
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/release/tikv.tar.gz --overwrite --offline -R tikv
else
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/debug/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/pd-cse/bin/pd.tar.gz --overwrite --offline -R pd
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/tidb-cse/bin/tidb.tar.gz --overwrite --offline -R tidb
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/cloud-storage-engine/target/debug/tikv.tar.gz --overwrite --offline -R tikv
fi

tiup cluster start calvin-cse-s3 -R pd
tiup cluster start calvin-cse-s3 -R tikv
tiup ctl:v6.5.2 pd -u http://127.0.0.1:11003 config set replication.max-replicas 1
# 如果 pd 配置文件中有 pre-alloc 提前分配租户，这里就不用再手动创建了
# curl -X POST http://localhost:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"tenant-1"}'

tiup cluster start calvin-cse-s3

# sh make_data.sh