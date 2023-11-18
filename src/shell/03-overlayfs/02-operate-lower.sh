#!/usr/bin/env bash
# sudo apt install attr
# sudo ./src/shell/03-overlayfs/02-operate-lower.sh

# 创建并进入测试目录
exp_base_dir=/tmp/overlayfs-exp/02-operate-lower
umount $exp_base_dir/merged >/dev/null 2>&1
rm -rf $exp_base_dir && mkdir -p $exp_base_dir
cd $exp_base_dir

# 准备 lower、merged、upper、work 目录
mkdir -p lower1 lower2 merged upper work
mkdir -p lower1/from-lower1-dir lower1/from-lower1-dir2 lower1/from-lower1-lowner2-dir lower1/from-lower2-opaquedir
mkdir -p lower1/from-lower1-dir/subdir
echo 'from-lower1' > lower1/from-lower1-dir/file
echo 'from-lower1' > lower1/from-lower1-dir2/file
echo 'from-lower1' > lower1/from-lower1-lowner2-dir/file
echo 'from-lower1' > lower1/file1
echo 'from-lower1' > lower1/file2
echo 'from-lower1' > lower1/from-lower2-opaquedir/file1

# 生成 merged
mount -t overlay overlay -olowerdir=lower2:lower1,upperdir=upper,workdir=work merged
mkdir -p merged/from-merged-dir
echo 'from-merged' > merged/from-merged-dir/file1

# 操作之前
echo '=== before ==='
echo '>>> cat merged/file1'
cat merged/file1
echo
echo '>>> ls -al merged/from-lower1-lowner2-dir'
ls -al merged/from-lower1-lowner2-dir
echo
echo '>>> ls -al merged/from-lower2-opaquedir'
ls -al merged/from-lower2-opaquedir
echo

# lower2 新增文件
echo 'from-lower2' > lower2/from-lower2-file
mkdir -p lower2/from-lower2-dir
mkdir -p lower2/from-lower1-lowner2-dir
echo 'from-lower2' > lower2/from-lower1-lowner2-dir/from-lower2-file
mkdir -p lower2/from-lower1-lowner2-dir/subdir
echo 'from-lower2' > lower2/from-lower1-lowner2-dir/subdir/from-lower2-file
mkdir -p lower2/from-merged-dir
echo 'from-lower2' > lower2/from-merged-dir/file2
touch -t 197001010000 lower2/from-merged-dir

# lower2 覆盖文件
echo 'from-lower2' > lower2/file1
echo 'from-lower2' > lower2/file2
echo 'from-lower2' > lower2/from-lower1-lowner2-dir/file

# lower2 隐藏文件目录
mknod lower2/from-lower1-dir c 0 0
mkdir -p lower2/from-lower1-dir2
mknod lower2/from-lower1-dir2/file c 0 0

# lower2 opaque 目录
mkdir -p lower2/from-lower2-opaquedir
echo 'from-lower2' > lower2/from-lower2-opaquedir/file2
setfattr -n 'trusted.overlay.opaque' -v 'y' lower2/from-lower2-opaquedir

# 操作之后
echo '=== after ==='
echo '>>> tree merged'
tree merged
echo
echo '>>> cat merged/file1'
cat merged/file1
echo
echo '>>> cat merged/file2'
cat merged/file2
echo
echo '>>> cat merged/from-lower1-lowner2-dir/file'
cat merged/from-lower1-lowner2-dir/file
echo


# 清理缓存后
echo 2 > /proc/sys/vm/drop_caches
echo '=== after clear cache ==='
echo '>>> tree merged'
tree merged
echo
echo '>>> cat merged/file1'
cat merged/file1
echo
echo '>>> cat merged/file2'
cat merged/file2
echo
echo '>>> cat merged/from-lower1-lowner2-dir/file'
cat merged/from-lower1-lowner2-dir/file
echo


# 输出如下:
# === before ===
# >>> cat merged/file1
# from-lower1

# >>> ls -al merged/from-lower1-lowner2-dir
# 总用量 12
# drwxr-xr-x 2 root root 4096 11月 19 02:57 .
# drwxr-xr-x 1 root root 4096 11月 19 02:57 ..
# -rw-r--r-- 1 root root   12 11月 19 02:57 file

# >>> ls -al merged/from-lower2-opaquedir
# 总用量 12
# drwxr-xr-x 2 root root 4096 11月 19 02:57 .
# drwxr-xr-x 1 root root 4096 11月 19 02:57 ..
# -rw-r--r-- 1 root root   12 11月 19 02:57 file1

# === after ===
# >>> tree merged
# merged
# ├── file1
# ├── file2
# ├── from-lower1-dir2
# ├── from-lower1-lowner2-dir
# │   └── file
# ├── from-lower2-dir
# ├── from-lower2-file
# ├── from-lower2-opaquedir
# │   └── file1
# └── from-merged-dir
#     └── file1

# 5 directories, 6 files

# >>> cat merged/file1
# from-lower1

# >>> cat merged/file2
# from-lower2

# >>> cat merged/from-lower1-lowner2-dir/file
# from-lower1

# === after clear cache ===
# >>> tree merged
# merged
# ├── file1
# ├── file2
# ├── from-lower1-dir2
# ├── from-lower1-lowner2-dir
# │   ├── file
# │   ├── from-lower2-file
# │   └── subdir
# │       └── from-lower2-file
# ├── from-lower2-dir
# ├── from-lower2-file
# ├── from-lower2-opaquedir
# │   └── file2
# └── from-merged-dir
#     └── file1

# 6 directories, 8 files

# >>> cat merged/file1
# from-lower2

# >>> cat merged/file2
# from-lower2

# >>> cat merged/from-lower1-lowner2-dir/file
# from-lower2
