import subprocess
import time

def handle(output):
    lines = output.split("\n")
    def seekTo(s, fr=-1):
        for i, line in enumerate(lines):
            if i < fr:
                continue
            if line.strip().startswith(s):
                return i
        return None

    def seekAndRead(s, fr=-1):
        i = seekTo(s, fr)
        if i is not None:
            start = lines[i].index(s)
            return lines[i][start + len(s):].strip()
        return None

    cont = seekTo("Containers:")
    if cont is None:
        print("cont is None")
    tiflash = seekTo("tiflash:", cont)
    if tiflash is None:
        print("tiflash is None")
    restart = seekAndRead("Restart Count:", tiflash)

    return restart

def monitor(names):
    prev_restart = [0 for i in names]
    while True:
        for (node, name) in enumerate(names):
            command = '''
            kubectl describe pod {}
            '''.format(name)
            result = subprocess.run(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            stdout = result.stdout
            # print ("stdout: {}".format(stdout))

            restart = handle(stdout)
            restart = int(restart)

            print ("restart: {}, prev_restart: {}".format(restart, prev_restart[node]))

            kube_exec_command = '''curl "127.0.0.1:20292/debug/pprof/heap_activate?interval=30"'''
            exec_command = '''kubectl exec -ti {} -c tiflash -- {}'''.format(name, kube_exec_command)
            print (exec_command)
            if restart != prev_restart[node]:
                exec_result = subprocess.run(exec_command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                exec_output = exec_result.stdout
                print ("exec_output: {}".format(exec_output))

            prev_restart[node] = restart
        time.sleep( 5 )

if __name__ == '__main__':
    # with open("a.txt", "r") as f:
    #     handle(f.read().split("\n"))

    monitor(["tc-tiflash-0", "tc-tiflash-1"])



