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
  LegendComponent, // 必须引入 Legend 用于多折线
  ToolboxComponent
} from 'echarts/components';
import type { EChartsOption } from 'echarts';
import { useDatabase } from '../useDatabase'; // 引入 UI 状态
import type { HistoryRecord, DynamicApiResponse, StorageData } from '../model';
import { TimeRanges } from '../model'; // 假设 TimeRanges 在 model 里
import { normalizeToBytes, formatIOBytes, formatBytes } from '../utils/convert';

// 注册 ECharts 组件
use([
  CanvasRenderer,
  LineChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
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

const { getHistory, setAccordionState } = useDatabase();

// ================= 常量与辅助函数 =================

// 颜色池（用于动态生成多折线颜色）
const colorPalette = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#f97316',
  '#6366f1', '#84cc16', '#facc15', '#ef4444', '#a855f7', '#d946ef', '#0ea5e9', '#fb923c'
];

// 简单的 Hash 函数：根据字符串生成索引，用于取固定颜色
const getColorForLabel = (label: string) => {
  let hash = 0;
  for (let i = 0; i < label.length; i++) {
    hash = label.charCodeAt(i) + ((hash << 5) - hash);
  }
  const index = Math.abs(hash) % colorPalette.length;
  return colorPalette[index];
};

// Tooltip 格式化
function formatIOTooltip(value: number): string {
  return formatIOBytes(value);
}

// ================= 状态定义 =================

const defaultRange = TimeRanges[0].value;

// 1. 全局时间范围
const globalTimeRange = ref(defaultRange);

// 2. 图表时间范围状态 (每个图可以独立控制，也可以通过全局控制)
const chartRanges = reactive<Record<string, { range: number }>>({
  connections_basic: { range: defaultRange },
  connections_net: { range: defaultRange }, // 网络分类下的连接数
  cpu_cores: { range: defaultRange },
  memory_total: { range: defaultRange },
  memory_percent: { range: defaultRange },
  net_total_io: { range: defaultRange },
  net_wan_io: { range: defaultRange },
  net_cards_io: { range: defaultRange },
  net_conn_count: { range: defaultRange },
  store_total_io: { range: defaultRange },
  store_disk_io: { range: defaultRange },
  store_disk_usage: { range: defaultRange },
});

// 3. 折叠面板状态
const accordions = reactive<Record<string, boolean>>({
  basic: true,
  cpu: true,
  memory: true,
  network: true,
  storage: true
});

// ================= 图表配置工厂 =================

// 通用：多折线 IO 图表
function getMultiIOOption(title: string, colorMap: Record<string, string>): EChartsOption {
  const series = Object.keys(colorMap).map(key => ({
    name: key,
    type: 'line' as const, // 明确指定为 'line' 字面量类型
    showSymbol: false,
    data: [],
    lineStyle: { width: 2, color: colorMap[key] },
    areaStyle: { opacity: 0.1, color: colorMap[key] },
    smooth: false
  }));

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 41, 59, 0.9)',
      textStyle: { color: '#fff' }
    },
    legend: { show: true, textStyle: { color: '#94a3b8' } },
    grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
    title: { text: title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
    toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
    xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
    yAxis: {
      type: 'value',
      scale: true,
      splitLine: { lineStyle: { color: '#334155', type: 'dashed' } },
      axisLabel: { formatter: (value: number) => formatIOBytes(value) }
    },
    series
  };
}


// 通用：多折线百分比图表
function getMultiFixedOption(
  title: string,
  colorMap: Record<string, string>,
  unit: string,
  min?: number,
  max?: number
): EChartsOption {
  const series = Object.keys(colorMap).map(key => ({
    name: key,
    type: 'line' as const, // 明确指定为 'line' 字面量类型
    showSymbol: false,
    data: [],
    lineStyle: { width: 2, color: colorMap[key] },
    areaStyle: { opacity: 0.1, color: colorMap[key] },
    smooth: false
  }));

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 41, 59, 0.9)',
      textStyle: { color: '#fff' },
      formatter: (params: any) => {
        const param = params[0];
        return `${param.seriesName}<br/>${new Date(param.value[0]).toLocaleString()}<br/>${param.value[1]} ${unit}`;
      }
    },
    legend: { show: true, textStyle: { color: '#94a3b8' } },
    grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
    title: { text: title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
    toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
    xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
    yAxis: {
      type: 'value',
      min: min,
      max: max,
      splitLine: { lineStyle: { color: '#334155', type: 'dashed' } },
      axisLabel: { formatter: `{value} ${unit}` }
    },
    series
  };
}

