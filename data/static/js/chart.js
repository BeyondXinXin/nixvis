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

    // 图表实例
    let visitsChart;

    // 获取完整的统计数据
    function fetchStatsData() {
        fetch('/api/stats-data')
            .then(response => {
                if (!response.ok) {
                    throw new Error('网络响应不正常');
                }
                return response.json();
            })
            .then(data => {
                console.log('从服务器获取的数据:', data);
                statsDataCache = data;
                updateChart(); // 获取数据后更新图表
            })
            .catch(error => {
                console.error('获取统计数据失败:', error);
                // 显示错误消息，不使用备用数据
                displayErrorMessage('无法获取统计数据，请稍后再试');
            });
    }

    // 显示错误消息
    function displayErrorMessage(message) {
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

    // 为日期选择下拉框添加事件
    dateRange.addEventListener('change', function () {
        // 如果没有数据，不执行任何操作
        if (!statsDataCache) {
            return;
        }
        updateChart();
    });

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
        renderChart(chartData, range);

        // 新增：更新整体统计数据
        updateOverallStats(statsDataCache, range);
    }

    // 处理statsData为图表所需格式
    function processStatsData(statsData, timeRange, viewType) {
        // 处理按小时的数据
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

    // 计算和显示整体统计
    function updateOverallStats(statsData, timeRange) {
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
        let trafficDisplay = '';
        if (totalTraffic < 1024) {
            trafficDisplay = totalTraffic.toFixed(2) + ' B';
        } else if (totalTraffic < 1024 * 1024) {
            trafficDisplay = (totalTraffic / 1024).toFixed(2) + ' KB';
        } else if (totalTraffic < 1024 * 1024 * 1024) {
            trafficDisplay = (totalTraffic / (1024 * 1024)).toFixed(2) + ' MB';
        } else if (totalTraffic < 1024 * 1024 * 1024 * 1024) {
            trafficDisplay = (totalTraffic / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
        } else {
            trafficDisplay = (totalTraffic / (1024 * 1024 * 1024 * 1024)).toFixed(2) + ' TB';
        }

        // 更新DOM
        document.getElementById('total-uv').textContent = totalUV.toLocaleString();
        document.getElementById('total-pv').textContent = totalPV.toLocaleString();
        document.getElementById('total-traffic').textContent = trafficDisplay;
    }

    // 获取周一日期
    function getStartOfWeek(date) {
        const d = new Date(date);
        const day = d.getDay();
        const diff = d.getDate() - day + (day === 0 ? -6 : 1); // 调整周日的情况
        return new Date(d.setDate(diff));
    }

    // 获取两个日期之间的所有日期
    function getDaysArray(start, end) {
        const arr = [];
        const dt = new Date(start);

        while (dt <= end) {
            arr.push(formatDate(new Date(dt)));
            dt.setDate(dt.getDate() + 1);
        }

        return arr;
    }

    // 渲染图表
    function renderChart(data, range) {
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

    // 格式化日期为YYYY-MM-DD
    function formatDate(date) {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }

    // 根据时间范围获取日期
    function getDateByTimeRange(timeRange) {
        const today = new Date();

        switch (timeRange) {
            case 'today':
                return formatDate(today);
            case 'week':
                // 周视图显示今天的数据
                return formatDate(today);
            case 'last7days':
                // 显示今天的数据
                return formatDate(today);
            case 'month':
                // 月视图显示今天的数据
                return formatDate(today);
            case 'last30days':
                // 显示今天的数据
                return formatDate(today);
            default:
                return formatDate(today);
        }
    }

    // 获取日期范围的显示文本
    function getRangeText(range) {
        switch (range) {
            case 'today': return '今天';
            case 'week': return '本周';
            case 'last7days': return '最近7天';
            case 'month': return '本月';
            case 'last30days': return '最近30天';
            default: return '全部';
        }
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


    // 检查初始日期范围并设置按钮状态
    function checkInitialDateRange() {
        const initialRange = dateRange.value;
        if (initialRange === 'today') {
            const dailyBtn = document.querySelector('.toggle-btn[data-view="daily"]');
            dailyBtn.classList.add('disabled');
            dailyBtn.disabled = true;
        }
    }

    checkInitialDateRange();
    fetchStatsData();
});


