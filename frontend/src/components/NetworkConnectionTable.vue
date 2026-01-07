<script setup lang="ts">
import { ref } from 'vue';
import type { ConnectionApiResponse } from '../model';
import { compressIPv6 } from '../utils/ipv6';

// 定义接收的属性
const props = defineProps<{
    connectionData?: ConnectionApiResponse;
}>();

// 内部状态：控制折叠面板 (不再依赖父组件的 accordions.conn)
const isOpen = ref(true);

// 👇 新增：智能格式化 IP 地址
// 如果包含 ':' (IPv6) 则压缩，否则原样返回 (IPv4)
const formatIP = (ip: string | undefined, family: string | undefined): string => {
    if (!ip) return '-';
    // 👇 直接判断 ip_family 是否为 IPv6
    if (family?.toUpperCase() === 'IPV6') {
        return compressIPv6(ip);
    }
    return ip;
};
</script>

<template>
    <div class="flex flex-col h-full">
        <!-- Counts -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-5 mb-8">
            <div
                class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-blue-500 flex items-center justify-between">
                <div>
                    <div class="text-slate-400 text-sm">TCP 连接</div>
                    <div class="text-3xl font-bold">{{ connectionData?.counts?.tcp || 0 }}</div>
                </div>
                <div class="text-blue-500/20 text-4xl">T</div>
            </div>
            <div
                class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-violet-500 flex items-center justify-between">
                <div>
                    <div class="text-slate-400 text-sm">UDP 连接</div>
                    <div class="text-3xl font-bold">{{ connectionData?.counts?.udp || 0 }}</div>
                </div>
                <div class="text-violet-500/20 text-4xl">U</div>
            </div>
            <div
                class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-white flex items-center justify-between">
                <div>
                    <div class="text-slate-400 text-sm">其他连接</div>
                    <div class="text-3xl font-bold">{{ connectionData?.counts?.other || 0 }}</div>
                </div>
                <div class="text-white/20 text-4xl">?</div>
            </div>
        </div>

        <!-- Table -->
        <div @click="isOpen = !isOpen"
            class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
            <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">连接列表</h3>
            <span class="text-slate-500 transition-transform duration-300" :class="{ 'rotate-180': isOpen }">▼</span>
        </div>
        <div v-show="isOpen" class="bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full text-sm text-center border-collapse">
                    <thead class="bg-slate-700/50 text-slate-300">
                        <tr>
                            <th class="px-5 py-3 font-medium">地址族</th>
                            <th class="px-5 py-3 font-medium">协议</th>
                            <th class="px-5 py-3 font-medium">源地址</th>
                            <th class="px-5 py-3 font-medium">目标地址</th>
                            <th class="px-5 py-3 font-medium">状态</th>
                            <th class="px-5 py-3 font-medium">传输情况</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-700">
                        <tr v-for="(c, i) in connectionData?.connections" :key="i"
                            class="hover:bg-slate-700/30 transition-colors">
                            <td class="px-5 py-3">
                                <span class="bg-slate-700 px-2 py-1 rounded text-xs text-slate-200">{{
                                    c.ip_family?.toUpperCase() }}</span>
                            </td>
                            <td class="px-5 py-3">
                                <span class="bg-slate-700 px-2 py-1 rounded text-xs text-slate-200">{{
                                    c.protocol?.toUpperCase() }}</span>
                            </td>
                            <td class="px-5 py-3 font-mono text-slate-300">{{ formatIP(c.source_ip, c.ip_family) }}{{
                                c.source_port > 0 ? ':' +
                                    c.source_port
                                :
                                '' }}</td>
                            <td class="px-5 py-3 font-mono text-slate-300">{{ formatIP(c.destination_ip, c.ip_family)
                                }}{{ c.destination_port > 0
                                    ?
                                ':' + c.destination_port : '' }}</td>
                            <td class="px-5 py-3 text-slate-300 ">{{ c.state || '-' }}</td>
                            <td class="px-5 py-3 text-slate-300 ">{{ c.traffic?.value.toFixed(2) }} {{ c.traffic?.unit
                                }} ({{
                                    c.packets
                                }}
                                Pkgs.)
                            </td>
                        </tr>
                        <tr v-if="!connectionData?.connections || connectionData.connections.length === 0">
                            <td colspan="6" class="px-5 py-8 text-center text-slate-500">暂无连接数据</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>