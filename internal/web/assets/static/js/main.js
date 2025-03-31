import {
    fetchOverallStats,
    fetchUrlStats,
    fetchRefererStats,
    fetchBrowserStats,
    fetchOSStats,
    fetchDeviceStats,
} from './api.js';

import {
    initChart,
    updateChartWebsiteIdAndRange,
    displayErrorMessage,
} from './charts.js';

import {
    updateUrlRankingTable,
    updaterefererRankingTable,
    updateBrowserTable,
    updateOsTable,
    updateDeviceTable,
} from './ranking.js';

import {
    initWebsiteSelector,
} from './sites.js';

import {
    initGeoMap,
    updateGeoMapWebsiteIdAndRange,
} from './maps.js';

import {
    formatTraffic,
} from './utils.js';

import {
    initThemeManager,
} from './theme.js';

// 模块级变量
let websiteSelector = null;
let dateRange = null;
let currentWebsiteId = '';

// 初始化应用
function initApp() {
    // 获取控件元素
    websiteSelector = document.getElementById('website-selector');
    dateRange = document.getElementById('date-range');

    initThemeManager(); // 初始化主题
    initChart(); // 初始化图表
    initGeoMap(); // 初始化地图
    initSites(); // 初始化网站选择器并绑定回调
    bindEventListeners();  // 绑定事件监听器
}

// 初始化网站选择器
async function initSites() {
    try {
        currentWebsiteId = await initWebsiteSelector(websiteSelector, handleWebsiteSelected);
        refreshData();

    } catch (error) {
        console.error('初始化网站失败:', error);
        displayErrorMessage('无法初始化网站选择器，请刷新页面重试');
    }
}

// 网站选择变化处理回调
function handleWebsiteSelected(websiteId) {
    currentWebsiteId = websiteId;
    refreshData();
}

// 绑定事件监听器
function bindEventListeners() {
    dateRange.addEventListener('change', handleDateRangeChange);
}

// 处理日期范围变化
function handleDateRangeChange() {
    const range = dateRange.value;
    refreshData();
}


// 加载网站数据
async function refreshData() {
    try {
        // 获取统计数据
        const range = dateRange.value;

        updateChartWebsiteIdAndRange(currentWebsiteId, range);
        updateGeoMapWebsiteIdAndRange(currentWebsiteId, range);

        const [overallData, urlStats, refererStats,
            browserStats, osStats, deviceStats] =
            await Promise.all([
                fetchOverallStats(currentWebsiteId, range),
                fetchUrlStats(currentWebsiteId, range, 10),
                fetchRefererStats(currentWebsiteId, range, 10),
                fetchBrowserStats(currentWebsiteId, range, 10),
                fetchOSStats(currentWebsiteId, range, 10),
                fetchDeviceStats(currentWebsiteId, range, 10)
            ]);

        updateOverallStats(overallData);
        updateUrlRankingTable(urlStats);
        updaterefererRankingTable(refererStats);
        updateBrowserTable(browserStats);
        updateOsTable(osStats);
        updateDeviceTable(deviceStats);

    } catch (error) {
        console.error('加载网站数据失败:', error);
        displayErrorMessage(`无法获取"${websiteSelector.options[websiteSelector.selectedIndex].text}"的统计数据`, chartCanvas);
    }
}

// 更新整体统计数据
function updateOverallStats(overall) {
    // 格式化流量显示
    const trafficDisplay = formatTraffic(overall.traffic);

    // 更新DOM
    document.getElementById('total-uv').textContent = overall.uv.toLocaleString();
    document.getElementById('total-pv').textContent = overall.pv.toLocaleString();
    document.getElementById('total-traffic').textContent = trafficDisplay;
}

// 页面加载时初始化应用
document.addEventListener('DOMContentLoaded', initApp);