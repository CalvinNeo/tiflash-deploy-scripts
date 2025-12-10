


# into ~/.tiup/data/ctici/tiflash-0/config.toml
 tiup cluster:v1.16.2-feature.fts deploy ctici nightly /data2/calvin_81/topos/topology.yaml.0.another.tici

rm -rf /DATA/disk1/calvin/.tiup/data/ctici
sudo rm -rf /data3/luorongzhen/s3_data/
kill `ps aux | grep "minio server /data3/luorongzhen/s3_data/" | grep -v grep | awk '{print $2}' `

tiup cluster:v1.16.2-feature.fts tls ctici enable

tiup cluster:v1.16.2-feature.fts start ctici

tiup cluster:v1.16.2-feature.fts display ctici

tiup cluster:v1.16.2-feature.fts destroy ctici -y


./target-build/debug/tici-server meta --config ci/meta-ctici.toml

export PATH=$PATH:/data2/calvin_81/c
sudo mkdir -p /data3/luorongzhen/s3_data/
sudo chmod 777 /data3/luorongzhen/s3_data/
minio server /data3/luorongzhen/s3_data/ --console-address "0.0.0.0:11998" --address "0.0.0.0:11999" > /dev/null &
sleep 2
mc config host add localminio http://127.0.0.1:11999 minioadmin minioadmin
mc mb localminio/logbucket

tiup --tag ctici playground nightly --db.binpath=/data2/calvin_81/c/tici/private-tidb/bin/tidb-server  --tiflash.binpath=/data2/calvin_81/c/tici/private-tiflash/dbg/dbms/src/Server/tiflash


vim ~/.tiup/data/ctici/tiflash-0/tiflash.toml

netstat -tulnp | grep pdpid

# 选择第二小的，或者

select * from information_schema.cluster_info;

cp /data2/calvin_81/c/tici/tici/target-build/debug/tici-server /DATA/disk4/luorongzhen/tidb-deploy/tici-meta-8500/bin/tici-server

```
[s3]
endpoint = "http://localhost:11999"
region = "us-east-1"
access_key = "minioadmin"
secret_key = "minioadmin"
bucket = "logbucket"
use_path_style = true
```

tiup cdc server --addr=127.0.0.1:8559 --pd=http://127.0.0.1:34523

tiup cdc cli changefeed create --server=http://127.0.0.1:8559 --sink-uri="s3://logbucket/storage_test?protocol=canal-json&access-key=minioadmin&secret-access-key=minioadmin&endpoint=http://127.0.0.1:11999"

# Build the project
cargo build

# Run the gRPC server
cargo run --bin indexer_server


use test;
create table t3(id varchar(100), a varchar(100), b int, primary key(id));
insert into t3 values ("va", "bonjour", 10);

alter table t3 set tiflash replica 1;
alter table t3 add fulltext index ft_index(a) with parser ngram;
