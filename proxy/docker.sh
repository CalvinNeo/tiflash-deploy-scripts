sudo docker run -it --name c1 hub.pingcap.net/tiflash/tiflash-llvm-base:amd64  /bin/sh
git clone https://github.com/CalvinNeo/tidb-engine-ext.git -b merge-master-beyond-8.4
cd tidb-engine-ext
make ci_test


docker run -it --name c3 ghcr.io/pingcap-qe/cd/builders/tiflash:v2024.11.7  /bin/sh
git clone https://github.com/CalvinNeo/tidb-engine-ext.git -b fix-arm-b
cd tidb-engine-ext
make ci_test
