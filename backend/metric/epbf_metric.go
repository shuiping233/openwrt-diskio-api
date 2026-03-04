//go:build linux

package metric

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"openwrt-diskio-api/backend/model"
	bpf "openwrt-diskio-api/backend/pkg/ebpf"
	"openwrt-diskio-api/backend/utils"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type IPMetrics struct {
	IP            string
	UploadRate    float64
	DownloadRate  float64
	TotalUpload   uint64
	TotalDownload uint64
}

type EbpfNetTrafficService struct {
	captureInterface string
	keyExpiredTime   time.Duration
	activeChan       chan struct{}
	objs             *bpf.BpfObjects
	link             netlink.Link
	metricsMap       map[uint32]*IPMetrics
}

func NewEbpfNetTrafficService(keyExpiredTime time.Duration) *EbpfNetTrafficService {
	return &EbpfNetTrafficService{
		keyExpiredTime: keyExpiredTime,
		activeChan:     make(chan struct{}, 1),
		metricsMap:     make(map[uint32]*IPMetrics),
	}
}

func (svc *EbpfNetTrafficService) InitEbpfInterfaceDevice(targetInterface string) error {
	svc.captureInterface = targetInterface
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("无法解除内存限制: %w", err)
	}

	objs := bpf.BpfObjects{}
	if err := bpf.LoadBpfObjects(&objs, nil); err != nil {
		return fmt.Errorf("加载 BPF 对象失败: %w", err)
	}

	link, err := netlink.LinkByName(targetInterface)
	if err != nil {
		return fmt.Errorf("🔴 错误: 未找到网卡 %s: %w", targetInterface, err)
	}

	if err := attachTCObjects(link, objs.CountFlow.FD()); err != nil {
		log.Fatalf("❌ 挂载网卡 %s 失败: %s", targetInterface, err)
	}
	fmt.Printf("✅ 正在监控局域网流量: %s\n", targetInterface)
	startCapture(&objs)
	svc.link = link
	svc.objs = &objs

	return nil
}

func (svc *EbpfNetTrafficService) Run(ctx context.Context) {

	objs := svc.objs

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	keyExpiredTime := svc.keyExpiredTime
	lastSnapshots := make(map[bpf.BpfFlowKey]uint64)
	metricsMap := svc.metricsMap
	lastRequestTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			svc.Close()
			return

		case <-svc.activeChan:
			// 1. 收到接口请求信号，刷新最后活跃时间
			lastRequestTime = time.Now()
			if !isCapturing(objs) {
				startCapture(objs)
			}

		case <-ticker.C:
			if isCapturing(objs) {
				// 如果超过 n 秒没收到新请求，则关停以节省资源
				if time.Since(lastRequestTime) > keyExpiredTime {
					stopCapture(objs)
					clearFlowMap(objs.FlowMap)
					clear(metricsMap)
					clear(lastSnapshots)
					continue
				}
			}

			nowKtime := getKtimeNS()
			timeout := uint64(keyExpiredTime) // n秒无流量老化

			// 重置每秒速率
			for _, m := range metricsMap {
				m.UploadRate = 0
				m.DownloadRate = 0
			}

			var key bpf.BpfFlowKey
			var val bpf.BpfFlowStats
			iter := objs.FlowMap.Iterate()

			for iter.Next(&key, &val) {
				// 老化处理
				if nowKtime-val.LastSeen > timeout {
					objs.FlowMap.Delete(key)
					delete(lastSnapshots, key)
					continue
				}

				currentKey := key
				currentBytes := val.Bytes
				delta := uint64(0)
				if lastBytes, ok := lastSnapshots[currentKey]; ok {
					if currentBytes >= lastBytes {
						delta = currentBytes - lastBytes
					}
				} else {
					delta = currentBytes
				}
				lastSnapshots[currentKey] = currentBytes

				trafficAggregate(key, delta, metricsMap)
			}
		}
	}
}
func (svc *EbpfNetTrafficService) ActiveSignal() {
	svc.activeChan <- struct{}{}
}

