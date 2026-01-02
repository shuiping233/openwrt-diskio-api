package model

// ProcfsPathsInterface 接口：所有可能访问的路径都走这里
type ProcfsPathsInterface interface {
	CpuTemp() string
	CpuUsage() string
	SystemUptime() string
	DefaultDns() string
	DefaultGateway() string
	StorageDeviceMounts() string
	StorageDeviceIo() string
	NetworkDeviceIo() string
	SystemMemoryInfo() string
	NetworkConnection() []string
	SystemVersion() string
	HardwareName() string
	SystemHostname() string
}

// ProcfsPaths 生产环境路径
type ProcfsPaths struct{}

func (p ProcfsPaths) CpuTemp() string             { return "/sys/class/thermal/thermal_zone0/temp" }
func (p ProcfsPaths) CpuUsage() string            { return "/proc/stat" }
func (p ProcfsPaths) SystemUptime() string        { return "/proc/uptime" }
func (p ProcfsPaths) DefaultDns() string          { return "/tmp/resolv.conf.ppp" }
func (p ProcfsPaths) DefaultGateway() string      { return "/proc/net/route" }
func (p ProcfsPaths) StorageDeviceMounts() string { return "/proc/mounts" }
func (p ProcfsPaths) StorageDeviceIo() string     { return "/proc/diskstats" }
func (p ProcfsPaths) NetworkDeviceIo() string     { return "/proc/net/dev" }
func (p ProcfsPaths) SystemMemoryInfo() string    { return "/proc/meminfo" }
func (p ProcfsPaths) NetworkConnection() []string {
	return []string{"/proc/net/nf_conntrack", "/proc/net/ip_conntrack"}
}
func (p ProcfsPaths) SystemVersion() string  { return "/proc/version" }
func (p ProcfsPaths) HardwareName() string   { return "/proc/device-tree/model" }
func (p ProcfsPaths) SystemHostname() string { return "/proc/sys/kernel/hostname" }
