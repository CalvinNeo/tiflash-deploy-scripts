# Prepare data with scale factor 1
./go-tpc tpch --sf=1 prepare
# Prepare data with scale factor 1, create tiflash replica, and analyze table after data loaded
./go-tpc tpch --sf 1 --analyze --tiflash-replica 1 prepare

./go-tpc tpch --sf=1 --check=true run -H 172.31.7.1 -P 4000

./go-tpc tpch --sf=1 run -H 172.31.7.1 -P 4000




./go-tpc tpcc --warehouses 200 prepare -T 30 -H 172.31.7.1 -P 4000

mysql --host 172.31.7.1 --port 4000 -u root -e "alter database tpcc set tiflash replica 2;"