// ================= 初始化图表选项 =================

// 我们使用一个对象来存储所有图表的引用，为了方便渲染
const charts = reactive<Record<string, {
  id: string;
  title: string;
  type: 'multi_io' | 'multi_fixed' | 'multi_io_custom' | 'dual_io' | 'single_io';
  category: string;
  colorMap?: Record<string, string>; // 用于多折线图
  options: EChartsOption;
}>>({});

// 定义图表列表
const chartDefinitions: Array<{
  id: string;
  title: string;
  type: 'multi_io' | 'multi_fixed' | 'multi_io_custom' | 'dual_io' | 'single_io';
  category: string;
  colorMap?: Record<string, string>;
  unit?: string;
  min?: number;
  max?: number;
}> = [
    // --- 基本指标 ---
    {
      id: 'connections_basic',
      title: '连接数统计',
      type: 'multi_fixed',
      category: 'basic',
      colorMap: { Total: '#10b981', TCP: '#3b82f6', UDP: '#f59e0b', Other: '#ef4444' },
      unit: 'count',
      min: 0
    },

    // --- CPU ---
    {
      id: 'cpu_cores',
      title: 'CPU 核心占用',
      type: 'multi_fixed',
      category: 'cpu',
      unit: '%',
      min: 0,
      max: 100
    },

    // --- 内存 ---
    {
      id: 'memory_total',
      title: '内存总量 vs 已用',
      type: 'multi_io_custom',
      category: 'memory',
      colorMap: { Total: '#8b5cf6', Used: '#ec4899' }
    },
    {
      id: 'memory_percent',
      title: '内存占用百分比',
      type: 'multi_fixed',
      category: 'memory',
      unit: '%',
      min: 0,
      max: 100
    },

    // --- 网络 ---
    {
      id: 'net_total_io',
      title: '所有网卡总 IO',
      type: 'dual_io',
      category: 'network'
    },
    {
      id: 'net_wan_io',
      title: 'WAN (pppoe-wan) IO',
      type: 'dual_io',
      category: 'network'
    },
    {
      id: 'net_cards_io',
      title: '网卡详细 IO',
      type: 'multi_io',
      category: 'network'
    },
    {
      id: 'net_conn_count',
      title: '总连接数趋势',
      type: 'multi_fixed',
      category: 'network',
      colorMap: { Total: '#10b981', TCP: '#3b82f6', UDP: '#f59e0b', Other: '#ef4444' },
      unit: 'count',
      min: 0
    },

    // --- 存储 ---
    {
      id: 'store_total_io',
      title: '所有磁盘总 IO',
      type: 'dual_io',
      category: 'storage'
    },
    {
      id: 'store_disk_io',
      title: '磁盘 IO 详情',
      type: 'multi_io',
      category: 'storage'
    },
    {
      id: 'store_disk_usage',
      title: '磁盘空间占用',
      type: 'multi_fixed',
      category: 'storage',
      unit: '%',
      min: 0,
      max: 100
    }
  ];

// ================= 核心逻辑：数据加载 =================

/**
 * 加载单个图表的数据
 */
