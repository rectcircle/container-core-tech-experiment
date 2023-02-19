package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/unix"
)

const (
	MB                  = 1024 * 1024
	memoryDemoCgroupDir = "/sys/fs/cgroup/memory/demo"
)

const (
	SubCommandMonitor = "monitor"
	SubCommandAlloc   = "alloc"
)

var PinnedAllocMemory [][]byte

func allocMemory(size int64) {
	b := make([]byte, size)
	for i := int64(0); i < size; i++ {
		b[i] = byte(i)
	}
	PinnedAllocMemory = append(PinnedAllocMemory, b)
}

func printRSSMemory(p *process.Process) {
	m, err := p.MemoryInfo()
	if err != nil {
		panic(err)
	}
	s := 0
	for i := 0; i < len(PinnedAllocMemory[0]); i++ {
		s += int(PinnedAllocMemory[0][i])
	}
	fmt.Printf("  pid %d: %s\n", p.Pid, m.String())
}

func handleOOMEvent() {
	fmt.Println("=== handle oom event ...")
	usage_in_bytes, err := os.ReadFile(path.Join(memoryDemoCgroupDir, `memory.usage_in_bytes`))
	if err != nil {
		panic(err)
	}
	fmt.Printf("memory.usage_in_bytes: \n%s\n", string(usage_in_bytes))
	ommControlBytes, err := os.ReadFile(path.Join(memoryDemoCgroupDir, `memory.oom_control`))
	if err != nil {
		panic(err)
	}
	fmt.Printf("memory.oom_control: \n%s\n", string(ommControlBytes))
	procsBytes, err := os.ReadFile(path.Join(memoryDemoCgroupDir, "cgroup.procs"))
	if err != nil {
		panic(err)
	}
	procs := strings.Split(string(procsBytes), "\n")
	for _, pidStr := range procs {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			break
		}
		pid64, _ := strconv.ParseInt(pidStr, 10, 32)
		p, err := process.NewProcess(int32(pid64))
		if err != nil {
			panic(err)
		}
		status, err := p.Status()
		if err != nil {
			panic(err)
		}
		oom_score, _ := os.ReadFile(fmt.Sprintf("/proc/%d/oom_score", p.Pid))
		oom_adj, _ := os.ReadFile(fmt.Sprintf("/proc/%d/oom_adj ", p.Pid))
		oom_score_adj, _ := os.ReadFile(fmt.Sprintf("/proc/%d/oom_score_adj", p.Pid))
		fmt.Printf("pid %d state is: %v (oom_adj=%s, oom_score=%s, oom_score_adj=%s)\n", p.Pid, status, strings.TrimSpace(string(oom_adj)), strings.TrimSpace(string(oom_score)), strings.TrimSpace(string(oom_score_adj)))
		// 如果要模拟内核的 oom-killer 的逻辑： oom_score + oom_adj + oom_score_adj 排序，并给最大的那个发送 SIGKILL(9) 信号。
		//
		// 但是 oom-killer 作为 Linux 内存分配的最后的保证，非常不建议禁用 oom-killer。
		//
		// 因此推荐如下：
		//   1. 通过调整 `oom_score_adj` 来调整进程的内存优先级，以自定义 oom 时 killer 的顺序。
		//   2. 可以通过监听 oom killer 事件，统计 oom 发生的频率，以辅助调度。
		//   2. 如需收集记录进程被 kill 的日志，只能通过 `dmesg` 或 /var/log/syslog 内核日志获取信息 （killed process）。
	}
}

func createOOMEventHandler() {
	// https://www.jianshu.com/p/f2403e33c766
	// https://access.redhat.com/documentation/zh-cn/red_hat_enterprise_linux/7/html/resource_management_guide/sec-memory
	var events [128]unix.EpollEvent
	var buf [8]byte
	// 创建epoll实例
	epollFd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		panic(err)
	}
	// 创建eventfd实例
	efd, _ := unix.Eventfd(0, unix.EFD_CLOEXEC)

	event := unix.EpollEvent{
		Fd:     int32(efd),
		Events: unix.EPOLLHUP | unix.EPOLLIN | unix.EPOLLERR,
	}
	// 将eventfd添加到epoll中进行监听
	unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, int(efd), &event)

	// 打开oom_control文件
	evtFile, _ := os.Open(path.Join(memoryDemoCgroupDir, "memory.oom_control"))

	// 注册oom事件，当有oom事件时，eventfd将会有数据可读
	data := fmt.Sprintf("%d %d", efd, evtFile.Fd())
	ioutil.WriteFile(path.Join(memoryDemoCgroupDir, "cgroup.event_control"), []byte(data), 0o700)

	for {
		// 开始监听oom事件
		n, err := unix.EpollWait(epollFd, events[:], -1)
		if err == nil {
			for i := 0; i < n; i++ {
				// 消费掉eventfd的数据
				unix.Read(int(events[i].Fd), buf[:])
				// 处理 oom envent
				handleOOMEvent()
			}
		}
	}
}

