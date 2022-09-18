//go:build linux

// sudo go run src/go/01-namespace/05-network/main.go

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	sub = "sub"
)

func runTestScript(tip string, script string) error {
	fmt.Println(tip)
	cmd := exec.Command("/bin/bash", "-cx", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func newNamespaceProccess() (<-chan error, int) {
	cmd := exec.Command(os.Args[0], "sub")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNET,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	result := make(chan error)
	cmd.Start()
	go func() {
		result <- cmd.Wait()
	}()
	return result, cmd.Process.Pid
}

func newNamespaceProccessFunc() error {
	// 时序 1: 刚创建的 Network Namespace， ip addr 只能看到 lo 接口
	if err := runTestScript("(1) === new namespace process ===", "ip addr"); err != nil {
		return err
	}
	fmt.Println()
	time.Sleep(2 * time.Second)
	// 时序 3: 此时已经配置好了 veth，ip addr 可以看到 veth 接口
	if err := runTestScript("(3) === new namespace process ===", "ip addr && ip route"); err != nil {
		return err
	}
	fmt.Println()
	// 时序 4: ping veth 另一端
	if err := runTestScript("(4) === new namespace process ===", "ping -c 1 172.16.0.1"); err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func oldNamespaceProccess(pid int) error {
	time.Sleep(1 * time.Second)
	// 时序 2: 配置 veth
	err := configVeth(pid)
	if err != nil {
		return err
	}
	if err := runTestScript("(2) === old namespace process ===", "ip addr show veth0"); err != nil {
		return err
	}
	fmt.Println()
	time.Sleep(2 * time.Second)
	return nil
}

func configVeth(pid int) error {
	const (
		vethName     = "veth0"
		vethPeerName = "veth0container"
		vethNet      = "172.16.0.1/16"
		gatewayIP    = "172.16.0.1"
		vethPeerNet  = "172.16.0.2/16"
	)
	// 1. 创建并配置位于根 Network Namespace 的一侧
	//    a. 创建 veth
	la := netlink.NewLinkAttrs()
	la.Name = vethName // 当前 veth 的命令
	// la.MasterIndex = br.Attrs().Index  // 如果是要和 bridge 连接，可以配置该属性
	if err := netlink.LinkAdd(&netlink.Veth{
		LinkAttrs: la,
		PeerName:  vethPeerName, // 当前 veth 另一端的名字
	}); err != nil {
		return err
	}
	ipNet, err := netlink.ParseIPNet(vethNet)
	if err != nil {
		return err
	}
	//    b. 给一侧 veth 设置 ip
	netlink.AddrAdd(netlink.NewLinkBond(netlink.LinkAttrs{Name: vethName}), &netlink.Addr{IPNet: ipNet})
	//    c. 启动一侧 veth
	netlink.LinkSetUp(netlink.NewLinkBond(netlink.LinkAttrs{Name: vethName}))

	// 2. 将 veth 的另一侧加入新的 Network Namespace
	//     a. 获取到要加入到新的 Network Namespace 的 veth 的另一侧
	peerLink, err := netlink.LinkByName(vethPeerName)
	if err != nil {
		return err
	}
	//     b. 获取到新的 Network Namespace 的 proc 上的引用
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/ns/net", pid), os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	//     c. 将 veth 的另一侧加入新的 Network Namespace
	if err = netlink.LinkSetNsFd(peerLink, int(f.Fd())); err != nil {
		return err
	}

	// 3. 让当前的进程 (父进程) 进入新的 Network Namespace
	//     a. 记录当前的 Network Namespace
	origns, err := netns.Get()
	if err != nil {
		return err
	}
	defer origns.Close()
	//     b. 后文 netns.Set 利用的是 setns 系统调用配置的线程，因此需要禁止 go 将当前协程调度到其他操作系统线程中。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	//     c. 当前进程 (父进程) 加入到新的 Network Namespace 中。
	if err = netns.Set(netns.NsHandle(f.Fd())); err != nil {
		return err
	}
	//     d. 在当前函数执行完成后，恢复现场
	defer netns.Set(origns)

	// 4. 当前进程已经在新的 Network Namespace 中了，去配置已经在新的 Network Namespace 中的另一侧 veth
	//     a. veth 配置 ip、子网
	ipNet, err = netlink.ParseIPNet(vethPeerNet)
	if err != nil {
		return nil
	}
	if err = netlink.AddrAdd(netlink.NewLinkBond(netlink.LinkAttrs{Name: vethPeerName}), &netlink.Addr{IPNet: ipNet}); err != nil {
		return err
	}
	//     b. 启动 veth 和 lo 设备
	if err = netlink.LinkSetUp(netlink.NewLinkBond(netlink.LinkAttrs{Name: vethPeerName})); err != nil {
		return nil
	}
	if err = netlink.LinkSetUp(netlink.NewLinkBond(netlink.LinkAttrs{Name: "lo"})); err != nil {
		return nil
	}
	//     c. 配置新的 Network Namespace 的路由表
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	gwIP := net.ParseIP(gatewayIP)
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        gwIP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

func main() {
	switch len(os.Args) {
	case 1:
		// 1. 执行 newNamespaceExec，启动一个具有新的 Network Namespace 的进程
		r1, pid := newNamespaceProccess()
		// 2. 在根 Network Namespace 中执行。
		err2 := oldNamespaceProccess(pid)
		if err2 != nil {
			panic(err2)
		}
		err1 := <-r1
		if err1 != nil {
			panic(err1)
		}
		if err := runTestScript("(5) === old namespace process ===", "ip addr show veth0 || true"); err != nil {
			panic(err)
		}
		return
	case 2:
		// 2. 该进程执行 newNamespaceProccessFunc，binding 文件系统，并执行测试脚本
		if os.Args[1] == sub {
			if err := newNamespaceProccessFunc(); err != nil {
				panic(err)
			}
			return
		}
	}
	log.Fatalf("usage: %s [sub]", os.Args[0])
}
