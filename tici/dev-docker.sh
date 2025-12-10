mc config host add localminio http://127.0.0.1:9000 minioadmin minioadmin


sudo docker build -f ci/Dockerfile -t ctici .

docker kill calvin-ctici || docker rm calvin-ctici
docker run -d --name calvin-ctici -v /data2/calvin_81/c/tici/tici:/tici ctici
docker run -it --name calvin-ctici -v /data2/calvin_81/c/tici/tici:/tici ctici sh
docker run -it --rm --name calvin-ctici --cpus=32 -v /data2/calvin_81/c/tici/tici:/tici -v /data2/calvin_81/c/tici/cargo_home:/root/.cargo ctici sh
docker exec -it calvin-ctici bash

docker save -o ctici.tar ctici:latest
podman load -i ctici.tar

podman run -d --name calvin-systemd \
  --privileged \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v /data2/calvin_81/c/tici/tici:/tici \
  ctici \
  /sbin/init


podman run -d --name calvin-systemd \
  --privileged \
  --cgroupns=host \
  --tmpfs /tmp \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v /run:/run:rw \
  -v /etc/machine-id:/etc/machine-id:ro \
  --hostname calvin-systemd \
  ctici \
  /sbin/init

podman kill calvin-systemd || podman rm calvin-systemd


docker compose  -f scripts/nextgen/cluster.yaml -f scripts/nextgen/tici-ci.yaml down
docker compose  -f scripts/nextgen/cluster.yaml -f scripts/nextgen/tici-ci.yaml up -d


docker compose  -f scripts/nextgen/cluster.yaml -f scripts/nextgen/tici-ci.yaml ps


docker exec -it nextgen-tici-meta0-1  sh

docker run -it --name calvin-tiflash -v /data2/calvin_81/c/tics:/tici hub.pingcap.net/tiflash/tiflash-llvm-base:rocky8-llvm-17.0.6-v2 sh


docker run -it --name calvin-nextgen-tici --privileged -v /sys/fs/cgroup:/sys/fs/cgroup:ro -v /data2/calvin_81/c/tici/tici:/tici ctici /sbin/init

docker kill calvin-nextgen-ctici || docker rm calvin-nextgen-ctici



docker run --rm -it -v /data2/calvin_81/c/tici/tici:/tici ctici sh

yum install -y wget

wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc

curl --proto '=https' --tlsv1.2 -sSf https://tiup-mirrors.pingcap.com/install.sh | sh

export RUSTFLAGS="-Awarnings"
chmod 777 -R oss
./minio server oss > /dev/null &
./mc config host add localminio http://127.0.0.1:9000 minioadmin minioadmin
./mc mb localminio/logbucket
/root/.tiup/bin/tiup playground &
./mc anonymous set public localminio/logbucket

python3 tici/scripts/read_hdfs.py hdfs_10w

cargo run --bin index_e2e_test

/root/.tiup/bin/tiup cdc server
/root/.tiup/bin/tiup cdc cli changefeed create --server=http://127.0.0.1:8300 --sink-uri="s3://logbucket/storage_test?protocol=canal-json&access-key=minioadmin&secret-access-key=minioadmin&endpoint=http://127.0.0.1:9000"

./mc config host ls

docker kill calvin-ctici || docker rm calvin-ctici
docker run --rm --name calvin-ctici -v /data2/calvin_81/c/tici/tici:/tici ctici tici/scripts/test

docker run -it --rm --name calvin-ctici -v /data2/calvin_81/c/tici/tici:/tici -v /data2/calvin_81/c/tici/cargo_home:/root/.cargo ctici sh
# docker run -it --rm --name calvin-ctici --cpus=5 -v /data2/calvin_81/c/tici/tici:/tici -v /data2/calvin_81/c/tici/cargo_home:/root/.cargo ctici sh

docker run -it --rm --name calvin-ctici --cpus=32 -v /data2/calvin_81/c/tici/tici:/tici -v /data2/calvin_81/c/tici/cargo_home:/root/.cargo ctici sh


docker run -it --rm --name calvin-ctici2 -v /data2/calvin_81/c/tici/tici:/tici ctici tici/scripts/test

docker exec -it calvin-ctici bash

cd tici
ASSET_DIR=/tici ./scripts/test-prepare.sh
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_REGION=us-east-1
tiup cdc:nightly cli changefeed create --server=http://127.0.0.1:8300 --sink-uri="s3://ticidefaultbucket/tici_default_prefix/cdc?protocol=canal-json&endpoint=http://127.0.0.1:9000&enable-tidb-extension=true&output-row-key=true"
nohup python3 ci/read_hdfs.py hdfs_10w > "./input.log" 2>&1 &


/data2/calvin_81/c/tici/deploy/tiup/bin/tiup-playground --pd 1 --kv 1 --db 1 --tiflash 1  \
  --db.binpath /data2/calvin_81/c/tici/deploy/tidb/bin/tidb-server \
  --tiflash.binpath /data2/calvin_81/c/tici/deploy/tiflash/build/dbms/src/Server/tiflash \
  --ticdc 1 --ticdc.binpath /data2/calvin_81/c/tici/deploy/ticdc/bin/cdc \
  --tici.meta 1 --tici.worker 1 --tici.binpath /data2/calvin_81/c/tici/deploy/tici \
  --tag calvin-tici-1

./tiup/bin/tiup-playground --pd 1 --kv 1 --db 1 --tiflash 0  \
  --db.binpath ./tidb/bin/tidb-server \
  --ticdc 1 --ticdc.binpath ./ticdc/bin/cdc \
  --tici.meta 1 --tici.worker 1 --tici.binpath ./tici \
  --tag calvin-tici-1

./tiup/bin/tiup-playground --pd 1 --kv 1 --db 1 --tiflash 0  \
  --db.binpath ./tidb/bin/tidb-server \
  --ticdc 1 --ticdc.binpath ./ticdc/bin/cdc \
  --tici.meta 1 --tici.worker 1 --tici.binpath ./tici \
  --tag calvin-tici-1

echo -e '[tici]\nenable = true\n[tici.frag_reader]\nlocal_data_path = "/data2/calvin_81/c/tici/deploy/data"' > ~/.tiup/data/calvin-tici-1/tiflash-0/tiflash.toml