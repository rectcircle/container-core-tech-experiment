#!/usr/bin/env bash
# sudo apt install attr
# sudo ./src/shell/03-overlayfs/03-operate-upper.sh

# 创建并进入测试目录
exp_base_dir=/tmp/overlayfs-exp/03-operate-upper
umount $exp_base_dir/merged >/dev/null 2>&1
rm -rf $exp_base_dir && mkdir -p $exp_base_dir
cd $exp_base_dir

# 准备 lower、merged、upper、work 目录
mkdir -p lower merged upper work
mkdir -p lower/from-lower-dir1
echo 'from-lower' > lower/from-lower-dir1/from-lower-file
mkdir -p lower/from-lower-dir2
echo 'from-lower' > lower/from-lower-dir2/from-lower-file
mkdir -p lower/from-lower-dir3
echo 'from-lower' > lower/from-lower-dir3/from-lower-file
mkdir -p lower/from-lower-dir4
echo 'from-lower' > lower/from-lower-dir4/from-lower-file
mkdir -p lower/from-lower-dir5
echo 'from-lower' > lower/from-lower-dir5/from-lower-file
echo 'from-lower' > lower/from-lower-file1
echo 'from-lower' > lower/from-lower-file2
echo 'from-lower' > lower/from-lower-file3

# 生成 merged
mount -t overlay overlay -olowerdir=lower,upperdir=upper,workdir=work merged

# 操作之前
echo '=== before ==='
echo '>>> cat merged/from-lower-file1'
cat merged/from-lower-file1
echo
echo '>>> ls merged/from-lower-dir1'
cat merged/from-lower-dir1
echo

# 新增
echo 'from-upper' > upper/from-upper-file
mkdir -p upper/from-lower-dir1
echo 'from-upper' > upper/from-lower-dir1/from-upper-file
mkdir -p upper/from-lower-dir2
echo 'from-upper' > upper/from-lower-dir2/from-upper-file
mkdir -p upper/from-upper-dir

# 覆盖
echo 'from-upper' > upper/from-lower-file1
echo 'from-upper' > upper/from-lower-file2
echo 'from-upper' > upper/from-lower-dir1/from-lower-file
echo 'from-upper' > upper/from-lower-dir2/from-lower-file

# 删除
mknod upper/from-lower-file3 c 0 0
mknod upper/from-lower-dir3 c 0 0
mkdir upper/from-lower-dir4
mknod upper/from-lower-dir4/from-lower-file c 0 0

# 透明
mkdir upper/from-lower-dir5
setfattr -n 'trusted.overlay.opaque' -v 'y' upper/from-lower-dir5  # 不能用 attr 命令，因为 attr 会自动添加 user. 前缀
echo 'from-upper' > upper/from-lower-dir5/from-upper-file


# 观察
# 操作后
echo '=== after ==='
echo '>>> tree merged/'
tree merged/
echo

echo '>>> cat merged/from-lower-file1'
cat merged/from-lower-file1
echo

echo '>>> cat merged/from-lower-file2'
cat merged/from-lower-file2
echo

echo '>>> cat merged/from-lower-dir1/from-lower-file'
cat merged/from-lower-dir1/from-lower-file
echo

echo '>>> cat merged/from-lower-dir2/from-lower-file'
cat merged/from-lower-dir2/from-lower-file
echo


# 清理缓存后
echo 2 > /proc/sys/vm/drop_caches
echo '=== after clear cache ==='
echo '>>> tree merged/'
tree merged/
echo

echo '>>> cat merged/from-lower-file1'
cat merged/from-lower-file1
echo

echo '>>> cat merged/from-lower-file2'
cat merged/from-lower-file2
echo

echo '>>> cat merged/from-lower-dir1/from-lower-file'
cat merged/from-lower-dir1/from-lower-file
echo

echo '>>> cat merged/from-lower-dir2/from-lower-file'
cat merged/from-lower-dir2/from-lower-file
echo

# 输出如下:
# === before ===
# >>> cat merged/from-lower-file1
# from-lower

# >>> ls merged/from-lower-dir1
# cat: merged/from-lower-dir1: 是一个目录

# === after ===
# >>> tree merged/
# merged/
# ├── from-lower-dir1
# │   └── from-lower-file
# ├── from-lower-dir2
# │   ├── from-lower-file
# │   └── from-upper-file
# ├── from-lower-dir4
# ├── from-lower-dir5
# │   └── from-upper-file
# ├── from-lower-file1
# ├── from-lower-file2
# ├── from-upper-dir
# └── from-upper-file

# 5 directories, 7 files

# >>> cat merged/from-lower-file1
# from-lower

# >>> cat merged/from-lower-file2
# from-upper

# >>> cat merged/from-lower-dir1/from-lower-file
# from-lower

# >>> cat merged/from-lower-dir2/from-lower-file
# from-upper

# === after clear cache ===
# >>> tree merged/
# merged/
# ├── from-lower-dir1
# │   ├── from-lower-file
# │   └── from-upper-file
# ├── from-lower-dir2
# │   ├── from-lower-file
# │   └── from-upper-file
# ├── from-lower-dir4
# ├── from-lower-dir5
# │   └── from-upper-file
# ├── from-lower-file1
# ├── from-lower-file2
# ├── from-upper-dir
# └── from-upper-file

# 5 directories, 8 files

# >>> cat merged/from-lower-file1
# from-upper

# >>> cat merged/from-lower-file2
# from-upper

# >>> cat merged/from-lower-dir1/from-lower-file
# from-upper

# >>> cat merged/from-lower-dir2/from-lower-file
# from-upper