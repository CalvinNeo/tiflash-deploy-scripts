terraform init

aws sso login
terraform apply -auto-approve

terraform output -raw ssh-center

tiup cluster deploy test nightly ./topology.yaml \
     --user ubuntu -i ~/.ssh/id_rsa --yes


sudo apt install lrzsz unzip
wget -qO - https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | sudo bash
sudo apt install -y sysbench

rsync -P ~/Desktop/bench/proactive-tests-added.tar.gz ubuntu@13.59.232.253:/home/ubuntu/proactive-tests-added.tar.gz

nohup sysbench /usr/share/sysbench/oltp_update_non_index.lua --tables=32 --table-size=10000000 --mysql-host=172.31.7.1,172.31.7.2,172.31.7.3,172.31.7.4,172.31.7.5,172.31.7.6 --mysql-port=4000 --threads=3500 --report-interval=10 --mysql-user=root --mysql-db=test --time=6000 run > sysbench.output 2>&1 &

tiup br:nightly backup db --pd http://172.31.8.1:2379 \
  --send-credentials-to-tikv=false \
  --storage s3://xxxxxxx --db rtdb


ssh ubuntu@172.31.9.1 '/tidb-deploy/tiflash-9000/bin/tiflash/tiflash version'

tiup cluster start test -R pd
tiup ctl:nightly pd config set cluster-version 7.2.0 --pd http://172.31.8.1:2379
tiup ctl:nightly pd config show cluster-version --pd http://172.31.8.1:2379


nohup ./tiflash-u2 -address 172.31.7.1:4000 -ratio 0 -update-thread 500 -insert-count 3000000000 -batch-insert 10 -trace-back-days 15 -trace-back-unsign-days 15 -replica 1 >/dev/null 2>&1 &
nohup ./tiflash-u2 -address 172.31.7.1:4000 -ratio 0.8  -update-thread 200 -insert-count 4294967295 -batch-insert 10 -trace-back-days 15 -trace-back-unsign-days 15 -replica 1 -query-mode simple -query-date "2023-06-19" -query-thread 1 -verify true  &

curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install