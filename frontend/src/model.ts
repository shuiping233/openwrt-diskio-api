// 基础指标结构
export interface Metric {
  value: number;
  unit: string;
}

// 动态数据结构
export interface StorageData {
  read: Metric;
  write: Metric;
  total: Metric;
  used: Metric;
  used_percent: Metric;
}

export interface CpuData {
  usage: Metric;
  temperature: Metric;
}

export interface NetworkDynamicData {
  incoming: Metric;
  outgoing: Metric;
}

export interface SystemDynamicData {
  uptime: string;
}

export interface DynamicApiResponse {
  storage?: { [key: string]: StorageData };
  cpu?: { [key: string]: CpuData }; // key 是 cpu0, cpu1...
  network?: { total: NetworkDynamicData;[key: string]: NetworkDynamicData };
  memory?: {
    total: Metric;
    used: Metric;
    used_percent: Metric;
  };
  system?: SystemDynamicData;
}

// 静态数据结构
export interface IpConfig {
  ipv4: string[];
  ipv6: string[];
}

export interface NetworkStaticData {
  [key: string]: IpConfig & { gateway?: string ; dns?: string[] }  ;
}

export interface SystemStaticData {
  hostname: string;
  kernel: string;
  os: string;
  arch: string;
  timezone: string;
}

export interface StaticApiResponse {
  network?: NetworkStaticData;
  system?: SystemStaticData;
}

// 连接数据结构
export interface Connection {
  ip_family: string;
  source_ip: string;
  source_port: number;
  destination_ip: string;
  destination_port: number;
  protocol: string;
  state: string;
  traffic: Metric;
  packets: number;
}

export interface ConnectionApiResponse {
  counts?: { tcp: number; udp: number; other: number };
  connections?: Connection[];
}

// ================= 历史数据结构 =================
export interface HistoryRecord {
  id?: number; // Dexie 自增 ID
  timestamp: number; // 时间戳
  metric: 'cpu' | 'cpu_temp' | 'memory' | 'network_in' | 'network_out' | 'storage_io';
  value: number; // 数值
  unit: string; // 单位
}

// ================= 用户配置结构 =================
export interface UserSetting {
  key: string; // 键名，例如 'retention_days', 'theme'
  value: string | number | boolean; // 值
}