// 配置
const API = {
    dynamic: '/metric/dynamic',
    connection: '/metric/network_connection',
    static: '/metric/static'
};
const DEFAULT_REFRESH = 2000;

// 全局状态
let refreshTimer = null;
let currentTab = 'tab-system';

// DOM 元素缓存
const els = {
    tabs: document.querySelectorAll('.tab-btn'),
    contents: document.querySelectorAll('.tab-content'),
    refreshSelect: document.getElementById('refresh-select'),
    statusTime: document.getElementById('status-time'),
    statusText: document.getElementById('status-text'),
    statusDot: document.getElementById('status-dot'),
    spinner: document.getElementById('loading-spinner'),
    sysSummary: document.getElementById('system-summary-grid'),
    sysDetails: document.getElementById('system-details-container'),
    netSummary: document.getElementById('network-summary-grid'),
    connTable: document.getElementById('connection-table-body'),
    toastContainer: document.getElementById('toast-container')
};

// --- 工具函数 ---

function formatTime() {
    const now = new Date();
    return now.toISOString().replace('T', ' ').substring(0, 19);
}

function showToast(msg) {
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = msg;
    els.toastContainer.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

// --- 核心逻辑 ---

async function fetchData() {
    els.spinner.style.display = 'block';
    els.statusText.textContent = '刷新中...';
    
    // 记录请求开始时间，用于更新右上角时间
    const reqTime = formatTime();

    try {
        // 并行请求所有接口
        const [dynamicRes, connRes, staticRes] = await Promise.all([
            fetch(API.dynamic),
            fetch(API.connection),
            fetch(API.static)
        ]);

        if (!dynamicRes.ok || !connRes.ok || !staticRes.ok) {
            throw new Error('一个或多个接口请求失败');
        }

        const dynamic = await dynamicRes.json();
        const conn = await connRes.json();
        const stat = await staticRes.json();

        renderSystemTab(dynamic, stat);
        renderNetworkTab(conn);

        els.statusDot.className = 'status-dot active';
        els.statusText.textContent = '运行中';

    } catch (error) {
        console.error(error);
        els.statusDot.className = 'status-dot error';
        els.statusText.textContent = '错误';
        showToast(error.message);
    } finally {
        els.spinner.style.display = 'none';
        els.statusTime.textContent = reqTime;
    }
}

// --- 渲染逻辑: 系统概览 ---

function renderSystemTab(dynamic, stat) {
    // 1. 渲染顶部汇总
    let summaryHTML = '';
    
    // CPU Total
    // 修正: usage 是一个对象 {value: ..., unit: ...}，需要取 .value
    const cpuUsageObj = dynamic.cpu?.total?.usage;
    if (cpuUsageObj && cpuUsageObj.value !== undefined) {
        summaryHTML += createSummaryCard('CPU 总使用率', `${cpuUsageObj.value.toFixed(1)}`, '%', 'var(--accent-cpu)');
    }

    // Memory Total
    // 修正: used_percent 是一个对象 {value: ..., unit: ...}，需要取 .value
    const memObj = dynamic.memory?.used_percent;
    if (memObj && memObj.value !== undefined) {
        summaryHTML += createSummaryCard('内存使用率', `${memObj.value.toFixed(1)}`, '%', 'var(--accent-mem)');
    }

    // Network Total
    const netIn = dynamic.network?.total?.incoming;
    const netOut = dynamic.network?.total?.outgoing;
    if (netIn && netOut) {
        summaryHTML += createSummaryCard('网络下行', formatBytes(netIn.value), netIn.unit, 'var(--accent-read)');
        summaryHTML += createSummaryCard('网络上行', formatBytes(netOut.value), netOut.unit, 'var(--accent-write)');
    }

    // System Info (Uptime, Hostname)
    const uptime = dynamic.system?.uptime;
    const hostname = stat.system?.hostname;
    if (uptime) summaryHTML += createSummaryCard('系统运行时间', uptime, '', '#fff');
    if (hostname) summaryHTML += createSummaryCard('主机名', hostname, '', '#fff');

    els.sysSummary.innerHTML = summaryHTML;

    // 2. 渲染详细折叠区域
    let detailsHTML = '';

    // Storage Section
    if (dynamic.storage) {
        let storageCards = '';
        for (const [devName, devData] of Object.entries(dynamic.storage)) {
            // 过滤掉可能存在的 total 键（如果它是一个设备名）
            if (devName === 'total') continue; 
            
            storageCards += `
                <div class="metric-row" style="display:block">
                    <div style="font-weight:bold; margin-bottom:5px">${devName}</div>
                    <div style="display:grid; grid-template-columns:1fr 1fr; gap:10px">
                        <div><span class="metric-label">读:</span> ${formatBytes(devData.read.value)} ${devData.read.unit}</div>
                        <div><span class="metric-label">写:</span> ${formatBytes(devData.write.value)} ${devData.write.unit}</div>
                        <div><span class="metric-label">容量:</span> ${devData.total.value} ${devData.total.unit}</div>
                        <div><span class="metric-label">使用率:</span> ${devData.used_percent.value.toFixed(1)}%</div>
                    </div>
                    <div style="height:4px; background:#333; margin-top:5px; border-radius:2px">
                        <div style="width:${Math.min(devData.used_percent.value, 100)}%; height:100%; background:var(--accent-read); border-radius:2px"></div>
                    </div>
                </div>
            `;
        }
        detailsHTML += createAccordionSection('存储详情', storageCards, true); // 默认展开
    }

    // CPU Section
    if (dynamic.cpu) {
        let cpuCards = '';
        for (const [core, data] of Object.entries(dynamic.cpu)) {
            if (core === 'total') continue;
            cpuCards += `
                <div class="metric-row">
                    <span class="metric-label">${core}</span>
                    <span class="metric-val">${data.usage.value.toFixed(1)}% ${data.temperature.value > 0 ? `(${data.temperature.value.toFixed(0)}°C)` : ''}</span>
                </div>
            `;
        }
        detailsHTML += createAccordionSection('CPU 核心详情', cpuCards, false);
    }

    // Network Details (合并 Static IP)
    if (dynamic.network || stat.network) {
        let netCards = '';
        // 动态 IO
        for (const [iface, data] of Object.entries(dynamic.network)) {
            if (iface === 'total') continue;
            netCards += `
                <div class="metric-row">
                    <span class="metric-label" style="width:60px">${iface} IO</span>
                    <span class="metric-val" style="color:var(--accent-read)">↓${formatBytes(data.incoming.value)}</span>
                    <span class="metric-val" style="color:var(--accent-write)">↑${formatBytes(data.outgoing.value)}</span>
                </div>
            `;
        }
        // 静态 IP
        if (stat.network) {
            for (const [iface, data] of Object.entries(stat.network)) {
                if (iface === 'global' || iface === 'lo') continue; // lo 和 global 单独处理
                netCards += `
                    <div class="metric-row">
                        <span class="metric-label" style="width:60px">${iface} IP</span>
                        <span class="metric-val" style="font-size:0.8rem">${data.ipv4.join(', ')} ${data.ipv6.length ? '...' : ''}</span>
                    </div>
                `;
            }
        }
        // Gateway
        if (stat.network?.global?.gateway && stat.network.global.gateway !== 'unknown') {
            netCards += `
                <div class="metric-row">
                    <span class="metric-label">网关</span>
                    <span class="metric-val">${stat.network.global.gateway}</span>
                </div>
            `;
        }
        detailsHTML += createAccordionSection('网络配置详情', netCards, false);
    }

    // System Info Details
    if (stat.system) {
        let sysInfo = '';
        sysInfo += `<div class="metric-row"><span class="metric-label">OS</span><span class="metric-val">${stat.system.os}</span></div>`;
        sysInfo += `<div class="metric-row"><span class="metric-label">Kernel</span><span class="metric-val">${stat.system.kernel}</span></div>`;
        sysInfo += `<div class="metric-row"><span class="metric-label">Arch</span><span class="metric-val">${stat.system.arch}</span></div>`;
        sysInfo += `<div class="metric-row"><span class="metric-label">Timezone</span><span class="metric-val">${stat.system.timezone}</span></div>`;
        detailsHTML += createAccordionSection('系统信息详情', sysInfo, false);
    }

    els.sysDetails.innerHTML = detailsHTML;
}

// --- 渲染逻辑: 网络连接 ---

function renderNetworkTab(connData) {
    // 1. 汇总 Counts
    const counts = connData.counts || {};
    els.netSummary.innerHTML = `
        ${createSummaryCard('TCP 连接', counts.tcp, '', 'var(--accent-mem)')}
        ${createSummaryCard('UDP 连接', counts.udp, '', 'var(--accent-cpu)')}
        ${createSummaryCard('其他连接', counts.other, '', '#fff')}
    `;

    // 2. 连接列表表格
    const conns = connData.connections || [];
    if (conns.length === 0) {
        els.connTable.innerHTML = '<tr><td colspan="4" style="text-align:center">暂无连接数据</td></tr>';
        return;
    }

    // 限制显示数量以防卡顿，比如只显示前 50 条，或者全部显示
    const displayConns = conns; 

    els.connTable.innerHTML = displayConns.map(c => `
        <tr>
            <td><span class="badge-proto">${c.protocol.toUpperCase()}</span></td>
            <td>${c.source_ip}:${c.source_port > 0 ? c.source_port : ''}</td>
            <td>${c.destination_ip}:${c.destination_port > 0 ? c.destination_port : ''}</td>
            <td>${c.state || '-'}</td>
        </tr>
    `).join('');
}

// --- 辅助 HTML 生成器 ---

function createSummaryCard(label, value, unit, color) {
    return `
        <div class="summary-card" style="border-top: 3px solid ${color}">
            <div class="label">${label}</div>
            <div>
                <span class="value">${value}</span>
                <span class="unit">${unit}</span>
            </div>
        </div>
    `;
}

function createAccordionSection(title, contentHTML, isOpen) {
    const openClass = isOpen ? 'open' : '';
    const displayStyle = isOpen ? 'display:block' : 'display:none';
    return `
        <div class="accordion-item ${openClass}">
            <div class="accordion-header" onclick="toggleAccordion(this)">
                <h3>${title}</h3>
                <span class="toggle-icon">▼</span>
            </div>
            <div class="accordion-body" style="${displayStyle}">
                <div class="sub-grid">
                    ${contentHTML}
                </div>
            </div>
        </div>
    `;
}

// 格式化字节
function formatBytes(bytes) {
    if (bytes === 0 || bytes === -1) return '0';
    if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(2) + ' GB';
    if (bytes >= 1048576) return (bytes / 1048576).toFixed(2) + ' MB';
    if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB';
    return bytes.toFixed(0);
}

// --- 交互控制 ---

// 切换折叠面板
window.toggleAccordion = function(header) {
    const item = header.parentElement;
    item.classList.toggle('open');
    const body = item.querySelector('.accordion-body');
    body.style.display = item.classList.contains('open') ? 'block' : 'none';
};

// 切换标签页
els.tabs.forEach(btn => {
    btn.addEventListener('click', () => {
        // UI 切换
        els.tabs.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        
        const targetId = btn.getAttribute('data-target');
        els.contents.forEach(c => c.classList.remove('active'));
        document.getElementById(targetId).classList.add('active');
        
        currentTab = targetId;
    });
});

// 刷新间隔控制
function startPolling(interval) {
    if (refreshTimer) clearInterval(refreshTimer);
    refreshTimer = setInterval(fetchData, interval);
}

els.refreshSelect.addEventListener('change', (e) => {
    startPolling(parseInt(e.target.value));
    fetchData();
});

// 初始化
function init() {
    fetchData();
    startPolling(DEFAULT_REFRESH);
}

init();