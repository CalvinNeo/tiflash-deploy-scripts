https://github.com/pingcap/tiflash/tree/master/tests/docker/next-gen-utils

https://github.com/pingcap-inc/tici/pull/382/files#diff-8a357b4873f33df125d3ef06542b9e210ca903d0270759c171246bae6e4fbc08

tiup cluster deploy calvin-nextgen v8.5.2 ./tests/scripts/nextgen/nextgen-topo.yaml --ignore-config-check
tiup cluster patch calvin-nextgen -R tidb ./tests/scripts/nextgen/binaries/package/tidb.tar.gz --overwrite --offline -y
tiup cluster patch calvin-nextgen -R pd ./tests/scripts/nextgen/binaries/package/pd.tar.gz --overwrite --offline -y
tiup cluster patch calvin-nextgen -R tikv ./tests/scripts/nextgen/binaries/package/tikv.tar.gz --overwrite --offline -y
tiup cluster patch calvin-nextgen -R tiflash ./tests/scripts/nextgen/binaries/package/tiflash.tar.gz --overwrite --offline -y
tiup cluster start calvin-nextgen

sudo cp /data1/calvin/T/tici/tidb/bin/tidb-server ./tests/scripts/nextgen/binaries/tidb/tidb-server
cd ./tests/scripts/nextgen/ && make package && cd -
tiup cluster patch calvin-nextgen -R tidb ./tests/scripts/nextgen/binaries/package/tidb.tar.gz --overwrite --offline -y


./target-build/debug/tici-server meta --config ci/meta-template.toml
./target-build/debug/tici-server worker --config ci/worker-template.toml

mc mb localminio/calvin-nextgen-logbucket
mc mb localminio/tici-nextgen-test


cd ./tests/scripts/nextgen/ && make package && cd -


# system
mysql --host 127.0.0.1 --port 12480 -uroot

# keyspace
mysql --host 127.0.0.1 --port 12482 -uroot

select shard_writer,count(*) from tici.tici_shard_meta group by shard_writer;


use test;
create table t3(id varchar(100), a varchar(100), b int, primary key(id));
insert into t3 values ("bon", "jour", 10);

alter table t3 set tiflash replica 1;
alter table t3 add fulltext index ft_index(a) with parser ngram;

SHOW INDEX from t3;

DROP INDEX ft_index ON t3;

make server NEXT_GEN=1


# Run

export TIDB_USER=root
export TIDB_PASSWORD=""
export TIDB_HOST=127.0.0.1
export TIDB_PORT=12482
export TIDB_DB=eth