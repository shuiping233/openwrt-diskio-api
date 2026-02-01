//go:build linux
// +build linux

package metric

import (
	"sync"
	"time"

	"openwrt-diskio-api/backend/model"
)

var connectionMetricPool = sync.Pool{
	New: func() interface{} {
		return &model.NetworkConnectionMetric{}
	},
}

type BackgroundService struct {
	MutexStatic             sync.RWMutex
	MutexDynamic            sync.RWMutex
	MutexNetwork            sync.RWMutex
	StaticMetric            *model.StaticMetric
	DynamicMetric           *model.DynamicMetric
	NetworkConnectionMetric *model.NetworkConnectionMetric
	Reader                  FsReaderInterface
	Runner                  CommandRunnerInterface
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

		b.MutexStatic.Lock()
		b.StaticMetric = &model.StaticMetric{
			Network: staticNetworkMetric,
			System:  staticSystemMetric,
		}
		b.MutexStatic.Unlock()

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

		b.MutexDynamic.Lock()
		b.DynamicMetric = &model.DynamicMetric{
			Cpu:     cpuMetric,
			Memory:  memoryMetric,
			Network: networkMetric,
			Storage: storageMetric,
			System:  systemMetric,
		}
		b.MutexDynamic.Unlock()

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

		networkConnectionMetric := connectionMetricPool.Get().(*model.NetworkConnectionMetric)
		networkConnectionMetric.Details = networkConnectionMetric.Details[:0]
		ReadConnectionMetric(b.Reader, networkConnectionMetric, privateCidr)

		b.MutexNetwork.Lock()
		b.NetworkConnectionMetric = networkConnectionMetric
		b.MutexNetwork.Unlock()

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}
