docker kill calvin-ctici-tiup || docker rm calvin-ctici-tiup


docker exec -it calvin-ctici-tiup bash

docker run -it --name calvin-ctici-tiup -p 30154:30154 -v /data2/calvin_81/c/tici/tici:/tici ctici sh



docker run -it --name calvin-ctici-tiup -p 30154:30154 -v /mnt/data:/data -v /data2/calvin_81/c/tici/tici:/tici ctici sh


nohup minio server /tici/tiupdata --console-address ":9001" &
sleep 2
mc alias set myminio http://localhost:9000 minioadmin minioadmin
mc mb myminio/logbucket
tiup mirror set http://tiup.pingcap.net:8988

tiup playground:v1.16.2-feature.fts v9.0.0-feature.fts --ticdc 1 --tici.meta 1 --tici.worker 1


socat TCP-LISTEN:30154,fork,reuseaddr TCP:127.0.0.1:3000


nginx -c ./nginx.conf -g 'daemon off;'
```
events {}
http {
    server {
        listen 0.0.0.0:30154;
        location / {
            proxy_pass http://127.0.0.1:3000;
        }
    }
}
```

