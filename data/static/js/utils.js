export function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}


export function getDaysArray(start, end) {
    const arr = [];
    const dt = new Date(start);

    while (dt <= end) {
        arr.push(formatDate(new Date(dt)));
        dt.setDate(dt.getDate() + 1);
    }

    return arr;
}

export function getStartOfWeek(date) {
    const d = new Date(date);
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1);
    return new Date(d.setDate(diff));
}

export function getEndOfWeek(date) {
    const d = new Date(date);
    const day = d.getDay();
    const diff = d.getDate() + (day === 0 ? 0 : 7 - day);
    return new Date(d.setDate(diff));
}

export function getDateByTimeRange(timeRange) {
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


export function getRangeText(range) {
    switch (range) {
        case 'today': return '今天';
        case 'week': return '本周';
        case 'last7days': return '最近7天';
        case 'month': return '本月';
        case 'last30days': return '最近30天';
        default: return '全部';
    }
}

// 格式化流量数据
export function formatTraffic(traffic) {
    if (traffic < 1024) {
        return traffic.toFixed(2) + ' B';
    } else if (traffic < 1024 * 1024) {
        return (traffic / 1024).toFixed(2) + ' KB';
    } else if (traffic < 1024 * 1024 * 1024) {
        return (traffic / (1024 * 1024)).toFixed(2) + ' MB';
    } else if (traffic < 1024 * 1024 * 1024 * 1024) {
        return (traffic / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
    } else {
        return (traffic / (1024 * 1024 * 1024 * 1024)).toFixed(2) + ' TB';
    }
}

// 设置加载状态相关函数
export function setLoadingState(isLoading) {
    const statsElements = [
        document.getElementById('total-uv'),
        document.getElementById('total-pv'),
        document.getElementById('total-traffic')
    ];

    statsElements.forEach(element => {
        if (element) {
            element.textContent = isLoading ? '加载中...' : '-';
        }
    });
}

// 重置统计信息
export function resetStatistics() {
    const statsElements = [
        document.getElementById('total-uv'),
        document.getElementById('total-pv'),
        document.getElementById('total-traffic')
    ];

    statsElements.forEach(element => {
        if (element) {
            element.textContent = '-';
        }
    });
}

// 保存用户选择到本地存储
export function saveUserPreference(key, value) {
    localStorage.setItem(key, value);
}

// 从本地存储获取用户选择
export function getUserPreference(key, defaultValue) {
    const saved = localStorage.getItem(key);
    return saved || defaultValue;
}