#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

/* * 流量 Key 结构体：显式对齐以防止 Hash 计算不一致
 */
struct flow_key {
    __u32 src_addr[4];
    __u32 dst_addr[4];
    __u16 src_port;
    __u16 dst_port;
    __u8  family; // AF_INET (2) 或 AF_INET6 (10)
    __u8  proto;  
    __u8  _pad[2]; // 填充至 4 字节对齐
};

struct flow_stats {
    __u64 packets;
    __u64 bytes;
    __u64 last_seen;
};

// 配置 Map：控制开关
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u32);
} config_map SEC(".maps");

/* * 核心改进：使用 PERCPU_HASH 
 * 1. 消除多核争用造成的 CPU 抖动。
 * 2. 提高在大流量压测下的数据稳定性。
 */
struct {
    __uint(type, BPF_MAP_TYPE_LRU_PERCPU_HASH); // 改为 LRU 类型
    __uint(max_entries, 32768); // 减小一点容量以换取更高的缓存命中率
    __type(key, struct flow_key);
    __type(value, struct flow_stats);
} flow_map SEC(".maps");

SEC("classifier")
int count_flow(struct __sk_buff *skb) {
    // 1. 检查采集开关
    __u32 config_key = 0;
    __u32 *enabled = bpf_map_lookup_elem(&config_map, &config_key);
    if (!enabled || *enabled == 0) {
        return TC_ACT_OK;
    }

    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    struct ethhdr *eth = data;
    if ((void *)eth + sizeof(*eth) > data_end) return TC_ACT_OK;

    struct flow_key key = {0};
    void *l4_header = NULL;

    // 2. 解析网络层
    if (eth->h_proto == bpf_htons(ETH_P_IP)) {
        struct iphdr *ip = data + sizeof(*eth);
        if ((void *)ip + sizeof(*ip) > data_end) return TC_ACT_OK;
        
        key.family = 2; 
        key.src_addr[0] = ip->saddr;
        key.dst_addr[0] = ip->daddr;
        key.proto = ip->protocol;
        l4_header = (void *)ip + (ip->ihl * 4);
    } 
    else if (eth->h_proto == bpf_htons(ETH_P_IPV6)) {
        struct ipv6hdr *ip6 = data + sizeof(*eth);
        if ((void *)ip6 + sizeof(*ip6) > data_end) return TC_ACT_OK;

        key.family = 10; 
        __builtin_memcpy(key.src_addr, &ip6->saddr, 16);
        __builtin_memcpy(key.dst_addr, &ip6->daddr, 16);
        key.proto = ip6->nexthdr;
        l4_header = (void *)ip6 + sizeof(*ip6);
    } 
    else {
        return TC_ACT_OK;
    }

    // 3. 解析传输层端口
    if (l4_header) {
        if (key.proto == IPPROTO_TCP) {
            struct tcphdr *tcp = l4_header;
            if ((void *)tcp + sizeof(*tcp) <= data_end) {
                key.src_port = bpf_ntohs(tcp->source);
                key.dst_port = bpf_ntohs(tcp->dest);
            }
        } else if (key.proto == IPPROTO_UDP) {
            struct udphdr *udp = l4_header;
            if ((void *)udp + sizeof(*udp) <= data_end) {
                key.src_port = bpf_ntohs(udp->source);
                key.dst_port = bpf_ntohs(udp->dest);
            }
        }
    }

    // 4. 更新统计
    // 在 Per-CPU Map 中，lookup 返回的是当前 CPU 核心对应的私有存储区
    struct flow_stats *val = bpf_map_lookup_elem(&flow_map, &key);
    __u64 now = bpf_ktime_get_ns();

    if (val) {
        // 无需 __sync_fetch_and_add，直接累加性能最高
        val->packets += 1;
        val->bytes += skb->len;
        val->last_seen = now;
    } else {
        struct flow_stats new_stats = {
            .packets = 1, 
            .bytes = skb->len, 
            .last_seen = now
        };
        // 第一次见到该 Flow，初始化当前 CPU 核心的统计
        bpf_map_update_elem(&flow_map, &key, &new_stats, BPF_ANY);
    }

    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";