func (svc *EbpfNetTrafficService) Close() {
	fmt.Println("\n正在清理并退出...")
	stopCapture(svc.objs)
	cleanUpTC(svc.link)
	svc.objs.Close()
}
func (svc *EbpfNetTrafficService) GetAggregationTrafficMetric() model.AggregationTrafficMetric {
	metricsMap := svc.metricsMap
	result := model.AggregationTrafficMetric{
		CaptureInterface: svc.captureInterface,
		Details:          make([]model.AggregationTrafficDetails, len(metricsMap)),
	}

	for ip, value := range metricsMap {
		ipStr := formatIP(ip)
		rate, unit := utils.ConvertBytes(value.DownloadRate, model.BSecond)
		incoming := model.MetricUnit{
			Value: rate,
			Unit:  unit,
		}
		rate, unit = utils.ConvertBytes(value.UploadRate, model.BSecond)
		outgoing := model.MetricUnit{
			Value: rate,
			Unit:  unit,
		}
		rate, unit = utils.ConvertBytes(value.DownloadRate+value.UploadRate, model.BSecond)
		totalThroughput := model.MetricUnit{
			Value: rate,
			Unit:  unit,
		}

		total, unit := utils.ConvertBytes(float64(value.TotalDownload), model.Byte)
		totalIncoming := model.MetricUnit{
			Value: total,
			Unit:  unit,
		}
		total, unit = utils.ConvertBytes(float64(value.TotalUpload), model.Byte)
		totalOutgoing := model.MetricUnit{
			Value: total,
			Unit:  unit,
		}
		total, unit = utils.ConvertBytes(float64(value.TotalDownload+value.TotalUpload), model.Byte)
		totalTraffic := model.MetricUnit{
			Value: total,
			Unit:  unit,
		}

		result.Details = append(result.Details, model.AggregationTrafficDetails{
			Ip:              ipStr,
			IpType:          model.IpAddressTypeLan, // TODO 先写死,因为其他类型的ip流量抓取还没做
			Incoming:        incoming,
			Outgoing:        outgoing,
			TotalThroughput: totalThroughput,
			TotalIncoming:   totalIncoming,
			TotalOutgoing:   totalOutgoing,
			TotalTraffic:    totalTraffic,
			Tcp:             -1, // TODO 先这样,后面会找NetworkConnection里的值统计好之后再填进去
			Udp:             -1, // TODO 先这样,后面会找NetworkConnection里的值统计好之后再填进去
			Other:           -1, // TODO 先这样,后面会找NetworkConnection里的值统计好之后再填进去
		})
	}
	return result
}

func drawUI(metrics map[uint32]*IPMetrics) {
	var keys []uint32
	for k := range metrics {
		keys = append(keys, k)
	}

	// 排序：按累计总流量降序
	sort.Slice(keys, func(i, j int) bool {
		mi, mj := metrics[keys[i]], metrics[keys[j]]
		return (mi.TotalUpload + mi.TotalDownload) > (mj.TotalUpload + mj.TotalDownload)
	})

	fmt.Printf("\033[H\033[2J") // 清屏
	fmt.Printf("【 局域网流量统计 (eBPF) 】- %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("%-18s | %-12s | %-12s | %-12s\n", "内网 IP 地址", "上传(KB/s)", "下载(KB/s)", "累计总流量")
	fmt.Println(strings.Repeat("-", 65))

	if len(keys) == 0 {
		fmt.Println("  等待流量...")
		return
	}

	for _, key := range keys {
		m := metrics[key]
		totalMB := float64(m.TotalUpload+m.TotalDownload) / 1024 / 1024

		// 过滤掉没跑过流量且没累计数据的 IP (保持界面干净)
		if m.UploadRate == 0 && m.DownloadRate == 0 && totalMB < 0.01 {
			continue
		}

		fmt.Printf("%-18s | %10.2f | %10.2f | %10.2f MB\n",
			m.IP, m.UploadRate, m.DownloadRate, totalMB)
	}
}

func trafficAggregate(key bpf.BpfFlowKey, delta uint64, res map[uint32]*IPMetrics) {
	rateKB := float64(delta) / 1024.0

	// 只统计局域网段 IP 的流量
	if isPrivateIP(key.SrcIp) {
		m := getOrCreateMetrics(key.SrcIp, res)
		m.UploadRate += rateKB
		m.TotalUpload += delta
	}

	if isPrivateIP(key.DstIp) {
		m := getOrCreateMetrics(key.DstIp, res)
		m.DownloadRate += rateKB
		m.TotalDownload += delta
	}
}

func getOrCreateMetrics(ip uint32, res map[uint32]*IPMetrics) *IPMetrics {
	if m, ok := res[ip]; ok {
		return m
	}
	m := &IPMetrics{IP: formatIP(ip)}
	res[ip] = m
	return m
}

func getKtimeNS() uint64 {
	var ts unix.Timespec
	unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts)
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec)
}

