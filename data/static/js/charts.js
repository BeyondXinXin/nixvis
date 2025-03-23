import { getStartOfWeek, getDaysArray, getDateByTimeRange, formatTraffic } from './utils.js';

// 用于存储图表实例的模块级变量
let visitsChart = null;

// 显示错误消息
export function displayErrorMessage(message, chartCanvas) {
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

// 处理数据
export function processStatsData(statsData, timeRange, viewType) {
    if (viewType === 'hourly') {
        // 处理按小时的数据
        if (timeRange === 'week' || timeRange === 'last7days' || timeRange === 'last30days' || timeRange === 'month') {
            // 为"本周"、"最近7天"、"本月"或"最近30天"处理多天的小时数据
            let startDate, endDate;

            switch (timeRange) {
                case 'week':
                    // 本周(周一到今天)
                    startDate = getStartOfWeek(new Date());
                    endDate = new Date();
                    break;
                case 'last7days':
                    // 最近7天(今天和前6天)
                    endDate = new Date();
                    startDate = new Date();
                    startDate.setDate(endDate.getDate() - 6);
                    break;
                case 'month':
                    // 本月(1号到当前)
                    startDate = new Date();
                    startDate.setDate(1);
                    endDate = new Date();
                    break;
                case 'last30days':
                    // 最近30天(今天和前29天)
                    endDate = new Date();
                    startDate = new Date();
                    startDate.setDate(endDate.getDate() - 29);
                    break;
            }

            // 获取日期数组
            const days = getDaysArray(startDate, endDate);

            // 创建标签，格式为 "日期 小时"
            const labels = [];
            const visitors = [];
            const pageviews = [];

            // 为每一天的每一小时创建数据点
            days.forEach(dateStr => {
                for (let hour = 0; hour < 24; hour++) {
                    // 标签格式：MM.DD HH:00
                    const d = new Date(dateStr);
                    const label = `${d.getMonth() + 1}.${d.getDate()} ${hour}:00`;
                    labels.push(label);

                    // 如果有数据，使用实际数据，否则为0
                    if (statsData.days && statsData.days[dateStr] &&
                        statsData.days[dateStr].hourly) {
                        const hourData = statsData.days[dateStr].hourly.find(h => h.hour === hour);
                        if (hourData) {
                            visitors.push(hourData.stats.uv || 0);
                            pageviews.push(hourData.stats.pv || 0);
                        } else {
                            visitors.push(0);
                            pageviews.push(0);
                        }
                    } else {
                        visitors.push(0);
                        pageviews.push(0);
                    }
                }
            });

            return { labels, visitors, pageviews };
        } else {
            // 单天的小时数据处理
            const labels = Array.from({ length: 24 }, (_, i) => `${i}:00`);
            const visitors = new Array(24).fill(0);
            const pageviews = new Array(24).fill(0);

            // 根据timeRange获取日期
            const date = getDateByTimeRange(timeRange);

            // 如果有该日期的数据，填充小时数据
            if (statsData.days && statsData.days[date]) {
                const dayData = statsData.days[date];
                // 处理小时数据
                if (dayData.hourly) {
                    dayData.hourly.forEach(hourData => {
                        const hour = hourData.hour;
                        visitors[hour] = hourData.stats.uv || 0;
                        pageviews[hour] = hourData.stats.pv || 0;
                    });
                }
            }

            return { labels, visitors, pageviews };
        }
    } else if (viewType === 'daily') {
        // 处理按天的数据
        let startDate, endDate;

        switch (timeRange) {
            case 'today':
                startDate = new Date();
                endDate = new Date();
                break;
            case 'week':
                // 本周(周一到今天)
                startDate = getStartOfWeek(new Date());
                endDate = new Date();
                break;
            case 'last7days':
                // 最近7天(今天和前6天)
                endDate = new Date();
                startDate = new Date();
                startDate.setDate(endDate.getDate() - 6);
                break;
            case 'month':
                // 本月(1号到月底)
                startDate = new Date();
                startDate.setDate(1);
                // 获取当月最后一天
                endDate = new Date();
                const currentMonth = endDate.getMonth();
                const currentYear = endDate.getFullYear();
                // 设置为下个月的第0天，即当月最后一天
                endDate = new Date(currentYear, currentMonth + 1, 0);
                break;
            case 'last30days':
                // 最近30天(今天和前29天)
                endDate = new Date();
                startDate = new Date();
                startDate.setDate(endDate.getDate() - 29);
                break;
            default:
                // 默认显示最近7天
                endDate = new Date();
                startDate = new Date();
                startDate.setDate(endDate.getDate() - 6);
        }

        const days = getDaysArray(startDate, endDate);
        const labels = days.map(date => {
            const d = new Date(date);

            // 根据不同时间范围使用不同的日期格式
            if (timeRange === 'month' || timeRange === 'last30days') {
                return `${d.getMonth() + 1}.${d.getDate()}`;
            } else if (timeRange === 'week' || timeRange === 'last7days') {
                const dayNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
                return `${d.getMonth() + 1}.${d.getDate()} ${dayNames[d.getDay()]}`;
            } else {
                return `${d.getMonth() + 1}.${d.getDate()}`;
            }
        });

        const visitors = [];
        const pageviews = [];

        days.forEach(dateStr => {
            if (statsData.days && statsData.days[dateStr]) {
                // 使用新的数据结构中的total字段
                visitors.push(statsData.days[dateStr].total.uv || 0);
                pageviews.push(statsData.days[dateStr].total.pv || 0);
            } else {
                visitors.push(0);
                pageviews.push(0);
            }
        });

        return { labels, visitors, pageviews };
    }

    // 默认返回空数据
    return { labels: [], visitors: [], pageviews: [] };
}

// 更新整体统计数据
export function updateOverallStats(statsData, timeRange) {
    // 根据timeRange计算日期范围
    let startDate, endDate;
    const today = new Date();

    switch (timeRange) {
        case 'today':
            startDate = new Date();
            endDate = new Date();
            break;
        case 'week':
            startDate = getStartOfWeek(new Date());
            endDate = new Date();
            break;
        case 'last7days':
            endDate = new Date();
            startDate = new Date();
            startDate.setDate(endDate.getDate() - 6);
            break;
        case 'month':
            startDate = new Date();
            startDate.setDate(1);
            endDate = new Date();
            break;
        case 'last30days':
            endDate = new Date();
            startDate = new Date();
            startDate.setDate(endDate.getDate() - 29);
            break;
        default:
            endDate = new Date();
            startDate = new Date();
            startDate.setDate(endDate.getDate() - 6);
    }

    // 获取日期数组
    const days = getDaysArray(startDate, endDate);

    // 计算总和
    let totalUV = 0;
    let totalPV = 0;
    let totalTraffic = 0; // 单位为MB

    days.forEach(dateStr => {
        if (statsData.days && statsData.days[dateStr]) {
            // 如果有这一天的统计数据
            const dayData = statsData.days[dateStr];
            if (dayData.total) {
                totalUV += dayData.total.uv || 0;
                totalPV += dayData.total.pv || 0;
                totalTraffic += dayData.total.traffic || 0;
            }
        }
    });

    // 格式化流量显示
    const trafficDisplay = formatTraffic(totalTraffic);

    // 更新DOM
    document.getElementById('total-uv').textContent = totalUV.toLocaleString();
    document.getElementById('total-pv').textContent = totalPV.toLocaleString();
    document.getElementById('total-traffic').textContent = trafficDisplay;
}

// 渲染图表
export function renderChart(ctx, data, range, currentView) {
    // 清除错误消息
    const errorMsg = document.querySelector('.chart-error-message');
    if (errorMsg) {
        errorMsg.remove();
    }

    // 计算每个时间点的PV-UV差值（PV总是大于等于UV）
    const pvMinusUv = data.pageviews.map((pv, i) => pv - data.visitors[i]);

    // 设置x轴配置
    let xAxisConfig = {
        stacked: true,
        grid: {
            display: false
        },
    };

    // 如果是本周或最近7天的按小时视图，数据点会很多
    if ((range === 'week' || range === 'last7days' || range === 'last30days' || range === 'month')
        && currentView === 'hourly') {
        xAxisConfig = {
            stacked: true,
            grid: {
                display: false
            },
            ticks: {
                // 自定义标签显示，只显示日期，不显示小时
                callback: function (val, index) {
                    const label = this.getLabelForValue(val);
                    // 提取小时部分判断是否为0点（一天的开始）
                    const hourPart = parseInt(label.split(' ')[1]);
                    // 只在每天的0点（午夜）显示日期
                    if (hourPart === 0) {
                        // 只返回日期部分 (MM.DD)
                        return label.split(' ')[0];
                    }
                    return ''; // 其他小时不显示标签
                },
                maxRotation: 0, // 标签不旋转
                autoSkip: false // 不自动跳过标签
            }
        };
    }

    // 准备图表配置
    const chartConfig = {
        type: 'bar',
        data: {
            labels: data.labels,
            datasets: [
                {
                    label: '访客数(UV)',
                    data: data.visitors,
                    backgroundColor: '#1a5599', // 深蓝色
                    borderColor: '#1a5599',
                    borderWidth: 1,
                    stack: 'Stack 0',
                },
                {
                    label: '浏览量(PV)',  // 修改标签名称，更清晰
                    data: pvMinusUv,     // 初始时仍然使用差值
                    backgroundColor: '#7fb9ff', // 淡蓝色
                    borderColor: '#7fb9ff',
                    borderWidth: 1,
                    stack: 'Stack 0',
                    // 存储原始完整PV值，用于切换显示
                    originalData: data.pageviews
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
                x: xAxisConfig
            },
            plugins: {
                tooltip: {
                    callbacks: {
                        // 自定义提示信息，显示完整的PV和UV值以及日期时间
                        label: function (context) {
                            const datasetIndex = context.datasetIndex;
                            const index = context.dataIndex;
                            const fullLabel = data.labels[index];

                            if (datasetIndex === 0) {
                                return `${fullLabel} - 访客数(UV): ${data.visitors[index]}`;
                            } else {
                                return `${fullLabel} - 浏览量(PV): ${data.pageviews[index]}`;
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

                        // 获取当前数据集的可见状态
                        const meta = ci.getDatasetMeta(index);
                        const currentlyHidden = meta.hidden;

                        // 标准的图例点击行为 - 切换可见性
                        meta.hidden = !currentlyHidden;

                        // 特殊处理: 当UV数据集的可见性改变时
                        if (index === 0) { // UV数据集
                            const pvDataset = ci.data.datasets[1];
                            const pvMeta = ci.getDatasetMeta(1);

                            if (currentlyHidden) { // UV将要显示
                                // 如果UV要显示，则PV数据改回差值
                                pvDataset.data = pvMinusUv;
                            } else { // UV将要隐藏
                                // 如果UV要隐藏，则PV数据改为完整值
                                pvDataset.data = pvDataset.originalData;
                            }
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

// 更新图表
export function updateChart(ctx, statsData, range, view) {
    // 如果没有数据，不渲染图表
    if (!statsData) {
        return;
    }

    // 处理数据并渲染图表
    const chartData = processStatsData(statsData, range, view);
    renderChart(ctx, chartData, range, view);

    // 更新整体统计数据
    updateOverallStats(statsData, range);
}

// 更新视图切换按钮状态
export function updateViewToggleButtons(range, currentView, viewToggleBtns) {
    // 如果选择了"今天"，强制使用"按小时"视图并禁用"按天"按钮
    if (range === 'today') {
        const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
        const hourlyBtn = document.querySelector('.toggle-btn[data-view="hourly"]');

        dailyBtn.classList.add('disabled');
        dailyBtn.disabled = true;

        // 如果当前是按天视图，切换到按小时视图
        if (currentView === 'daily') {
            // 更新按钮状态
            viewToggleBtns.forEach(btn => btn.classList.remove('active'));
            hourlyBtn.classList.add('active');
            return 'hourly'; // 返回新的视图模式
        }
    } else {
        // 其他日期范围，启用"按天"按钮
        const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
        dailyBtn.classList.remove('disabled');
        dailyBtn.disabled = false;
    }
    return currentView; // 返回不变的视图模式
}