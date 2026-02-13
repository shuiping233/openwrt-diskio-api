//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	frontend "openwrt-diskio-api"
	"openwrt-diskio-api/backend/metric"
	"openwrt-diskio-api/backend/model"

	"github.com/spf13/afero"
)

var (
	reader     = metric.FsReader{Fs: afero.NewOsFs()}
	runner     = metric.CommandRunner{}
	background = metric.BackgroundService{
		Reader: reader,
		Runner: runner,
	}
)

func DynamicMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonBytes := background.GetJsonBytes(model.JsonCacheKeyDynamicMetric)
	if len(jsonBytes) == 0 {
		var err error
		jsonBytes, err = json.Marshal(&model.DynamicMetric{})
		if err != nil {
			errMsg := fmt.Sprintf("json marshal error : %s", err.Error())
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
	}
	w.Write(jsonBytes)
}
func NetworkConnectionMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonBytes := background.GetJsonBytes(model.JsonCacheKeyNetworkConnectionMetric)
	if len(jsonBytes) == 0 {
		var err error
		jsonBytes, err = json.Marshal(&model.DynamicMetric{})
		if err != nil {
			errMsg := fmt.Sprintf("json marshal error : %s", err.Error())
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
	}
	w.Write(jsonBytes)
}
func StaticMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	jsonBytes := background.GetJsonBytes(model.JsonCacheKeyStaticMetric)
	if len(jsonBytes) == 0 {
		var err error
		jsonBytes, err = json.Marshal(&model.DynamicMetric{})
		if err != nil {
			errMsg := fmt.Sprintf("json marshal error : %s", err.Error())
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
	}
	w.Write(jsonBytes)
}

func main() {
	var (
		host                      = flag.String("host", "127.0.0.1", "listen host")
		port                      = flag.Int("port", 8080, "listen port")
		dynamicMetricInterval     = flag.Uint("dynamic-metric-interval", 1, " metric update interval")
		networkConnectionInterval = flag.Uint("network-connection-interval", 10, " network connection details update interval")
		staticMetricInterval      = flag.Uint("static-metric-interval", 60, " metric update interval")
	)
	flag.Parse()

	log.Println("print input config : ")
	log.Printf("host : %s", *host)
	log.Printf("port : %d", *port)
	log.Printf("dynamicMetricInterval : %v", *dynamicMetricInterval)
	log.Printf("networkConnectionInterval : %v", *networkConnectionInterval)
	log.Printf("staticMetricInterval : %v", *staticMetricInterval)

	go background.UpdateDynamicMetric(*dynamicMetricInterval)
	go background.UpdateNetworkConnectionDetails(*networkConnectionInterval)
	go background.UpdateStaticMetric(*staticMetricInterval)

	webFS, _ := fs.Sub(frontend.WebEmb, "dist/frontend")
	http.Handle("/", http.FileServer(http.FS(webFS)))

	addr := *host + ":" + strconv.Itoa(*port)

	http.HandleFunc("/metric/dynamic", DynamicMetricHandler)
	http.HandleFunc("/metric/network_connection", NetworkConnectionMetricHandler)
	http.HandleFunc("/metric/static", StaticMetricHandler)

	log.Printf("listen http://%s/", addr)
	log.Printf("Interface url : http://%s/metric/dynamic", addr)
	log.Printf("Interface url : http://%s/metric/network_connection", addr)
	log.Printf("Interface url : http://%s/metric/static", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
