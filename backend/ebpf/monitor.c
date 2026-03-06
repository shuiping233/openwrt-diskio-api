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

/* * 统一的流量 Key 结构体
 * 兼容 IPv4 (使用 addr[0]) 和 IPv6 (使用 addr[0-3])
 */
struct flow_key {
    __u32 src_addr[4];
    __u32 dst_addr[4];
    __u16 src_port;
    __u16 dst_port;
    __u8  family; // AF_INET (2) 或 AF_INET6 (10)
    __u8  proto;  // IPPROTO_TCP, IPPROTO_UDP 等
    __u8  _pad[2]; // 显式对齐 
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

// 流量统计 Map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 65535);
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

    // 2. 解析以太网帧头
    struct ethhdr *eth = data;
    if (data + sizeof(*eth) > data_end) return TC_ACT_OK;

    struct flow_key key = {0};
    void *l4_header = NULL;

    // 3. 分支处理 IPv4 和 IPv6
    if (eth->h_proto == bpf_htons(ETH_P_IP)) {
        struct iphdr *ip = data + sizeof(*eth);
        if ((void *)ip + sizeof(*ip) > data_end) return TC_ACT_OK;
        
        key.family = 2; // AF_INET
        key.src_addr[0] = ip->saddr;
        key.dst_addr[0] = ip->daddr;
        key.proto = ip->protocol;
        
        // 计算 L4 偏移 (注意 IHL 是 4 字节单位)
        l4_header = (void *)ip + (ip->ihl * 4);
    } 
    else if (eth->h_proto == bpf_htons(ETH_P_IPV6)) {
        struct ipv6hdr *ip6 = data + sizeof(*eth);
        if ((void *)ip6 + sizeof(*ip6) > data_end) return TC_ACT_OK;

        key.family = 10; // AF_INET6
        // 拷贝 128 位地址
        __builtin_memcpy(key.src_addr, &ip6->saddr, 16);
        __builtin_memcpy(key.dst_addr, &ip6->daddr, 16);
        key.proto = ip6->nexthdr;
        
        l4_header = (void *)ip6 + sizeof(*ip6);
    } 
    else {
        // 忽略 ARP, PPPoE 等其他协议
        return TC_ACT_OK;
    }

    // 4. 解析端口号 (TCP/UDP)
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

    // 5. 更新统计 Map
    struct flow_stats *val = bpf_map_lookup_elem(&flow_map, &key);
    __u64 now = bpf_ktime_get_ns();

    if (val) {
        __sync_fetch_and_add(&val->packets, 1);
        __sync_fetch_and_add(&val->bytes, skb->len);
        val->last_seen = now;
    } else {
        struct flow_stats new_stats = {
            .packets = 1, 
            .bytes = skb->len, 
            .last_seen = now
        };
        bpf_map_update_elem(&flow_map, &key, &new_stats, BPF_ANY);
    }

    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";