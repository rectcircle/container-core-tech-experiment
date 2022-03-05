//go:build linux

// sudo go run src/go/01-namespace/04-pid/main.go

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

const (
	sub = "sub"
)

var proccess_a_args = []string{
	"/bin/bash",
	"-xc",
	"bash -c 'nohup sleep infinity >/dev/null 2>&1 &' " +
		"&& echo $$ " +
		"&& ls /proc " +
		"&& ps -o pid,ppid,cmd " +
		"&& kill -9 1 " +
		"&& ps -o pid,ppid,cmd " +
		"&& exec sleep infinity ",
}

var proccess_e_args = []string{
	"/bin/bash",
	"-xc",
	"ls /proc " +
		"&& ps -eo pid,ppid,cmd | grep sleep | grep -v grep " +
		"&& kill -9 $(ps -eo pid,ppid | grep $PPID | awk '{print $1}' | sed -n '2p') " +
		"&& ps -eo pid,ppid,cmd | grep sleep | grep -v grep ",
}

func asyncExec(name string, arg ...string) int {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	return cmd.Process.Pid
}

func newNamespaceProccess() int {
	cmd := exec.Command(os.Args[0], "sub")
	// 创建新进程，并为该进程创建一个 PID Namespace（syscall.CLONE_NEWPID
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()
	return cmd.Process.Pid
}

func newNamespaceProccessFunc() {
	// seq: 0s
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
	time.Sleep(3 * time.Second)

	// seq: 3s
	fmt.Println("=== new pid namespace process ===")
	if err := syscall.Exec(proccess_a_args[0], proccess_a_args, nil); err != nil {
		panic(err)
	}
}

func registerSignalhandler() {
	// 处理 SIGCHLD 信号，解决僵尸进程阻塞 Namespace 进程退出的情况。
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGCHLD)
	go func() {
		for {
			<-sigs
			for {
				var wstatus syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
				if err != nil || pid == -1 || pid == 0 {
					break
				}
				fmt.Printf("*** pid %d exit by %d signal\n", pid, wstatus.Signal())
			}
		}
	}()
}

func mainProccess() {
	// seq: 0s
	fmt.Printf("=== main: %d\n", os.Getpid())
	// 注册 SIGCHLD 处理程序，会产生僵尸进程，而导致 PID Namespace 无法退出
	registerSignalhandler()
	// 1. 执行 newNamespaceExec，启动一个具有新的 PID Namespace 的进程
	pa := newNamespaceProccess()
	fmt.Printf("=== PA: %d\n", pa)

	time.Sleep(1 * time.Second)
	// seq: 1s

	// 构造 进程 b
	// 通过 nsenter 进入进程 a 的 PID Namespace
	pbp := asyncExec("/bin/bash", "-c", fmt.Sprintf("exec nsenter -p -t %d bash -c 'echo === PB: \"$$ in new pid namespace\" && exec sleep infinity'", pa))
	time.Sleep(1 * time.Second)
	// 此时 kill 掉 nsenter 进程，sleep infinity 就能称为满足条件的进程 b
	syscall.Kill(pbp, syscall.SIGKILL)

	// seq: 2s
	// 构造进程 c
	// Go 不能直接使用 setns 系统调用（因为 setns 不支持多线程调用，而 go runtime 是多线程），因此还是通过 nsenter 命令实现
	_ = asyncExec("/bin/bash", "-c", fmt.Sprintf("exec nsenter -p -t %d bash -c 'echo === PC: \"$$ in new pid namespace\" && exec sleep infinity'", pa))

	time.Sleep(2 * time.Second)
	// seq: 4s
	fmt.Println("=== old pid namespace process ===")
	_ = asyncExec(proccess_e_args[0], proccess_e_args[1:]...)

	time.Sleep(1 * time.Second)
	// seq: 5s

	return
}

func main() {
	switch len(os.Args) {
	case 1:
		mainProccess()
		return
	case 2:
		if os.Args[1] == sub {
			newNamespaceProccessFunc()
			return
		}
	}
	log.Fatalf("usage: %s [sub]", os.Args[0])
}
