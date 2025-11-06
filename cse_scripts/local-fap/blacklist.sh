

tiup ctl:v8.1.0 pd store -u 10.2.12.79:11003

tiup ctl:v8.1.0 pd store -u 10.2.12.79:11003 | jq '.stores[] | select(.store.labels[] | .key == "engine" and .value == "tiflash") | {store_id: .store.id, state: .store.state_name, address: .store.address}'

tiup ctl:v8.1.0 pd -u 10.2.12.79:11003 region 131


mysql --host 10.2.12.79 --port 11005 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 10.2.12.79 --port 11005 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 10.2.12.79 --port 11005 -u root -e "ALTER DATABASE narrow set tiflash replica 2;"
mysql --host 10.2.12.79 --port 11005 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 10.2.12.79 --port 11005 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"
mysql --host 10.2.12.79 --port 11005 -u root --comments -e "select * from information_schema.tiflash_replica;"

tcr3 -N 10.2.12.79:12021

tcs3

tiup cluster start calvin-cse-s3 -R pd
tiup cluster start calvin-cse-s3 -R tikv
curl -H "Content-Type: application/json" -d '{"name":"b"}' http://10.2.12.79:11003/pd/api/v2/keyspaces
tiup ctl:nightly pd -u http://127.0.0.1:11003 config set replication.max-replicas 1
tiup cluster start calvin-cse-s3


tcst3 -R tiflash
tiup cluster patch -y calvin-cse-s3 /data3/calvin_81/disk1/tiflash/cse/tiflash-cse/build/release/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tcs3 -R tiflash


cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash.log | grep "cklist peer remove"
cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash.log | grep "fake remote fail"
cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash.log | grep "remote_regions"


cd build/release && make tiflash -j40 && make install && cd install_tiflash && rm -rf tiflash/bin && tar -czvf tiflash.tar.gz ./tiflash && cd ../../..

tail -n 100 /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash.log
tail -n 100 /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash_tikv.log

sudo vim /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/conf/tiflash.toml
sudo cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/conf/tiflash.toml

cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/conf/tiflash.toml | grep "disagg_blacklist_wn_store_id"

mysql --host 10.2.12.79 --port 11005 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 10.2.12.79 --port 11005 -u root -e "ALTER DATABASE narrow set tiflash replica 1;"
mysql --host 10.2.12.79 --port 11005 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 10.2.12.79 --port 11005 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"

/data3/luorongzhen/tidb-deploy-s3/tiflash-12021/bin/tiflash/tiflash client --host 10.2.12.79 --port 12024 

DBGInvoke __enable_fail_point(force_remote_read_for_batch_cop)

DBGInvoke echo("1");

cat /data3/luorongzhen/tidb-deploy-s3/tiflash-12021/log/tiflash.log | grep "DBGInvoke"

/data4/luorongzhen/tidb-deploy/tiflash-5761/bin/tiflash/tiflash client --host 10.2.12.79 --port 5764

sudo scp dbms/src/Server/tiflash  root@10.2.12.79:/data3/luorongzhen/tidb-deploy-s3/tiflash-12021/bin/tiflash/tiflash
sudo scp contrib/tiflash-proxy-cmake/debug/libtiflash_proxy.so   root@10.2.12.79:/data3/luorongzhen/tidb-deploy-s3/tiflash-12021/bin/tiflash/libtiflash_proxy.so
sudo scp contrib/GmSSL/lib/libgmssld.so.3 root@10.2.12.79:/data3/luorongzhen/tidb-deploy-s3/tiflash-12021/bin/tiflash/libgmssld.so.3


