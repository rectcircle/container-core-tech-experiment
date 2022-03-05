#!/usr/bin/env bash

# 注意：该脚本运行于进程为 main

# seq: 0s

echo "=== main: $$"
# bash 默认处理了 SIGCHLD 信号，因此不需要处理信号

### 构造进程 a
# 创建一个新的 mount namespace
unshare -m bash -c 'mount -t proc proc /proc \
    && sleep 3 \
    && echo "=== new pid namespace process ===" \
    && set -x \
    && bash -c "nohup sleep infinity >/dev/null 2>&1 &" \
    && echo $$ \
    && ls /proc \
    && ps -o pid,ppid,cmd \
    && kill -9 1 \
    && ps -o pid,ppid,cmd \
    && exec sleep infinity \
' &

pa=$!
echo "=== PA: $pa"
sleep 1

# seq: 1s

# 恢复 main 进程 /proc/[pid]/ns/pid_for_children 为初始状态
exec nsenter -p -t 1 bash $(cd $(dirname "$0"); pwd)/seq01.sh $pa
