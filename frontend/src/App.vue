<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, type Ref } from 'vue';
import type {
  DynamicApiResponse, StaticApiResponse, ConnectionApiResponse
} from './model';



// ================= 2. 状态定义 =================

const data = reactive({
  dynamic: {} as DynamicApiResponse,
  static: {} as StaticApiResponse,
  connection: {} as ConnectionApiResponse
});

const uiState = reactive({
  activeTab: 'system' as 'system' | 'network',
  refreshInterval: 2000,
  lastUpdated: '--',
  isLoading: false,
  status: '初始化...',
  statusColor: '#94a3b8',
  // 折叠面板状态
  accordions: {
    storage: true,
    cpu: true,
    network: true,
    sysinfo: true,
    conn: true
  }
});

let timer: Ref<number | null> = ref(null);

// ================= 3. 辅助函数 =================

const formatTime = (): string => {
  const now = new Date();
  return now.toISOString().replace('T', ' ').substring(0, 19);
};

// const formatBytes = (bytes: number | undefined): string => {
//   if (!bytes || bytes === 0 || bytes === -1) return '0';
//   if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(2) + ' GB';
//   if (bytes >= 1048576) return (bytes / 1048576).toFixed(2) + ' MB';
//   if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB';
//   return bytes.toFixed(0);
// };

// ================= 4. 核心逻辑 =================

const fetchData = async () => {
  uiState.isLoading = true;
  uiState.status = '刷新中...';
  const reqTime = formatTime();

  try {
    const [dRes, cRes, sRes] = await Promise.all([
      fetch('/metric/dynamic'),
      fetch('/metric/network_connection'),
      fetch('/metric/static')
    ]);

    if (!dRes.ok || !cRes.ok || !sRes.ok) throw new Error('接口请求失败');

    data.dynamic = (await dRes.json()) as DynamicApiResponse;
    data.connection = (await cRes.json()) as ConnectionApiResponse;
    data.static = (await sRes.json()) as StaticApiResponse;

    uiState.status = '运行中';
    uiState.statusColor = '#10b981'; // green
  } catch (e) {
    console.error(e);
    uiState.status = '错误';
    uiState.statusColor = '#ef4444'; // red
  } finally {
    uiState.isLoading = false;
    uiState.lastUpdated = reqTime;
  }
};

const startPolling = () => {
  if (timer.value) clearInterval(timer.value);
  timer.value = window.setInterval(fetchData, uiState.refreshInterval);
};

// ================= 5. 生命周期 =================

onMounted(() => {
  fetchData();
  startPolling();
});

onUnmounted(() => {
  if (timer.value) clearInterval(timer.value);
});
</script>

