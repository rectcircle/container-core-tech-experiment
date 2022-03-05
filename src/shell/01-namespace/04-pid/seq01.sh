#!/usr/bin/env bash

# 注意：该脚本运行于 main 进程的子进程，其 PID Namespace 和 main 进程相同

pa=$1

# seq: 1s

### 构造进程 b
nsenter -p -t $pa bash -c 'echo "=== PB: $$ in new pid namespace" && exec sleep infinity' &
pbp=$! # 进程 b 的父进程
sleep 1

# seq: 2s

kill -9 $pbp # kill 进程 b 的父进程，进程 b 构造完成

### 构造进程 c
nsenter -p -t $pa bash -c 'echo "=== PC: $$ in new pid namespace" && exec sleep infinity' &

sleep 2

# seq: 4s

echo "=== old pid namespace process ==="
set -x
ls /proc
ps -eo pid,ppid,cmd | grep sleep | grep -v grep
kill -9 $pa
ps -eo pid,ppid,cmd | grep sleep | grep -v grep
