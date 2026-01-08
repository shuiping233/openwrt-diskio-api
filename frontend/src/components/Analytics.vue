<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useDatabase } from '../useDatabase';
import type { HistoryRecord } from '../model';

const { getConfig, setConfig, getHistory, addHistory, clearHistory } = useDatabase();

// 状态
const retentionDays = ref(7);
const historyList = ref<HistoryRecord[]>([]);
const selectedMetric = ref('cpu'); // 下拉选
const isLoading = ref(false);
const message = ref('');

// 初始化加载配置
onMounted(async () => {
  const savedDays = await getConfig<number>('retention_days');
  if (savedDays) retentionDays.value = savedDays;
  await fetchHistoryData();
});

// 操作：保存配置
const saveSetting = async () => {
  await setConfig('retention_days', retentionDays.value);
  message.value = '配置已保存';
  setTimeout(() => message.value = '', 2000);
};

// 操作：查询数据
const fetchHistoryData = async () => {
  isLoading.value = true;
  historyList.value = await getHistory(
    selectedMetric.value as any, 
    24 * 60 * 60 * 1000 // 查最近24小时
  );
  isLoading.value = false;
};

// 操作：模拟写入测试数据 (点击按钮生成随机数据入库)
const testInsert = async () => {
  const record: Omit<HistoryRecord, 'id'> = {
    timestamp: Date.now(),
    metric: selectedMetric.value as any,
    value: Math.random() * 100,
    unit: '%'
  };
  await addHistory(record);
  message.value = '测试数据已写入';
  await fetchHistoryData(); // 刷新列表
};

// 操作：清空
const handleClear = async () => {
  if (confirm(`确定清空 ${selectedMetric.value} 的历史数据吗？`)) {
    await clearHistory(selectedMetric.value as any);
    message.value = '数据已清空';
    await fetchHistoryData();
  }
};
</script>

<template>
  <div class="p-6 text-slate-200">
    <h2 class="text-2xl font-bold mb-6">历史数据分析</h2>

    <!-- 1. 配置区域 -->
    <div class="bg-slate-800 p-5 rounded-xl mb-6 border border-slate-700">
      <h3 class="text-lg font-semibold mb-4 text-blue-400">用户配置</h3>
      <div class="flex items-center gap-4">
        <div>
          <label class="block text-sm text-slate-400 mb-1">数据保留天数</label>
          <input 
            v-model.number="retentionDays" 
            type="number" 
            min="1" max="365"
            class="bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-white w-32 outline-none focus:border-blue-500"
          />
        </div>
        <button @click="saveSetting" class="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 rounded text-sm transition">
          保存配置
        </button>
        <span v-if="message" class="text-green-400 text-sm">{{ message }}</span>
      </div>
    </div>

    <!-- 2. 数据操作区域 -->
    <div class="bg-slate-800 p-5 rounded-xl mb-6 border border-slate-700">
      <h3 class="text-lg font-semibold mb-4 text-orange-400">数据操作</h3>
      
      <div class="flex flex-wrap gap-4 mb-6">
        <select v-model="selectedMetric" class="bg-slate-900 border border-slate-600 rounded px-3 py-1.5 outline-none">
          <option value="cpu">CPU</option>
          <option value="cpu_temp">CPU 温度</option>
          <option value="memory">内存</option>
          <option value="network_in">网络入站</option>
          <option value="network_out">网络出站</option>
        </select>

        <button @click="fetchHistoryData" class="px-4 py-1.5 bg-slate-600 hover:bg-slate-500 rounded text-sm transition">
          查询数据
        </button>
        <button @click="testInsert" class="px-4 py-1.5 bg-green-600 hover:bg-green-500 rounded text-sm transition">
          写入测试数据
        </button>
        <button @click="handleClear" class="px-4 py-1.5 bg-red-600 hover:bg-red-500 rounded text-sm transition">
          清空历史
        </button>
      </div>

      <div class="text-xs text-slate-500 mb-2">数据预览 (最近24小时):</div>
      <div class="bg-slate-900 rounded-lg overflow-hidden border border-slate-700 max-h-64 overflow-y-auto">
        <table class="w-full text-xs text-left border-collapse">
          <thead class="bg-slate-800 text-slate-400">
            <tr>
              <th class="px-3 py-2">时间</th>
              <th class="px-3 py-2">指标</th>
              <th class="px-3 py-2">数值</th>
              <th class="px-3 py-2">单位</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800">
            <tr v-for="item in historyList" :key="item.id">
              <td class="px-3 py-2 font-mono">{{ new Date(item.timestamp).toLocaleString() }}</td>
              <td class="px-3 py-2">{{ item.metric }}</td>
              <td class="px-3 py-2 font-mono">{{ item.value.toFixed(2) }}</td>
              <td class="px-3 py-2">{{ item.unit }}</td>
            </tr>
            <tr v-if="historyList.length === 0">
              <td colspan="4" class="px-3 py-4 text-center text-slate-500">暂无数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 3. 图表占位区域 -->
    <div class="bg-slate-800 p-5 rounded-xl border border-slate-700 h-96 flex items-center justify-center">
      <div class="text-center">
        <p class="text-xl font-bold text-slate-400">ECharts 图表区域</p>
        <p class="text-sm text-slate-500 mt-2">等待下一步实现...</p>
      </div>
    </div>

  </div>
</template>