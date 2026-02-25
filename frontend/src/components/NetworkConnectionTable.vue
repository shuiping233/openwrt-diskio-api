<script setup lang="ts">
import { ref, computed, h, watch, reactive } from 'vue';
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
import type { ConnectionApiResponse, Connection } from '../model';
import { compressIPv6 } from '../utils/ipv6';
import { convertToBytes, BytesFixed, formatIOBytes, normalizeToBytes } from '../utils/convert';
import { useToast } from '../useToast';

// Props
const props = defineProps<{
  connectionData?: ConnectionApiResponse;
}>();

// ================= 1. 折叠状态管理 =================
const uiState = reactive({
  accordions: {
    aggregation: true,  // 聚合统计折叠状态
    connectionList: true,  // 连接列表折叠状态
  },
  ipGroupCollapsed: {
    lan: false,  // 局域网IP组折叠状态
    other: false,  // 其他IP组折叠状态
  }
});

// ================= 2. 全局搜索词 =================
const globalFilter = ref('');
const aggregationFilter = ref(''); // 聚合统计的搜索词

watch(globalFilter, (newFilter) => {
  table.setGlobalFilter(newFilter);
});

// ================= 3. 聚合统计排序状态 =================
type SortDirection = 'asc' | 'desc' | null;
type SortColumn = 'ip' | 'traffic' | 'upload' | 'download' | 'tcp' | 'udp' | 'other';

const aggregationSort = reactive<{
  column: SortColumn;
  direction: SortDirection;
}>({
  column: 'traffic',
  direction: 'desc'
});

const toggleAggregationSort = (column: SortColumn) => {
  if (aggregationSort.column === column) {
    // 切换方向
    if (aggregationSort.direction === 'desc') {
      aggregationSort.direction = 'asc';
    } else if (aggregationSort.direction === 'asc') {
      aggregationSort.direction = null;
    } else {
      aggregationSort.direction = 'desc';
    }
  } else {
    // 新列，默认降序
    aggregationSort.column = column;
    aggregationSort.direction = 'desc';
  }
};

const getSortIcon = (column: SortColumn): string => {
  if (aggregationSort.column !== column) return '';
  if (aggregationSort.direction === 'asc') return '↑';
  if (aggregationSort.direction === 'desc') return '↓';
  return '';
};

// ================= 4. 判断IP是否为局域网IP =================
const isLanIP = (ip: string): boolean => {
  // IPv4 局域网地址范围
  const lanPatterns = [
    /^10\./,                                      // 10.0.0.0/8
    /^172\.(1[6-9]|2[0-9]|3[01])\./,             // 172.16.0.0/12
    /^192\.168\./,                               // 192.168.0.0/16
    /^127\./,                                     // 127.0.0.0/8 (loopback)
    /^169\.254\./,                               // 169.254.0.0/16 (link-local)
    /^0\./,                                       // 0.0.0.0/8
  ];
  
  // IPv6 局域网地址
  const ipv6LanPatterns = [
    /^::1$/,                                      // loopback
    /^fc00:/i,                                    // unique local
    /^fe80:/i,                                    // link-local
    /^fd00:/i,                                    // unique local
  ];
  
  for (const pattern of lanPatterns) {
    if (pattern.test(ip)) return true;
  }
  
  for (const pattern of ipv6LanPatterns) {
    if (pattern.test(ip)) return true;
  }
  
  return false;
};

// ================= 5. 处理连接数据（去重） =================
const displayData = computed(() => {
  const list = props.connectionData?.connections || [];
  if (list.length === 0) return [];

  // 使用 Set 来跟踪已见过的连接标识符，防止重复
  const seen = new Set();
  return list.filter(connection => {
    const endpointA = `${connection.source_ip}:${connection.source_port}`;
    const endpointB = `${connection.destination_ip}:${connection.destination_port}`;
    // 对端点进行排序以处理双向连接
    const endpoints = [endpointA, endpointB].sort();
    const key = `${endpoints[0]}<->${endpoints[1]}-${connection.protocol}`;

    if (seen.has(key)) {
      return false; // 过滤掉重复项
    }
    seen.add(key);
    return true;
  });
});

// ================= 6. 聚合统计数据计算 =================
interface IPStats {
  ip: string;
  trafficBytes: number;  // 总流量（字节）
  uploadBytes: number;   // 上行流量（字节）
  downloadBytes: number; // 下行流量（字节）
  tcpCount: number;
  udpCount: number;
  otherCount: number;
  connections: Connection[];
}

interface GroupStats {
  name: string;
  ips: IPStats[];
  totalTraffic: number;
  totalUpload: number;
  totalDownload: number;
  totalTcp: number;
  totalUdp: number;
  totalOther: number;
}

