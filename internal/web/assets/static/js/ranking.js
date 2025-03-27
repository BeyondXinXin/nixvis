// 更新引荐来源排名表格
export function updaterefererRankingTable(data) {
    updateClientTable('referer-ranking-table', data.url, data.url_overall);
}

// 更新浏览器统计表格
export function updateBrowserTable(data) {
    updateClientTable('browser-ranking-table', data.url, data.url_overall);
}

// 更新操作系统统计表格
export function updateOsTable(data) {
    updateClientTable('os-ranking-table', data.url, data.url_overall);
}

// 更新设备统计表格
export function updateDeviceTable(data) {
    updateClientTable('device-ranking-table', data.url, data.url_overall);
}

// 更新URL排名表格
export function updateUrlRankingTable(urlStats) {
    updateClientTable('url-ranking-table', urlStats.url, urlStats.url_overall, true);
}


// 通用客户端表格更新函数 - 简化版本
function updateClientTable(tableId, itemLabs, itemOverall, showPv = false) {
    const tableBody = document.querySelector(`#${tableId} tbody`);

    // 清空表格内容
    tableBody.innerHTML = '';

    if (!itemLabs || !itemOverall || itemLabs.length === 0) {
        const row = document.createElement('tr');
        row.classList.add('loading-row');
        row.innerHTML = '<td colspan="2">暂无数据</td>';
        tableBody.appendChild(row);
        return;
    }

    // 填充表格数据
    itemLabs.forEach((itemlab, index) => {
        const stats = itemOverall[index];
        const row = document.createElement('tr');

        if (showPv) {
            row.innerHTML = `
                <td class="item-path" title="${itemlab}">${itemlab}</td>
                <td>${stats.uv.toLocaleString()}</td>
                <td>${stats.pv.toLocaleString()}</td>`;
        } else {
            row.innerHTML = `
                <td class="item-path" title="${itemlab}">${itemlab}</td>
                <td>${stats.uv.toLocaleString()}</td>`;
        }
        tableBody.appendChild(row);
    });
}