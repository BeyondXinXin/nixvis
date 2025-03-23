export async function fetchWebsites() {
    try {
        const response = await fetch('/api/websites');
        if (!response.ok) {
            throw new Error('网络响应不正常');
        }
        const data = await response.json();
        console.log('从服务器获取的网站列表:', data);
        return data.websites || [];
    } catch (error) {
        console.error('获取网站列表失败:', error);
        throw error;
    }
}

export async function fetchStatsData(websiteId) {
    if (!websiteId) {
        throw new Error('必须提供网站ID');
    }

    try {
        const response = await fetch(`/api/stats-data?id=${encodeURIComponent(websiteId)}`);
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