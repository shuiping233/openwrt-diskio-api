package model

const (
	StringDefault              = "unknown"
	NetConnectionIndexIpFamily = 0 // ipv4/ipv6
	NetConnectionIndexProto    = 2 // tcp/udp/icmp
	NetConnectionIndexState    = 5 // 只有 TCP 有
)

const (
	BSecond  = "B/S"
	KbSecond = "KB/S"
	MbSecond = "MB/S"
	GbSecond = "GB/S"
	TbSecond = "TB/S"
	PbSecond = "PB/S"
	Byte     = "B"
	KiloByte = "KB"
	MegaByte = "MB"
	GigaByte = "GB"
	TeraByte = "TB"
	PetaByte = "PB"
	Percent  = "%"
	Celsius  = "°C"
)

var (
	RateUnitList                        = []string{BSecond, KbSecond, MbSecond, GbSecond, TbSecond, PbSecond}
	DataUnitList                        = []string{Byte, KiloByte, MegaByte, GigaByte, TeraByte, PetaByte}
	InternalNetworkDeviceNamePrefixList = []string{"br-lan", "docker", "tun"}
)

type NetSnapUnit struct {
	RxBytes float64
	TxBytes float64
}
type NetSnap struct {
	Interfaces map[string]NetSnapUnit
}

type CpuSnap struct {
	AllCycles   uint64        // 所有核心时间片总和("cpu"一行)
	AllCoreIdle uint64        // 所有核心idle总和("cpu"一行)
	Cores       []CpuSnapUnit // 各核心时间片总和和idle("cpu0","cpu1"等行)
}
type CpuSnapUnit struct {
	Cycles uint64
	Idle   uint64
}

type DiskSnap map[string]DiskSnapUnit
type DiskSnapUnit struct{ ReadBytes, WriteBytes float64 }

type DynamicMetric struct {
	Storage StorageMetric `json:"storage"`
	Cpu     CpuMetric     `json:"cpu"`
	Network NetworkMetric `json:"network"`
	Memory  MemoryMetric  `json:"memory"`
	System  SystemMetric  `json:"system"`
}

type NetworkConnectionMetric struct {
	Counts  NetworkConnectionCounts `json:"counts"`
	Details []NetworkConnection     `json:"connections"`
}

type NetworkConnectionCounts struct {
	Tcp   uint `json:"tcp"`
	Udp   uint `json:"udp"`
	Other uint `json:"other"`
}

func (n *NetworkConnectionCounts) AddCountTcp() {
	n.Tcp += 1
}

func (n *NetworkConnectionCounts) AddCountUdp() {
	n.Udp += 1
}

func (n *NetworkConnectionCounts) AddCountOther() {
	n.Other += 1
}

// because tcp/udp/icmp counts are counted income and outcome connection ,
// divide them by 2 is true connection counts
func (n *NetworkConnectionCounts) DivideAllCounts() {
	if n.Tcp > 0 {
		n.Tcp /= 2
	}
	if n.Udp > 0 {
		n.Udp /= 2
	}
	if n.Other > 0 {
		n.Other /= 2
	}
}

type NetworkConnection struct {
	IpFamily        string     `json:"ip_family"`
	SourceIp        string     `json:"source_ip"`
	SourcePort      int        `json:"source_port"`
	DestinationIp   string     `json:"destination_ip"`
	DestinationPort int        `json:"destination_port"`
	Protocol        string     `json:"protocol"`
	State           string     `json:"state"`
	Traffic         MetricUnit `json:"traffic"`
	Packets         int64      `json:"packets"`
}

type StorageMetric map[string]StorageIoMetric

type StorageIoMetric struct {
	Read  MetricUnit `json:"read"`
	Write MetricUnit `json:"write"`
	// if not read storage device usage , fill -1
	Total MetricUnit `json:"total,omitempty"`
	// if not read storage device usage , fill -1
	Used MetricUnit `json:"used,omitempty"`
	// if not read storage device usage , fill -1
	UsedPercent MetricUnit `json:"used_percent,omitempty"`
}

func (s StorageMetric) SetTotal(read float64, readUnit string, write float64, writeUnit string) {
	s["total"] = StorageIoMetric{
		Read:  MetricUnit{read, readUnit},
		Write: MetricUnit{write, writeUnit},
	}
}

type CpuMetric map[string]CpuUsageMetric

type CpuUsageMetric struct {
	Usage       MetricUnit `json:"usage"`
	Temperature MetricUnit `json:"temperature"`
}

func (c CpuMetric) SetTotal(usage float64, usageUnit string, temperature float64, temperatureUnit string) {
	c["total"] = CpuUsageMetric{
		Usage:       MetricUnit{usage, usageUnit},
		Temperature: MetricUnit{temperature, temperatureUnit},
	}
}

type NetworkMetric map[string]NetworkIoMetric

type NetworkIoMetric struct {
	Incoming MetricUnit `json:"incoming"`
	Outgoing MetricUnit `json:"outgoing"`
}

func (c NetworkMetric) SetTotal(incoming float64, incomingUnit string, outgoing float64, outgoingUnit string) {
	c["total"] = NetworkIoMetric{
		Incoming: MetricUnit{incoming, incomingUnit},
		Outgoing: MetricUnit{outgoing, outgoingUnit},
	}
}

type MemoryMetric struct {
	Total       MetricUnit `json:"total"`
	Used        MetricUnit `json:"used"`
	UsedPercent MetricUnit `json:"used_percent"`
}

type SystemMetric struct {
	Uptime string `json:"uptime"`
}

type StaticMetric struct {
	Network StaticNetworkMetric `json:"network"`
	System  StaticSystemMetric  `json:"system"`
}

type StaticNetworkMetric map[string]StaticNetworkInterfaceMetric

type StaticNetworkInterfaceMetric struct {
	Ipv4    []string `json:"ipv4"`
	Ipv6    []string `json:"ipv6"`
	Dns     []string `json:"dns,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
}

func (s StaticNetworkMetric) SetGlobal(Ipv4 []string, Ipv6 []string, dns []string, gateway string) {
	s["global"] = StaticNetworkInterfaceMetric{
		Ipv4:    Ipv4,
		Ipv6:    Ipv6,
		Dns:     dns,
		Gateway: gateway,
	}
}

type StaticSystemMetric struct {
	Hostname   string `json:"hostname"`
	Kernel     string `json:"kernel"`
	Os         string `json:"os"`
	DeviceName string `json:"device_name"`
	Arch       string `json:"arch"`
	Timezone   string `json:"timezone"`
}

type MetricUnit struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}
