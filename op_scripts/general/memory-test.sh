tiup bench ycsb load tidb -p tidb.instances="127.0.0.1:4000" -p recordcount=10000
tiup bench ycsb run tidb -p tidb.instances="10.2.12.81:5711" -p operationcount=10000