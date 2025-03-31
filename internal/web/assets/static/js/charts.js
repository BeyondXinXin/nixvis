import {
    fetchTimeSeriesStats,
} from './api.js';

// 用于存储图表实例的模块级变量
let ctx = null;
let visitsChart = null;
let chartCanvas = null;
let viewToggleBtns = null;

let currentView = 'hourly';
let range = 'today';
let currentWebsiteId = '';

// 初始化地图
export function initChart() {
    chartCanvas = document.getElementById('visitsChart');
    ctx = chartCanvas.getContext('2d');
    bindEventListeners();
}

// 更新 WebsiteId 和 range
export function updateChartWebsiteIdAndRange(websiteId, newRange) {
    currentWebsiteId = websiteId;
    range = newRange;

    const dailyBtn = document.querySelector('.data-view-toggle-btn[data-view="daily"]');
    const hourlyBtn = document.querySelector('.data-view-toggle-btn[data-view="hourly"]');

    if (range === 'today' || range === 'yesterday') {
        dailyBtn.classList.add('disabled');
        dailyBtn.disabled = true;

        viewToggleBtns.forEach(btn => btn.classList.remove('active'));
        hourlyBtn.classList.add('active');

        currentView = 'hourly';
    } else {
        dailyBtn.classList.remove('disabled');
        dailyBtn.disabled = false;

        viewToggleBtns.forEach(btn => btn.classList.remove('active'));
        dailyBtn.classList.add('active');

        currentView = 'daily';
    }

    updateChart();
}


function bindEventListeners() {
    viewToggleBtns = document.querySelectorAll('.data-view-toggle-btn');
    viewToggleBtns.forEach(btn => {
        btn.addEventListener('click', function () {
            // 更新当前视图
            currentView = this.dataset.view;

            // 更新按钮状态
            viewToggleBtns.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');

            // 刷新数据
            updateChart();
        });
    });
}

// 渲染图表
async function updateChart() {
    // 清除错误消息
    const errorMsg = document.querySelector('.chart-error-message');
    if (errorMsg) {
        errorMsg.remove();
    }

    const statsData = await fetchTimeSeriesStats(currentWebsiteId, range, currentView)
    const pvMinusUv = statsData.pvMinusUv;

    // 准备图表配置
    const chartConfig = {
        type: 'bar',
        data: {
            labels: statsData.labels,
            datasets: [
                {
                    label: '访客数(UV)',
                    data: statsData.visitors,
                    backgroundColor: '#4a6fff', // 深蓝色 97,147,234
                    borderColor: '#4a6fff',
                    borderWidth: 1,
                    stack: 'Stack 0',
                },
                {
                    label: '浏览量(PV)',  // 修改标签名称，更清晰
                    data: pvMinusUv,     // 初始时仍然使用差值
                    backgroundColor: '#83c9f9', // 淡蓝色
                    borderColor: '#83c9f9',
                    borderWidth: 1,
                    stack: 'Stack 0',
                    // 存储原始完整PV值，用于切换显示
                    originalData: statsData.pageviews
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    stacked: true,
                    grid: {
                        display: true
                    }
                },
                x: {
                    stacked: true,
                    grid: {
                        display: false
                    }, ticks: {
                        callback: function (val, index) {
                            // 获取当前标签
                            const currentLabel = this.getLabelForValue(val);

                            // 检查是否是第一次出现
                            const labels = this.chart.data.labels;
                            const firstIndex = labels.indexOf(currentLabel);

                            // 只有第一次出现的标签才显示，其余相同标签返回空字符串
                            return firstIndex === index ? currentLabel : '';
                        }
                    }
                }
            },
            plugins: {
                tooltip: {
                    callbacks: {
                        // 自定义提示信息，显示完整的PV和UV值以及日期时间
                        label: function (context) {
                            const datasetIndex = context.datasetIndex;
                            const index = context.dataIndex;
                            const fullLabel = statsData.labels[index];

                            if (datasetIndex === 0) {
                                return `${fullLabel} - 访客数(UV): ${statsData.visitors[index]}`;
                            } else {
                                return `${fullLabel} - 浏览量(PV): ${statsData.pageviews[index]}`;
                            }
                        }
                    }
                },
                legend: {
                    position: 'bottom',
                    align: 'center',
                    labels: {
                        padding: 20,
                        boxWidth: 15,
                        usePointStyle: true,
                        generateLabels: function (chart) {
                            const originalLabels = Chart.defaults.plugins.legend.labels.generateLabels(chart);
                            if (originalLabels.length > 1) {
                                originalLabels[1].text = '浏览量(PV)';
                            }
                            return originalLabels;
                        }
                    },
                    onClick: function (e, legendItem, legend) {
                        const index = legendItem.datasetIndex;
                        const ci = legend.chart;
                        const meta = ci.getDatasetMeta(index);
                        const currentlyHidden = meta.hidden;
                        meta.hidden = !currentlyHidden;
                        const pvDataset = ci.data.datasets[1];

                        if (index === 0) {
                            if (!currentlyHidden) {
                                pvDataset.data = pvDataset.originalData;
                            } else {
                                pvDataset.data = pvMinusUv;
                            }
                        } else if (index === 1) {
                        }

                        // 更新图表
                        ci.update();
                    }
                }
            }
        }
    };

    // 如果已存在图表，先销毁
    if (visitsChart) {
        visitsChart.destroy();
    }

    // 创建新图表
    visitsChart = new Chart(ctx, chartConfig);
}

// 显示错误消息
export function displayErrorMessage(message) {
    // 清除现有图表
    if (visitsChart) {
        visitsChart.destroy();
        visitsChart = null;
    }

    // 在图表区域显示错误消息
    const container = chartCanvas.parentElement;
    const errorDiv = document.createElement('div');
    errorDiv.className = 'chart-error-message';
    errorDiv.textContent = message;
    errorDiv.style.textAlign = 'center';
    errorDiv.style.padding = '40px';
    errorDiv.style.color = '#721c24';
    errorDiv.style.backgroundColor = '#f8d7da';
    errorDiv.style.border = '1px solid #f5c6cb';
    errorDiv.style.borderRadius = '4px';
    errorDiv.style.marginTop = '20px';

    // 插入错误消息，替换或添加到图表容器
    if (container.querySelector('.chart-error-message')) {
        container.replaceChild(errorDiv, container.querySelector('.chart-error-message'));
    } else {
        container.appendChild(errorDiv);
    }
}


