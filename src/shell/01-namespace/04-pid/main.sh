#!/usr/bin/env bash

# sudo ./src/shell/01-namespace/04-pid/main.sh

# 注意：该脚本运行于进程为 main
# 设置 main 进程的 /proc/[pid]/ns/pid_for_children
# mount namespace 不能在此设置，因为 mount namespace 会立即生效 
exec unshare -p bash $(cd $(dirname "$0"); pwd)/seq00.sh
