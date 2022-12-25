package main

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

const MB = 1024 * 1024
const (
	SubCommandMonitor = "monitor"
	SubCommandAlloc   = "alloc"
)

func main() {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		panic(err)
	}
	printRSSMemory(p)
	allocMemory(10 * MB)
	printRSSMemory(p)
}

var PinnedAllocMemory [][]byte

func allocMemory(size int64) {
	b := make([]byte, size)
	for i := int64(0); i < size; i++ {
		b[i] = 1
	}
	PinnedAllocMemory = append(PinnedAllocMemory, b)
}

func printRSSMemory(p *process.Process) {
	m, err := p.MemoryInfo()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%f MB\n", float64(m.RSS)/MB)
}
