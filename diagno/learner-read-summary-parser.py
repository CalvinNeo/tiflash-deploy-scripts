import re
import sys

before2 = '''
["[Learner Read] Learner Read Summary, regions_info=(region_id=482 to_wait=1373535 applied_index=1373535 bypass_locks=);(region_id=3106 to_wait=6 applied_index=6 bypass_locks=);(region_id=8984 to_wait=6 applied_index=6 bypass_locks=);(region_id=6348 to_wait=6 applied_index=6 bypass_locks=);(region_id=1930 to_wait=6 applied_index=6 bypass_locks=);(region_id=5663 to_wait=11 applied_index=11 bypass_locks=);(region_id=4218 to_wait=11 applied_index=11 bypass_locks=);(region_id=1303 to_wait=12 applied_index=12 bypass_locks=);(region_id=7196 to_wait=6 applied_index=6 bypass_locks=);(region_id=6726 to_wait=6 applied_index=6 bypass_locks=);(region_id=4870 to_wait=6 applied_index=6 bypass_locks=);(region_id=7784 to_wait=6 applied_index=6 bypass_locks=);(region_id=4067 to_wait=6 applied_index=6 bypass_locks=);(region_id=3926 to_wait=6 applied_index=6 bypass_locks=);(region_id=8673 to_wait=6 applied_index=6 bypass_locks=);(region_id=2454 to_wait=6 applied_index=6 bypass_locks=);(region_id=8282 to_wait=6 applied_index=6 bypass_locks=);(region_id=8836 to_wait=6 applied_index=6 bypass_locks=);(region_id=3243 to_wait=6 applied_index=6 bypass_locks=);(region_id=2967 to_wait=6 applied_index=6 bypass_locks=);(region_id=5043 to_wait=6 applied_index=6 bypass_locks=);(region_id=6876 to_wait=6 applied_index=6 bypass_locks=);(region_id=7640 to_wait=6 applied_index=6 bypass_locks=);(region_id=1752 to_wait=6 applied_index=6 bypass_locks=);(region_id=5330 to_wait=6 applied_index=6 bypass_locks=);(region_id=3628 to_wait=11 applied_index=11 bypass_locks=);(region_id=4412 to_wait=6 applied_index=6 bypass_locks=);(region_id=5493 to_wait=6 applied_index=6 bypass_locks=), unavailable_regions_info={ids=[] locks=[]}, start_ts=449297655564861463"] [source="MPP<gather_id:1, query_ts:1713934538141218160, local_query_id:32444, server_id:488, start_ts:449297655564861463,task_id:1>"] [thread_id=678]
'''

after2 = '''
["[Learner Read] Learner Read Summary, regions_info=(region_id=9512 to_wait=6 applied_index=6 bypass_locks=);(region_id=5493 to_wait=6 applied_index=6 bypass_locks=);(region_id=2644 to_wait=6 applied_index=6 bypass_locks=);(region_id=482 to_wait=1373535 applied_index=1374030 bypass_locks=);(region_id=2454 to_wait=6 applied_index=6 bypass_locks=);(region_id=8282 to_wait=6 applied_index=6 bypass_locks=);(region_id=3106 to_wait=6 applied_index=6 bypass_locks=);(region_id=8836 to_wait=6 applied_index=6 bypass_locks=);(region_id=3243 to_wait=6 applied_index=6 bypass_locks=);(region_id=2967 to_wait=6 applied_index=6 bypass_locks=);(region_id=6082 to_wait=7 applied_index=7 bypass_locks=);(region_id=6726 to_wait=6 applied_index=6 bypass_locks=);(region_id=1930 to_wait=6 applied_index=6 bypass_locks=);(region_id=9605 to_wait=6 applied_index=6 bypass_locks=);(region_id=4683 to_wait=6 applied_index=6 bypass_locks=);(region_id=3628 to_wait=11 applied_index=11 bypass_locks=);(region_id=1533 to_wait=8 applied_index=8 bypass_locks=);(region_id=4870 to_wait=6 applied_index=6 bypass_locks=);(region_id=2082 to_wait=7 applied_index=7 bypass_locks=);(region_id=5043 to_wait=6 applied_index=6 bypass_locks=);(region_id=5943 to_wait=6 applied_index=6 bypass_locks=);(region_id=8542 to_wait=6 applied_index=6 bypass_locks=);(region_id=4218 to_wait=11 applied_index=11 bypass_locks=);(region_id=7640 to_wait=6 applied_index=6 bypass_locks=);(region_id=2833 to_wait=6 applied_index=6 bypass_locks=);(region_id=2292 to_wait=6 applied_index=6 bypass_locks=);(region_id=8457 to_wait=6 applied_index=6 bypass_locks=);(region_id=6613 to_wait=6 applied_index=6 bypass_locks=), unavailable_regions_info={ids=[] locks=[]}, start_ts=449297655564861463"] [source="MPP<gather_id:1, query_ts:1713934540489884111, local_query_id:32451, server_id:488, start_ts:449297655564861463,task_id:2>"] [thread_id=235]
'''

