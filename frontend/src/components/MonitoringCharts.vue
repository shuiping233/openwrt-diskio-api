<script setup lang="ts">
import { ref, reactive, onMounted, watch, nextTick } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  ToolboxComponent
} from 'echarts/components';
import type { EChartsOption } from 'echarts';
import { useDatabase } from '../useDatabase';
import type { HistoryRecord, DynamicApiResponse, StorageData } from '../model';
import { normalizeToBytes, formatIOBytes, formatBytes } from '../utils/convert';

// 注册 ECharts 组件
use([
  CanvasRenderer,
  LineChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  ToolboxComponent
]);

// Props
const props = defineProps<{
  data: {
    dynamic: DynamicApiResponse,
    static: any,
    connection: any
  };
}>();

const { getHistory } = useDatabase();

// ================= 常量与辅助函数 =================
// 3. Tooltip 格式化
function formatIOTooltip(value: number): string {
  return formatIOBytes(value);
}

// ================= 状态定义 =================

const timeRanges = [
  { label: '1m', value: 60 * 1000 },
  { label: '10m', value: 10 * 60 * 1000 },
  { label: '30m', value: 30 * 60 * 1000 },
  { label: '1h', value: 60 * 60 * 1000 },
  { label: '12h', value: 12 * 60 * 60 * 1000 },
  { label: '1d', value: 24 * 60 * 60 * 1000 },
  { label: '7d', value: 7 * 24 * 60 * 60 * 1000 },
  { label: '30d', value: 30 * 24 * 60 * 60 * 1000 },
];

const defaultRange = timeRanges[0].value;

// 新增全局时间范围控制
const globalTimeRange = ref(defaultRange);

const chartStates = reactive<Record<string, { range: number }>>({
  cpu: { range: globalTimeRange.value },
  cpu_temp: { range: globalTimeRange.value },
  memory: { range: globalTimeRange.value },
  network_in: { range: globalTimeRange.value },
  network_out: { range: globalTimeRange.value },
  storage_io: { range: globalTimeRange.value },
  storage_usage: { range: globalTimeRange.value },
});

// ================= ECharts Option 生成 =================

// 百分比/温度图表 (Y轴固定单位)
function getFixedAxisOption(title: string, color: string, unit: string, min?: number, max?: number): EChartsOption {
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 41, 59, 0.9)',
      textStyle: { color: '#fff' },
      formatter: (params: any) => {
        const param = params[0];

        return `${param.seriesName}<br/>${new Date(param.value[0]).toLocaleString()}<br/>${formatBytes(param.value[1], unit)} ${unit}`;
      }
    },
    grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
    title: { text: title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
    toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
    xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
    yAxis: {
      type: 'value',
      min: min, max: max,
      splitLine: { lineStyle: { color: '#334155', type: 'dashed' } },
      axisLabel: { formatter: `{value} ${unit}` }
    },
    series: [{
      type: 'line',
      name: title,
      showSymbol: false,
      data: [],
      lineStyle: { width: 2, color: color },
      areaStyle: { opacity: 0.1, color: color },
      smooth: false
    }]
  };
}

// IO 类图表 (Y轴自动归一化显示，这里接收的是 Bytes/s)
function getIOOption(title: string, color: string): EChartsOption {
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 41, 59, 0.9)',
      textStyle: { color: '#fff' },
      formatter: (params: any) => {
        const param = params[0];
        // param.value[1] 是已经归一化后的 Bytes/s
        const displayValue = formatIOTooltip(param.value[1]);
        return `${param.seriesName}<br/>${new Date(param.value[0]).toLocaleString()}<br/>${displayValue}`;
      }
    },
    grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
    title: { text: title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
    toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
    xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
    yAxis: {
      type: 'value',
      scale: true, // 启用自动缩放
      splitLine: { lineStyle: { color: '#334155', type: 'dashed' } },
      // Y轴标签使用格式化函数
      axisLabel: { formatter: (value: number) => formatIOBytes(value) }
    },
    series: [{
      type: 'line',
      name: title,
      showSymbol: false,
      data: [],
      lineStyle: { width: 2, color: color },
      areaStyle: { opacity: 0.1, color: color },
      smooth: false
    }]
  };
}

const chartOptions = reactive<Record<string, EChartsOption>>({
  cpu: getFixedAxisOption('CPU 占用', '#3b82f6', '%', 0, 100),
  cpu_temp: getFixedAxisOption('CPU 温度', '#f59e0b', '°C', 0, 120),
  memory: getFixedAxisOption('内存占用', '#8b5cf6', '%', 0, 100),
  network_out: getIOOption('网络上行', '#f97316'),
  network_in: getIOOption('网络下行', '#10b981'),
  storage_io: getIOOption('存储 IO', '#ec4899'),
  storage_usage: getFixedAxisOption('存储占用', '#06b6d4', '%', 0, 100),
});

// ================= 数据加载与处理 =================

function filterDataByTimeRange(data: [number, number][], range: number): [number, number][] {
  const now = Date.now();
  const cutoffTime = now - range;
  return data.filter(([timestamp]) => timestamp >= cutoffTime);
}

const loadHistoryAndRender = async (key: string) => {
  const range = chartStates[key].range;
  // 从 DB 获取原始数据
  const data = await getHistory(key as any, range);

  // 针对图表类型进行归一化
  let seriesData: [number, number][];

  // 判断是否为 IO 类型图表
  const isIO = ['network_in', 'network_out', 'storage_io'].includes(key);

  if (isIO) {
    // IO 图表：假设 DB 里存的可能是任意单位，再次进行归一化 (Bytes/s)
    seriesData = data.map(item => {
      // item.unit 是当时存入的单位
      const normalizedValue = normalizeToBytes(item.value, item.unit);
      return [item.timestamp, normalizedValue] as [number, number];
    });
  } else {
    // 百分比/温度图表：直接用原始值
    seriesData = data.map(item => [item.timestamp, item.value] as [number, number]);
  }

  (chartOptions[key].series as any)[0].data = seriesData;
};

