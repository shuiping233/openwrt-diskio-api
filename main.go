//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type diskStat struct {
	name      string
	readSect  uint64
	writeSect uint64
	timestamp time.Time
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		fmt.Println("Usage: diskio")
		fmt.Println("Print disk I/O (KB/s) every second.")
		return
	}

	prev := map[string]diskStat{}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		curr, err := readDiskStats()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read diskstats: %v\n", err)
			continue
		}

		var totReadKB, totWriteKB float64

		// 计算差值并打印
		for name, now := range curr {
			before, ok := prev[name]
			if !ok {
				continue // 第一次出现，没有历史
			}
			elapsed := now.timestamp.Sub(before.timestamp).Seconds()
			if elapsed <= 0 {
				continue
			}
			// 1 扇区 = 512 字节
			readKB := float64(now.readSect-before.readSect) * 512 / 1024 / elapsed
			writeKB := float64(now.writeSect-before.writeSect) * 512 / 1024 / elapsed
			fmt.Printf("%-10s  read=%7.1f KB/s  write=%7.1f KB/s\n",
				name, readKB, writeKB)

			totReadKB += readKB
			totWriteKB += writeKB
		}
		fmt.Printf("%-10s  read=%7.1f KB/s  write=%7.1f KB/s\n",
			"TOTAL", totReadKB, totWriteKB)
		prev = curr
	}
}

// readDiskStats 解析 /proc/diskstats，返回各设备最新统计
func readDiskStats() (map[string]diskStat, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	ts := time.Now()
	res := make(map[string]diskStat)
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		// 简单过滤：只保留整盘（无数字后缀）和 mmcblk0 这类常见名
		name := f[2]
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") {
			continue
		}
		readSect, _ := strconv.ParseUint(f[5], 10, 64)
		writeSect, _ := strconv.ParseUint(f[9], 10, 64)
		res[name] = diskStat{
			name:      name,
			readSect:  readSect,
			writeSect: writeSect,
			timestamp: ts,
		}
	}
	return res, nil
}
