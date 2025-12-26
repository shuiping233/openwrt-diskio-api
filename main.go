//go:build linux
// +build linux

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type devIO struct {
	Read  string `json:"read"`
	Write string `json:"write"`
	Unit  string `json:"unit"`
	// 新增：磁盘占用信息
	TotalGB     string `json:"total_gb,omitempty"`     // 总容量 GB
	UsedGB      string `json:"used_gb,omitempty"`      // 已用 GB
	UsedPercent string `json:"used_percent,omitempty"` // 占用百分比
}

var (
	// 实时速率（每秒更新）
	ioMap  = map[string]devIO{}
	ioLock sync.RWMutex
)

//go:embed web
var webEmb embed.FS

func main() {
	var (
		host = flag.String("host", "127.0.0.1", "listen host")
		port = flag.Int("port", 8080, "listen port")
	)
	flag.Parse()

	go backgroundUpdate()

	webFS, _ := fs.Sub(webEmb, "web")
	http.Handle("/", http.FileServer(http.FS(webFS)))

	addr := *host + ":" + strconv.Itoa(*port)
	http.HandleFunc("/io", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "only GET", http.StatusMethodNotAllowed)
			return
		}
		ioLock.RLock()
		out := make(map[string]devIO, len(ioMap))
		for k, v := range ioMap {
			out[k] = v
		}
		ioLock.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	log.Printf("listen http://%s/io", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// 后台每秒算一次差值，更新 ioMap
func backgroundUpdate() {
	prevSnap := map[string]diskSnap{}
	prevTime := time.Now()

	for {
		time.Sleep(1 * time.Second)
		currTime := time.Now()
		currSnap, err := readDiskStats()
		if err != nil {
			continue
		}
		elapsed := currTime.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			continue
		}

		// 1. 计算 I/O 速率
		tmp := make(map[string]devIO)
		var totReadKB, totWriteKB float64

		for name, snap := range currSnap {
			old, ok := prevSnap[name]
			if !ok {
				continue
			}
			readKB := float64(snap.readSect-old.readSect) * 512 / 1024 / elapsed
			writeKB := float64(snap.writeSect-old.writeSect) * 512 / 1024 / elapsed
			tmp[name] = devIO{
				Read:  strconv.FormatFloat(readKB, 'f', 1, 64),
				Write: strconv.FormatFloat(writeKB, 'f', 1, 64),
				Unit:  "KB/s",
			}
			totReadKB += readKB
			totWriteKB += writeKB
		}
		tmp["total"] = devIO{
			Read:  strconv.FormatFloat(totReadKB, 'f', 1, 64),
			Write: strconv.FormatFloat(totWriteKB, 'f', 1, 64),
			Unit:  "KB/s",
		}

		// 2. 获取并合并磁盘占用数据
		// 获取 map[设备名]占用信息
		usageMap := getDiskUsage()

		// 遍历 usageMap，如果设备名在 tmp 中存在，则补全占用信息
		// 注意：OpenWrt 下有些设备名可能带 /dev/ 前缀，需要处理一下
		for devName, usage := range usageMap {
			// 尝试直接匹配
			if ioData, exists := tmp[devName]; exists {
				ioData.TotalGB = usage.TotalGB
				ioData.UsedGB = usage.UsedGB
				ioData.UsedPercent = usage.UsedPercent
				tmp[devName] = ioData
			}
			// 尝试匹配 /dev/ 前缀的情况 (有些挂载信息记录的是完整路径)
			cleanName := strings.TrimPrefix(devName, "/dev/")
			if ioData, exists := tmp[cleanName]; exists {
				ioData.TotalGB = usage.TotalGB
				ioData.UsedGB = usage.UsedGB
				ioData.UsedPercent = usage.UsedPercent
				tmp[cleanName] = ioData
			}
		}

		ioLock.Lock()
		ioMap = tmp
		ioLock.Unlock()

		prevSnap, prevTime = currSnap, currTime
	}
}

// 获取磁盘占用情况
func getDiskUsage() map[string]devIO {
	usageMap := make(map[string]devIO)

	// 读取 /proc/mounts 获取挂载信息
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return usageMap
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		devSource := fields[0] // 设备名，如 /dev/mmcblk2p2 或 overlay
		mountPoint := fields[1]
		fsType := fields[2]

		// 过滤掉虚拟文件系统和不需要的
		if strings.HasPrefix(devSource, "none") ||
			fsType == "proc" ||
			fsType == "sysfs" ||
			fsType == "devtmpfs" ||
			fsType == "tmpfs" ||
			fsType == "cgroup" ||
			fsType == "debugfs" {
			continue
		}

		// OpenWrt 特例：overlayfs 通常基于 /dev/...，但在 /proc/mounts 里可能是 "overlay"
		// 我们可以暂时忽略 overlay 的统计，或者通过 cat /proc/mounts 找到其下层的设备
		// 这里为了通用性，如果 devSource 看起来是个设备路径（以 /dev 开头），我们就统计它
		if !strings.HasPrefix(devSource, "/dev/") {
			// 如果是 /dev/root 这种软链接情况，也可以处理，这里简化为只处理绝对路径
			// 实际上 OpenWrt 的 rootfs_data 通常在 /dev/mmcblk2p2 之类
			continue
		}

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free

		if total == 0 {
			continue
		}

		totalGB := float64(total) / 1024 / 1024 / 1024
		usedGB := float64(used) / 1024 / 1024 / 1024
		percent := float64(used) / float64(total) * 100

		usageMap[devSource] = devIO{
			TotalGB:     strconv.FormatFloat(totalGB, 'f', 2, 64),
			UsedGB:      strconv.FormatFloat(usedGB, 'f', 2, 64),
			UsedPercent: strconv.FormatFloat(percent, 'f', 1, 64),
		}
	}

	return usageMap
}

// 以下结构与之前相同
type diskSnap struct{ readSect, writeSect uint64 }

func readDiskStats() (map[string]diskSnap, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	res := map[string]diskSnap{}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		name := f[2]
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		readSect, _ := strconv.ParseUint(f[5], 10, 64)
		writeSect, _ := strconv.ParseUint(f[9], 10, 64)
		res[name] = diskSnap{readSect: readSect, writeSect: writeSect}
	}
	return res, nil
}
