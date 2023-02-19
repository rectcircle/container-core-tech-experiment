package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/unix"
)

func usage100PercentCPU() {
	i := 0
	for {
		i = i + 1
	}
}

func printSelfCPUPercent(p *process.Process) {
	cpuPercent, err := p.Percent(1 * time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%.2f%s\n", cpuPercent, "%")
}

func attachSelfToCgroup(cgroupPath string) error {
	return os.WriteFile(path.Join(cgroupPath, "cgroup.procs"), []byte(fmt.Sprint(os.Getpid())), 0o644)
}

func printSelfCgroup(subsys string) {
	selfCgroupBytes, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(selfCgroupBytes), "\n") {
		subSystems := strings.Split(strings.Split(line, ":")[1], ",")
		for _, now := range subSystems {
			if now == subsys {
				fmt.Println(line)
				return
			}
		}
	}
	panic(fmt.Errorf("subsys not found: %s", subsys))
}

// go build ./src/go/01-namespace/07-cgroup/v1/01-cpu && sudo ./01-cpu; rm -rf ./01-cpu

func main() {
	// 启动 CPU 负载
	go usage100PercentCPU()

	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		panic(err)
	}

	// 打印默认的情况
	fmt.Printf("当前进程的 cpu cgroup 为: ")
	printSelfCgroup("cpu")
	fmt.Printf("当前进程的 cpu 使用率为: ")
	printSelfCPUPercent(p)
	fmt.Println()

	// 在默认 cpu 层级，根 cgroup 创建一个名为 demo 的 cgroup。
	cpuDemoCgroupDir := "/sys/fs/cgroup/cpu/demo"
	err = os.Mkdir(cpuDemoCgroupDir, 777)
	if err != nil {
		panic(err)
	}
	defer func() {
		// 使用 unix 的 rmdir 系统调用。
		// 参考 https://github.com/opencontainers/runc/blob/main/libcontainer/cgroups/utils.go#L231
		err = unix.Rmdir(cpuDemoCgroupDir)
		if err != nil && err != unix.ENOENT {
			panic(err)
		}
	}()
	// 配置 CPU quota 为 0.2 核，0.2 * 100000 = 20000
	cpuCFSQuotaPath := path.Join(cpuDemoCgroupDir, "cpu.cfs_quota_us")
	cpuCFSPeriodPath := path.Join(cpuDemoCgroupDir, "cpu.cfs_period_us")
	cpuCFSPeriod, err := os.ReadFile(cpuCFSPeriodPath) // 100000
	if err != nil {
		panic(err)
	}
	cpuCFSPeriodInt, err := strconv.ParseInt(strings.TrimSpace(string(cpuCFSPeriod)), 10, 32)
	if err != nil {
		panic(err)
	}
	cpuCore := 0.2
	cpuCFSQuotaInt := int(cpuCore * float64(cpuCFSPeriodInt))
	err = os.WriteFile(cpuCFSQuotaPath, []byte(fmt.Sprint(cpuCFSQuotaInt)), 0o644)
	if err != nil {
		panic(err)
	}
	cpuCFSQuota, err := os.ReadFile(cpuCFSQuotaPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("创建 demo cpu cgroup, 配置 core: %f, 即\n  cpu.cfs_quota_us: %s\n  cpu.cfs_quota_us: %s\n",
		cpuCore,
		strings.TrimSpace(string(cpuCFSQuota)),
		strings.TrimSpace(string(cpuCFSPeriod)),
	)
	// 将当前进程加入到当前 cgroup
	err = attachSelfToCgroup(cpuDemoCgroupDir)
	if err != nil {
		panic(err)
	}
	defer func() {
		// 进程退出前，将当前进程脱离 demo cgroup，如果不处理，删除 cgroup 将报错 device or resource busy
		err = attachSelfToCgroup("/sys/fs/cgroup/cpu")
		if err != nil {
			panic(err)
		}
	}()
	fmt.Println("当前进程已加入 demo cpu cgroup")
	fmt.Println()

	fmt.Printf("当前进程的 cpu cgroup 为: ")
	printSelfCgroup("cpu")
	fmt.Printf("当前进程的 cpu 使用率为: ")
	printSelfCPUPercent(p)
}
