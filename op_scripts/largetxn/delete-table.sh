
region_ids=$(mysql --host $HOST --port $PORT --user root --comments --skip-column-names -B -e "select s.region_id from information_schema.tables t, information_schema.tikv_region_status s where t.table_schema = '$SCHEMA_NAME' AND t.table_name = '$TABLE_NAME' AND t.tidb_table_id = s.table_id;")

sudo chmod 777 -R $TIFLASH_PATH/page/kvstore


partions=$(mysql --host $HOST --port $PORT --user root --comments --skip-column-names -B -e "select tidb_partition_id from information_schema.partitions where table_schema = '$SCHEMA_NAME' AND table_name = '$TABLE_NAME';")
IFS=$'\n' read -r -d '' -a lines <<< "$partions"
line_count=${#lines[@]}

if [[ line_count == 1 && ${lines[0]} == "NULL" ]]; then
    echo "Non partition table"
    curl -XDELETE "http://$PD_ADDR/pd/api/v1/config/rule/tiflash/table-$TABLE_ID-r"
else
    echo "Partition table"
    
fi


export IFS=$'\n'
for rid in $region_ids; do
    tiup ctl:v8.4.0 pd -u $PD_ADDR operator add remove-peer $rid $STORE_ID
    ./dbms/src/Server/tiflash pagectl -V 3 --mode 8 --path $TIFLASH_PATH/page/kvstore --page_id $rid -N 1000000
done

