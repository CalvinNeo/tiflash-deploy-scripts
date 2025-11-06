
import os, subprocess, json

X = 200
DEST = X
SCHEDULE_LIMIT = X
MAX_SNAPSHOT = X

DEST = 120
SCHEDULE_LIMIT = 64
MAX_SNAPSHOT = 256

os.system("tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set replica-schedule-limit {}".format(SCHEDULE_LIMIT))
os.system("tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set patrol-region-interval 1ms")
os.system("tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set max-pending-peer-count {}".format(SCHEDULE_LIMIT))
os.system("tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 config set max-snapshot-count {}".format(MAX_SNAPSHOT))


r = subprocess.check_output(["tiup", "ctl:v7.1.0", "pd", "-u", "http://172.31.8.1:2379", "store"])
print("GET result {}".format(r))
j = json.loads(r)

for store in j['stores']:
    info = store['store']
    store_id = info['id']
    labels = info['labels'] if 'labels' in info else []
    ignore = False
    for kv in labels:
        if kv[u'key'] == u'engine_role' and kv[u'value'] != u'write':
            ignore = True
    if not ignore:
        print("Set store {} to {}".format(store_id, DEST))
        os.system("tiup ctl:v7.1.0 pd -u http://172.31.8.1:2379 store limit {} {}".format(store_id, DEST))
    else:
        print("Ignore store {}".format(store_id))