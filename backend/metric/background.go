//go:build linux
// +build linux

package metric

import (
	"sync/atomic"
	"time"

	"openwrt-diskio-api/backend/model"
)

type BackgroundService struct {
	StaticMetric            atomic.Pointer[model.StaticMetric]
	DynamicMetric           atomic.Pointer[model.DynamicMetric]
	NetworkConnectionMetric atomic.Pointer[model.NetworkConnectionMetric]
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

		b.StaticMetric.Store(&model.StaticMetric{
			Network: staticNetworkMetric,
			System:  staticSystemMetric,
		})

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

		b.DynamicMetric.Store(&model.DynamicMetric{
			Cpu:     cpuMetric,
			Memory:  memoryMetric,
			Network: networkMetric,
			Storage: storageMetric,
			System:  systemMetric,
		})

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

		b.NetworkConnectionMetric.Store(networkConnectionMetric)

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}
