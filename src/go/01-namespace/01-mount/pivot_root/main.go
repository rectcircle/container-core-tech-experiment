//go:build linux

// sudo go run ./src/go/01-namespace/01-mount/pivot_root/main.go

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

const (
	sub = "sub"

	newroot = "data/busybox/rootfs"

	script = "export PATH=/bin && ls / && ls /bin"
)

func newNamespaceExec() <-chan error {
	cmd := exec.Command(os.Args[0], "sub")
	// 创建新进程，并为该进程创建一个 Mount Namespace（syscall.CLONE_NEWNS）
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	result := make(chan error)
	go func() {
		result <- cmd.Run()
	}()
	return result
}

func pivotRootAndRun() {
	// 首先，需要阻止挂载事件传播到其他 Mount Namespace，参见：https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#NOTES
	// 如果不执行这个语句， cat /proc/self/mountinfo 所有行将会包含 shared，这样在这个子进程中执行 mount 其他进程也会受影响
	// 关于 Shared subtrees 更多参见：
	//   https://segmentfault.com/a/1190000006899213
	//   https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#SHARED_SUBTREES
	// 下面语句的含义是：重新递归挂（MS_REC）载 / ，并设置为不共享（MS_SLAVE 或 MS_PRIVATE）
	// 说明：
	//   MS_SLAVE 换成 MS_PRIVATE 也能达到同样的效果
	//   等价于执行：mount --make-rslave / 命令
	if err := syscall.Mount("", "/", "", syscall.MS_SLAVE|syscall.MS_REC, ""); err != nil {
		panic(err)
	}
	// 确保 new_root 是一个挂载点
	if err := syscall.Mount(newroot, newroot, "", syscall.MS_BIND, ""); err != nil {
		panic(err)
	}
	// 切换根挂载目录，将 new_root 挂载到根目录，将旧的根目录挂载到 put_old 目录下
	// 可以通过 pivot_root(".", ".") 来实现免除创建临时目录，参见： https://github.com/opencontainers/runc/commit/f8e6b5af5e120ab7599885bd13a932d970ccc748
	// - new_root 和 put_old 必须是一个目录
	// - new_root 和 put_old 不能和当前根目录相同。
	// - put_old 必须是 new_root 的子孙目录
	// - new_root 必须是挂载点的路径，但不能是根目录。如果不是的话，可以通过 mount bind 方式转换为一个挂载点（参见上一个命令）。
	// - 旧的根目录必须是挂载点。
	if err := os.Chdir(newroot); err != nil {
		panic(err)
	}
	if err := syscall.PivotRoot(".", "."); err != nil {
		panic(err)
	}
	// 根目录已经切换了，所以之前的工作目录已经不存在了，所以需要将 working directory 切换到根目录
	if err := os.Chdir("/"); err != nil {
		panic(err)
	}
	fmt.Println("=== new mount namespace and pivot_root process ===")
	cmd := exec.Command("/bin/sh", "-xc", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func main() {
	// 1. 执行 newNamespaceExec，启动一个具有新的 Mount Namespace 的进程
	// 2. 该进程执行 pivotRootAndRun，配置 Mount，调用 pivotRoot 并运行测试脚本
	switch len(os.Args) {
	case 1:
		r1 := newNamespaceExec()
		err1 := <-r1
		if err1 != nil {
			panic(err1)
		}
		return
	case 2:
		if os.Args[1] == sub {
			pivotRootAndRun()
			return
		}
	}
	log.Fatalf("usage: %s [sub]", os.Args[0])
}
