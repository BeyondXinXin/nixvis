import { fetchStatsData } from './api.js';
import {
    renderChart,
    updateViewToggleButtons,
    displayErrorMessage,
    updateOverallStats
} from './charts.js';
import { initWebsiteSelector } from './sites.js';
import {
    resetStatistics,
    saveUserPreference
} from './utils.js';

// 模块级变量
let currentView = 'hourly';
let statsDataCache = null;
let ctx = null;
let websiteSelector = null;
let dateRange = null;
let viewToggleBtns = null;
let chartCanvas = null;
let currentWebsiteId = '';

// 初始化应用
function initApp() {
    // 获取控件元素
    websiteSelector = document.getElementById('website-selector');
    dateRange = document.getElementById('date-range');
    viewToggleBtns = document.querySelectorAll('.toggle-btn');
    chartCanvas = document.getElementById('visitsChart');

    if (!chartCanvas) {
        console.error('找不到图表canvas元素');
        return;
    }

    ctx = chartCanvas.getContext('2d');

    // 初始化网站选择器并绑定回调
    initSites();

    // 绑定事件监听器
    bindEventListeners();
}

// 初始化网站选择器
async function initSites() {
    try {
        // 使用sites.js中的函数初始化网站选择器
        // 当网站选择变化时，会调用onWebsiteSelected回调
        currentWebsiteId = await initWebsiteSelector(
            websiteSelector,
            handleWebsiteSelected,
            chartCanvas
        );

        refreshData();

    } catch (error) {
        console.error('初始化网站失败:', error);
        displayErrorMessage('无法初始化网站选择器，请刷新页面重试', chartCanvas);
    }
}

// 网站选择变化处理回调
function handleWebsiteSelected(websiteId) {
    currentWebsiteId = websiteId;
    refreshData();
}

// 绑定事件监听器
function bindEventListeners() {
    // 日期范围变化事件
    dateRange.addEventListener('change', handleDateRangeChange);

    // 视图切换按钮点击事件
    viewToggleBtns.forEach(btn => {
        btn.addEventListener('click', handleViewToggle);
    });
}

// 处理日期范围变化
function handleDateRangeChange() {
    const range = dateRange.value;
    const newView = updateViewToggleButtons(range, currentView, viewToggleBtns);
    if (newView !== currentView) {
        currentView = newView;
    }

    refreshData();
}

// 处理视图切换
function handleViewToggle() {
    viewToggleBtns.forEach(b => b.classList.remove('active'));
    this.classList.add('active');
    currentView = this.dataset.view;
    saveUserPreference('preferredView', currentView);

    refreshData();
}

// 加载网站数据
async function refreshData() {
    try {
        // 获取统计数据
        const range = dateRange.value;
        const statsData = await fetchStatsData(currentWebsiteId, range, currentView);
        statsDataCache = statsData;

        renderChart(ctx, statsDataCache.charts, range, currentView);

        setTimeout(() => {
            updateOverallStats(statsDataCache.overall);
        }, 50);


    } catch (error) {
        console.error('加载网站数据失败:', error);
        displayErrorMessage(`无法获取"${websiteSelector.options[websiteSelector.selectedIndex].text}"的统计数据`, chartCanvas);
        // 重置统计信息
        resetStatistics();

    }
}

// 页面加载时初始化应用
document.addEventListener('DOMContentLoaded', initApp);