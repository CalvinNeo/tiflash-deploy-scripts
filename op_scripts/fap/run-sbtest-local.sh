tiup cluster clean calvin-test --all -y
sudo cp /DATA/disk1/calvin/tiflash/tics/rel/dbms/src/Server/tiflash  /data4/luorongzhen/tidb-deploy/tiflash-5761/bin/tiflash/tiflash
sudo cp /DATA/disk1/calvin/tiflash/tics/rel/contrib/tiflash-proxy-cmake/release/libtiflash_proxy.so /data4/luorongzhen/tidb-deploy/tiflash-5761/bin/tiflash/libtiflash_proxy.so
sudo cp /DATA/disk1/calvin/tiflash/tics/rel/contrib/GmSSL/lib/libgmssl.so.3 /data4/luorongzhen/tidb-deploy/tiflash-5761/bin/tiflash/libgmssl.so.3
tiup cluster start calvin-test
mysql --host 10.2.12.81 --port 5711 -e'create database sbtest;'
timeout 300s sysbench --config-file=/DATA/disk1/calvin/calvin_test/config oltp_update_index --tables=1 --table-size=100000000 --db-ps-mode=auto --rand-type=uniform prepare
mysql --host 10.2.12.81 --port 5711 -e'select count(*) from sbtest.sbtest1;'
mysql --host 10.2.12.81 --port 5711 -e'alter database sbtest set tiflash replica 1;'