func monitor() {
	// 1.1 在默认 memory 层级，根 cgroup 创建一个名为 demo 的 cgroup。
	if err := os.Mkdir(memoryDemoCgroupDir, 0o755); err != nil {
		panic(err)
	}
	defer func() {
		// 使用 unix 的 rmdir 系统调用。
		// 参考 https://github.com/opencontainers/runc/blob/main/libcontainer/cgroups/utils.go#L231
		if err := unix.Rmdir(memoryDemoCgroupDir); err != nil && err != unix.ENOENT {
			panic(err)
		}
	}()
	// 1.2 配置该 cgroup 的 memory.limit_in_bytes 为 100 MB
	if err := os.WriteFile(filepath.Join(memoryDemoCgroupDir, "memory.limit_in_bytes"), []byte("100M"), 0o644); err != nil {
		panic(err)
	}

	// 2.1 创建两个进程，先加入 cgroup，再分配 70MB 左右内存（注意：实测先分配内存再加入不会被 cgroup 感知？）。
	proc1 := exec.Command("/proc/self/exe", fmt.Sprint(70*MB))
	proc1.Stdout = os.Stdout
	proc1.Stderr = os.Stderr
	if err := proc1.Start(); err != nil {
		panic(err)
	}
	proc1Done := make(chan error, 1)
	go func() { proc1Done <- proc1.Wait(); close(proc1Done) }()
	defer func() { proc1.Process.Kill(); proc1.Wait() }()
	fmt.Printf("proc1 %d start ...\n", proc1.Process.Pid)
	time.Sleep(1 * time.Second)

	proc2 := exec.Command("/proc/self/exe", fmt.Sprint(70*MB))
	proc2.Stdout = os.Stdout
	proc2.Stderr = os.Stderr
	if err := proc2.Start(); err != nil {
		panic(err)
	}
	proc2Done := make(chan error, 1)
	go func() { proc2Done <- proc2.Wait(); close(proc2Done) }()
	defer func() { proc2.Process.Kill(); proc2.Wait() }()
	fmt.Printf("proc2 %d start ...\n", proc2.Process.Pid)
	time.Sleep(1 * time.Second)

	// 2.3 观察两个进程的存活状态
	select {
	case <-proc1Done:
		fmt.Printf("proc1 %d has exited: %s\n", proc1.Process.Pid, proc1.ProcessState.String())
	default:
		fmt.Printf("proc1 %d running\n", proc1.Process.Pid)
	}
	select {
	case <-proc2Done:
		fmt.Printf("proc2 %d has exited: %s\n", proc2.Process.Pid, proc2.ProcessState.String())
	default:
		fmt.Printf("proc2 %d is running\n", proc2.Process.Pid)
	}
	fmt.Println()

	// 3.1 再创建一个子进程，加入上面创建的 demo memory cgroup，并设置该进程的 oom_score_adj 为 1000，并申请 50 MB 内存
	proc3 := exec.Command("/proc/self/exe", fmt.Sprint(50*MB), "1000")
	proc3.Stdout = os.Stdout
	proc3.Stderr = os.Stderr
	if err := proc3.Start(); err != nil {
		panic(err)
	}
	proc3Done := make(chan error, 1)
	go func() { proc3Done <- proc3.Wait(); close(proc3Done) }()
	defer func() { proc3.Process.Kill(); proc3.Wait() }()
	fmt.Printf("proc3 %d start ...\n", proc3.Process.Pid)
	time.Sleep(1 * time.Second)

	select {
	case <-proc1Done:
		fmt.Printf("proc1 %d has exited: %s\n", proc1.Process.Pid, proc1.ProcessState.String())
	default:
		fmt.Printf("proc1 %d running\n", proc1.Process.Pid)
	}
	select {
	case <-proc2Done:
		fmt.Printf("proc2 %d has exited: %s\n", proc2.Process.Pid, proc2.ProcessState.String())
	default:
		fmt.Printf("proc2 %d is running\n", proc2.Process.Pid)
	}
	select {
	case <-proc3Done:
		fmt.Printf("proc3 %d has exited: %s\n", proc3.Process.Pid, proc3.ProcessState.String())
	default:
		fmt.Printf("proc3 %d is running\n", proc3.Process.Pid)
	}
	fmt.Println()

	// ! 注意：这里只是一个演示，在生产环境不建议禁用 oom-killer
	// 4.1 监控进程写入 1 到 `memory.oom_control` 禁用内核 oom killer，并通过 `cgroup.event_control` 文件配置当前进程接受 OOM 事件。
	if err := os.WriteFile(filepath.Join(memoryDemoCgroupDir, "memory.oom_control"), []byte("1"), 0o644); err != nil {
		panic(err)
	}
	go createOOMEventHandler()
	// 5.1 监控再创建一个子进程，加入上面创建的 demo memory cgroup，并申请 50 MB 内存
	proc4 := exec.Command("/proc/self/exe", fmt.Sprint(50*MB))
	proc4.Stdout = os.Stdout
	proc4.Stderr = os.Stderr
	if err := proc4.Start(); err != nil {
		panic(err)
	}
	proc4Done := make(chan error, 1)
	go func() { proc4Done <- proc4.Wait(); close(proc4Done) }()
	defer func() { proc4.Process.Kill(); proc4.Wait() }()
	fmt.Printf("proc4 %d start ...\n", proc4.Process.Pid)
	time.Sleep(1 * time.Second)
	// time.Sleep(60 * time.Second)
}

// sudo swapoff -a
// go build ./src/go/01-namespace/07-cgroup/v1/02-memory && sudo ./02-memory; rm -rf ./02-memory
// sudo dmesg -T | egrep -i 'killed process'
// sudo swapon -a

func main() {
	if len(os.Args) == 1 {
		monitor()
	} else {
		s, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			panic(err)
		}
		// 2.1.a 加入 Cgroup。
		if err = os.WriteFile(path.Join(memoryDemoCgroupDir, "cgroup.procs"), []byte(fmt.Sprint(os.Getpid())), 0o644); err != nil {
			panic(err)
		}
		// 3.1.a 配置进程的 `oom_score_adj`
		if len(os.Args) == 3 {
			adj, err := strconv.ParseInt(os.Args[2], 10, 64)
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile("/proc/self/oom_score_adj", []byte(fmt.Sprint(adj)), 0o644); err != nil {
				panic(err)
			}
		}
		// 2.2.b 申请内存
		allocMemory(s)
		p, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			panic(err)
		}
		printRSSMemory(p)
		time.Sleep(60 * time.Second)
	}
}
