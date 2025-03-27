import { fetchWebsites } from './api.js';
import { saveUserPreference, getUserPreference } from './utils.js';
import { displayErrorMessage } from './charts.js';


export async function initWebsiteSelector(selector, onWebsiteSelected) {
    try {
        // 获取网站列表
        const websites = await fetchWebsites();

        // 清空网站选择器
        selector.innerHTML = '';

        if (websites.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = '没有可用的网站';
            selector.appendChild(option);
            return '';
        }

        // 填充网站选项
        websites.forEach(website => {
            const option = document.createElement('option');
            option.value = website.id;
            option.textContent = website.name;
            selector.appendChild(option);
        });

        // 尝试从localStorage获取上次选择的网站
        const lastSelected = getUserPreference('selectedWebsite', '');
        let currentWebsiteId = '';

        if (lastSelected && selector.querySelector(`option[value="${lastSelected}"]`)) {
            selector.value = lastSelected;
            currentWebsiteId = lastSelected;
        } else {
            // 如果没有保存的选择或者保存的选择不在列表中，选择第一个网站
            currentWebsiteId = websites[0].id;
            selector.value = currentWebsiteId;
        }

        // 设置变更事件监听器
        selector.addEventListener('change', function () {
            const websiteId = this.value;
            saveUserPreference('selectedWebsite', websiteId);

            if (typeof onWebsiteSelected === 'function') {
                onWebsiteSelected(websiteId);
            }
        });

        return currentWebsiteId;
    } catch (error) {
        console.error('初始化网站选择器失败:', error);
        return '';
    }
}