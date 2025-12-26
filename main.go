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
	"time"
)

type devIO struct {
	Read  string `json:"read"`
	Write string `json:"write"`
	Unit  string `json:"unit"`
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

		ioLock.Lock()
		ioMap = tmp
		ioLock.Unlock()

		prevSnap, prevTime = currSnap, currTime
	}
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
