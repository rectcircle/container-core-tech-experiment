//go:build linux

// sudo go run ./src/go/01-namespace/01-mount/main.go

package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

const script = "ls data/binding/target " +
	"&& readlink /proc/self/ns/mnt " +
	"&& cat /proc/self/mounts | grep data/binding/target || true" +
	"&& cat /proc/self/mountinfo | grep data/binding/target || true " +
	"&& cat /proc/self/mountstats | grep data/binding/target || true " +
	"&& sleep 10"

func newNamespaceExec() <-chan error {
	cmd := exec.Command("/bin/bash", "-c",
		// 首先，需要阻止挂载事件传播到其他 Mount Namespace，参见：https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#NOTES
		// 如果不执行这个语句， cat /proc/self/mountinfo 所有行将会包含 shared，这样在这个子进程中执行 mount 其他进程也会受影响
		// 关于 Shared subtrees 更多参见：
		//   https://segmentfault.com/a/1190000006899213
		//   https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#SHARED_SUBTREES
		// 下面语句的含义是：重新递归挂（r）载 / ，并设置为私有
		// 说明：
		//   --make-rprivate 换成 --make-rslave 也能达到同样的效果
		//   等价于系统调用：mount(NULL, "/", NULL , MS_PRIVATE | MS_REC, NULL)
		// Go 语言对应 api 为：syscall.Mount
		"mount --make-rprivate /"+
			// 将 data/binding/source 挂载（绑定）到 data/binding/target
			// 因为在新的 Mount Namespace 中执行，所有其他进程的目录树不受影响
			// 等价系统调用为：mount("data/binding/source", "data/binding/target", NULL, MS_BIND, NULL);
			// 更多参见：https://man7.org/linux/man-pages/man8/mount.8.html
			"&& mount --bind data/binding/source data/binding/target"+
			"&& echo '=== new mount namespace process ===' "+
			"&& set -x &&"+
			script)
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

func oldNamespaceExec() <-chan error {
	cmd := exec.Command("/bin/bash", "-c", "echo '=== old namespace process ===' && set -x && "+script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	result := make(chan error)
	go func() {
		result <- cmd.Run()
	}()
	return result
}

func main() {
	r1 := newNamespaceExec()
	time.Sleep(5 * time.Second)
	// 创建新的进程（不创建 Namespace），并执行测试命令
	r2 := oldNamespaceExec()
	err1, err2 := <-r1, <-r2
	if err1 != nil {
		panic(err1)
	}
	if err2 != nil {
		panic(err2)
	}
}
