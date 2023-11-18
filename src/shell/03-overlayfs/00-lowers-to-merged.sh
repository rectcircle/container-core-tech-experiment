#!/usr/bin/env bash
# sudo ./src/shell/03-overlayfs/00-lowers-to-merged.sh

# 创建并进入测试目录
exp_base_dir=/tmp/overlayfs-exp/00-lowers-to-merged
umount $exp_base_dir/merged >/dev/null 2>&1
rm -rf $exp_base_dir && mkdir -p $exp_base_dir
cd $exp_base_dir

# 准备 lower、merged 目录
mkdir -p lower1 lower2 merged
# lower1/
mkdir -p lower1/from-lower1-dir lower1/from-lower1-dir/subdir
echo 'from-lower1' > lower1/from-lower1-dir/file
echo 'from-lower1' > lower1/from-lower1-file
echo 'from-lower1' > lower1/from-lower1-lower2-file
mkdir -p lower1/from-lower1-lower2-dir lower1/from-lower1-lower2-dir/subdir
echo 'from-lower1' > lower1/from-lower1-lower2-dir/file
echo 'from-lower1' > lower1/from-lower1-lower2-dir/from-lower1-file
# lower2/
mkdir -p lower2/from-lower2-dir lower2/from-lower2-dir/subdir
echo 'from-lower2' > lower2/from-lower2-dir/file
echo 'from-lower2' > lower2/from-lower2-file
echo 'from-lower2' > lower2/from-lower1-lower2-file
mkdir -p lower2/from-lower1-lower2-dir lower2/from-lower1-lower2-dir/subdir
echo 'from-lower2' > lower2/from-lower1-lower2-dir/file
echo 'from-lower2' > lower2/from-lower1-lower2-dir/from-lower2-file
touch -t 197001010000 lower2/from-lower1-lower2-dir

# 生成 merged
mount -t overlay overlay -olowerdir=lower2:lower1 merged

# 观察情况
echo '>>> tree lower1/'
tree lower1
echo

echo '>>> tree lower2/'
tree lower2
echo

echo '>>> tree merged/'
tree merged/
echo

echo '>>> cat merged/from-lower1-lower2-file'
cat merged/from-lower1-lower2-file
echo

echo '>>> cat merged/from-lower1-lower2-dir/file'
cat merged/from-lower1-lower2-dir/file
echo

echo '>>> stat merged/from-lower1-lower2-dir'
stat merged/from-lower1-lower2-dir
echo


# 输出如下:

# >>> tree lower1/
# lower1
# ├── from-lower1-dir
# │   ├── file
# │   └── subdir
# ├── from-lower1-file
# ├── from-lower1-lower2-dir
# │   ├── file
# │   ├── from-lower1-file
# │   └── subdir
# └── from-lower1-lower2-file

# 4 directories, 5 files

# >>> tree lower2/
# lower2
# ├── from-lower1-lower2-dir
# │   ├── file
# │   ├── from-lower2-file
# │   └── subdir
# ├── from-lower1-lower2-file
# ├── from-lower2-dir
# │   ├── file
# │   └── subdir
# └── from-lower2-file

# 4 directories, 5 files

# >>> tree merged/
# merged/
# ├── from-lower1-dir
# │   ├── file
# │   └── subdir
# ├── from-lower1-file
# ├── from-lower1-lower2-dir
# │   ├── file
# │   ├── from-lower1-file
# │   ├── from-lower2-file
# │   └── subdir
# ├── from-lower1-lower2-file
# ├── from-lower2-dir
# │   ├── file
# │   └── subdir
# └── from-lower2-file

# 6 directories, 8 files

# >>> stat merged/from-lower1-lower2-dir

# >>> cat merged/from-lower1-lower2-file
# from-lower2

# >>> cat merged/from-lower1-lower2-dir/file
# from-lower2

# >>> stat merged/from-lower1-lower2-dir
#   文件：merged/from-lower1-lower2-dir
#   大小：4096            块：8          IO 块：4096   目录
# 设备：36h/54d   Inode：1625488     硬链接：1
# 权限：(0755/drwxr-xr-x)  Uid：(    0/    root)   Gid：(    0/    root)
# 最近访问：2023-11-18 21:52:10.813161498 +0800
# 最近更改：1970-01-01 00:00:00.000000000 +0800
# 最近改动：2023-11-18 21:52:10.809161535 +0800
# 创建时间：2023-11-18 21:52:10.809161535 +0800