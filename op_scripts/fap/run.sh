# 创建
terraform init
terraform apply -auto-approve


tiup cluster destroy test -y

tiup cluster deploy test nightly ./topology.yaml --user ubuntu -i ~/.ssh/id_rsa --yes

# 部署
cd ../cmake-build-Release
rm -rf tiflash-cloud-native-linux-amd64.tar.gz
mkdir -p package/tiflash
cp artifacts/* package/tiflash/
cd package
tar czf ../tiflash-cloud-native-linux-amd64.tar.gz *
cd ..
scp tiflash-cloud-native-linux-amd64.tar.gz ubuntu@999.999.999.99:/home/ubuntu/tiflash-cloud-native-linux-amd64.tar.gz
cd ../.devcontainer


# 测试
tiup cluster start test
python s.py
tiup cluster patch test tiflash-cloud-native-linux-amd64.tar.gz -R tiflash -y
tiup br:v7.1.0 restore full --pd http://172.31.8.1:2379 --send-credentials-to-tikv=false --check-requirements=false --storage=s3://qa-workload-datasets/benchmark/ch-1k-v5 --s3.region=us-west-2 --merge-region-size-bytes=100663290 --merge-region-key-count=960001

tiup br:v7.1.0 restore full --pd http://172.31.8.1:2379 --send-credentials-to-tikv=false --check-requirements=false --storage=s3://qa-workload-datasets/benchmark/ch-1k-v5 --s3.region=us-west-2 

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 0' | mysql  -u root --host 172.31.7.1 --port 4000


echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 1' | mysql -u root --host 172.31.7.1 --port 4000

tiup cluster restart test -R tiflash -y

echo 'select * from information_schema.tiflash_replica' | mysql -u root --host 172.31.7.1 --port 4000

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 2' | mysql -u root --host 172.31.7.1 --port 4000
echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 1' | mysql -u root --host 172.31.7.1 --port 4000


# 检查结果
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tiflash[tpcc.orders])*/ count(*) from tpcc.orders;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tikv[tpcc.orders])*/ count(*) from tpcc.orders;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tiflash[tpcc.stock])*/ count(*) from tpcc.stock;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tikv[tpcc.stock])*/ count(*) from tpcc.stock;"

mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tiflash[tpcc.warehouse])*/ count(*) from tpcc.warehouse;"
mysql --host 172.31.7.1 --port 4000 -u root --comments -e "select /*+read_from_storage(tikv[tpcc.warehouse])*/ count(*) from tpcc.warehouse;"


# 改变测试环境
# 禁用 fap
ssh 172.31.9.1
ssh 172.31.9.2
sudo chmod 777 /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml

sudo sed -i 's/enable-fast-add-peer = false/enable-fast-add-peer = true/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml

ssh 172.31.9.1 "sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"
ssh 172.31.9.2 "sudo sed -i 's/enable-fast-add-peer = true/enable-fast-add-peer = false/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"


ssh 172.31.9.1 "cat /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml"


sudo chmod 777 /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml
sudo sed -i 's/region-worker-tick-interval = "1s"/region-worker-tick-interval = "5s"/' /tidb-deploy/tiflash-9000/conf/tiflash-learner.toml

# 清理
tiup cluster clean test --all -y

terraform destroy -auto-approve