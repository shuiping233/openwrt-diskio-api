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
import { LabelLayout, UniversalTransition } from 'echarts/features';
import type { EChartsOption } from 'echarts';
import { useDatabase } from '../useDatabase';
import type { HistoryRecord, DynamicApiResponse, StaticApiResponse, ConnectionApiResponse } from '../model';

// 注册 ECharts 组件 (必须步骤，否则报错)
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
        static: StaticApiResponse,
        connection: ConnectionApiResponse
    };
}>();

const { addHistoryBatch, getHistory } = useDatabase();

// ================= 状态定义 =================

// 时间范围
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

// 1. 每个图表的独立配置 (范围)
const chartStates = reactive<Record<string, { range: number }>>({
    cpu: { range: defaultRange },
    cpu_temp: { range: defaultRange },
    memory: { range: defaultRange },
    network_in: { range: defaultRange },
    network_out: { range: defaultRange },
    storage_io: { range: defaultRange },
    storage_usage: { range: defaultRange },
});

// 2. 图表的 Option (核心)
// 使用 reactive 对象，vue-echarts 会自动响应变化并渲染
const chartOptions = reactive<Record<string, EChartsOption>>({
    cpu: getBaseOption('CPU 占用', '#3b82f6'),
    cpu_temp: getBaseOption('CPU 温度', '#f59e0b'),
    memory: getBaseOption('内存占用', '#8b5cf6'),
    network_in: getBaseOption('网络下行', '#10b981'),
    network_out: getBaseOption('网络上行', '#f97316'),
    storage_io: getBaseOption('存储 IO', '#ec4899'),
    storage_usage: getBaseOption('存储占用', '#06b6d4'),
});

// ================= 辅助函数 =================

function getBaseOption(title: string, color: string): EChartsOption {
    return {
        backgroundColor: 'transparent',
        tooltip: { trigger: 'axis', backgroundColor: 'rgba(30, 41, 59, 0.9)', textStyle: { color: '#fff' } },
        grid: { left: 40, right: 20, bottom: 30, top: 60, containLabel: false },
        title: { text: title, textStyle: { color: '#94a3b8', fontSize: 14 }, left: 'center' },
        toolbox: { show: true, feature: { saveAsImage: { show: true, title: '保存图片' } } },
        xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#64748b' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: '#334155', type: 'dashed' } } },
        series: [{
            type: 'line',
            showSymbol: false,
            data: [], // 初始空数据
            lineStyle: { width: 2, color: color },
            areaStyle: { opacity: 0.1, color: color },
            smooth: true
        }]
    };
}

// 从 DB 加载历史数据，并直接赋值给 Option
const loadHistoryAndRender = async (key: string) => {
    const range = chartStates[key].range;
    const data = await getHistory(key as any, range);

    // ECharts Time Axis 格式: [[t1, v1], [t2, v2]]
    const seriesData = data.map(item => [item.timestamp, item.value]);

    // 获取单位
    const unit = data.length > 0 ? data[0].unit : '%';

    // 直接修改 reactive 属性，vue-echarts 自动更新
    (chartOptions[key].series as any)[0].data = seriesData;
    (chartOptions[key].yAxis as any).axisLabel = { formatter: `{value} ${unit}` };
};

// ================= 核心逻辑：流式更新 =================

// 这个函数直接修改 reactive 数组，不会导致图表重置，只会平滑更新
const appendDataPoint = (key: string, timestamp: number, value: number, unit: string) => {
    const seriesArr = (chartOptions[key].series as any)[0].data as Array<[number, number]>;

    // 1. 推入新数据
    seriesArr.push([timestamp, value]);

    // 2. 内存保护：如果点太多，移除旧点，防止浏览器卡顿
    // 真正的历史数据在 DB 里，图表只展示最近 500 个点足够
    if (seriesArr.length > 500) {
        seriesArr.shift();
    }

    // 3. 异步存入 DB
    addHistoryBatch([{
        timestamp,
        metric: key as any,
        value,
        unit
    }]).catch(console.error);
};

// 监听 App.vue 传来的数据流
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

    // Network
    const netIn = newData.network?.total?.incoming;
    const netOut = newData.network?.total?.outgoing;
    if (netIn?.value !== undefined) appendDataPoint('network_in', now, netIn.value, netIn.unit);
    if (netOut?.value !== undefined) appendDataPoint('network_out', now, netOut.value, netOut.unit);

    // Storage IO
    if (newData.storage) {
        let totalIO = 0, unit = 'KB/s';
        Object.values(newData.storage).forEach((d: any) => {
            if (d.read.value > 0 && d.write.value > 0){
                totalIO += d.read.value + d.write.value;
            }
            unit = d.read.unit;
        });
        appendDataPoint('storage_io', now, totalIO, unit);
    }

    // Storage Usage
    const storageKeys = Object.keys(newData.storage || {}).filter(k => k !== 'total');
    if (storageKeys.length > 0) {
        const usage = newData.storage[storageKeys[0]].used_percent;
        appendDataPoint('storage_usage', now, usage.value, usage.unit);
    }
}, { deep: true });

// 切换时间范围
const handleRangeChange = (key: string) => {
    loadHistoryAndRender(key);
};

// 清空
const clearAllData = async () => {
    if (confirm('确定清空所有历史数据吗？')) {
        const { clearHistory } = useDatabase();
        await clearHistory();
        // 重置图表数据
        Object.keys(chartOptions).forEach(k => {
            (chartOptions[k].series as any)[0].data = [];
        });
    }
};

// ================= 生命周期 =================

onMounted(async () => {
    await nextTick();
    // 初始化加载数据
    Object.keys(chartOptions).forEach(k => loadHistoryAndRender(k));
});
</script>

<template>
    <div class="w-full h-full flex flex-col gap-6">
        <!-- 头部 -->
        <div class="flex justify-between items-center bg-slate-800 p-4 rounded-xl border border-slate-700">
            <h2 class="text-xl font-bold text-slate-200">历史数据监控</h2>
            <button @click="clearAllData"
                class="text-xs bg-red-900/50 text-red-400 px-3 py-1 rounded hover:bg-red-900 transition">
                清空所有数据
            </button>
        </div>

        <!-- 图表网格 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">

            <!-- 循环生成图表 -->
            <div v-for="(opt, key) in chartOptions" :key="key"
                class="bg-slate-800 border border-slate-700 rounded-xl p-4 relative group">
                <!-- 时间选择器 -->
                <select v-model="chartStates[key].range" @change="handleRangeChange(key)"
                    class="absolute top-4 right-4 z-10 bg-slate-900 border border-slate-600 text-xs text-slate-300 px-2 py-1 rounded outline-none opacity-0 group-hover:opacity-100 transition-opacity">
                    <option v-for="r in timeRanges" :key="r.value" :value="r.value">{{ r.label }}</option>
                </select>

                <!-- vue-echarts 组件 -->
                <!-- 注意：设置 height 否则可能显示不全 -->
                <v-chart :option="opt" :autoresize="true" style="height: 320px;" />
            </div>

        </div>
    </div>
</template>

<style scoped>
/* 可选：增加一些过渡动画 */
</style>