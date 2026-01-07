<script setup lang="ts">
import { ref, computed, h, watch } from 'vue';
import {
  useVueTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  FlexRender,
  createColumnHelper,
  SortingState,
  ColumnFiltersState
} from '@tanstack/vue-table';
import type { ConnectionApiResponse } from '../model';
import { compressIPv6 } from '../utils/ipv6';
import { useToast } from '../useToast';

// Props
const props = defineProps<{
  connectionData?: ConnectionApiResponse;
}>();

// 全局搜索词
const globalFilter = ref('');

watch(globalFilter, (newFilter) => {
  table.setGlobalFilter(newFilter);
});

// ================= 1. 数据聚合逻辑 =================
const aggregatedData = computed(() => {
  const list = props.connectionData?.connections || [];
  if (list.length === 0) return [];

  const groups = new Map<string, any>();

  list.forEach(c => {
    const endpointA = `${c.source_ip}:${c.source_port}`;
    const endpointB = `${c.destination_ip}:${c.destination_port}`;
    const endpoints = [endpointA, endpointB].sort();
    const key = `${c.protocol}:${endpoints[0]}<->${endpoints[1]}`;

    if (!groups.has(key)) {
      groups.set(key, {
        ...c,
        _sumTraffic: 0,
        _sumPackets: 0
      });
    }

    const item = groups.get(key);
    if (c.traffic.value < 0) {
      return Array.from(groups.values());
    }
    item._sumTraffic += c.traffic.value;
    item._sumPackets += c.packets;

    item.traffic.value = item._sumTraffic;
    item.packets = item._sumPackets;
  });

  return Array.from(groups.values());
});

// ================= 2. 辅助函数 =================
const formatIP = (ip: string | undefined, family: string | undefined): string => {
  if (!ip) return '-';
  if (family?.toUpperCase() === 'IPV6') {
    return compressIPv6(ip);
  }
  return ip;
};

const formatBytes = (bytes: number): string => {
  return bytes < 0 ? "-1" : bytes.toFixed(2);
};

// 复制功能
const copyInfo = (row: any) => {
  const text = `[${row.ip_family}] ${row.protocol} ${row.source_ip}:${row.source_port} -> ${row.destination_ip}:${row.destination_port} | 状态: ${row.state} | 流量: ${row.traffic.value.toFixed(2)} ${row.traffic.unit} (${row.packets} Pkgs)`;
  navigator.clipboard.writeText(text).then(() => {
    const { success } = useToast();
    success('连接信息已复制！');
  });
};

// ================= 3. TanStack Table 配置 (使用 h 函数代替 JSX 以避免 TS 解析错误) =================
const columnHelper = createColumnHelper<any>();

