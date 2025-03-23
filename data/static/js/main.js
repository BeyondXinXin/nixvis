import { fetchStatsData } from './api.js';
import { renderChart, updateOverallStats, processStatsData, displayErrorMessage } from './charts.js';
import { getStartOfWeek, getDaysArray, getDateByTimeRange, formatTraffic } from './utils.js';

document.addEventListener('DOMContentLoaded', function () {
    // 获取控件元素
    const dateRange = document.getElementById('date-range');
    const viewToggleBtns = document.querySelectorAll('.toggle-btn');
    const chartCanvas = document.getElementById('visitsChart');

    if (!chartCanvas) {
        console.error('找不到图表canvas元素');
        return;
    }

    const ctx = chartCanvas.getContext('2d');

    // 当前视图模式 (hourly 或 daily)
    let currentView = 'hourly';

    // 用于存储从服务器获取的数据
    let statsDataCache = null;

    // 更新图表函数
    function updateChart() {
        // 如果没有数据，不渲染图表
        if (!statsDataCache) {
            return;
        }

        // 获取当前选择的日期范围
        const range = dateRange.value;

        // 处理数据并渲染图表
        const chartData = processStatsData(statsDataCache, range, currentView);
        renderChart(ctx, chartData, range, currentView);

        // 更新整体统计数据
        updateOverallStats(statsDataCache, range);
    }

    // 为日期选择下拉框添加事件
    dateRange.addEventListener('change', function () {
        // 如果没有数据，不执行任何操作
        if (!statsDataCache) {
            return;
        }

        // 获取当前选择的日期范围
        const range = dateRange.value;

        // 如果选择了"今天"，强制使用"按小时"视图并禁用"按天"按钮
        if (range === 'today') {
            // 找到按天的按钮并禁用它
            const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
            const hourlyBtn = document.querySelector('.toggle-btn[data-view="hourly"]');

            dailyBtn.classList.add('disabled');
            dailyBtn.disabled = true;

            // 如果当前是按天视图，切换到按小时视图
            if (currentView === 'daily') {
                // 更新按钮状态
                viewToggleBtns.forEach(btn => btn.classList.remove('active'));
                hourlyBtn.classList.add('active');

                // 更新当前视图
                currentView = 'hourly';
            }
        } else {
            // 其他日期范围，启用"按天"按钮
            const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
            dailyBtn.classList.remove('disabled');
            dailyBtn.disabled = false;
        }

        updateChart();
    });

    // 为切换按钮添加事件
    viewToggleBtns.forEach(btn => {
        btn.addEventListener('click', function () {
            // 如果没有数据，不执行任何操作
            if (!statsDataCache) {
                return;
            }

            // 更新活动按钮状态
            viewToggleBtns.forEach(b => b.classList.remove('active'));
            this.classList.add('active');

            // 更新当前视图
            currentView = this.dataset.view;

            // 重新渲染图表
            updateChart();
        });
    });

    // 检查初始日期范围并设置按钮状态
    function checkInitialDateRange() {
        const initialRange = dateRange.value;
        if (initialRange === 'today') {
            const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
            dailyBtn.classList.add('disabled');
            dailyBtn.disabled = true;
        }
    }

    // 获取统计数据并初始化图表
    fetchStatsData()
        .then(data => {
            console.log('从服务器获取的数据:', data);
            statsDataCache = data;
            updateChart(); // 获取数据后更新图表
            checkInitialDateRange(); // 检查初始日期范围并设置按钮状态
        })
        .catch(error => {
            console.error('获取统计数据失败:', error);
            // 显示错误消息
            displayErrorMessage('无法获取统计数据，请稍后再试', chartCanvas);
        });
});