func isPrivateIP(ip uint32) bool {
	b1 := byte(ip & 0xFF)
	b2 := byte((ip >> 8) & 0xFF)
	return b1 == 10 || (b1 == 172 && b2 >= 16 && b2 <= 31) || (b1 == 192 && b2 == 168)
}

func formatIP(n uint32) string {
	return net.IPv4(byte(n), byte(n>>8), byte(n>>16), byte(n>>24)).String()
}

// --- TC 控制 ---

func attachTCObjects(link netlink.Link, fd int) error {
	cleanUpTC(link)
	qdisc := &netlink.GenericQdisc{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Handle:    netlink.MakeHandle(0xffff, 0),
			Parent:    netlink.HANDLE_CLSACT,
		},
		QdiscType: "clsact",
	}
	if err := netlink.QdiscAdd(qdisc); err != nil {
		return err
	}

	parents := []uint32{netlink.HANDLE_MIN_INGRESS, netlink.HANDLE_MIN_EGRESS}
	for _, parent := range parents {
		filter := &netlink.BpfFilter{
			FilterAttrs: netlink.FilterAttrs{
				LinkIndex: link.Attrs().Index,
				Parent:    parent,
				Priority:  1,
				Protocol:  unix.ETH_P_ALL,
			},
			Fd:           fd,
			DirectAction: true,
		}
		if err := netlink.FilterAdd(filter); err != nil {
			return err
		}
	}
	return nil
}

func cleanUpTC(link netlink.Link) {
	qdiscs, _ := netlink.QdiscList(link)
	for _, q := range qdiscs {
		if q.Attrs().Parent == netlink.HANDLE_CLSACT {
			netlink.QdiscDel(q)
		}
	}
}

func startCapture(objs *bpf.BpfObjects) {
	log.Println("Enable ebpf network traffic capture")
	key := uint32(0)
	val := uint32(1)
	objs.ConfigMap.Update(&key, &val, ebpf.UpdateAny)
}

func stopCapture(objs *bpf.BpfObjects) {
	log.Println("Disable ebpf network traffic capture")
	key := uint32(0)
	val := uint32(0)
	objs.ConfigMap.Update(&key, &val, ebpf.UpdateAny)
}

func isCapturing(objs *bpf.BpfObjects) bool {
	key := uint32(0)
	val := uint32(0)

	err := objs.ConfigMap.Lookup(&key, &val)
	if err != nil {
		return false
	}
	return val == 1
}

func clearFlowMap(m *ebpf.Map) {
	var key bpf.BpfFlowKey
	iter := m.Iterate()
	// 边读边删，这是 eBPF Map 推荐的清空方式
	for iter.Next(&key, nil) {
		if err := m.Delete(key); err != nil {
			// 忽略已经不存在的 key（并发可能导致这个）
			continue
		}
	}
}
