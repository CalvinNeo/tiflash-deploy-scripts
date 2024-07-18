tiup cluster destroy calvin-cse-s3 -y
rm -rf /data3/luorongzhen/s3_data/
kill `ps aux | grep "minio server /data3/luorongzhen/s3_data/" | grep -v grep | awk '{print $2}' `