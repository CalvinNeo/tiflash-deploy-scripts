mysql --host 10.2.12.81 --port 5711 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 10.2.12.81 --port 5711 -u root -e "ALTER DATABASE narrow set tiflash replica 1;"
repeat 300 mysql --host 10.2.12.81 --port 5711 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"

mysql --host 10.2.12.81 --port 5711 -u root -e "ALTER DATABASE narrow set tiflash replica 0;"


mysql --host 10.2.12.81 --port 5711 -u root -e "CREATE TABLE narrow2.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 10.2.12.81 --port 5711 -u root -e "ALTER DATABASE narrow2 set tiflash replica 1;"
repeat 10 mysql --host 10.2.12.81 --port 5711 -u root -e "insert into narrow2.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"

mysql --host 10.2.12.81 --port 5711 -u root -e "select * from information_schema.tiflash_replica;"


mysql --host 10.2.12.79 --port 5711 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 10.2.12.79 --port 5711 -u root -e "ALTER DATABASE narrow set tiflash replica 1;"
mysql --host 10.2.12.79 --port 5711 -u root -e "ALTER DATABASE narrow set tiflash replica 0;"
repeat 300 mysql --host 10.2.12.79 --port 5711 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
repeat 10000 mysql --host 10.2.12.79 --port 5711 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
repeat 1000 mysql --host 10.2.12.79 --port 5711 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"

mysql --host 10.2.12.79 --port 5711 -u root -e "select * from information_schema.tiflash_replica;"
mysql --host 10.2.12.79 --port 5711 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"



mysql --host 172.31.7.1 --port 4000 -u root -e "CREATE DATABASE narrow; CREATE TABLE narrow.t(id int NOT NULL AUTO_INCREMENT,ins VARCHAR(4),ts int,r VARCHAR(4),s VARCHAR(4),v int,incr int);"
mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 172.31.7.1 --port 4000 -u root -e "insert into narrow.t (ins, ts, r, s, v, incr) values ('aaa', 1, 'bbb', 'ccc', 8, 9);"
mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER DATABASE narrow set tiflash replica 1;"
mysql --host 172.31.7.1 --port 4000 -u root -e "select count(*) from narrow.t;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+ read_from_storage(tiflash[narrow.t]) */ count(*) from narrow.t;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select * from information_schema.tiflash_replica;"
mysql --host 172.31.7.1 --port 4000 -u root -e "ALTER DATABASE narrow set tiflash replica 2;"

