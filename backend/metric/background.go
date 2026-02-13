//go:build linux
// +build linux

package metric

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"openwrt-diskio-api/backend/model"
)

var semaphore = make(chan struct{}, 3)

type BackgroundService struct {
	Reader                                 FsReaderInterface
	Runner                                 CommandRunnerInterface
	jsonCache                              sync.Map
	UpdateStaticMetricInterval             uint
	UpdateDynamicMetricInterval            uint
	UpdateNetworkConnectionDetailsInterval uint
	updatingStatusMap                      sync.Map
	UpdateEventChan                        chan string
	wg                                     sync.WaitGroup
}

func (b *BackgroundService) SetUpdateStaticMetricInterval(interval uint) {
	b.UpdateStaticMetricInterval = interval
}
func (b *BackgroundService) SetUpdateDynamicMetricInterval(interval uint) {
	b.UpdateDynamicMetricInterval = interval
}
func (b *BackgroundService) SetUpdateNetworkConnectionDetailsInterval(interval uint) {
	b.UpdateNetworkConnectionDetailsInterval = interval
}

func (b *BackgroundService) UpdateStaticMetric() {
	updateInterval := b.UpdateStaticMetricInterval

	staticSystemMetric := ReadStaticSystemMetric(b.Reader, b.Runner)
	staticNetworkMetric := ReadStaticNetworkMetric(b.Reader, b.Runner)

	jsonBytes, err := json.Marshal(&model.StaticMetric{
		Network: staticNetworkMetric,
		System:  staticSystemMetric,
	})
	if err != nil {
		log.Fatalf("StaticMetric json marshal error : %s", err)
	}
	b.setJsonBytes(
		model.JsonCacheKeyStaticMetric,
		time.Duration(updateInterval)*time.Second,
		jsonBytes,
	)

}
func (b *BackgroundService) UpdateDynamicMetric() {
	updateInterval := b.UpdateDynamicMetricInterval
	diskSnap := model.DiskSnap{}
	cpuSnap := model.CpuSnap{}
	netSnap := model.NetSnap{
		Interfaces: map[string]model.NetSnapUnit{},
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
		log.Fatalf("DynamicMetric json marshal error : %s", err)
	}
	b.setJsonBytes(
		model.JsonCacheKeyDynamicMetric,
		time.Duration(updateInterval)*time.Second,
		jsonBytes,
	)
}

func (b *BackgroundService) UpdateNetworkConnectionDetails() {
	updateInterval := b.UpdateNetworkConnectionDetailsInterval

	privateCidr := ReadPrivateIpv4Addresses(b.Runner)

	networkConnectionMetric := &model.NetworkConnectionMetric{}
	ReadConnectionMetric(b.Reader, networkConnectionMetric, privateCidr)

	jsonBytes, err := json.Marshal(networkConnectionMetric)
	if err != nil {
		log.Fatalf("NetworkConnectionDetails json marshal error : %s", err)
	}
	b.setJsonBytes(
		model.JsonCacheKeyNetworkConnectionMetric,
		time.Duration(updateInterval)*time.Second,
		jsonBytes,
	)
}

func (b *BackgroundService) Worker(index int) {
	b.wg.Add(1)
	defer b.wg.Done()
	for key := range b.UpdateEventChan {
		switch key {
		case model.JsonCacheKeyDynamicMetric:
			b.UpdateDynamicMetric()
		case model.JsonCacheKeyStaticMetric:
			b.UpdateStaticMetric()
		case model.JsonCacheKeyNetworkConnectionMetric:
			b.UpdateNetworkConnectionDetails()
		}
		b.updatingStatusMap.Delete(key)
	}
	log.Printf("worker %d exit", index)
}

func (b *BackgroundService) setJsonBytes(key string, updateInterval time.Duration, value []byte) {
	now := time.Now().UTC()
	b.jsonCache.Store(key,
		model.CacheValue{
			UpdateAt: now,
			ExpireAt: now.Add(updateInterval),
			Data:     value,
		},
	)
}
func (b *BackgroundService) GetJsonBytes(key string) []byte {
	rawCache, ok := b.jsonCache.Load(key)
	if !ok {
		log.Printf("get json cache failed : %s not found", key)
		return []byte{}
	}
	cache, ok := rawCache.(model.CacheValue)
	if !ok {
		log.Fatalf("get json cache %q failed : not valid %T ", key, model.CacheValue{})
	}
	now := time.Now().UTC()
	if now.After(cache.ExpireAt) {
		if _, loading := b.updatingStatusMap.LoadOrStore(key, true); !loading {
			select {
			case b.UpdateEventChan <- key:
			default:
				b.updatingStatusMap.Delete(key)
			}
		}
	}
	return cache.Data
}

func (b *BackgroundService) Close() {
	if b.UpdateEventChan != nil {
		close(b.UpdateEventChan)
	}
	b.wg.Wait()
}
