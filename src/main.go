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
	"strconv"
	"sync"

	"openwrt-diskio-api/src/metric"
	"openwrt-diskio-api/src/model"

	"github.com/spf13/afero"
)

var (
	dynamicMetric     = &model.DynamicMetric{}
	staticMetric      = &model.StaticMetric{}
	dynamicMetricLock = &sync.RWMutex{}
	staticMetricLock  = &sync.RWMutex{}
	reader            = metric.FsReader{Fs: afero.NewOsFs()}
	runner            = metric.CommandRunner{}
	background        = metric.BackgroundService{
		StaticMetric:  staticMetric,
		DynamicMetric: dynamicMetric,
		Reader:        reader,
		Runner:        runner,
	}
)

//go:embed web
var webEmb embed.FS

func DynamicMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET", http.StatusMethodNotAllowed)
		return
	}

	// dynamicMetricLock .RLock()
	// out := dynamicMetric
	// dynamicMetricLock .RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(background.DynamicMetric)
}
func StaticMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET", http.StatusMethodNotAllowed)
		return
	}

	// dynamicMetricLock .RLock()
	// out := dynamicMetric
	// dynamicMetricLock .RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(background.StaticMetric)
}

func main() {
	var (
		host                  = flag.String("host", "127.0.0.1", "listen host")
		port                  = flag.Int("port", 8080, "listen port")
		dynamicMetricInterval = *flag.Uint("dynamic-metric-interval", 1, " metric update interval")
		staticMetricInterval  = *flag.Uint("static-metric-interval", 60, " metric update interval")
	)
	flag.Parse()

	go background.UpdateDynamicMetric(dynamicMetricInterval, dynamicMetricLock)
	go background.UpdateStaticMetric(staticMetricInterval, staticMetricLock)

	webFS, _ := fs.Sub(webEmb, "web")
	http.Handle("/", http.FileServer(http.FS(webFS)))

	addr := *host + ":" + strconv.Itoa(*port)

	http.HandleFunc("/metric/dynamic", DynamicMetricHandler)
	http.HandleFunc("/metric/static", StaticMetricHandler)

	log.Printf("listen http://%s/", addr)
	log.Printf("Interface url : http://%s/metric/dynamic", addr)
	log.Printf("Interface url : http://%s/metric/static", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
