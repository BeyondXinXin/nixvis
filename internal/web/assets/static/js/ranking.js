// 更新引荐来源排名表格
export function updaterefererRankingTable(data) {
    updateClientTable('referer-ranking-table', data);
}

// 更新浏览器统计表格
export function updateBrowserTable(data) {
    updateClientTable('browser-ranking-table', data);
}

// 更新操作系统统计表格
export function updateOsTable(data) {
    updateClientTable('os-ranking-table', data);
}

// 更新设备统计表格
export function updateDeviceTable(data) {
    updateClientTable('device-ranking-table', data);
}

// 更新URL排名表格
export function updateUrlRankingTable(data) {
    updateClientTable('url-ranking-table', data, true);
}


// 通用客户端表格更新函数 - 简化版本
function updateClientTable(tableId, data, showPv = false) {
    const tableBody = document.querySelector(`#${tableId} tbody`);

    // 清空表格内容
    tableBody.innerHTML = '';

    const itemLabs = data.key || [];
    const itemUV = data.uv || [];
    const itemUvPercent = data.uv_percent;

    if (!data || itemLabs.length === 0 || itemUV.length === 0) {
        const row = document.createElement('tr');
        row.classList.add('loading-row');
        row.innerHTML = '<td colspan="2">暂无数据</td>';
        tableBody.appendChild(row);
        return;
    }

    // 填充表格数据
    itemLabs.forEach((itemlab, index) => {
        const row = document.createElement('tr');
        if (showPv) {
            const itemPV = data.pv || [];
            const itemPvPercent = data.pv_percent[index] || 0;
            const percentage = itemUvPercent[index] || 0;
            row.innerHTML = `
                <td class="item-path" title="${itemlab}">${itemlab}</td>
                <td class="item-count">
                    <div class="bar-container">
                        <span class="bar-label">${itemUV[index]}</span>
                        <div class="bar">
                            <div class="bar-fill" style="width: ${itemPvPercent}%;"></div>
                            <span class="bar-percentage">${itemPvPercent}%</span>
                        </div>
                    </div>
                </td>
                <td class="item-count">
                    <div class="bar-container">
                        <span class="bar-label">${itemPV[index]}</span>
                        <div class="bar">
                            <div class="bar-fill" style="width: ${percentage}%;"></div>
                            <span class="bar-percentage">${percentage}%</span>
                        </div>
                    </div>
                </td>`;

        } else {
            row.innerHTML = `
                <td class="item-path" title="${itemlab}">${itemlab}</td>
                <td class="item-count">${itemUV[index].toLocaleString()}</td>`;
            const percentage = itemUvPercent[index] || 0;
            row.innerHTML = `
                <td class="item-path" title="${itemlab}">${itemlab}</td>
                <td class="item-count">
                    <div class="bar-container">
                        <span class="bar-label">${itemUV[index]}</span>
                        <div class="bar">
                            <div class="bar-fill" style="width: ${percentage}%;"></div>
                            <span class="bar-percentage">${percentage}%</span>
                        </div>
                    </div>
                </td>`;
        }
        tableBody.appendChild(row);
    });
}