before0 = '''
[2024/04/24 12:55:38.164 +08:00] [DEBUG] [LearnerReadWorker.cpp:487] ["[Learner Read] Learner Read Summary, regions_info=(region_id=2833 to_wait=6 applied_index=6 bypass_locks=);(region_id=8457 to_wait=6 applied_index=6 bypass_locks=);(region_id=8542 to_wait=6 applied_index=6 bypass_locks=);(region_id=9106 to_wait=6 applied_index=6 bypass_locks=);(region_id=3518 to_wait=6 applied_index=6 bypass_locks=);(region_id=3753 to_wait=7 applied_index=7 bypass_locks=);(region_id=6458 to_wait=6 applied_index=6 bypass_locks=);(region_id=9231 to_wait=6 applied_index=6 bypass_locks=);(region_id=6082 to_wait=7 applied_index=7 bypass_locks=);(region_id=4550 to_wait=6 applied_index=6 bypass_locks=);(region_id=7504 to_wait=11 applied_index=11 bypass_locks=);(region_id=9390 to_wait=6 applied_index=6 bypass_locks=);(region_id=9512 to_wait=6 applied_index=6 bypass_locks=);(region_id=5799 to_wait=6 applied_index=6 bypass_locks=);(region_id=3361 to_wait=6 applied_index=6 bypass_locks=);(region_id=5943 to_wait=6 applied_index=6 bypass_locks=);(region_id=6246 to_wait=6 applied_index=6 bypass_locks=);(region_id=4683 to_wait=6 applied_index=6 bypass_locks=);(region_id=8054 to_wait=6 applied_index=6 bypass_locks=);(region_id=5187 to_wait=6 applied_index=6 bypass_locks=);(region_id=9605 to_wait=6 applied_index=6 bypass_locks=);(region_id=7023 to_wait=6 applied_index=6 bypass_locks=);(region_id=2082 to_wait=7 applied_index=7 bypass_locks=);(region_id=6613 to_wait=6 applied_index=6 bypass_locks=);(region_id=2292 to_wait=6 applied_index=6 bypass_locks=);(region_id=1533 to_wait=8 applied_index=8 bypass_locks=);(region_id=7313 to_wait=11 applied_index=11 bypass_locks=);(region_id=2644 to_wait=6 applied_index=6 bypass_locks=), unavailable_regions_info={ids=[] locks=[]}, start_ts=449297655564861463"] [source="MPP<gather_id:1, query_ts:1713934538141218160, local_query_id:32444, server_id:488, start_ts:449297655564861463,task_id:3>"] [thread_id=1003]
'''

after0 = '''
[2024/04/24 12:55:40.517 +08:00] [DEBUG] [LearnerReadWorker.cpp:487] ["[Learner Read] Learner Read Summary, regions_info=(region_id=9106 to_wait=6 applied_index=6 bypass_locks=);(region_id=6876 to_wait=6 applied_index=6 bypass_locks=);(region_id=1303 to_wait=12 applied_index=12 bypass_locks=);(region_id=6246 to_wait=6 applied_index=6 bypass_locks=);(region_id=7784 to_wait=6 applied_index=6 bypass_locks=);(region_id=9231 to_wait=6 applied_index=6 bypass_locks=);(region_id=5330 to_wait=6 applied_index=6 bypass_locks=);(region_id=6458 to_wait=6 applied_index=6 bypass_locks=);(region_id=8984 to_wait=6 applied_index=6 bypass_locks=);(region_id=7023 to_wait=6 applied_index=6 bypass_locks=);(region_id=4067 to_wait=6 applied_index=6 bypass_locks=);(region_id=3926 to_wait=6 applied_index=6 bypass_locks=);(region_id=8673 to_wait=6 applied_index=6 bypass_locks=);(region_id=5663 to_wait=11 applied_index=11 bypass_locks=);(region_id=7196 to_wait=6 applied_index=6 bypass_locks=);(region_id=3753 to_wait=7 applied_index=7 bypass_locks=);(region_id=3518 to_wait=6 applied_index=6 bypass_locks=);(region_id=7313 to_wait=11 applied_index=11 bypass_locks=);(region_id=5187 to_wait=6 applied_index=6 bypass_locks=);(region_id=8054 to_wait=6 applied_index=6 bypass_locks=);(region_id=1752 to_wait=6 applied_index=6 bypass_locks=);(region_id=7504 to_wait=11 applied_index=11 bypass_locks=);(region_id=9390 to_wait=6 applied_index=6 bypass_locks=);(region_id=4412 to_wait=6 applied_index=6 bypass_locks=);(region_id=4550 to_wait=6 applied_index=6 bypass_locks=);(region_id=5799 to_wait=6 applied_index=6 bypass_locks=);(region_id=3361 to_wait=6 applied_index=6 bypass_locks=);(region_id=6348 to_wait=6 applied_index=6 bypass_locks=), unavailable_regions_info={ids=[] locks=[]}, start_ts=449297655564861463"] [source="MPP<gather_id:1, query_ts:1713934540489884111, local_query_id:32451, server_id:488, start_ts:449297655564861463,task_id:1>"] [thread_id=123]
'''