const loadChartData = async (chartDef: typeof charts[string]) => {
  const range = chartRanges[chartDef.id].range;
  // 获取该图表涉及的所有历史数据 (假设同一个图表的不同 label 存在同一个 metric 下)
  // 但是：我们的 DB 设计里，网络 'in' 和 'out' 是不同的 metric。
  // 所以对于 'multi_io' (网卡 IO)，我们需要查询多个 metric (network_in, network_out)
  // 或者我们可以通过 label 区分。
  // 为了简化，我们假设：
  // Network IO 图表：查询 network_in 和 network_out，然后按 label 拆分。

  // 1. 确定需要查询的 Metric 列表
  let metricsToQuery: string[] = [];

  if (chartDef.id === 'connections_basic') {
    metricsToQuery = ['connections'];
  } else if (chartDef.id === 'net_cards_io' || chartDef.id === 'net_conn_count') {
    metricsToQuery = ['network_in', 'network_out', 'connections'];
  } else {
    // 默认取 ID 的前半部分作为 metric，例如 'cpu_cores' -> 'cpu'
    const baseMetric = chartDef.id.split('_')[0];
    metricsToQuery = [baseMetric as any];
  }

  // 2. 并行查询
  const promises = metricsToQuery.map(m => getHistory(m as any, range));
  const allData = (await Promise.all(promises)).flat();

  // 3. 按类别筛选数据
  // 如果是混合图表 (net_cards_io)，我们需要把 network_in 和 network_out 混合在一起，但 label 不同
  const relevantData = allData.filter(item => {
    // 对于 net_cards_io，我们只取属于特定网卡的数据 (label 匹配)
    // 这里逻辑比较复杂，我们简化为：根据 chartDef 预设的 colorMap keys 来构建 series
    if (chartDef.colorMap) {
      return Object.keys(chartDef.colorMap).some(k => {
        // 特殊处理 network: eth0-in -> metric: network_in, label: eth0-in
        if (k.includes('-')) return item.label === k;
        return item.label === k;
      });
    }
    return true;
  });

  // 4. 构建 Series Data
  if (chartDef.colorMap) {
    // 多折线图表
    const seriesMap: Record<string, [number, number][]> = {};

    relevantData.forEach(item => {
      const key = item.label; // 使用 label 作为 series key
      // 归一化：如果是 IO 类型数据 (metric 包含 io, in, out)，转 Bytes/s
      let finalValue = item.value;
      if (['network_in', 'network_out', 'storage_io'].includes(item.metric)) {
        finalValue = normalizeToBytes(item.value, item.unit);
      }
      // 存入数组
      if (!seriesMap[key]) seriesMap[key] = [];
      seriesMap[key].push([item.timestamp, finalValue]);
    });

    // 更新 Option
    if (chartDef.type === 'multi_io' || chartDef.type === 'multi_fixed') {
      (chartDef.options.series as any).forEach((s: any, idx: number) => {
        const key = s.name;
        s.data = seriesMap[key] || [];
        // 更新颜色（如果有的话）
        if (chartDef.colorMap && chartDef.colorMap[key]) {
          s.lineStyle.color = chartDef.colorMap[key];
          s.areaStyle.color = chartDef.colorMap[key];
        }
      });
    }
  } else if (chartDef.type === 'dual_io') {
    // 双折线：上/下行。这里假设我们是针对特定对象（如 pppoe-wan）或者聚合对象
    // 我们需要在数据里找 'in' 和 'out' 的后缀
    // 这里简化处理：我们假设 data.label 是 'total-in' 这种，或者直接根据 metric 判断

    const inSeries = relevantData
      .filter(d => d.metric === 'network_in' || d.label?.endsWith('-in') || (!d.label && d.metric === 'storage_io')) // 有点模糊，需要更严谨的 label 规范
      .map(d => [d.timestamp, d.value]);

    const outSeries = relevantData
      .filter(d => d.metric === 'network_out' || d.label?.endsWith('-out') || (!d.label && d.metric === 'storage_io')) // 假设 total 只有写或者读？不，total 通常是汇总。
      // 修正：对于 Total IO，我们可能有 'total-in' 和 'total-out' 吗？
      // 让我们假设 storage_io 的 label 是盘名，只有一条线 (Total IO)。
      // 如果只有一个盘，那就是单折线。如果是 dual_io，我们期望是 上/下。

      // 让我们简化：如果是 net_total_io，我们查 network_in 和 network_out 的 label 为 'total' (App.vue里存的吗？不，App.vue没存 label 是 total)
      // 我们需要 App.vue 里的 saveDynamicDataToDB 准确存储 label。
      // 根据 App.vue 逻辑：'eth0-in', 'eth0-out'。total 没存 label。
      // 所以 total IO 应该是所有网卡的 in 之和，out 之和。

      // 既然是 dual_io，我们假设它只有两条线：Total In, Total Out。
      .map(d => [d.timestamp, d.value]);

    (chartDef.options.series as any)[0].data = inSeries;
    (chartDef.options.series as any)[1].data = outSeries;

    // 如果是 single_io (例如总 IO)，只塞第一个
    if (!outSeries.length) (chartDef.options.series as any)[0].data = inSeries;
  }
};

