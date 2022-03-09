#!/usr/bin/env bash

abs_dir=$(cd $(dirname $0); pwd)
cd $abs_dir

# 开始测试
echo '=== origin ==='
tree

sudo mount --bind source1 target1
echo '=== bind ./source1 ./target1 ==='
tree

sudo mount --bind source2 target1/target2
echo '=== / is share & ./target1 is share ==='
echo '=== bind ./source2 ./target1/target2 : ./source1/target2 ✅  ==='
cat /proc/self/mountinfo | grep "/ / "
cat /proc/self/mountinfo | grep "propagation"
tree

sudo umount target1/target2
sudo mount --make-slave target1
sudo mount --bind source2 source1/target2
echo '=== / is share & ./target1 is slave ==='
echo '=== bind ./source2 ./source1/target2 : ./target1/target2/ ✅  ==='
cat /proc/self/mountinfo | grep "/ / "
cat /proc/self/mountinfo | grep "propagation"
tree

sudo umount source1/target2
sudo mount --bind source2 target1/target2
echo '=== bind ./source2 ./target1/target2 : ./source1/target2 ❌ ==='
cat /proc/self/mountinfo | grep "/ / "
cat /proc/self/mountinfo | grep "propagation"
tree

sudo umount target1/target2
sudo umount target1
