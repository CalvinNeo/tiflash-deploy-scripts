tiup playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1 --tici.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --tici.worker.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --db.port 4012

export TIDB_USER=root
export TIDB_PASSWORD=""
export TIDB_HOST=127.0.0.1
export TIDB_PORT=4012
export TIDB_DB=eth

mysql --comments --host 127.0.0.1 --port 4012 -u root -e "create database eth;"

python eth_import_transactions.py   --start-date 2025-10-24 --days 3   --source local   --chunksize 1000 --batch-size 1000 -v 

python eth_import_transactions.py   --start-date 2025-10-24 --days 3   --source local   --chunksize 1000 --batch-size 1000 -v --index-type fts

python monitor_latest_fts_count.py -v

cat calvin-tici/tiflash-0/tici_searchlib.log | grep warmup

cat calvin-tici/tici-meta-0/tici-meta.log  | grep "shard_id=4"  | grep -v "shard_task_runner" | grep -v "append_frag_meta"

 rm -rf ~/.tiup/data/calvin-tici/

 cd .. && cd calvin-tici


 tiup -T calvin-tici playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1 --tici.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --tici.worker.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --db.port 4012 --tici.meta.config /data2/calvin_81/c/tici/tici/ci/meta-local.toml

# /data2/calvin_81/c/tici/tiflash/

 cat tici-meta-0/tici-meta.log  | grep -E "(shard_id=4|emove reader context)"  | grep -v "shard_task_runner" | grep -v "append_frag_meta"


# tiup -T calvin-tici playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1 --tici.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --tici.worker.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --db.port 4012 --tiflash.binpath /data2/calvin_81/c/tici/tiflash/dbg/artifact/tiflash --tici.meta.config /data2/calvin_81/c/tici/tici/ci/meta-local.toml
 
 tiup -T calvin-tici playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1 --db.binpath /data2/calvin_81/c/tidb/bin/tidb-server --tici.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --tici.worker.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --db.port 4012 --tiflash.binpath /data2/calvin_81/c/tici/tiflash/dbg/dbms/src/Server/tiflash --tici.meta.config /data2/calvin_81/c/tici/tici/ci/meta-local.toml


# tics3
 tiup -T calvin-tici playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1 --tici.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --tici.worker.binpath /data2/calvin_81/c/tici/tici/target-build/debug/tici-server --db.port 4012 --tiflash.binpath /data2/calvin_81/c/tics3/dbg/dbms/src/Server/tiflash --tici.meta.config /data2/calvin_81/c/tici/tici/ci/meta-local.toml --tici.worker.config /data2/calvin_81/c/tici/tici/ci/worker-local.toml


curl "http://127.0.0.1:8501/op/meta/reschedule_range?start_key=74800000000000007C&end_key=74800000000000007D&writer=&table_id=131&index_id=2"

select shard_writer,count(*) from tici.tici_shard_meta group by shard_writer;

select table_id,index_id from tici.tici_shard_meta;
