//go:build linux
// +build linux

package metric

import (
	"sync"
	"time"

	"openwrt-diskio-api/src/model"
)

func UpdateStaticMetric(
	updateInterval uint,
	lock *sync.RWMutex,
	staticMetric *model.StaticMetric,
) {

	prevTime := time.Now()
	for {
		currTime := time.Now()
		elapsed := currTime.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			continue
		}

		staticSystemMetric := ReadStaticSystemMetric()
		staticNetworkMetric := ReadStaticNetworkMetric()

		staticMetric = &model.StaticMetric{
			Network: staticNetworkMetric,
			System:  staticSystemMetric,
		}

		prevTime = currTime

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}

func UpdateDynamicMetric(
	updateInterval uint,
	lock *sync.RWMutex,
	dynamicMetric *model.DynamicMetric,
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

		// TODO 需要传入 elapsed 算 rate
		networkMetric := ReadNetworkMetric(&netSnap, updateInterval)
		cpuMetric := ReadCpuMetric(&cpuSnap)
		storageMetric := ReadStorageMetric(diskSnap, updateInterval)
		memoryMetric := ReadMemoryMetric()
		systemMetric := ReadSystemMetric()
		networkConnectionMetric := model.NetworkConnectionMetric{}
		ReadConnectionMetric(&networkConnectionMetric)

		dynamicMetric = &model.DynamicMetric{
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
