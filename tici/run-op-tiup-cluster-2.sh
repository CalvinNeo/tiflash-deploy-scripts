ps aux | grep "minio server /data3/luorongzhen/s3_data/"

tiup cluster deploy ctici nightly topology.yaml.0.another.tici

tiup cluster deploy ctici nightly --tls topology.yaml.0.another.tici


# into ~/.tiup/data/ctici/tiflash-0/config.toml

tcc ctici
sudo rm -rf /data3/luorongzhen/s3_data/
kill `ps aux | grep "minio server /data3/luorongzhen/s3_data/" | grep -v grep | awk '{print $2}' `

export PATH=$PATH:/data2/calvin_81/c
sudo mkdir -p /data3/luorongzhen/s3_data/
sudo chmod 777 /data3/luorongzhen/s3_data/
minio server /data3/luorongzhen/s3_data/ --console-address "0.0.0.0:11998" --address "0.0.0.0:11999" > /dev/null &
mc mb localminio/logbucket

sudo cp /data2/calvin_81/c/tici/private-tidb/bin/tidb-server  /DATA/disk4/luorongzhen/tidb-deploy/tidb-5811/bin/tidb-server
sudo cp /data2/calvin_81/c/tici/private-tiflash/dbg/dbms/src/Server/tiflash  /DATA/disk4/luorongzhen/tidb-deploy/tiflash-5861/bin/tiflash/tiflash
sudo cp /data2/calvin_81/c/tici/private-tiflash/dbg/contrib/tiflash-proxy-cmake/debug/libtiflash_proxy.so  /DATA/disk4/luorongzhen/tidb-deploy/tiflash-5861/bin/tiflash/libtiflash_proxy.so

vim /DATA/disk4/luorongzhen/tidb-deploy/tiflash-5861/conf/tiflash.toml

netstat -tulnp | grep pdpid

# 选择第二小的，或者

select * from information_schema.cluster_info;



```
[s3]
endpoint = "http://localhost:11999"
region = "us-east-1"
access_key = "minioadmin"
secret_key = "minioadmin"
bucket = "logbucket"
use_path_style = true
```

tiup cdc server --addr=127.0.0.1:8559 --pd=http://127.0.0.1:5845

tiup cdc cli changefeed create --server=http://127.0.0.1:8559 --sink-uri="s3://logbucket/storage_test?protocol=canal-json&access-key=minioadmin&secret-access-key=minioadmin&endpoint=http://127.0.0.1:11999"

# Build the project
cargo build

# Run the gRPC server
cargo run --bin indexer_server


mysql --comments --host 127.0.0.1 --port 5811 -u root

drop database if exists test; create database test;use test; CREATE TABLE `test`.`t1` (  `id` int unsigned NOT NULL AUTO_INCREMENT,  `title` varchar(200) DEFAULT NULL,  `body` text DEFAULT NULL,  PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin; insert into `test`.`t1` (title, body)value("title1", "body1");ALTER TABLE t1 ADD FULLTEXT INDEX ft_index (title) WITH PARSER ngram;

use test;
create table t3(id varchar(100), a varchar(100), b int, primary key(id));
insert into t3 values ("va", "bonjour", 10);

alter table t3 set tiflash replica 1;
alter table t3 add fulltext index ft_index(a) with parser ngram;

select count(*) from test.t3;

select count(*) from test.t3 where fts_match_word("2025", a);
