<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, type Ref } from 'vue';
import dayjs from 'dayjs';
import type {
  DynamicApiResponse, StaticApiResponse, ConnectionApiResponse
} from './model';
import NetworkConnectionTable from './components/NetworkConnectionTable.vue';
import { useToast } from './useToast';
import Toaster from './components/Toaster.vue';



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
  }
});

let timer: Ref<number | null> = ref(null);

// ================= 3. 辅助函数 =================

const formatTime = (): string => {
  return dayjs().format('YYYY-MM-DD HH:mm:ss')
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

    // 直接抛出原生错误，而不是自定义错误
    if (!dRes.ok) throw new Error(`动态数据接口错误: ${dRes.status} ${dRes.statusText}`);
    if (!cRes.ok) throw new Error(`网络连接接口错误: ${cRes.status} ${cRes.statusText}`);
    if (!sRes.ok) throw new Error(`静态数据接口错误: ${sRes.status} ${sRes.statusText}`);

    data.dynamic = (await dRes.json()) as DynamicApiResponse;
    data.connection = (await cRes.json()) as ConnectionApiResponse;
    data.static = (await sRes.json()) as StaticApiResponse;

    uiState.status = '运行中';
    uiState.statusColor = '#10b981'; // green
  } catch (e) {
    console.error(e);
    uiState.status = '错误';
    uiState.statusColor = '#ef4444'; // red
    const { error } = useToast();
    
    // 根据错误类型显示不同消息
    if (e instanceof TypeError) {
      // 网络错误，如连接失败
      error(`网络错误: ${e.message}`);
    } else if (e.message.includes('接口错误')) {
      // HTTP 错误状态
      error(e.message);
    } else {
      // 其他错误
      error(`请求失败: ${e.message}`);
    }
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
    <div v-if="uiState.activeTab === 'network'" class="p-0"> <!-- p-0 可以根据需要调整 -->
      <!-- 👇 使用组件，传入连接数据 -->
      <NetworkConnectionTable :connection-data="data.connection" />
    </div>

    <Toaster />
  </div>
  
</template>

<style></style>