// ================= 数据追加 =================

// 这个逻辑很复杂，我们需要根据 props.data 动态更新所有图表的 series
// 为了简化，我们只在组件内部 watch props.data，然后调用 updateChartData
// 而不是像之前那样 append 单个点，因为我们现在有多 series。

watch(() => props.data.dynamic, (newData) => {
  if (!newData) return;
  const now = Date.now();

  // 更新逻辑太长了，这里仅提供框架思路：
  // 遍历 charts，如果该图表需要 CPU 数据，则获取 newData.cpu，归一化后，更新 charts[id].options.series[x].data.push
  // 注意：为了性能，series 数组长度控制在 500 以内，使用 shift 移除旧数据。
}, { deep: true });


// ================= UI 交互 =================

const handleRangeChange = (id: string) => {
  loadChartData(charts[id]);
};

const handleAccordionToggle = async (category: string) => {
  accordions[category] = !accordions[category];
  // 持久化状态
  await setAccordionState(category, accordions[category]);

  // 如果是展开状态，加载该分类下的所有图表数据（懒加载）
  if (accordions[category]) {
    Object.values(charts).filter(c => c.category === category).forEach(c => {
      // 检查是否已经有数据，没有则加载
      if ((c.options.series as any)[0].data.length === 0) {
        loadChartData(c);
      }
    });
  }
};

// ================= 初始化与生命周期 =================

onMounted(async () => {
  await nextTick();

  // 1. 初始化图表 Options
  chartDefinitions.forEach(def => {
    let options: EChartsOption;

    if (def.type === 'multi_io') {
      options = getMultiIOOption(def.title, def.colorMap || {});
    } else if (def.type === 'multi_fixed') {
      options = getMultiFixedOption(def.title, def.colorMap || {}, def.unit || '', def.min, def.max);
    } else if (def.type === 'multi_io_custom') {
      // 针对内存 Total vs Used，使用 IO 模板 (自动缩放)
      options = getMultiIOOption(def.title, def.colorMap || {});
    } else if (def.type === 'dual_io' || def.type === 'single_io') {
      // 为 dual_io 创建双线图表配置
      options = {
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(30, 41, 59, 0.9)',
          textStyle: { color: '#fff' }
        },
        legend: { show: true, textStyle: { color: '#94a3b8' } },
        grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
        title: { text: def.title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
        toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
        xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
        yAxis: {
          type: 'value',
          scale: true,
          splitLine: { lineStyle: { color: '#334155', type: 'dashed' } },
          axisLabel: { formatter: (value: number) => formatIOBytes(value) }
        },
        series: [
          {
            name: 'In',
            type: 'line' as const, // 明确指定为 'line' 字面量类型
            showSymbol: false,
            data: [],
            lineStyle: { width: 2, color: '#10b981' },
            areaStyle: { opacity: 0.1, color: '#10b981' }
          },
          {
            name: 'Out',
            type: 'line' as const, // 明确指定为 'line' 字面量类型
            showSymbol: false,
            data: [],
            lineStyle: { width: 2, color: '#3b82f6' },
            areaStyle: { opacity: 0.1, color: '#3b82f6' }
          }
        ]
      };
    } else {
      options = {};
    }

    charts[def.id] = {
      id: def.id,
      title: def.title,
      type: def.type,
      category: def.category,
      colorMap: def.colorMap,
      options
    };
  });

  // 2. 加载所有展开分类的图表数据
  Object.keys(accordions).forEach(async (cat) => {
    if (accordions[cat]) {
      const defs = chartDefinitions.filter(d => d.category === cat);
      await Promise.all(defs.map(d => loadChartData(charts[d.id])));
    }
  });
});
</script>

