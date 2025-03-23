export async function fetchStatsData() {
    try {
        const response = await fetch('/api/stats-data');
        if (!response.ok) {
            throw new Error('网络响应不正常');
        }
        const data = await response.json();
        console.log('从服务器获取的数据:', data);
        return data;
    } catch (error) {
        console.error('获取统计数据失败:', error);
        throw error;
    }
}