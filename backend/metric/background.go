//go:build linux
// +build linux

package metric

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"openwrt-diskio-api/backend/model"
)

type BackgroundService struct {
	Reader    FsReaderInterface
	Runner    CommandRunnerInterface
	jsonCache sync.Map
}

func (b *BackgroundService) UpdateStaticMetric(
	updateInterval uint,
) {
	prevTime := time.Now()
	for {
		currTime := time.Now()
		elapsed := currTime.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			continue
		}

		staticSystemMetric := ReadStaticSystemMetric(b.Reader, b.Runner)
		staticNetworkMetric := ReadStaticNetworkMetric(b.Reader, b.Runner)

		jsonBytes, err := json.Marshal(&model.StaticMetric{
			Network: staticNetworkMetric,
			System:  staticSystemMetric,
		})
		if err != nil {
			panic(fmt.Errorf("StaticMetric json marshal error : %s", err))
		}
		b.jsonCache.Store(model.JsonCacheKeyStaticMetric, jsonBytes)

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}
func (b *BackgroundService) UpdateDynamicMetric(
	updateInterval uint,
) {
	diskSnap := model.DiskSnap{}
	cpuSnap := model.CpuSnap{}
	netSnap := model.NetSnap{
		Interfaces: map[string]model.NetSnapUnit{},
	}

	prevTime := time.Now()

	for {
		currTime := time.Now()
		elapsed := currTime.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			continue
		}
		networkMetric := ReadNetworkMetric(b.Reader, &netSnap, updateInterval)
		cpuMetric := ReadCpuMetric(b.Reader, &cpuSnap)
		storageMetric := ReadStorageMetric(b.Reader, diskSnap, updateInterval)
		memoryMetric := ReadMemoryMetric(b.Reader)
		systemMetric := ReadSystemMetric(b.Reader)

		jsonBytes, err := json.Marshal(&model.DynamicMetric{
			Cpu:     cpuMetric,
			Memory:  memoryMetric,
			Network: networkMetric,
			Storage: storageMetric,
			System:  systemMetric,
		})
		if err != nil {
			panic(fmt.Errorf("DynamicMetric json marshal error : %s", err))
		}
		b.jsonCache.Store(model.JsonCacheKeyDynamicMetric, jsonBytes)

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}

func (b *BackgroundService) UpdateNetworkConnectionDetails(
	updateInterval uint,
) {
	prevTime := time.Now()
	for {
		currTime := time.Now()
		elapsed := currTime.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			continue
		}

		privateCidr := ReadPrivateIpv4Addresses(b.Runner)

		networkConnectionMetric := &model.NetworkConnectionMetric{}
		ReadConnectionMetric(b.Reader, networkConnectionMetric, privateCidr)

		jsonBytes, err := json.Marshal(networkConnectionMetric)
		if err != nil {
			panic(fmt.Errorf("NetworkConnectionDetails json marshal error : %s", err))
		}
		b.jsonCache.Store(model.JsonCacheKeyNetworkConnectionMetric, jsonBytes)

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}

func (b *BackgroundService) GetJsonBytes(key string) []byte {
	cache, ok := b.jsonCache.Load(key)
	if !ok {
		return []byte{}
	}
	return cache.([]byte)
}
