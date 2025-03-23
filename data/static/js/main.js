import { fetchStatsData } from './api.js';
import {
    updateChart,
    updateViewToggleButtons,
    displayErrorMessage,
    updateOverallStats
} from './charts.js';
import { initWebsiteSelector } from './sites.js';
import {
    setLoadingState,
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

        // 如果成功获取网站ID，加载数据
        if (currentWebsiteId) {
            loadWebsiteData(currentWebsiteId);
        }
    } catch (error) {
        console.error('初始化网站失败:', error);
        displayErrorMessage('无法初始化网站选择器，请刷新页面重试', chartCanvas);
    }
}

// 网站选择变化处理回调
function handleWebsiteSelected(websiteId) {
    currentWebsiteId = websiteId;
    loadWebsiteData(websiteId);
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
    // 如果没有数据，不执行任何操作
    if (!statsDataCache) {
        return;
    }

    // 获取当前选择的日期范围
    const range = dateRange.value;

    // 更新视图切换按钮状态，可能会改变currentView
    const newView = updateViewToggleButtons(range, currentView, viewToggleBtns);
    if (newView !== currentView) {
        currentView = newView;
    }

    // 更新图表
    refreshChart();
}

// 处理视图切换
function handleViewToggle() {
    // 如果没有数据，不执行任何操作
    if (!statsDataCache) {
        return;
    }

    // 更新活动按钮状态
    viewToggleBtns.forEach(b => b.classList.remove('active'));
    this.classList.add('active');

    // 更新当前视图
    currentView = this.dataset.view;

    // 保存用户偏好
    saveUserPreference('preferredView', currentView);

    // 重新渲染图表
    refreshChart();
}

// 加载网站数据
async function loadWebsiteData(websiteId) {
    try {
        // 显示加载中状态
        setLoadingState(true);

        // 获取统计数据
        const data = await fetchStatsData(websiteId);
        statsDataCache = data;

        // 获取当前选择的日期范围
        const range = dateRange.value;

        // 先检查日期范围设置
        checkInitialDateRange();

        // 直接更新总览数据
        console.log("直接更新总览数据");
        updateOverallStats(data, range);

        // 然后更新图表
        refreshChart();

        setTimeout(() => {
            const currentRange = dateRange.value;
            updateOverallStats(statsDataCache, currentRange);
        }, 50);

        // 隐藏加载状态
        setLoadingState(false);
    } catch (error) {
        console.error('加载网站数据失败:', error);
        displayErrorMessage(`无法获取"${websiteSelector.options[websiteSelector.selectedIndex].text}"的统计数据`, chartCanvas);
        // 重置统计信息
        resetStatistics();
        // 隐藏加载状态
        setLoadingState(false);
    }
}

// 刷新图表
function refreshChart() {
    // 如果没有数据，不渲染图表
    if (!statsDataCache) {
        return;
    }

    // 获取当前选择的日期范围
    const range = dateRange.value;

    // 使用charts.js中的函数更新图表
    updateChart(ctx, statsDataCache, range, currentView);

    // 确保总览数据更新 - 即使charts.js的updateChart内部已调用，这里再次调用以确保更新
    updateOverallStats(statsDataCache, range);
}

// 检查初始日期范围并设置按钮状态
function checkInitialDateRange() {
    const initialRange = dateRange.value;
    const newView = updateViewToggleButtons(initialRange, currentView, viewToggleBtns);
    if (newView !== currentView) {
        currentView = newView;
    }
}

// 页面加载时初始化应用
document.addEventListener('DOMContentLoaded', initApp);