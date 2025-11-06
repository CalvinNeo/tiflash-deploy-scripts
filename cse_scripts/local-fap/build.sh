git clone git@github.com:tidbcloud/pd-cse.git
cd pd-cse
git fetch origin release-7.1-keyspace
git checkout origin/release-7.1-keyspace
make build
cd bin
tar -czvf pd.tar.gz pd-server

git clone git@github.com:tidbcloud/tidb-cse.git
cd tidb-cse
git fetch origin release-7.1-keyspace
git checkout origin/release-7.1-keyspace
export MIN_TIKV_VERSION=6.1.0 && export REGISTER_METRICS_INIT=false && make server
cd bin
tar -czvf tidb.tar.gz tidb-server

git clone git@github.com:tidbcloud/cloud-storage-engine.git
cd cloud-storage-engine
git fetch origin cloud-engine
make build
cd target/debug/ && tar -czvf tikv.tar.gz tikv-server

git clone git@github.com:tidbcloud/tiflash-cse.git
cd tiflash-cse
git fetch origin cloud-engine-on-release-7.5
git checkout -b cloud-engine-on-release-7.5 remotes/origin/cloud-engine-on-release-7.4
git submodule update --init --recursive
mkdir -p build/debug

# cmake ../ -DHAVE_STD_REGEX=ON -DRUN_HAVE_STD_REGEX=1 -DCMAKE_CXX_COMPILER=clang++ -DCMAKE_C_COMPILER=clang -DCMAKE_BUILD_TYPE=Debug -DCMAKE_INSTALL_PREFIX=./install_tiflash/tiflash -DCMAKE_PREFIX_PATH=/data1/ra_common/tiflash-env-17/sysroot/


cd dbg-install/
cmake .. -DCMAKE_CXX_COMPILER=clang++ -DCMAKE_C_COMPILER=clang -DENABLE_TESTS=on -DCMAKE_PREFIX_PATH=/DATA/disk1/ra_common/tiflash-env-17/sysroot -DCMAKE_BUILD_TYPE=Debug -DCMAKE_INSTALL_PREFIX=./install_tiflash/tiflash
make tiflash -j40 && make install && cd install_tiflash && rm -rf tiflash/bin && tar -czvf tiflash.tar.gz ./tiflash && cd ..
tiup cluster patch -y calvin-cse-s3 /data2/calvin_81/c/cse/tiflash-cse/dbg-install/install_tiflash/tiflash.tar.gz --overwrite -R tiflash



cd build/release
cmake ../.. -DCMAKE_CXX_COMPILER=clang++ -DENABLE_FAILPOINTS=ON -DCMAKE_C_COMPILER=clang -DCMAKE_BUILD_TYPE=RELWITHDEBINFO -DCMAKE_INSTALL_PREFIX=./install_tiflash/tiflash -DCMAKE_PREFIX_PATH=/data1/ra_common/tiflash-env-17/sysroot/
make tiflash -j40 && make install
cd install_tiflash && rm -rf tiflash/bin
tar -czvf tiflash.tar.gz ./tiflash
cd ../../..

tiup cluster stop calvin-cse-s3 -y -R tiflash
tiup cluster patch -y calvin-cse-s3 /DATA/disk1/calvin/tiflash/cse/tiflash-cse/build/debug/install_tiflash/tiflash.tar.gz --overwrite --offline -R tiflash
tiup cluster start calvin-cse-s3 -y -R tiflash


source /data1/ra_common/.tiflash_env_17