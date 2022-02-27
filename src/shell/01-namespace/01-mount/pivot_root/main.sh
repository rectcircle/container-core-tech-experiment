#!/usr/bin/env bash

# sudo ./src/shell/01-namespace/01-mount/pivot_root/main.sh

new_root="data/busybox/rootfs"
script="ls / && ls /bin"

# unshare -m: 创建新进程，并为该进程创建一个 Mount Namespace（-m）
# 更多参见：https://man7.org/linux/man-pages/man1/unshare.1.html\
# 注意 unshare 会自动取消进程的所有共享，因此不需要手动执行：mount --make-rprivate /
# 更多参见：https://man7.org/linux/man-pages/man1/unshare.1.html 的 --propagation 参数说明

# mount --bind: 确保 new_root 是一个挂载点
# cd $new_root: 确保 working directory 是新的 rootfs
# pivot_root: 切换 rootfs
# cd /: 根目录已经切换了，所以之前的工作目录已经不存在了，所以需要将 working directory 切换到根目录
unshare -m /bin/bash -c "mount --bind $new_root $new_root \
	&& cd $new_root \
	&& pivot_root . . \
	&& cd / \
	&& echo '=== new mount namespace and pivot_root process ===' \
	&& /bin/sh -xc \"$script\"" &
pid1=$!

wait $pid1
