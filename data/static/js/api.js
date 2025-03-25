export async function fetchWebsites() {
    try {
        const response = await fetch('/api/websites');
        if (!response.ok) {
            throw new Error('网络响应不正常');
        }
        const data = await response.json();
        return data.websites || [];
    } catch (error) {
        console.error('获取网站列表失败:', error);
        throw error;
    }
}

export async function fetchStatsData(websiteId, timeRange, viewType) {
    try {
        // 创建URL参数对象
        const params = new URLSearchParams();
        params.append('id', websiteId);
        params.append('timeRange', timeRange);
        params.append('viewType', viewType);

        const response = await fetch(`/api/stats-data?${params.toString()}`);

        if (!response.ok) {
            throw new Error('网络响应不正常');
        }
        const statsData = await response.json();

        return statsData;
    } catch (error) {
        console.error('获取统计数据失败:', error);
        throw error;
    }
}