#!/usr/bin/env bash

# sudo ./src/shell/01-namespace/01-mount/main.sh

script="ls data/binding/target  \
	&& readlink /proc/self/ns/mnt  \
	&& cat /proc/self/mounts | grep data/binding/target || true \
	&& cat /proc/self/mountinfo | grep data/binding/target || true  \
	&& cat /proc/self/mountstats | grep data/binding/target || true  \
	&& sleep 10"

# 创建新进程，并为该进程创建一个 Mount Namespace（-m）
# 更多参见：https://man7.org/linux/man-pages/man1/unshare.1.html

# 注意 unshare 会自动取消进程的所有共享，因此不需要手动执行：mount --make-rprivate /
# 更多参见：https://man7.org/linux/man-pages/man1/unshare.1.html 的 --propagation 参数说明

# 将 data/binding/source 挂载（绑定）到 data/binding/target
# 因为在新的 Mount Namespace 中执行，所有其他进程的目录树不受影响
# 等价系统调用为：mount("data/binding/source", "data/binding/target", NULL, MS_BIND, NULL);
# 更多参见：https://man7.org/linux/man-pages/man8/mount.8.html
unshare -m /bin/bash -c "mount --bind data/binding/source data/binding/target \
	&& echo '=== new mount namespace process ===' && set -x && $script" &
pid1=$!

sleep 5
# 创建新的进程（不创建 Namespace），并执行测试命令
/bin/bash -c "echo '=== old namespace process ===' && set -x && $script" &
pid2=$!

wait $pid1
wait $pid2
