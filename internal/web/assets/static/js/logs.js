import {
    fetchLogs,
    fetchWebsites
} from './api.js';

import {
    formatTraffic,
    saveUserPreference,
    getUserPreference
} from './utils.js';

import {
    initThemeManager,
} from './theme.js';

// 状态变量
let currentWebsiteId = '';
let currentPage = 1;
let pageSize = 100;
let totalPages = 0;
let sortField = 'timestamp';
let sortOrder = 'desc';
let searchFilter = '';

// DOM 元素
let websiteSelector;
let searchInput;
let searchButton;
let sortFieldSelect;
let sortOrderSelect;
let pageSizeSelect;
let logsTable;
let prevPageBtn;
let nextPageBtn;
let currentPageSpan;
let totalPagesSpan;
let pageJumpInput;
let pageJumpBtn;

// 初始化应用
async function initApp() {
    // 获取DOM元素
    websiteSelector = document.getElementById('website-selector');
    searchInput = document.getElementById('logs-search');
    searchButton = document.getElementById('search-btn');
    sortFieldSelect = document.getElementById('sort-field');
    sortOrderSelect = document.getElementById('sort-order');
    pageSizeSelect = document.getElementById('page-size');
    logsTable = document.getElementById('logs-table');
    prevPageBtn = document.getElementById('prev-page');
    nextPageBtn = document.getElementById('next-page');
    currentPageSpan = document.getElementById('current-page');
    totalPagesSpan = document.getElementById('total-pages');
    pageJumpInput = document.getElementById('page-jump-input');
    pageJumpBtn = document.getElementById('page-jump-btn');

    // 初始化主题
    initThemeManager();

    // 初始化网站选择器
    await initWebsiteSelector();

    // 从本地存储获取保存的设置
    pageSize = parseInt(getUserPreference('logsPageSize', '100'));
    sortField = getUserPreference('logsSortField', 'timestamp');
    sortOrder = getUserPreference('logsSortOrder', 'desc');

    // 设置下拉框默认值
    pageSizeSelect.value = pageSize;
    sortFieldSelect.value = sortField;
    sortOrderSelect.value = sortOrder;

    // 绑定事件
    bindEventListeners();

    // 加载日志数据
    loadLogs();
}

// 初始化网站选择器
async function initWebsiteSelector() {
    try {
        // 获取网站列表
        const websites = await fetchWebsites();

        // 清空网站选择器
        websiteSelector.innerHTML = '';

        if (websites.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = '没有可用的网站';
            websiteSelector.appendChild(option);
            return;
        }

        // 填充网站选项
        websites.forEach(website => {
            const option = document.createElement('option');
            option.value = website.id;
            option.textContent = website.name;
            websiteSelector.appendChild(option);
        });

        // 尝试从localStorage获取上次选择的网站
        const lastSelected = getUserPreference('selectedWebsite', '');

        if (lastSelected && websiteSelector.querySelector(`option[value="${lastSelected}"]`)) {
            websiteSelector.value = lastSelected;
            currentWebsiteId = lastSelected;
        } else {
            currentWebsiteId = websites[0].id;
            websiteSelector.value = currentWebsiteId;
        }
    } catch (error) {
        console.error('初始化网站选择器失败:', error);
        alert('无法加载网站列表，请刷新页面重试。');
    }
}

// 绑定事件监听器
function bindEventListeners() {
    // 网站选择变化
    websiteSelector.addEventListener('change', function () {
        currentWebsiteId = this.value;
        saveUserPreference('selectedWebsite', currentWebsiteId);
        currentPage = 1;
        loadLogs();
    });

    // 搜索按钮点击
    searchButton.addEventListener('click', function () {
        searchFilter = searchInput.value.trim();
        currentPage = 1;
        loadLogs();
    });

    // 搜索框回车
    searchInput.addEventListener('keyup', function (event) {
        if (event.key === 'Enter') {
            searchFilter = this.value.trim();
            currentPage = 1;
            loadLogs();
        }
    });

    // 排序字段变化
    sortFieldSelect.addEventListener('change', function () {
        sortField = this.value;
        saveUserPreference('logsSortField', sortField);
        loadLogs();
    });

    // 排序顺序变化
    sortOrderSelect.addEventListener('change', function () {
        sortOrder = this.value;
        saveUserPreference('logsSortOrder', sortOrder);
        loadLogs();
    });

    // 每页行数变化
    pageSizeSelect.addEventListener('change', function () {
        pageSize = parseInt(this.value);
        saveUserPreference('logsPageSize', pageSize);
        currentPage = 1;
        loadLogs();
    });

    // 上一页按钮
    prevPageBtn.addEventListener('click', function () {
        if (currentPage > 1) {
            currentPage--;
            loadLogs();
        }
    });

    // 下一页按钮
    nextPageBtn.addEventListener('click', function () {
        if (currentPage < totalPages) {
            currentPage++;
            loadLogs();
        }
    });

    // 页面跳转按钮
    pageJumpBtn.addEventListener('click', function () {
        const pageNum = parseInt(pageJumpInput.value);
        if (pageNum && pageNum > 0 && pageNum <= totalPages) {
            currentPage = pageNum;
            loadLogs();
        } else {
            alert(`请输入有效的页码 (1-${totalPages})`);
        }
    });

    // 页面跳转输入框回车
    pageJumpInput.addEventListener('keyup', function (event) {
        if (event.key === 'Enter') {
            pageJumpBtn.click();
        }
    });
}

