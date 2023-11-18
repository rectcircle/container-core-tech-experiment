#!/usr/bin/env bash
# sudo apt install attr
# sudo ./src/shell/03-overlayfs/01-operate-merged.sh

# 创建并进入测试目录
exp_base_dir=/tmp/overlayfs-exp/01-operate-merged
umount $exp_base_dir/merged >/dev/null 2>&1
rm -rf $exp_base_dir && mkdir -p $exp_base_dir
cd $exp_base_dir

# 准备 lower、merged、upper、work 目录
mkdir -p lower1 merged upper work
mkdir -p lower1/from-lower1-dir lower1/from-lower1-dir2 lower1/from-lower1-dir3
mkdir -p lower1/from-lower1-dir/subdir
echo 'from-lower1' > lower1/from-lower1-dir/subdir/file
echo 'from-lower1' > lower1/from-lower1-dir/file
echo 'from-lower1' > lower1/from-lower1-file
echo 'from-lower1' > lower1/from-lower1-dir2/file1
echo 'from-lower1' > lower1/from-lower1-dir2/file2
echo 'from-lower1' > lower1/from-lower1-file2
echo 'from-lower1' > lower1/from-lower1-dir3/file1
echo 'from-lower1' > lower1/from-lower1-dir3/file2
echo 'from-lower1' > lower1/from-lower1-file3

# 生成 merged
mount -t overlay overlay -olowerdir=lower1,upperdir=upper,workdir=work merged

# 在 merged 新建
echo 'from-merged' > merged/from-merged-file
echo 'from-merged' > merged/from-lower1-dir/from-merged-file
mkdir -p merged/from-merged-dir/subdir
mkdir -p merged/from-lower1-dir/from-merged-dir/subdir

# 在 merged 修改
echo 'from-merged' >> merged/from-lower1-file
echo 'from-merged' >> merged/from-lower1-dir/file
touch -t 197001010000 merged/from-lower1-dir/subdir

# 在 merged 删除
rm -rf merged/from-lower1-dir2
rm -rf merged/from-lower1-file2

# 在 merged 移动
mv merged/from-lower1-file3 merged/from-lower1-file3-moved
mv merged/from-lower1-dir3 merged/from-lower1-dir3-moved

# 观察情况
echo '>>> tree merged/'
tree merged/
echo

echo '>>> tree upper/'
tree upper/
echo

echo '>>> cat upper/from-lower1-file'
cat upper/from-lower1-file
echo

echo '>>> cat upper/from-lower1-dir/file'
cat upper/from-lower1-dir/file
echo

echo '>>> stat upper/from-lower1-dir/subdir'
stat upper/from-lower1-dir/subdir
echo


echo '>>> stat upper/from-lower1-dir2'
stat upper/from-lower1-dir2
echo

echo '>>> attr -l upper/from-lower1-dir2'
attr -l upper/from-lower1-dir2
echo

echo '>>> stat upper/from-lower1-file2'
stat upper/from-lower1-file2
echo

echo '>>> attr -l upper/from-lower1-file2'
attr -l upper/from-lower1-file2
echo


# 输出如下:
# >>> tree merged/
# merged/
# ├── from-lower1-dir
# │   ├── file
# │   ├── from-merged-dir
# │   │   └── subdir
# │   ├── from-merged-file
# │   └── subdir
# │       └── file
# ├── from-lower1-dir3-moved
# │   ├── file1
# │   └── file2
# ├── from-lower1-file
# ├── from-lower1-file3-moved
# ├── from-merged-dir
# │   └── subdir
# └── from-merged-file

# 7 directories, 8 files

# >>> tree upper/
# upper/
# ├── from-lower1-dir
# │   ├── file
# │   ├── from-merged-dir
# │   │   └── subdir
# │   ├── from-merged-file
# │   └── subdir
# ├── from-lower1-dir2
# ├── from-lower1-dir3
# ├── from-lower1-dir3-moved
# │   ├── file1
# │   └── file2
# ├── from-lower1-file
# ├── from-lower1-file2
# ├── from-lower1-file3
# ├── from-lower1-file3-moved
# ├── from-merged-dir
# │   └── subdir
# └── from-merged-file

# 7 directories, 11 files

# >>> cat upper/from-lower1-file
# from-lower1
# from-merged

# >>> cat upper/from-lower1-dir/file
# from-lower1
# from-merged

# >>> stat upper/from-lower1-dir/subdir
#   文件：upper/from-lower1-dir/subdir
#   大小：4096            块：8          IO 块：4096   目录
# 设备：fe01h/65025d      Inode：1625520     硬链接：2
# 权限：(0755/drwxr-xr-x)  Uid：(    0/    root)   Gid：(    0/    root)
# 最近访问：2023-11-18 22:39:17.350643792 +0800
# 最近更改：1970-01-01 00:00:00.000000000 +0800
# 最近改动：2023-11-18 22:39:17.222644562 +0800
# 创建时间：2023-11-18 22:39:17.222644562 +0800

# >>> stat upper/from-lower1-dir2
#   文件：upper/from-lower1-dir2
#   大小：0               块：0          IO 块：4096   字符特殊文件
# 设备：fe01h/65025d      Inode：1625522     硬链接：4     设备类型：0,0
# 权限：(0000/c---------)  Uid：(    0/    root)   Gid：(    0/    root)
# 最近访问：2023-11-18 22:39:17.222644562 +0800
# 最近更改：2023-11-18 22:39:17.222644562 +0800
# 最近改动：2023-11-18 22:39:17.346643816 +0800
# 创建时间：2023-11-18 22:39:17.222644562 +0800

# >>> attr -l upper/from-lower1-dir2

# >>> stat upper/from-lower1-file2
#   文件：upper/from-lower1-file2
#   大小：0               块：0          IO 块：4096   字符特殊文件
# 设备：fe01h/65025d      Inode：1625522     硬链接：4     设备类型：0,0
# 权限：(0000/c---------)  Uid：(    0/    root)   Gid：(    0/    root)
# 最近访问：2023-11-18 22:39:17.222644562 +0800
# 最近更改：2023-11-18 22:39:17.222644562 +0800
# 最近改动：2023-11-18 22:39:17.346643816 +0800
# 创建时间：2023-11-18 22:39:17.222644562 +0800

# >>> attr -l upper/from-lower1-file2
