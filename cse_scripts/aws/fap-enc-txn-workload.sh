
ssh 172.31.6.2 'cat /tidb-deploy/tikv-20160/log/tikv.log | grep "239, 206, 254, 127"'
ssh 172.31.6.1 'cat /tidb-deploy/tikv-20160/log/tikv.log | grep "encryption" | grep "304" | grep "loading"'
ssh 172.31.6.1 'cat /tidb-deploy/tikv-20160/log/tikv.log | grep ":199"'

ssh 172.31.7.1 'cat /tidb-deploy/tidb-4000/log/tidb.log | grep "tiflash"'

ssh 172.31.7.1 'cat /tidb-deploy/tidb-4000/log/tidb.log | grep "Configure"'


 ssh 172.31.9.1 'cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep "encryption" | grep "304" | grep "loading"'
 ssh 172.31.9.2 'cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep "encryption" | grep "304" | grep "loading"'

 ssh 172.31.9.2 'tail /tidb-deploy/tiflash-9000/log/tiflash_tikv.log'

 ssh 172.31.9.2 'cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep "pb_snapshot"'
 ssh 172.31.9.2 'cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep "put snapshot txn_file_locks"'


set tidb_enable_txn_file ='on';
CREATE DATABASE narrow;
CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins TEXT,ts int,r TEXT,s TEXT,v int,incr int);
ALTER DATABASE narrow set tiflash replica 1;
insert into narrow.t (ins, ts, r, s, v, incr) values (REPEAT('aaaaaaaaaaaaa', 100), 1, REPEAT('aaaaaaaaaaaaa', 100), REPEAT('aaaaaaaaaaaaa', 100), 8, 9);


insert into narrow.t (ins, ts, r, s, v, incr) values ('a', 1, 'a', 'a', 8, 9);

begin;
insert into narrow.t (ins, ts, r, s, v, incr) values (REPEAT('aaaaaaaaaaaaa', 1000), 1, REPEAT('aaaaaaaaaaaaa', 1000), REPEAT('aaaaaaaaaaaaa', 1000), 8, 9);

 ssh 172.31.9.1 'cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep "put snapshot txn_file_lock"'
 insert into narrow.t (select * from narrow.t);



 mysql --host 172.31.7.1 --port 4000 -u root 

ssh 172.31.6.1 'tail /tidb-deploy/tikv-20160/log/tikv.log'

while True; do mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (select * from narrow.t limit 3000);"; sleep 1; done
