package test

import "github.com/stretchr/testify/mock"

const (
	UnitTestPathPrefix = "/ut"
)

type Case struct {
	TestName    string
	Input       []string
	Expected    interface{}
	IsReturnErr bool
}
type Cases []Case

type ProcfsPaths struct {
	prefix string // 例如 "/fake"
}

func (p ProcfsPaths) CpuTemp() string {
	return UnitTestPathPrefix + "/sys/class/thermal/thermal_zone0/temp"
}
func (p ProcfsPaths) CpuUsage() string            { return UnitTestPathPrefix + "/proc/stat" }
func (p ProcfsPaths) SystemUptime() string        { return UnitTestPathPrefix + "/proc/uptime" }
func (p ProcfsPaths) DefaultDns() string          { return UnitTestPathPrefix + "/tmp/resolv.conf.ppp" }
func (p ProcfsPaths) DefaultGateway() string      { return UnitTestPathPrefix + "/proc/net/route" }
func (p ProcfsPaths) StorageDeviceMounts() string { return UnitTestPathPrefix + "/proc/mounts" }
func (p ProcfsPaths) StorageDeviceIo() string     { return UnitTestPathPrefix + "/proc/diskstats" }
func (p ProcfsPaths) NetworkDeviceIo() string     { return UnitTestPathPrefix + "/proc/net/dev" }
func (p ProcfsPaths) SystemMemoryInfo() string    { return UnitTestPathPrefix + "/proc/meminfo" }
func (p ProcfsPaths) NetworkConnection() []string {
	return []string{UnitTestPathPrefix + "/proc/net/nf_conntrack", UnitTestPathPrefix + "/proc/net/ip_conntrack"}
}

func (p ProcfsPaths) SystemVersion() string { return UnitTestPathPrefix + "/proc/version" }
func (p ProcfsPaths) HardwareName() string {
	return UnitTestPathPrefix + "/proc/device-tree/model"
}
func (p ProcfsPaths) SystemHostname() string {
	return UnitTestPathPrefix + "/proc/sys/kernel/hostname"
}

type CommandRunner struct {
	mock.Mock
}

func (c *CommandRunner) Run(name string, args ...string) (string, error) {
	result := c.Called(name, args)
	return result.String(0), result.Error(1)
}