// 排序函数
const sortIPStats = (ips: IPStats[], column: SortColumn, direction: SortDirection): IPStats[] => {
  if (!direction) return ips;
  
  const sorted = [...ips];
  const multiplier = direction === 'desc' ? -1 : 1;
  
  sorted.sort((a, b) => {
    let comparison = 0;
    switch (column) {
      case 'ip':
        comparison = a.ip.localeCompare(b.ip);
        break;
      case 'traffic':
        comparison = a.trafficBytes - b.trafficBytes;
        break;
      case 'upload':
        comparison = a.uploadBytes - b.uploadBytes;
        break;
      case 'download':
        comparison = a.downloadBytes - b.downloadBytes;
        break;
      case 'tcp':
        comparison = a.tcpCount - b.tcpCount;
        break;
      case 'udp':
        comparison = a.udpCount - b.udpCount;
        break;
      case 'other':
        comparison = a.otherCount - b.otherCount;
        break;
    }
    return comparison * multiplier;
  });
  
  return sorted;
};

// 过滤函数
const filterIPStats = (ips: IPStats[], filter: string): IPStats[] => {
  if (!filter.trim()) return ips;
  const lowerFilter = filter.toLowerCase();
  return ips.filter(ip => {
    return ip.ip.toLowerCase().includes(lowerFilter) ||
           formatTraffic(ip.trafficBytes).toLowerCase().includes(lowerFilter) ||
           formatTraffic(ip.uploadBytes).toLowerCase().includes(lowerFilter) ||
           formatTraffic(ip.downloadBytes).toLowerCase().includes(lowerFilter) ||
           String(ip.tcpCount).includes(lowerFilter) ||
           String(ip.udpCount).includes(lowerFilter) ||
           String(ip.otherCount).includes(lowerFilter);
  });
};

const aggregationData = computed((): { lan: GroupStats; other: GroupStats } => {
  const connections = displayData.value;
  const ipMap = new Map<string, IPStats>();
  
  // 统计每个IP的数据
  connections.forEach(conn => {
    // 处理源IP
    const processIP = (ip: string, isSource: boolean) => {
      if (!ipMap.has(ip)) {
        ipMap.set(ip, {
          ip,
          trafficBytes: 0,
          uploadBytes: 0,
          downloadBytes: 0,
          tcpCount: 0,
          udpCount: 0,
          otherCount: 0,
          connections: []
        });
      }
      
      const stats = ipMap.get(ip)!;
      const trafficBytes = normalizeToBytes(conn.traffic.value, conn.traffic.unit);
      stats.trafficBytes += trafficBytes;
      stats.connections.push(conn);
      
      // 简化的上下行判断：假设源IP是上行，目标IP是下行
      if (isSource) {
        stats.uploadBytes += trafficBytes;
      } else {
        stats.downloadBytes += trafficBytes;
      }
      
      // 协议计数
      const protocol = conn.protocol.toUpperCase();
      if (protocol === 'TCP') {
        stats.tcpCount++;
      } else if (protocol === 'UDP') {
        stats.udpCount++;
      } else {
        stats.otherCount++;
      }
    };
    
    processIP(conn.source_ip, true);
    processIP(conn.destination_ip, false);
  });
  
  // 分组为局域网和其他
  let lanIPs: IPStats[] = [];
  let otherIPs: IPStats[] = [];
  
  ipMap.forEach((stats, ip) => {
    if (isLanIP(ip)) {
      lanIPs.push(stats);
    } else {
      otherIPs.push(stats);
    }
  });
  
  // 应用搜索过滤
  lanIPs = filterIPStats(lanIPs, aggregationFilter.value);
  otherIPs = filterIPStats(otherIPs, aggregationFilter.value);
  
  // 应用排序
  lanIPs = sortIPStats(lanIPs, aggregationSort.column, aggregationSort.direction);
  otherIPs = sortIPStats(otherIPs, aggregationSort.column, aggregationSort.direction);
  
  // 计算分组汇总（基于过滤后的数据）
  const calculateGroupTotal = (ips: IPStats[]): GroupStats => {
    return {
      name: '',
      ips,
      totalTraffic: ips.reduce((sum, ip) => sum + ip.trafficBytes, 0),
      totalUpload: ips.reduce((sum, ip) => sum + ip.uploadBytes, 0),
      totalDownload: ips.reduce((sum, ip) => sum + ip.downloadBytes, 0),
      totalTcp: ips.reduce((sum, ip) => sum + ip.tcpCount, 0),
      totalUdp: ips.reduce((sum, ip) => sum + ip.udpCount, 0),
      totalOther: ips.reduce((sum, ip) => sum + ip.otherCount, 0),
    };
  };
  
  return {
    lan: { ...calculateGroupTotal(lanIPs), name: '局域网IP' },
    other: { ...calculateGroupTotal(otherIPs), name: '其他IP' }
  };
});

