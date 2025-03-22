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

    // 示例数据 - 实际应用中应该从服务器获取
    const sampleData = {
        hourly: {
            labels: Array.from({ length: 24 }, (_, i) => `${i}:00`),
            visitors: [42, 25, 18, 10, 5, 3, 2, 15, 45, 68, 75, 82, 78, 85, 90, 92, 88, 75, 65, 58, 47, 35, 28, 20],
            pageviews: [85, 52, 35, 22, 12, 8, 5, 30, 90, 130, 145, 160, 155, 170, 185, 190, 175, 150, 130, 115, 95, 70, 55, 40]
        },
        daily: {
            labels: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
            visitors: [580, 620, 650, 710, 780, 850, 750],
            pageviews: [1200, 1300, 1350, 1450, 1600, 1750, 1550]
        }
    };

    // 图表实例
    let visitsChart;

    // 为切换按钮添加事件
    viewToggleBtns.forEach(btn => {
        btn.addEventListener('click', function () {
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
    dateRange.addEventListener('change', updateChart);

    // 更新图表函数
    function updateChart() {
        // 获取当前选择的日期范围
        const range = dateRange.value;

        // 根据当前视图获取数据
        const data = sampleData[currentView];

        // 计算每个时间点的PV-UV差值（PV总是大于等于UV）
        const pvMinusUv = data.pageviews.map((pv, i) => pv - data.visitors[i]);

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
                        label: '页面浏览(PV-UV)',
                        data: pvMinusUv,
                        backgroundColor: '#7fb9ff', // 淡蓝色
                        borderColor: '#7fb9ff',
                        borderWidth: 1,
                        stack: 'Stack 0',
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false, // 改为true以保持纵横比
                aspectRatio: 2, // 设置纵横比为2:1
                scales: {
                    y: {
                        beginAtZero: true,
                        stacked: true,
                    },
                    x: {
                        stacked: true,
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: `${getRangeText(range)} - ${currentView === 'hourly' ? '按小时' : '按天'}统计`,
                        font: {
                            size: 16
                        }
                    },
                    tooltip: {
                        callbacks: {
                            // 自定义提示信息，显示完整的PV和UV值
                            label: function (context) {
                                const datasetIndex = context.datasetIndex;
                                const index = context.dataIndex;

                                if (datasetIndex === 0) {
                                    return `访客数(UV): ${data.visitors[index]}`;
                                } else {
                                    return `浏览量(PV): ${data.pageviews[index]}`;
                                }
                            }
                        }
                    },
                    legend: {
                        position: 'bottom', // 将图例放置在图表下方
                        align: 'center',    // 居中对齐
                        labels: {
                            padding: 20,    // 为图例标签添加一些内边距
                            boxWidth: 15,   // 减小图例色块的宽度
                            usePointStyle: true, // 使用点状样式可以让图例更紧凑
                            // 自定义图例，使其更易于理解
                            generateLabels: function (chart) {
                                const originalLabels = Chart.defaults.plugins.legend.labels.generateLabels(chart);
                                // 修改第二个数据集的标签文本
                                if (originalLabels.length > 1) {
                                    originalLabels[1].text = '浏览量(PV)';
                                }
                                return originalLabels;
                            }
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

    // 初始化图表
    updateChart();
});