// ================= 数据追加 =================

const appendDataPoint = (key: string, timestamp: number, value: number, unit: string) => {
  const seriesArr = (chartOptions[key].series as { data: [number, number][] })[0].data;

  // 1. 归一化处理：如果是 IO 图表，转为 Bytes/s
  let finalValue = value;
  const isIO = ['network_in', 'network_out', 'storage_io'].includes(key);
  if (isIO) {
    finalValue = normalizeToBytes(value, unit);
    // 强制覆盖 unit，确保后续 DB 存入一致
    unit = 'B/S';
  }

  seriesArr.push([timestamp, finalValue]);

  if (seriesArr.length > 500) {
    seriesArr.shift();
  }

  const filteredData = filterDataByTimeRange(seriesArr, chartStates[key].range);
  (chartOptions[key].series as any)[0].data = filteredData;
};

// ================= 监听数据流 =================

watch(() => props.data.dynamic, (newData) => {
  if (!newData) return;
  const now = Date.now();

  // CPU
  const cpuUsage = newData.cpu?.total?.usage;
  if (cpuUsage?.value !== undefined) appendDataPoint('cpu', now, cpuUsage.value, cpuUsage.unit);

  // CPU Temp
  if (newData.cpu) {
    let totalTemp = 0, count = 0;
    Object.values(newData.cpu).forEach((c: any) => { if (c.temperature.value > 0) { totalTemp += c.temperature.value; count++ } });
    if (count > 0) {
      const unit = Object.values(newData.cpu)[0].temperature.unit;
      appendDataPoint('cpu_temp', now, totalTemp / count, unit);
    }
  }

  // Memory
  const memUsage = newData.memory?.used_percent;
  if (memUsage?.value !== undefined) appendDataPoint('memory', now, memUsage.value, memUsage.unit);

  // Network In
  // 修改：这里假设接口里是具体的网卡，根据你的代码是 pppoe-wan
  const netIn = newData.network?.['pppoe-wan']?.incoming;
  if (netIn?.value !== undefined) {
    // 这里 appendDataPoint 内部会自动处理归一化
    appendDataPoint('network_in', now, netIn.value, netIn.unit);
  }

  // Network Out
  const netOut = newData.network?.['pppoe-wan']?.outgoing;
  if (netOut?.value !== undefined) {
    appendDataPoint('network_out', now, netOut.value, netOut.unit);
  }

  // Storage IO
  if (newData.storage) {
    let totalBytes = 0;
    let unit = 'B/S'; // 默认

    Object.values(newData.storage).forEach((d: StorageData) => {
      // 分别对读和写进行归一化，然后相加
      // 这样可以兼容 read 是 KB，write 是 MB 的极端情况
      const readBytes = normalizeToBytes(d.read.value, d.read.unit);
      const writeBytes = normalizeToBytes(d.write.value, d.write.unit);
      if (readBytes > 0) {
        totalBytes += readBytes;
      }
      if (writeBytes > 0) {
        totalBytes += writeBytes;
      }
    });
    appendDataPoint('storage_io', now, totalBytes, unit);

  }

  // Storage Usage
  const storageKeys = Object.keys(newData.storage || {}).filter(k => k !== 'total');
  if (storageKeys.length > 0) {
    const usage = newData.storage[storageKeys[0]].used_percent;
    appendDataPoint('storage_usage', now, usage.value, usage.unit);
  }
}, { deep: true });

// ================= UI 交互 =================

const handleRangeChange = (key: string) => {
  loadHistoryAndRender(key);
};

// 全局时间范围变化处理
const handleGlobalRangeChange = () => {
  // 更新所有图表的时间范围
  Object.keys(chartStates).forEach(key => {
    chartStates[key].range = globalTimeRange.value;
  });

  // 重新加载所有图表的数据
  Object.keys(chartOptions).forEach(key => loadHistoryAndRender(key));
};

onMounted(async () => {
  await nextTick();
  Object.keys(chartOptions).forEach(k => loadHistoryAndRender(k));
});
</script>

<template>
  <div class="w-full h-full flex flex-col gap-6">
    <div class="py-2.5 border-b border-slate-700 mb-5 flex justify-between items-center">
      <div class="flex items-center gap-4">
        <h3 class="text-lg font-semibold text-slate-200">监控图表</h3>
      </div>

      <!-- 全局时间范围下拉列表 -->
      <div class="flex items-center gap-2">
        <div class="text-slate-400 text-sm text-right ">全局图表时间范围 :</div>
        <div class="relative">
          <select v-model="globalTimeRange" @change="handleGlobalRangeChange"
            class="bg-slate-900 border border-slate-600 text-white text-xs px-2 py-1 rounded outline-none focus:border-blue-500">
            <option v-for="r in timeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div v-for="(opt, key) in chartOptions" :key="key"
        class="bg-slate-800 border border-slate-700 rounded-xl p-4 relative group">
        <select v-model="chartStates[key].range" @change="handleRangeChange(key)"
          class="absolute top-6 right-16 z-10 bg-slate-900 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-100 transition-opacity">
          <option v-for="r in timeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
        </select>
        <v-chart :option="opt" :autoresize="true" style="height: 320px;" />
      </div>
    </div>
  </div>
</template>

<style scoped>
div[ref] {
  width: 100%;
  height: 100%;
}
</style>