// ================= 7. 辅助函数 =================
const formatIP = (ip: string | undefined, family: string | undefined): string => {
  if (!ip) return '-';
  if (family?.toUpperCase() === 'IPV6') {
    return `[${compressIPv6(ip)}]`;
  }
  return ip;
};

// 格式化流量显示
const formatTraffic = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  return formatIOBytes(bytes);
};

// 复制功能
const copyInfo = (row: any) => {
  let source_ip: string = row.source_ip
  let destination_ip: string = row.destination_ip

  if (row.ip_family?.toUpperCase() === 'IPV6') {
    source_ip = `[${compressIPv6(row.source_ip)}]`;
    destination_ip = `[${compressIPv6(row.destination_ip)}]`;
  }

  const text = `[${row.ip_family}] ${row.protocol} ${source_ip}:${row.source_port} -> ${destination_ip}:${row.destination_port} | 状态: ${row.state || '-'} | 流量: ${row.traffic.value.toFixed(2)} ${row.traffic.unit} (${row.packets} Pkgs)`;

  // 检查浏览器是否支持 Clipboard API
  if (navigator.clipboard && window.isSecureContext) {
    // 现代浏览器的安全上下文
    navigator.clipboard.writeText(text).then(() => {
      const { success } = useToast();
      success('连接信息已复制！');
    }).catch((err) => {
      console.error('复制失败:', err);
      // 降级到传统方法
      fallbackCopyTextToClipboard(text);
    });
  } else {
    // 降级到传统方法
    fallbackCopyTextToClipboard(text);
  }
};

// 传统复制方法（兼容不支持 Clipboard API 的浏览器）
const fallbackCopyTextToClipboard = (text: string) => {
  const textArea = document.createElement('textarea');
  textArea.value = text;

  // 避免滚动到底部
  textArea.style.top = '0';
  textArea.style.left = '0';
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';

  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();

  try {
    const successful = document.execCommand('copy');
    if (successful) {
      const { success } = useToast();
      success('连接信息已复制！');
    } else {
      const { error } = useToast();
      error('复制失败，请手动复制');
    }
  } catch (err) {
    console.error('传统复制方法失败:', err);
    const { error } = useToast();
    error('复制失败，请手动复制');
  }

  document.body.removeChild(textArea);
};

