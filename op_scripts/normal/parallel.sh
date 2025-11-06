tiup cluster deploy test v8.1.0 ./topology.yaml --user ubuntu -i ~/.ssh/id_rsa --yes
# 测试
tiup cluster start test

tiup br:v7.1.0 restore full --pd http://172.31.8.1:2379 --send-credentials-to-tikv=false --check-requirements=false --storage=s3://qa-workload-datasets/benchmark/ch-1k-v5 --s3.region=us-west-2 

echo 'use tpcc; show tables;' | mysql -u root --host 172.31.7.1 --port 4000


echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 0' | mysql -u root --host 172.31.7.1 --port 4000

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 1' | mysql -u root --host 172.31.7.1 --port 4000


echo 'ALTER TABLE tpcc.history SET TIFLASH REPLICA 1' | mysql -u root --host 172.31.7.1 --port 4000
echo 'ALTER TABLE tpcc.history SET TIFLASH REPLICA 0' | mysql -u root --host 172.31.7.1 --port 4000


tiup cluster restart test -R tiflash -y

echo 'select * from information_schema.tiflash_replica' | mysql -u root --host 172.31.7.1 --port 4000


tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set replica-schedule-limit 200
tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set patrol-region-interval 200
tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set max-pending-peer-count 200
tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set max-snapshot-count 200

tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 store limit all 200



ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep 'Transformed snapshot in SSTFile to DTFiles'"
ssh 172.31.9.1 "tail -n 2000 /tidb-deploy/tiflash-9000/log/tiflash.log | grep 'Transformed snapshot in SSTFile to DTFiles'"

ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash.log | grep 'Transformed snapshot in SSTFile to DTFiles' | grep '14:05:'"


ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"

ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/log/tiflash_tikv.log | grep 'snap_handle_pool_size'"