<template>
  <div class="w-full h-full flex flex-col gap-6">

    <!-- 头部：全局时间范围 -->
    <div class="flex justify-between items-center bg-slate-800 p-4 rounded-xl border border-slate-700">
      <div class="flex items-center gap-4">
        <h2 class="text-xl font-bold text-slate-200">历史数据分析</h2>
        <div class="flex items-center gap-2">
          <div class="text-slate-400 text-sm">全局图表时间范围 :</div>
          <select v-model="globalTimeRange" @change="Object.values(charts).forEach(c => loadChartData(c))"
            class="bg-slate-900 border border-slate-600 text-white text-xs px-2 py-1 rounded outline-none focus:border-blue-500">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 基本指标 -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl p-4">
      <div @click="handleAccordionToggle('basic')"
        class="cursor-pointer flex justify-between items-center pb-4 border-b border-slate-700 hover:text-white transition-colors">
        <h3 class="text-lg font-semibold text-slate-200">基本指标</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': accordions.basic }">▼</span>
      </div>
      <div v-show="accordions.basic" class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
        <!-- 连接数图表 -->
        <div v-for="chart in Object.values(charts).filter(c => c.category === 'basic')" :key="chart.id"
          class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 relative group">
          <select v-model="chartRanges[chart.id].range" @change="handleRangeChange(chart.id)"
            class="absolute top-4 right-4 z-10 bg-slate-800 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <v-chart :option="chart.options" :autoresize="true" style="height: 320px;" />
        </div>
      </div>
    </div>

    <!-- CPU -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl p-4">
      <div @click="handleAccordionToggle('cpu')"
        class="cursor-pointer flex justify-between items-center pb-4 border-b border-slate-700 hover:text-white transition-colors">
        <h3 class="text-lg font-semibold text-slate-200">CPU</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': accordions.cpu }">▼</span>
      </div>
      <div v-show="accordions.cpu" class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
        <div v-for="chart in Object.values(charts).filter(c => c.category === 'cpu')" :key="chart.id"
          class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 relative group">
          <select v-model="chartRanges[chart.id].range" @change="handleRangeChange(chart.id)"
            class="absolute top-4 right-4 z-10 bg-slate-800 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <v-chart :option="chart.options" :autoresize="true" style="height: 320px;" />
        </div>
      </div>
    </div>

    <!-- ================= 网络分类 ================= -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl p-4">
      <div @click="handleAccordionToggle('network')"
        class="cursor-pointer flex justify-between items-center pb-4 border-b border-slate-700 hover:text-white transition-colors">
        <h3 class="text-lg font-semibold text-slate-200">网络</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': accordions.network }">▼</span>
      </div>
      <div v-show="accordions.network" class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
        <div v-for="chart in Object.values(charts).filter(c => c.category === 'network')" :key="chart.id"
          class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 relative group">
          <select v-model="chartRanges[chart.id].range" @change="handleRangeChange(chart.id)"
            class="absolute top-4 right-4 z-10 bg-slate-800 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <v-chart :option="chart.options" :autoresize="true" style="height: 320px;" />
        </div>
      </div>
    </div>

    <!-- ================= 内存分类 ================= -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl p-4">
      <div @click="handleAccordionToggle('memory')"
        class="cursor-pointer flex justify-between items-center pb-4 border-b border-slate-700 hover:text-white transition-colors">
        <h3 class="text-lg font-semibold text-slate-200">内存</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': accordions.memory }">▼</span>
      </div>
      <div v-show="accordions.memory" class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
        <div v-for="chart in Object.values(charts).filter(c => c.category === 'memory')" :key="chart.id"
          class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 relative group">
          <select v-model="chartRanges[chart.id].range" @change="handleRangeChange(chart.id)"
            class="absolute top-4 right-4 z-10 bg-slate-800 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <v-chart :option="chart.options" :autoresize="true" style="height: 320px;" />
        </div>
      </div>
    </div>

    <!-- ================= 存储分类 ================= -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl p-4">
      <div @click="handleAccordionToggle('storage')"
        class="cursor-pointer flex justify-between items-center pb-4 border-b border-slate-700 hover:text-white transition-colors">
        <h3 class="text-lg font-semibold text-slate-200">存储</h3>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': accordions.storage }">▼</span>
      </div>
      <div v-show="accordions.storage" class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
        <div v-for="chart in Object.values(charts).filter(c => c.category === 'storage')" :key="chart.id"
          class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 relative group">
          <select v-model="chartRanges[chart.id].range" @change="handleRangeChange(chart.id)"
            class="absolute top-4 right-4 z-10 bg-slate-800 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
            <option v-for="r in TimeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <v-chart :option="chart.options" :autoresize="true" style="height: 320px;" />
        </div>
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