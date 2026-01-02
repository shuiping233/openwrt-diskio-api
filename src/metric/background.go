//go:build linux
// +build linux

package metric

import (
	"sync"
	"time"

	"openwrt-diskio-api/src/model"
)

type BackgroundService struct {
	StaticMetric  *model.StaticMetric
	DynamicMetric *model.DynamicMetric
	Reader        FsReader
	Runner        CommandRunnerInterface
}

func (b *BackgroundService) UpdateStaticMetric(
	updateInterval uint,
	lock *sync.RWMutex,
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

		b.StaticMetric = &model.StaticMetric{
			Network: staticNetworkMetric,
			System:  staticSystemMetric,
		}

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}

func (b *BackgroundService) UpdateDynamicMetric(
	updateInterval uint,
	lock *sync.RWMutex,
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
		networkConnectionMetric := model.NetworkConnectionMetric{}
		ReadConnectionMetric(b.Reader, &networkConnectionMetric)

		b.DynamicMetric = &model.DynamicMetric{
			Cpu:               cpuMetric,
			Memory:            memoryMetric,
			Network:           networkMetric,
			Storage:           storageMetric,
			System:            systemMetric,
			NetworkConnection: networkConnectionMetric,
		}
		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}