// 更新加载日志数据函数
async function loadLogs() {
    if (!currentWebsiteId) {
        displayError('请先选择网站');
        return;
    }

    // 显示加载状态
    const tableBody = logsTable.querySelector('tbody');
    tableBody.innerHTML = '<tr class="loading-row"><td colspan="11">加载中...</td></tr>';

    // 禁用分页按钮
    updatePaginationControls(true);

    try {
        // 请求日志数据，使用新的API函数
        const data = await fetchLogs(
            currentWebsiteId,
            currentPage,
            pageSize,
            sortField,
            sortOrder,
            searchFilter
        );

        // 更新分页信息
        totalPages = data.pagination.pages;
        currentPage = data.pagination.page;
        updatePaginationControls();

        // 渲染日志表格
        renderLogsTable(data.logs);
    } catch (error) {
        console.error('加载日志数据失败:', error);
        displayError('加载日志数据失败，请重试');
    }
}

// 渲染日志表格
function renderLogsTable(logs) {
    const tableBody = logsTable.querySelector('tbody');
    tableBody.innerHTML = '';

    if (!logs || logs.length === 0) {
        tableBody.innerHTML = '<tr class="loading-row"><td colspan="11">没有找到日志数据</td></tr>';
        return;
    }

    logs.forEach(log => {
        const row = document.createElement('tr');

        // 时间列
        let cell = document.createElement('td');
        cell.textContent = log.time;
        cell.title = log.time;
        row.appendChild(cell);

        // IP列
        cell = document.createElement('td');
        cell.textContent = log.ip;
        cell.title = log.ip;
        row.appendChild(cell);

        // 位置列
        cell = document.createElement('td');
        const location = log.domestic_location || log.global_location || '-';
        cell.textContent = location;
        cell.title = location;
        row.appendChild(cell);

        // 请求列
        cell = document.createElement('td');
        const request = `${log.method} ${log.url}`;
        cell.textContent = request;
        cell.title = request;
        row.appendChild(cell);

        // 状态码列
        cell = document.createElement('td');
        cell.textContent = log.status_code;
        if (log.status_code >= 400) {
            cell.style.color = 'var(--error-color)';
        } else if (log.status_code >= 300) {
            cell.style.color = 'var(--warning-color)';
        } else {
            cell.style.color = 'var(--success-color)';
        }
        row.appendChild(cell);

        // 流量列
        cell = document.createElement('td');
        cell.textContent = formatTraffic(log.bytes_sent);
        cell.title = `${log.bytes_sent} 字节`;
        row.appendChild(cell);

        // 来源列
        cell = document.createElement('td');
        cell.textContent = log.referer || '-';
        cell.title = log.referer || '-';
        row.appendChild(cell);

        // 浏览器列
        cell = document.createElement('td');
        cell.textContent = log.user_browser || '-';
        cell.title = log.user_browser || '-';
        row.appendChild(cell);

        // 系统列
        cell = document.createElement('td');
        cell.textContent = log.user_os || '-';
        cell.title = log.user_os || '-';
        row.appendChild(cell);

        // 设备列
        cell = document.createElement('td');
        cell.textContent = log.user_device || '-';
        cell.title = log.user_device || '-';
        row.appendChild(cell);

        // PV列
        cell = document.createElement('td');
        cell.textContent = log.pageview_flag ? '✓' : '-';
        cell.style.textAlign = 'center';
        if (log.pageview_flag) {
            cell.style.color = 'var(--success-color)';
        }
        row.appendChild(cell);

        tableBody.appendChild(row);
    });
}

// 更新分页控件
function updatePaginationControls(loading = false) {
    // 更新当前页和总页数显示
    currentPageSpan.textContent = currentPage;
    totalPagesSpan.textContent = totalPages;

    // 启用/禁用上一页按钮
    prevPageBtn.disabled = loading || currentPage <= 1;

    // 启用/禁用下一页按钮
    nextPageBtn.disabled = loading || currentPage >= totalPages;

    // 更新页面跳转输入框
    pageJumpInput.disabled = loading;
    pageJumpBtn.disabled = loading;

    if (!loading) {
        pageJumpInput.max = totalPages;
        pageJumpInput.placeholder = `1-${totalPages}`;
    }
}

// 显示错误信息
function displayError(message) {
    const tableBody = logsTable.querySelector('tbody');
    tableBody.innerHTML = `<tr class="loading-row"><td colspan="11">${message}</td></tr>`;
}

// 页面加载时启动应用
document.addEventListener('DOMContentLoaded', initApp);