const columns = [
  // 地址族
  columnHelper.accessor('ip_family', {
    header: '地址族',
    cell: (info) => h('span', { class: 'bg-slate-700 px-2 py-1 rounded text-xs text-slate-200' }, info.getValue()?.toUpperCase()),
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const valA = rowA.original.ip_family || '';
      const valB = rowB.original.ip_family || '';
      return valA.localeCompare(valB);
    },
  }),
  // 协议
  columnHelper.accessor('protocol', {
    header: '协议',
    cell: (info) => h('span', { class: 'bg-slate-700 px-2 py-1 rounded text-xs text-slate-200' }, info.getValue()?.toUpperCase()),
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const valA = rowA.original.protocol || '';
      const valB = rowB.original.protocol || '';
      return valA.localeCompare(valB);
    },
  }),
  // 源地址
  columnHelper.accessor('source_ip', {
    header: '源地址',
    cell: (info) => {
      const row = info.row.original;
      const ip = info.getValue();
      const port = row.source_port;
      return h('span', { class: 'font-mono text-slate-300' }, formatIP(ip, row.ip_family) + (port > 0 ? ':' + port : ''));
    },
    enableSorting: true, // 启用排序
    sortingFn: (rowA, rowB) => {
      const ipA = formatIP(rowA.original.source_ip, rowA.original.ip_family);
      const portA = rowA.original.source_port;
      const ipB = formatIP(rowB.original.source_ip, rowB.original.ip_family);
      const portB = rowB.original.source_port;

      // 首先按IP地址排序
      const ipComparison = ipA.localeCompare(ipB);
      if (ipComparison !== 0) {
        return ipComparison;
      }
      // IP相同时按端口号数值排序
      return portA - portB;
    },
    filterFn: (row, columnId, filterValue) => {
      const ip = row.getValue(columnId);
      const port = row.original.source_port;
      const family = typeof row.original.ip_family === 'string' ? row.original.ip_family : '';
      const fullAddress = `${formatIP(ip as string, family)}:${port}`;
      return fullAddress.toLowerCase().includes(filterValue.toLowerCase());
    },
  }),
  // 目标地址
  columnHelper.accessor('destination_ip', {
    header: '目标地址',
    cell: (info) => {
      const row = info.row.original;
      const ip = info.getValue();
      const port = row.destination_port;
      return h('span', { class: 'font-mono text-slate-300' }, formatIP(ip, row.ip_family) + (port > 0 ? ':' + port : ''));
    },
    enableSorting: true, // 启用排序
    sortingFn: (rowA, rowB) => {
      const ipA = formatIP(rowA.original.destination_ip, rowA.original.ip_family);
      const portA = rowA.original.destination_port;
      const ipB = formatIP(rowB.original.destination_ip, rowB.original.ip_family);
      const portB = rowB.original.destination_port;

      // 首先按IP地址排序
      const ipComparison = ipA.localeCompare(ipB);
      if (ipComparison !== 0) {
        return ipComparison;
      }
      // IP相同时按端口号数值排序
      return portA - portB;
    },
    filterFn: (row, columnId, filterValue) => {
      const ip = row.getValue(columnId);
      const port = row.original.destination_port;
      const family = typeof row.original.ip_family === 'string' ? row.original.ip_family : '';
      const fullAddress = `${formatIP(ip as string, family)}:${port}`;
      return fullAddress.toLowerCase().includes(filterValue.toLowerCase());
    },
  }),
  // 状态
  columnHelper.accessor('state', {
    header: '状态',
    cell: (info) => h('span', { class: 'text-slate-300' }, info.getValue() || '-'),
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const valA = rowA.original.state || '';
      const valB = rowB.original.state || '';
      return valA.localeCompare(valB);
    },
  }),
  // 传输情况
  columnHelper.accessor('traffic', {
    header: '传输情况',
    cell: (info) => {
      const row = info.row.original;
      return h('span', { class: 'text-slate-300' }, formatBytes(row.traffic.value) + ' ' + row.traffic.unit + ' (' + row.packets + ' Pkgs.)');
    },
    sortingFn: (rowA, rowB) => {
      const valA = rowA.original.traffic.value || 0;
      const valB = rowB.original.traffic.value || 0;
      return valA - valB;
    },
    enableSorting: true,
    filterFn: (row, columnId, filterValue) => {
      const trafficValue = row.original.traffic.value || 0;
      const trafficUnit = row.original.traffic.unit || '';
      const packets = row.original.packets || 0;

      // 格式化后的值
      const formattedValue = formatBytes(trafficValue);
      const fullDisplayValue = `${formattedValue} ${trafficUnit} (${packets} Pkgs.)`;

      // 转换为小写进行比较
      const lowerFilterValue = filterValue.toLowerCase();
      const lowerDisplayValue = fullDisplayValue.toLowerCase();

      // 检查是否包含过滤值（支持数字和单位的搜索，忽略空格）
      if (lowerDisplayValue.replace(/\s+/g, '').includes(lowerFilterValue.replace(/\s+/g, ''))) {
        return true;
      }

      // 检查是否包含过滤值（支持数字和单位的搜索，保留空格）
      if (lowerDisplayValue.includes(lowerFilterValue)) {
        return true;
      }

      // 检查数值部分
      if (String(trafficValue).toLowerCase().includes(lowerFilterValue)) {
        return true;
      }

      // 检查单位部分
      if (trafficUnit.toLowerCase().includes(lowerFilterValue)) {
        return true;
      }

      // 检查包数
      if (String(packets).toLowerCase().includes(lowerFilterValue)) {
        return true;
      }

      // 检查不带空格的组合
      const noSpaceValue = `${formattedValue}${trafficUnit}(${packets}Pkgs.)`.toLowerCase();
      if (noSpaceValue.includes(lowerFilterValue.replace(/\s+/g, ''))) {
        return true;
      }

      return false;
    },
  }),
  // 操作列
  columnHelper.display({
    id: 'actions',
    header: '操作',
    cell: ({ row }) => h('button', {
      onClick: () => copyInfo(row.original),
      class: 'text-xs bg-slate-700 hover:bg-blue-600 text-white px-2 py-1 rounded transition-colors',
      title: '复制连接信息'
    }, '复制'),
    enableSorting: false, // 禁用排序
  }),
];

