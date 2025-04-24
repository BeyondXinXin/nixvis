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

// 查询接口
async function fetchStats(type, params = {}) {
    try {
        const queryParams = new URLSearchParams();

        // 添加所有参数到查询字符串
        Object.entries(params).forEach(([key, value]) => {
            if (value !== undefined && value !== null) {
                queryParams.append(key, value);
            }
        });

        const url = `/api/stats/${type}?${queryParams.toString()}`;
        const response = await fetch(url);

        if (!response.ok) {
            throw new Error(`请求失败，状态码: ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        console.error(`获取${type}统计数据失败:`, error);
        throw error;
    }
}

// 以下是针对不同统计类型的专用函数
export async function fetchTimeSeriesStats(websiteId, timeRange, viewType) {
    return fetchStats('timeseries', { id: websiteId, timeRange, viewType });
}

export async function fetchOverallStats(websiteId, timeRange) {
    return fetchStats('overall', { id: websiteId, timeRange });
}

export async function fetchUrlStats(websiteId, timeRange, limit = 10) {
    return fetchStats('url', { id: websiteId, timeRange, limit });
}

export async function fetchRefererStats(websiteId, timeRange, limit = 10) {
    return fetchStats('referer', { id: websiteId, timeRange, limit });
}

export async function fetchBrowserStats(websiteId, timeRange, limit = 10) {
    return fetchStats('browser', { id: websiteId, timeRange, limit });
}

export async function fetchOSStats(websiteId, timeRange, limit = 10) {
    return fetchStats('os', { id: websiteId, timeRange, limit });
}

export async function fetchDeviceStats(websiteId, timeRange, limit = 10) {
    return fetchStats('device', { id: websiteId, timeRange, limit });
}

export async function fectchLocationStats(websiteId, timeRange, locationType, limit = 99) {
    return fetchStats('location', { id: websiteId, locationType, timeRange, limit });
}

export async function fetchLogs(websiteId, page, pageSize, sortField, sortOrder, filter) {
    const params = {
        id: websiteId,
        page: page,
        pageSize: pageSize,
        sortField: sortField,
        sortOrder: sortOrder
    };

    if (filter) {
        params.filter = filter;
    }

    return fetchStats('logs', params);
}