tiflash2 = '''
[Learner Read] Learner Read Summary, regions_info=(region_id=482 to_wait=1373535 applied_index=1373535 bypass_locks=);(region_id=3106 to_wait=6 applied_index=6 bypass_locks=);(region_id=8984 to_wait=6 applied_index=6 bypass_locks=);(region_id=6348 to_wait=6 applied_index=6 bypass_locks=);(region_id=1930 to_wait=6 applied_index=6 bypass_locks=);(region_id=5663 to_wait=11 applied_index=11 bypass_locks=);(region_id=4218 to_wait=11 applied_index=11 bypass_locks=);(region_id=1303 to_wait=12 applied_index=12 bypass_locks=);(region_id=7196 to_wait=6 applied_index=6 bypass_locks=);(region_id=6726 to_wait=6 applied_index=6 bypass_locks=);(region_id=4870 to_wait=6 applied_index=6 bypass_locks=);(region_id=7784 to_wait=6 applied_index=6 bypass_locks=);(region_id=4067 to_wait=6 applied_index=6 bypass_locks=);(region_id=3926 to_wait=6 applied_index=6 bypass_locks=);(region_id=8673 to_wait=6 applied_index=6 bypass_locks=);(region_id=2454 to_wait=6 applied_index=6 bypass_locks=);(region_id=8282 to_wait=6 applied_index=6 bypass_locks=);(region_id=8836 to_wait=6 applied_index=6 bypass_locks=);(region_id=3243 to_wait=6 applied_index=6 bypass_locks=);(region_id=2967 to_wait=6 applied_index=6 bypass_locks=);(region_id=5043 to_wait=6 applied_index=6 bypass_locks=);(region_id=6876 to_wait=6 applied_index=6 bypass_locks=);(region_id=7640 to_wait=6 applied_index=6 bypass_locks=);(region_id=1752 to_wait=6 applied_index=6 bypass_locks=);(region_id=5330 to_wait=6 applied_index=6 bypass_locks=);(region_id=3628 to_wait=11 applied_index=11 bypass_locks=);(region_id=4412 to_wait=6 applied_index=6 bypass_locks=);(region_id=5493 to_wait=6 applied_index=6 bypass_locks=), unavailable_regions_info={ids=[] locks=[]}, 
'''

def f(s):
    region_ids = re.findall(r"region_id=\d+", s)
    to_wait = re.findall(r"to_wait=\d+", s)
    assert len(region_ids) == len(to_wait)
    for i in range(len(region_ids)):
        print (region_ids[i], to_wait[i])

def pregion(s):
    region_ids = re.findall(r"region_id=\d+", s)
    arr = []
    for i in range(len(region_ids)):
        arr.append(region_ids[i].replace("region_id=", ""))
    print ('(' + '|'.join(arr) + ')')

def diff(b, a):
    br = re.findall(r"region_id=\d+", b)
    bw = re.findall(r"applied_index=\d+", b)
    ar = re.findall(r"region_id=\d+", a)
    aw = re.findall(r"applied_index=\d+", a)

    assert len(br) == len(bw)
    assert len(ar) == len(aw)

    bm = {}
    am = {}

    for i in range(len(br)):
        bm[br[i]] = bw[i]

    for i in range(len(ar)):
        am[ar[i]] = aw[i]

    for rid in bm:
        if rid in am:
            if bm[rid] != am[rid]:
                print (rid, bm[rid], am[rid])
        else:
            print (rid, bm[rid], '*')

    for rid in am:
        if rid not in bm:
            print (rid, '*', am[rid])

# pregion(tiflash2)
diff(before0, after0)
# f(before)
# print ("=======")
# f(after)