// ================= 8. TanStack Table 配置 (使用 h 函数代替 JSX 以避免 TS 解析错误) =================
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
      return h('span', { class: 'text-slate-300' }, BytesFixed(row.traffic.value, row.traffic.unit) + ' ' + row.traffic.unit + ' (' + row.packets + ' Pkgs.)');
    },
    sortingFn: (rowA, rowB) => {
      const valA = rowA.original.traffic.value || 0;
      const unitA = rowA.original.traffic.unit || 'B';
      const valB = rowB.original.traffic.value || 0;
      const unitB = rowB.original.traffic.unit || 'B';

      // 将不同单位转换为字节进行比较
      const bytesA = convertToBytes(valA, unitA);
      const bytesB = convertToBytes(valB, unitB);

      return bytesA - bytesB;
    },
    enableSorting: true,
    filterFn: (row, columnId, filterValue) => {
      const trafficValue = row.original.traffic.value || 0;
      const trafficUnit = row.original.traffic.unit || '';
      const packets = row.original.packets || 0;

      // 格式化后的值
      const formattedValue = BytesFixed(trafficValue, trafficUnit);
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
  data: displayData,
  columns,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getFilteredRowModel: getFilteredRowModel(),
  getRowId: (row, index, parent) => {
    // 为每个连接创建一个标准化的唯一ID
    const endpointA = `${row.source_ip}:${row.source_port}`;
    const endpointB = `${row.destination_ip}:${row.destination_port}`;
    const endpoints = [endpointA, endpointB].sort(); // 排序确保一致性
    const baseId = `${endpoints[0]}<->${endpoints[1]}-${row.protocol}`;

    // 添加一个稳定的唯一标识符，基于连接信息和原始索引
    return `${baseId}-${row.traffic.value}-${row.packets}-${index}`;
  },
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
  <div class="flex flex-col h-full space-y-6">
    <!-- Counts -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-5">
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

    <!-- 1. 聚合统计表格 -->
    <div>
      <!-- 聚合统计折叠栏 -->
      <div @click="uiState.accordions.aggregation = !uiState.accordions.aggregation"
        class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
        <div class="flex items-center gap-4">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">聚合统计</h3>
          <span class="text-xs text-slate-500">按 IP 地址聚合统计</span>
        </div>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': uiState.accordions.aggregation }">▼</span>
      </div>

      <!-- 聚合统计内容 -->
      <div v-show="uiState.accordions.aggregation" class="bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
        <!-- 全局搜索框（单独一行，居右） -->
        <div class="px-4 py-3 border-b border-slate-700 flex justify-end">
          <div class="relative">
            <input v-model="aggregationFilter" placeholder="搜索 IP、流量、连接数..."
              class="bg-slate-900 border border-slate-600 text-white text-xs px-3 py-1.5 rounded w-56 outline-none focus:border-blue-500" />
          </div>
        </div>
        
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-center border-collapse">
            <thead class="bg-slate-700/50 text-slate-300">
              <tr>
                <th @click="toggleAggregationSort('ip')" 
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    IP 地址
                    <span class="text-slate-400">{{ getSortIcon('ip') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('traffic')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    实时流量
                    <span class="text-slate-400">{{ getSortIcon('traffic') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('upload')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    实时上行
                    <span class="text-slate-400">{{ getSortIcon('upload') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('download')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    实时下行
                    <span class="text-slate-400">{{ getSortIcon('download') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('tcp')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    TCP 连接
                    <span class="text-slate-400">{{ getSortIcon('tcp') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('udp')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    UDP 连接
                    <span class="text-slate-400">{{ getSortIcon('udp') }}</span>
                  </div>
                </th>
                <th @click="toggleAggregationSort('other')"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap cursor-pointer select-none hover:text-white hover:bg-slate-700/50 transition-colors">
                  <div class="flex items-center justify-center gap-1">
                    其他连接
                    <span class="text-slate-400">{{ getSortIcon('other') }}</span>
                  </div>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-700">
              <!-- 局域网IP分组 -->
              <tr class="bg-slate-700/30 hover:bg-slate-700/50 transition-colors cursor-pointer"
                @click="uiState.ipGroupCollapsed.lan = !uiState.ipGroupCollapsed.lan">
                <td colspan="7" class="px-3 py-3 text-left">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <span class="text-slate-500 transition-transform duration-300"
                        :class="{ 'rotate-180': !uiState.ipGroupCollapsed.lan }">▼</span>
                      <span class="font-semibold text-slate-200">{{ aggregationData.lan.name }}</span>
                      <span class="text-xs text-slate-500">({{ aggregationData.lan.ips.length }} 个 IP)</span>
                    </div>
                    <div class="flex items-center gap-4 text-xs">
                      <span class="text-slate-400">总流量: <span class="text-slate-200 font-mono">{{ formatTraffic(aggregationData.lan.totalTraffic) }}</span></span>
                      <span class="text-slate-400">上行: <span class="text-orange-400 font-mono">{{ formatTraffic(aggregationData.lan.totalUpload) }}</span></span>
                      <span class="text-slate-400">下行: <span class="text-cyan-400 font-mono">{{ formatTraffic(aggregationData.lan.totalDownload) }}</span></span>
                      <span class="text-slate-400">TCP: <span class="text-slate-200 font-mono">{{ aggregationData.lan.totalTcp }}</span></span>
                      <span class="text-slate-400">UDP: <span class="text-slate-200 font-mono">{{ aggregationData.lan.totalUdp }}</span></span>
                      <span class="text-slate-400">其他: <span class="text-slate-200 font-mono">{{ aggregationData.lan.totalOther }}</span></span>
                    </div>
                  </div>
                </td>
              </tr>
              <!-- 局域网IP详细行 -->
              <tr v-for="ipStats in aggregationData.lan.ips" :key="ipStats.ip" v-show="!uiState.ipGroupCollapsed.lan"
                class="hover:bg-slate-700/30 transition-colors">
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-300">{{ ipStats.ip }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ formatTraffic(ipStats.trafficBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-orange-400">{{ formatTraffic(ipStats.uploadBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-cyan-400">{{ formatTraffic(ipStats.downloadBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.tcpCount }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.udpCount }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.otherCount }}</span>
                </td>
              </tr>
              <tr v-if="aggregationData.lan.ips.length === 0 && !uiState.ipGroupCollapsed.lan">
                <td colspan="7" class="px-5 py-4 text-center text-slate-500 text-xs">暂无局域网IP数据</td>
              </tr>

              <!-- 其他IP分组 -->
              <tr class="bg-slate-700/30 hover:bg-slate-700/50 transition-colors cursor-pointer"
                @click="uiState.ipGroupCollapsed.other = !uiState.ipGroupCollapsed.other">
                <td colspan="7" class="px-3 py-3 text-left">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <span class="text-slate-500 transition-transform duration-300"
                        :class="{ 'rotate-180': !uiState.ipGroupCollapsed.other }">▼</span>
                      <span class="font-semibold text-slate-200">{{ aggregationData.other.name }}</span>
                      <span class="text-xs text-slate-500">({{ aggregationData.other.ips.length }} 个 IP)</span>
                    </div>
                    <div class="flex items-center gap-4 text-xs">
                      <span class="text-slate-400">总流量: <span class="text-slate-200 font-mono">{{ formatTraffic(aggregationData.other.totalTraffic) }}</span></span>
                      <span class="text-slate-400">上行: <span class="text-orange-400 font-mono">{{ formatTraffic(aggregationData.other.totalUpload) }}</span></span>
                      <span class="text-slate-400">下行: <span class="text-cyan-400 font-mono">{{ formatTraffic(aggregationData.other.totalDownload) }}</span></span>
                      <span class="text-slate-400">TCP: <span class="text-slate-200 font-mono">{{ aggregationData.other.totalTcp }}</span></span>
                      <span class="text-slate-400">UDP: <span class="text-slate-200 font-mono">{{ aggregationData.other.totalUdp }}</span></span>
                      <span class="text-slate-400">其他: <span class="text-slate-200 font-mono">{{ aggregationData.other.totalOther }}</span></span>
                    </div>
                  </div>
                </td>
              </tr>
              <!-- 其他IP详细行 -->
              <tr v-for="ipStats in aggregationData.other.ips" :key="ipStats.ip" v-show="!uiState.ipGroupCollapsed.other"
                class="hover:bg-slate-700/30 transition-colors">
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-300">{{ ipStats.ip }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ formatTraffic(ipStats.trafficBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-orange-400">{{ formatTraffic(ipStats.uploadBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-cyan-400">{{ formatTraffic(ipStats.downloadBytes) }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.tcpCount }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.udpCount }}</span>
                </td>
                <td class="px-3 py-2 text-center">
                  <span class="font-mono text-slate-200">{{ ipStats.otherCount }}</span>
                </td>
              </tr>
              <tr v-if="aggregationData.other.ips.length === 0 && !uiState.ipGroupCollapsed.other">
                <td colspan="7" class="px-5 py-4 text-center text-slate-500 text-xs">暂无其他IP数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 2. 连接列表 -->
    <div>
      <!-- 连接列表折叠栏 -->
      <div @click="uiState.accordions.connectionList = !uiState.accordions.connectionList"
        class="py-2.5 border-b border-slate-700 mb-5 cursor-pointer select-none flex justify-between items-center group">
        <div class="flex items-center gap-4">
          <h3 class="text-lg font-semibold text-slate-200 group-hover:text-white">连接列表</h3>
        </div>
        <span class="text-slate-500 transition-transform duration-300"
          :class="{ 'rotate-180': uiState.accordions.connectionList }">▼</span>
      </div>

      <!-- 连接列表内容 -->
      <div v-show="uiState.accordions.connectionList" class="bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
        <!-- 全局搜索框（单独一行，居右） -->
        <div class="px-4 py-3 border-b border-slate-700 flex justify-end">
          <div class="relative">
            <input v-model="globalFilter" placeholder="全局搜索..."
              class="bg-slate-900 border border-slate-600 text-white text-xs px-3 py-1.5 rounded w-56 outline-none focus:border-blue-500" />
          </div>
        </div>
        
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-center border-collapse">
            <thead class="bg-slate-700/50 text-slate-300">
              <tr>
                <th v-for="column in table.getHeaderGroups()[0].headers" :key="column.id"
                  class="px-3 py-3 font-medium text-center whitespace-nowrap">
                  <div class="flex flex-col gap-1 items-center">
                    <div v-if="column.column.getCanSort()"
                      class="flex items-center gap-1 cursor-pointer select-none hover:text-white" @click="() => {
                        if (column.column.getCanSort()) {
                          column.column.toggleSorting(undefined, column.column.getIsSorted() === false)
                        }
                      }">
                      <FlexRender :render="column.column.columnDef.header" :props="column.getContext()" />
                      {{ { asc: '↑', desc: '↓' }[column.column.getIsSorted() as string] || '' }}
                    </div>
                    <div v-else class="flex items-center gap-1">
                      <FlexRender :render="column.column.columnDef.header" :props="column.getContext()" />
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
  </div>
</template>
