<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, type Ref } from 'vue';
import dayjs from 'dayjs';
import type {
  DynamicApiResponse, StaticApiResponse, ConnectionApiResponse
} from './model';
import SettingsModal from './components/SettingsModal.vue';
import NetworkConnectionTable from './components/NetworkConnectionTable.vue';
import SystemOverview from './components/SystemOverview.vue';
import Analytics from './components/Analytics.vue';
import { useToast } from './useToast';
import Toaster from './components/Toaster.vue';



// ================= 2. 状态定义 =================

const data = reactive({
  dynamic: {} as DynamicApiResponse,
  static: {} as StaticApiResponse,
  connection: {} as ConnectionApiResponse
});

const showSettings = ref(false);

const uiState = reactive({
  activeTab: 'system' as 'system' | 'network' | 'analytics',
  refreshInterval: 2000,
  lastUpdated: '--',
  isLoading: false,
  status: '初始化...'
});

let timer: Ref<number | null> = ref(null);

// ================= 3. 辅助函数 =================

const formatTime = (): string => {
  return dayjs().format('YYYY-MM-DD HH:mm:ss')
};


// 状态颜色映射
const getStatusColor = (status: string): string => {
  switch (status) {
    case '运行中':
      return '#10b981'; // 绿色
    case '刷新中...':
      return '#3b82f6'; // 蓝色
    case '错误':
      return '#ef4444'; // 红色
    default:
      return '#94a3b8'; // 灰色
  }
};

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
  } catch (e) {
    console.error(e);
    uiState.status = '错误';
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
  const { success } = useToast();
  success(`刷新间隔已调整为 ${uiState.refreshInterval / 1000} 秒`);
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
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
        </svg>
        <span class="text-xl font-bold">系统 I/O 监控仪表盘</span>
      </div>

      <div class="flex items-center gap-2 text-sm text-slate-400">
        <!-- Status Dot -->
        <div
          :style="{ width: '8px', height: '8px', borderRadius: '50%', background: getStatusColor(uiState.status), boxShadow: `0 0 8px ${getStatusColor(uiState.status)}` }">
        </div>
        <span>{{ uiState.status }}</span>
        <!-- Spinner: Using Tailwind animate-spin -->
        <div v-if="uiState.isLoading"
          class="w-3.5 h-3.5 border-2 border-slate-500 border-t-white rounded-full animate-spin"></div>

        <span class="font-mono">{{ uiState.lastUpdated }}</span>

        <!-- Select -->
        <select v-model.number="uiState.refreshInterval" @change="startPolling"
          class="bg-slate-800 text-white border border-slate-700 rounded px-2 py-1 outline-none focus:border-slate-500 cursor-pointer">
          <option :value="1000">1s</option>
          <option :value="2000">2s</option>
          <option :value="3000">3s</option>
          <option :value="5000">5s</option>
          <option :value="10000">10s</option>
          <option :value="30000">30s</option>
        </select>

        <!-- 👇 新增：设置齿轮按钮 -->
        <button @click="showSettings = true" class="text-slate-400 hover:text-white transition-colors" title="设置">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 4.085 1.584a1.724 1.724 0 001.145 2.163c1.543.94 3.31.826 4.085 1.584a1.724 1.724 0 002.573 1.066c1.543-.94 3.31-.826 4.085-1.584a1.724 1.724 0 001.145-2.163c1.543-.94 3.31-.826 4.085-1.584a1.724 1.724 0 002.573-1.066c-1.543.94-3.31.826-4.085-1.584a1.724 1.724 0 00-1.145-2.163c1.543-.94 3.31-.826 4.085-1.584a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31.826-4.085 1.584a1.724 1.724 0 00-1.145 2.163c1.543.94 3.31.826 4.085 1.584a1.724 1.724 0 002.573 1.066c1.543-.94 3.31-.826 4.085-1.584a1.724 1.724 0 001.145-2.163c-1.543-.94-3.31-.826-4.085 1.584a1.724 1.724 0 00-2.573-1.066zM15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.384-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 1.414L8.586 10H7a2 2 0 01-2-2V5a2 2 0 012-2z" />
          </svg>
        </button>

      </div>
    </header>

    <!-- Tabs -->
    <nav class="flex gap-2 mb-5">
      <button @click="uiState.activeTab = 'system'"
        class="px-5 py-2 text-sm font-semibold cursor-pointer border border-slate-700 rounded-lg transition-colors"
        :class="[
          uiState.activeTab === 'system'
            ? 'text-white border-b-2 border-blue-500 bg-transparent'
            : 'text-slate-400 bg-slate-800/50 hover:bg-slate-800'
        ]">
        系统概览
      </button>
      <button @click="uiState.activeTab = 'network'"
        class="px-5 py-2 text-sm font-semibold cursor-pointer border border-slate-700 rounded-lg transition-colors"
        :class="[
          uiState.activeTab === 'network'
            ? 'text-white border-b-2 border-blue-500 bg-transparent'
            : 'text-slate-400 bg-slate-800/50 hover:bg-slate-800'
        ]">
        网络连接
      </button>
      <button @click="uiState.activeTab = 'analytics'"
        class="px-5 py-2 text-sm font-semibold cursor-pointer border border-slate-700 rounded-lg transition-colors"
        :class="[
          uiState.activeTab === 'analytics'
            ? 'text-white border-b-2 border-blue-500 bg-transparent'
            : 'text-slate-400 bg-slate-800/50 hover:bg-slate-800'
        ]">
        历史数据分析
      </button>
    </nav>

    <!-- Tab: System Overview -->
    <div v-if="uiState.activeTab === 'system'">
      <SystemOverview :data="data" />
    </div>

    <!-- Tab: Network Connections -->
    <div v-if="uiState.activeTab === 'network'" class="p-0"> <!-- p-0 可以根据需要调整 -->
      <!-- 👇 使用组件，传入连接数据 -->
      <NetworkConnectionTable :connection-data="data.connection" />
    </div>
    <!-- Tab: Analytics -->
    <div v-if="uiState.activeTab === 'analytics'">
      <!-- 👇 传入 data -->
      <Analytics :data="data" />
    </div>

    <SettingsModal v-model:isOpen="showSettings" />
    
    <Toaster />
  </div>

</template>

<style></style>