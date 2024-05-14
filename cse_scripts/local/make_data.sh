if [[ $M -eq "INSTALL" ]]
wget https://github.com/pingcap/go-tpc/releases/download/latest-ba96e1e382089f25cbd75d41b832c5740a6e4b68/go-tpc_latest_linux_amd64.tar.gz
fi

if [[ $M -eq "BIG" ]]
./go-tpc tpcc --warehouses 200 prepare -T 30 -H 127.0.0.1 -P 11005 -D tpcc
./go-tpc tpcc --warehouses 10 run -T 30 -H 127.0.0.1 -P 11005 -D tpcc
else
mysql --host 127.0.0.1 --port 11005 -u root -e "create table test.t(z int, y varchar(64));"
for i in $(seq 1 100);
do
    mysql --host 127.0.0.1 --port 11005 -u root -e "insert into test.t values ($i, 'aaaaaaaaaaaaaaaaaaaaaaaaaaa');"
done
fi

