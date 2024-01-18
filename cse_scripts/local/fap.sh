mysql --host 127.0.0.1 --port 11005 -u root -e "alter table test.t set tiflash replica 1;"
mysql --host 127.0.0.1 --port 11005 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 127.0.0.1 --port 11005 -u root -e "show table tpcc.orders regions;"
sleep 10
sleep 10
sleep 10
mysql --host 127.0.0.1 --port 11005 -u root -e "alter table test.t set tiflash replica 2;"

mysql --host 127.0.0.1 --port 11005 -u root --comments -e "select /*+read_from_storage(tiflash[test.t])*/  count(*) from test.t;"


mysql --host 127.0.0.1 --port 11005 -u root -e "alter database tpcc set tiflash replica 1;"
mysql --host 127.0.0.1 --port 11005 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 127.0.0.1 --port 11005 -u root -e "alter database tpcc set tiflash replica 2;"

mysql --host 127.0.0.1 --port 11005 -u root --comments -e "select /*+read_from_storage(tiflash[tpcc.orders])*/  count(*) from tpcc.orders;"