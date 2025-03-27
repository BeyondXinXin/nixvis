export function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
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

// 保存用户选择到本地存储
export function saveUserPreference(key, value) {
    localStorage.setItem(key, value);
}

// 从本地存储获取用户选择
export function getUserPreference(key, defaultValue) {
    const saved = localStorage.getItem(key);
    return saved || defaultValue;
}