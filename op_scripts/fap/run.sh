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
tiup br:v7.1.0 restore full --pd http://172.31.8.1:2379 \
  --send-credentials-to-tikv=false --check-requirements=false \
  --storage=s3://qa-workload-datasets/benchmark/ch-1k-v5 --s3.region=us-west-2 \
  --merge-region-size-bytes=100663290 --merge-region-key-count=960001

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 0' | mysql \
  -u root --host 172.31.7.1 --port 4000

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 1' | mysql \
  -u root --host 172.31.7.1 --port 4000

tiup cluster restart test -R tiflash -y

echo 'select * from information_schema.tiflash_replica' | mysql \
  -u root --host 172.31.7.1 --port 4000

echo 'ALTER DATABASE tpcc SET TIFLASH REPLICA 2' | mysql \
-u root --host 172.31.7.1 --port 4000

# 清理
tiup cluster clean test --all -y

terraform destroy -auto-approve