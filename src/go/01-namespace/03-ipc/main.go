//go:build linux

// sudo go run ./src/go/01-namespace/03-ipc/main.go

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	sub = "sub"

	script = "ipcs -q"
)

func runTestScript(tip string) <-chan error {
	fmt.Println(tip)
	cmd := exec.Command("/bin/bash", "-cx", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	result := make(chan error)
	go func() {
		result <- cmd.Run()
	}()
	return result
}

func newNamespaceProccess() <-chan error {
	cmd := exec.Command(os.Args[0], "sub")
	// 创建新进程，并为该进程创建一个 IPC Namespace（syscall.CLONE_NEWIPC）
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	// 为了测试方便，同时创建 Mount Namespace 和 PID Namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC,
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

func createMsgQueue() {
	// key_t k = ftok("/system_v_msg_queue_test_1", 1);
	// int msgid = msgget(k, IPC_CREAT);

	if err := exec.Command("sh", "-c", "ipcmk -Q").Run(); err != nil {
		panic(err)
	}
}

func newNamespaceProccessFunc() <-chan error {
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
	// 挂载当前 PID Namespace 的 proc
	// 因为在新的 Mount Namespace 中执行，所有其他进程的目录树不受影响
	// 等价命令为：mount -t proc proc /proc
	// 更多参见：https://man7.org/linux/man-pages/man8/mount.8.html
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		panic(err)
	}
	// 创建一个 System V 消息队列
	createMsgQueue()
	return runTestScript("=== new ipc namespace process ===")
}

func oldNamespaceProccess() <-chan error {
	return runTestScript("=== old namespace process ===")
}

func main() {
	switch len(os.Args) {
	case 1:
		// 1. 执行 newNamespaceExec，启动一个具有新的 IPC Namespace 的进程
		r1 := newNamespaceProccess()
		time.Sleep(5 * time.Second)
		// 3. 创建新的进程（不创建 Namespace），并执行测试脚本
		r2 := oldNamespaceProccess()
		err1, err2 := <-r1, <-r2
		if err1 != nil {
			panic(err1)
		}
		if err2 != nil {
			panic(err2)
		}
		return
	case 2:
		if os.Args[1] == sub {
			if err := <-newNamespaceProccessFunc(); err != nil {
				panic(err)
			}
			return
		}
	}
	log.Fatalf("usage: %s [sub]", os.Args[0])
}
