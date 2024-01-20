tiup br:nightly restore full --pd http://172.31.8.1:2379 \
  --send-credentials-to-tikv=false --check-requirements=false \
  --storage=s3://xxx/rtdb-1.4b-with-pk-info2 --s3.region=ap-northeast-2

/usr/local/bin/aws s3 cp s3://yxxx/pks/pk0topk8.tar.gz pk0topk8.tar.gz
/usr/local/bin/aws s3 cp s3://xxx/pks/pk9topk13.tar.gz pk9topk13.tar.gz
/usr/local/bin/aws s3 cp s3://xxx/pks/pk_14.1.4b pk_14

./target/release/pressure --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/,mysql://root@172.31.7.3:4000/,mysql://root@172.31.7.4:4000/ --input-files /home/ubuntu/pk_0,/home/ubuntu/pk_1,/home/ubuntu/pk_2,/home/ubuntu/pk_3,/home/ubuntu/pk_4,/home/ubuntu/pk_5,/home/ubuntu/pk_6,/home/ubuntu/pk_7,/home/ubuntu/pk_8,/home/ubuntu/pk_9,/home/ubuntu/pk_10,/home/ubuntu/pk_11,/home/ubuntu/pk_12,/home/ubuntu/pk_13,/home/ubuntu/pk_14 --slot-size 140000 --update-interval-millis 500000 --workers 8 --tasks 400
./target/release/pressure --tidb-addrs mysql://root@172.31.7.1:4000/,mysql://root@172.31.7.2:4000/,mysql://root@172.31.7.3:4000/,mysql://root@172.31.7.4:4000/ --input-files /home/ubuntu/pk_0,/home/ubuntu/pk_1,/home/ubuntu/pk_2,/home/ubuntu/pk_3,/home/ubuntu/pk_4,/home/ubuntu/pk_5,/home/ubuntu/pk_6,/home/ubuntu/pk_7,/home/ubuntu/pk_8,/home/ubuntu/pk_9,/home/ubuntu/pk_10,/home/ubuntu/pk_11,/home/ubuntu/pk_12,/home/ubuntu/pk_13,/home/ubuntu/pk_14 --slot-size 140000 --update-interval-millis 500000 --workers 8 --tasks 200