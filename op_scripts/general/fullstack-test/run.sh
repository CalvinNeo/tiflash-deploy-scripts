tiup cluster clean calvin-test --all -y
tiup cluster start calvin-test -R pd,tidb,tikv,grafana,alertmanager,prometheus
/DATA/disk1/calvin/tiflash/tics/rel/dbms/src/Server/tiflash server --config-file=/DATA/disk1/calvin/tiflash/tics/tests/fullstack-test2/config/tiflash_dt.toml

sh ./run-test.sh fullstack-test2/ddl/alter_truncate_table.test
