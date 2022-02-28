//go:build linux

// sudo go run ./src/go/01-namespace/02-uts/main.go

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

	script = "hostname && hostname --nis || true"
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
	// 创建新进程，并为该进程创建一个 UTS Namespace（syscall.CLONE_NEWUTS）
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
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

func newNamespaceProccessFunc() <-chan error {
	if err := syscall.Sethostname([]byte("new-hostname")); err != nil {
		panic(err)
	}
	if err := syscall.Setdomainname([]byte("new-domainname")); err != nil {
		panic(err)
	}
	return runTestScript("=== new uts namespace process ===")
}

func oldNamespaceProccess() <-chan error {
	return runTestScript("=== old namespace process ===")
}

func main() {
	switch len(os.Args) {
	case 1:
		// 1. 执行 newNamespaceExec，启动一个具有新的 UTS Namespace 的进程
		r1 := newNamespaceProccess()
		time.Sleep(5 * time.Second)
		// 3. 创建新的进程（不创建 Namespace），并执行测试命令
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
		// 2. 该进程执行 newNamespaceProccessFunc，配置 hostname 和 domainname，并执行测试脚本
		if os.Args[1] == sub {
			if err := <-newNamespaceProccessFunc(); err != nil {
				panic(err)
			}
			return
		}
	}
	log.Fatalf("usage: %s [sub]", os.Args[0])
}