<template>
  <!-- 整个应用容器 -->
  <div class="max-auto mx-auto p-5 bg-slate-900 text-slate-50 min-h-screen">

    <!-- Header -->
    <header class="flex justify-between items-center mb-8 pb-5 border-b border-slate-700">
      <div class="flex items-center gap-2">
        <!-- <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          class="text-white">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg> -->
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
        </svg>
        <span class="text-xl font-bold">系统 I/O 监控仪表盘</span>
      </div>

      <div class="flex items-center gap-2 text-sm text-slate-400">
        <!-- Select -->
        <select v-model.number="uiState.refreshInterval" @change="startPolling"
          class="bg-slate-800 text-white border border-slate-700 rounded px-2 py-1 outline-none focus:border-slate-500 cursor-pointer">
          <option :value="1000">1s</option>
          <option :value="2000">2s</option>
          <option :value="5000">5s</option>
          <option :value="10000">10s</option>
        </select>

        <span class="font-mono">{{ uiState.lastUpdated }}</span>

        <!-- Status Dot -->
        <div
          :style="{ width: '8px', height: '8px', borderRadius: '50%', background: uiState.statusColor, boxShadow: `0 0 8px ${uiState.statusColor}` }">
        </div>
        <span>{{ uiState.status }}</span>

        <!-- Spinner: Using Tailwind animate-spin -->
        <div v-if="uiState.isLoading"
          class="w-3.5 h-3.5 border-2 border-slate-500 border-t-white rounded-full animate-spin"></div>
      </div>
    </header>

    <!-- Tabs -->
    <nav class="flex gap-2 mb-5">
      <button @click="uiState.activeTab = 'system'"
        class="px-5 py-2 text-sm font-semibold cursor-pointer border-none transition-colors" :class="[
          uiState.activeTab === 'system'
            ? 'text-white border-b-2 border-blue-500 bg-transparent'
            : 'text-slate-400 bg-slate-800/50 hover:bg-slate-800'
        ]">
        系统概览
      </button>
      <button @click="uiState.activeTab = 'network'"
        class="px-5 py-2 text-sm font-semibold cursor-pointer border-none transition-colors" :class="[
          uiState.activeTab === 'network'
            ? 'text-white border-b-2 border-blue-500 bg-transparent'
            : 'text-slate-400 bg-slate-800/50 hover:bg-slate-800'
        ]">
        网络连接
      </button>
    </nav>

    <!-- Tab: System Overview -->
    <div v-if="uiState.activeTab === 'system'">

      <!-- 1. Summary Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 mb-10">
        <!-- CPU -->
        <div v-if="data.dynamic.cpu?.total?.usage"
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl hover:border-slate-500">
          <div class="text-slate-400 text-sm mb-1">CPU 总使用率</div>
          <div class="text-xl font-bold mt-1">{{ data.dynamic.cpu.total.usage.value.toFixed(2) }} <span
              class="text-slate-400 text-sm">{{ data.dynamic.cpu.total.usage.unit }}</span></div>
          <div class="h-1 bg-slate-700 mt-3 rounded-full overflow-hidden">
            <div class="h-full bg-violet-500 transition-all duration-500"
              :style="{ width: data.dynamic.cpu.total.usage.value + '%' }"></div>
          </div>
        </div>

        <!-- Memory -->
        <div v-if="data.dynamic.memory?.used_percent"
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl hover:border-slate-500">
          <div class="text-slate-400 text-sm mb-1">内存使用率</div>

          <div class="flex justify-between items-baseline mt-1">
            <!-- 左侧：百分比 -->
            <div class="text-xl font-bold">
              {{ data.dynamic.memory.used_percent.value.toFixed(2) }}
              <span class="text-slate-300 text-sm ml-0.5">{{ data.dynamic.memory.used_percent.unit }}</span>
            </div>

            <!-- 右侧：具体使用量 -->
            <div class="text-right">
              <span class="font-bold">{{ data.dynamic.memory.used.value.toFixed(2) }}</span>
              <span class="text-slate-300 text-sm ml-0.5">{{ data.dynamic.memory.used.unit }}</span>
              <span class="text-slate-400 mx-1">/</span>
              <span class="font-bold">{{ data.dynamic.memory.total.value.toFixed(2) }}</span>
              <span class="text-slate-300 text-sm ml-0.5">{{ data.dynamic.memory.total.unit }}</span>
            </div>
          </div>

          <div class="h-1 bg-slate-700 mt-3 rounded-full overflow-hidden">
            <div class="h-full bg-blue-500 transition-all duration-500"
              :style="{ width: data.dynamic.memory.used_percent.value + '%' }"></div>
          </div>
        </div>

        <!-- Network In -->
        <div v-if="data.dynamic.network?.total?.incoming" class="bg-slate-800 border border-slate-700 rounded-xl p-5">
          <div class="flex items-center justify-between">
            <div class="text-slate-400 text-sm">网络下行</div>
            <div class="text-xl font-bold font-mono text-cyan-500">
              {{ data.dynamic.network.total.incoming.value.toFixed(2) }}
              <span class="text-slate-400 text-sm">{{ data.dynamic.network.total.incoming.unit }}</span>
            </div>
          </div>
        </div>

        <!-- Network Out -->
        <div v-if="data.dynamic.network?.total?.outgoing" class="bg-slate-800 border border-slate-700 rounded-xl p-5">
          <div class="flex items-center justify-between">
            <div class="text-slate-400 text-sm">网络上行</div>
            <div class="text-xl font-bold font-mono text-orange-500">
              {{ data.dynamic.network.total.outgoing.value.toFixed(2) }}
              <span class="text-slate-400 text-sm">{{ data.dynamic.network.total.outgoing.unit }}</span>
            </div>
          </div>
        </div>

        <!-- System Info (Smaller cards) -->
        <div v-if="data.dynamic.system?.uptime"
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 flex items-center justify-between">
          <div class="text-slate-400 text-sm">运行时间</div>
          <div class="text-lg font-bold">{{ data.dynamic.system.uptime }}</div>
        </div>
        <div v-if="data.static.system?.hostname"
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 flex items-center justify-between">
          <div class="text-slate-400 text-sm">主机名</div>
          <div class="text-lg font-bold font-mono">{{ data.static.system.hostname }}</div>
        </div>
      </div>

      <!-- 2. Detailed Sections (Accordions with Cards) -->

      <!-- Storage -->
      <div v-if="data.dynamic.storage">
        <div @click="uiState.accordions.storage = !uiState.accordions.storage"
          class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">存储详情</h3>
          <span class="text-slate-500 transition-transform duration-300"
            :class="{ 'rotate-180': uiState.accordions.storage }">▼</span>
        </div>
        <div v-show="uiState.accordions.storage" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          <div v-for="(dev, name) in data.dynamic.storage" :key="name"
            class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl">
            <div class="flex justify-between items-center mb-4">
              <span class="text-xl font-bold">{{ name }}</span>
              <span class="bg-slate-700 px-2 py-0.5 rounded text-xs font-mono text-slate-300">{{
                dev.used_percent.value.toFixed(2) }}%</span>
            </div>
            <div class="grid grid-cols-3 gap-2 text-sm mb-3">
              <div><span class="text-slate-500">读:</span> {{ dev.read.value }} <span class="font-mono text-slate-200">
                  {{ dev.read.unit }}</span></div>
              <div><span class="text-slate-500">写:</span> {{ dev.write.value }} <span class="font-mono text-slate-200">
                  {{ dev.write.unit }}</span></div>
              <div><span class="text-slate-500"></span></div>
              <div><span class="text-slate-500">使用量:</span> <span class="font-mono">{{ dev.used_percent.value.toFixed(2)
                  }} {{
                    dev.used_percent.unit }}</span></div>
              <div><span class="text-slate-500">总容量:</span> <span class="font-mono">{{ dev.total.value.toFixed(2) }} {{
                dev.total.unit }}</span></div>
              <div><span class="text-slate-500">已用:</span> <span class="font-mono">{{ dev.used.value.toFixed(2) }} {{
                dev.used.unit
                  }}</span></div>
            </div>
            <div class="h-1.5 bg-slate-900 rounded-full overflow-hidden mt-2">
              <div class="h-full bg-cyan-500 transition-all duration-500"
                :style="{ width: Math.min(dev.used_percent.value, 100) + '%' }"></div>
            </div>
          </div>
        </div>
      </div>

      <!-- CPU -->
      <div v-if="data.dynamic.cpu" class="mt-8">
        <div @click="uiState.accordions.cpu = !uiState.accordions.cpu"
          class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">CPU 核心详情</h3>
          <span class="text-slate-500 transition-transform duration-300"
            :class="{ 'rotate-180': uiState.accordions.cpu }">▼</span>
        </div>
        <div v-show="uiState.accordions.cpu" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
          <div v-for="(core, name) in data.dynamic.cpu" :key="name"
            class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl">
            <div class="flex justify-between mb-2">
              <span class="text-lg font-bold">{{ name }}</span>
              <span class="text-lg font-bold">{{ core.usage.value.toFixed(2) }}%</span>
            </div>
            <div class="h-1.5 bg-slate-900 rounded-full overflow-hidden mb-2">
              <div class="h-full bg-violet-500 transition-all duration-500"
                :style="{ width: Math.min(core.usage.value, 100) + '%' }"></div>
            </div>
            <div v-if="core.temperature.value > 0" class="text-xs text-slate-500 mt-2">温度: {{
              core.temperature.value.toFixed(0) }}°C</div>
          </div>
        </div>
      </div>

      <!-- Network Interfaces -->
      <div v-if="data.dynamic.network || data.static.network" class="mt-8">
        <div @click="uiState.accordions.network = !uiState.accordions.network"
          class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">网络配置详情</h3>
          <span class="text-slate-500 transition-transform duration-300"
            :class="{ 'rotate-180': uiState.accordions.network }">▼</span>
        </div>
        <div v-show="uiState.accordions.network" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          <!-- IO Cards -->
          <template v-for="(net, iface) in data.dynamic.network" :key="'io-'+iface">
            <div v-if="iface !== 'total'"
              class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl">
              <h3 class="text-lg font-bold mb-4">{{ iface }} <span class="text-slate-500 text-sm font-normal">IO</span>
              </h3>
              <div class="flex justify-between items-center">
                <div class="text-xl font-bold font-mono text-cyan-500">↓ {{ net.incoming.value.toFixed(2) }} {{
                  net.incoming.unit }} </div>
                <div class="text-xl font-bold font-mono text-orange-500">↑ {{ net.outgoing.value.toFixed(2) }} {{
                  net.outgoing.unit }} </div>
              </div>
            </div>
          </template>
          <!-- IP Cards -->
          <template v-for="(info, iface) in data.static.network" :key="'ip-' + iface">
            <div v-if="iface !== 'global' && iface !== 'lo'"
              class="bg-slate-800 border border-slate-700 rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-xl">
              <h3 class="text-lg font-bold mb-3">{{ iface }} <span class="text-slate-500 text-sm font-normal">IP</span>
              </h3>
              <div class="text-sm font-mono wrap-break-word space-y-1">
                <div v-for="ip in info.ipv4" :key="ip" class="text-slate-200">{{ ip }}</div>
                <div v-for="ip in info.ipv6" :key="ip" class="text-slate-200 text-xs">{{ ip }}</div>
              </div>
            </div>
          </template>
          <!-- Gateway -->
          <div v-if="data.static.network?.global?.gateway && data.static.network.global.gateway !== 'unknown'"
            class="bg-slate-800 border border-slate-700 rounded-xl p-5 flex flex-col justify-center">
            <h3 class="text-slate-400 text-sm mb-2">网关</h3>
            <div class="text-2xl font-bold font-mono">{{ data.static.network.global.gateway }}</div>
          </div>
        </div>
      </div>

      <!-- System Info -->
      <div v-if="data.static.system" class="mt-8">
        <div @click="uiState.accordions.sysinfo = !uiState.accordions.sysinfo"
          class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">系统信息详情</h3>
          <span class="text-slate-500 transition-transform duration-300"
            :class="{ 'rotate-180': uiState.accordions.sysinfo }">▼</span>
        </div>
        <div v-show="uiState.accordions.sysinfo" class="bg-slate-800 border border-slate-700 rounded-xl p-6">
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div><span class="text-slate-500 block text-sm mb-1">OS:</span> <span class="font-medium">{{
              data.static.system.os
                }}</span></div>
            <div><span class="text-slate-500 block text-sm mb-1">Kernel:</span> <span class="font-medium">{{
              data.static.system.kernel }}</span></div>
            <div><span class="text-slate-500 block text-sm mb-1">Arch:</span> <span class="font-medium">{{
              data.static.system.arch }}</span></div>
            <div><span class="text-slate-500 block text-sm mb-1">Timezone:</span> <span class="font-medium">{{
              data.static.system.timezone }}</span></div>
          </div>
        </div>
      </div>

    </div>

    <!-- Tab: Network Connections -->
    <div v-if="uiState.activeTab === 'network'">
      <!-- Counts -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-5 mb-8">
        <div
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-blue-500 flex items-center justify-between">
          <div>
            <div class="text-slate-400 text-sm">TCP 连接</div>
            <div class="text-3xl font-bold">{{ data.connection.counts?.tcp || 0 }}</div>
          </div>
          <div class="text-blue-500/20 text-4xl">T</div>
        </div>
        <div
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-violet-500 flex items-center justify-between">
          <div>
            <div class="text-slate-400 text-sm">UDP 连接</div>
            <div class="text-3xl font-bold">{{ data.connection.counts?.udp || 0 }}</div>
          </div>
          <div class="text-violet-500/20 text-4xl">U</div>
        </div>
        <div
          class="bg-slate-800 border border-slate-700 rounded-xl p-5 border-t-4 border-t-white flex items-center justify-between">
          <div>
            <div class="text-slate-400 text-sm">其他连接</div>
            <div class="text-3xl font-bold">{{ data.connection.counts?.other || 0 }}</div>
          </div>
          <div class="text-white/20 text-4xl">?</div>
        </div>
      </div>

      <!-- Table -->
      <div @click="uiState.accordions.conn = !uiState.accordions.conn"
        class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
        <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">连接列表</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': uiState.accordions.conn }">▼</span>
      </div>
      <div v-show="uiState.accordions.conn" class="bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
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
              <tr v-for="(c, i) in data.connection.connections" :key="i"
                class="hover:bg-slate-700/30 transition-colors">
                <td class="px-5 py-3">
                  <span class="bg-slate-700 px-2 py-1 rounded text-xs text-slate-200">{{ c.ip_family.toUpperCase()
                    }}</span>
                </td>
                <td class="px-5 py-3">
                  <span class="bg-slate-700 px-2 py-1 rounded text-xs text-slate-200">{{ c.protocol.toUpperCase()
                    }}</span>
                </td>
                <td class="px-5 py-3 font-mono text-slate-300">{{ c.source_ip }}{{ c.source_port > 0 ? ':' +
                  c.source_port
                  :
                  '' }}</td>
                <td class="px-5 py-3 font-mono text-slate-300">{{ c.destination_ip }}{{ c.destination_port > 0 ?
                  ':' + c.destination_port : '' }}</td>
                <td class="px-5 py-3 text-slate-300 ">{{ c.state || '-' }}</td>
                <td class="px-5 py-3 text-slate-300 ">{{ c.traffic.value.toFixed(2) }} {{ c.traffic.unit }} ({{
                  c.packets }}
                  Pkgs.)
                </td>
              </tr>
              <tr v-if="!data.connection.connections || data.connection.connections.length === 0">
                <td colspan="4" class="px-5 py-8 text-center text-slate-500">暂无连接数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

  </div>
</template>

<style></style>