// 初始状态
const initialSorting = [{ id: 'traffic', desc: true }];

const table = useVueTable({
  data: aggregatedData,
  columns,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getFilteredRowModel: getFilteredRowModel(),
  getRowId: (row) => `${row.source_ip}-${row.source_port}-${row.destination_ip}-${row.destination_port}-${row.protocol}`, // 使用连接的唯一标识
  initialState: {
    sorting: initialSorting,
    columnFilters: [],
    globalFilter: globalFilter.value,
  },
  globalFilterFn: (row, columnId, value) => {
    const search = String(value).toLowerCase();
    const rowStr = Object.values(row.original).join(' ').toLowerCase();
    return rowStr.includes(search);
  },
});
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

    <!-- Table Header & Search (移除折叠功能) -->
    <div class="py-2.5 border-b border-slate-700 mb-5 flex justify-between items-center">
      <div class="flex items-center gap-4">
        <h3 class="text-lg font-semibold text-slate-200">连接列表</h3>
      </div>
      <!-- 全局搜索框 -->
      <div class="flex items-center gap-4">
        <div class="relative">
          <input v-model="globalFilter" placeholder="全局搜索..."
            class="bg-slate-900 border border-slate-600 text-white text-xs px-2 py-1 rounded w-48 outline-none focus:border-blue-500" />
        </div>
      </div>
    </div>

    <!-- Table Content (移除 v-show，始终显示) -->
    <div class="bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-center border-collapse">
          <thead class="bg-slate-700/50 text-slate-300">
            <tr>
              <th v-for="column in table.getHeaderGroups()[0].headers" :key="column.id"
                class="px-3 py-3 font-medium text-center whitespace-nowrap">
                <div class="flex flex-col gap-1 items-center">
                  <div class="flex items-center gap-1 cursor-pointer select-none hover:text-white" @click="() => {
                    if (column.column.getCanSort()) {
                      column.column.toggleSorting(undefined, column.column.getIsSorted() === false)
                    }
                  }">
                    <FlexRender :render="column.column.columnDef.header" :props="column.getContext()" />
                    {{ { asc: '↑', desc: '↓' }[column.column.getIsSorted() as string] || '' }}
                  </div>

                  <!-- 列过滤器 -->
                  <input v-if="column.column.getCanFilter()" :value="column.column.getFilterValue() ?? ''"
                    @input="e => column.column.setFilterValue((e.target as HTMLInputElement).value)"
                    :placeholder="`过滤 ${column.column.columnDef.header as string}...`"
                    class="bg-slate-900 border border-slate-600 text-xs px-1 py-0.5 rounded w-24 text-slate-200 outline-none" />
                </div>
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700">
            <tr v-for="row in table.getRowModel().rows" :key="row.id" class="hover:bg-slate-700/30 transition-colors">
              <td v-for="cell in row.getVisibleCells()" :key="cell.id" class="px-3 py-2 text-center">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
            <tr v-if="table.getRowModel().rows.length === 0">
              <td colspan="7" class="px-5 py-8 text-center text-slate-500">暂无匹配数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>