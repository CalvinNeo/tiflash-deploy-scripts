
export PATH=$PATH:/data3/calvin_81/disk1/bin
sh burn.sh

tcc3

export PATH=$PATH:/data3/calvin_81/disk1/bin
sudo mkdir -p /data3/luorongzhen/disagg_data/
sudo chmod 777 /data3/luorongzhen/disagg_data/
minio server /data3/luorongzhen/disagg_data/ --console-address "0.0.0.0:12001" --address "0.0.0.0:12002" > /dev/null &
sleep 2
mc config host add localminio http://127.0.0.1:12002 minioadmin minioadmin
set -euxo pipefail
# set -ex
mc mb localminio/tiflash-disagg
mc mb localminio/tiflash-disagg

tiup cluster deploy -y tiflash-disagg 7.5.0 /data3/calvin_81/disk1/tiflash/cse/topology.yaml --skip-create-user --ignore-config-check


sudo cp dbms/src/Server/tiflash  /data4/luorongzhen/tidb-deploy/tiflash-5358/bin/tiflash/tiflash
sudo cp contrib/tiflash-proxy-cmake/debug/libtiflash_proxy.so   /data4/luorongzhen/tidb-deploy/tiflash-5358/bin/tiflash/libtiflash_proxy.so
sudo cp contrib/GmSSL/lib/libgmssld.so.3   /data4/luorongzhen/tidb-deploy/tiflash-5358/bin/tiflash/libgmssld.so.3

sudo cp dbms/src/Server/tiflash  /data4/luorongzhen/tidb-deploy/tiflash-5365/bin/tiflash/tiflash
sudo cp contrib/tiflash-proxy-cmake/debug/libtiflash_proxy.so   /data4/luorongzhen/tidb-deploy/tiflash-5365/bin/tiflash/libtiflash_proxy.so
sudo cp contrib/GmSSL/lib/libgmssld.so.3   /data4/luorongzhen/tidb-deploy/tiflash-5365/bin/tiflash/libgmssld.so.3

tiup cluster start tiflash-disagg -R pd
tiup cluster start tiflash-disagg -R tikv
tiup ctl:nightly pd -u http://127.0.0.1:11003 config set replication.max-replicas 1
# 如果 pd 配置文件中有 pre-alloc 提前分配租户，这里就不用再手动创建了
# curl -X POST http://localhost:2379/pd/api/v2/keyspaces -H 'Content-Type: application/json' -d '{"name":"a"}'

tiup cluster start tiflash-disagg

# sh make_data.sh