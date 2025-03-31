// 主题管理器
export function initThemeManager() {
    // 检查本地存储中的主题设置
    const isDarkMode = localStorage.getItem('darkMode') === 'true';

    // 应用初始主题
    if (isDarkMode) {
        document.body.classList.add('dark-mode');
    }


    updateThemeToggleButton(isDarkMode);
    bindEventListeners();
}

function bindEventListeners() {
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }
}

// 切换主题
export function toggleTheme() {
    const isDarkMode = document.body.classList.contains('dark-mode');

    if (isDarkMode) {
        // 切换到浅色模式
        document.body.classList.remove('dark-mode');
        localStorage.setItem('darkMode', 'false');
    } else {
        // 切换到深色模式
        document.body.classList.add('dark-mode');
        localStorage.setItem('darkMode', 'true');
    }

    updateThemeToggleButton(!isDarkMode);
    updateChartsTheme();
}

// 更新主题切换按钮状态
function updateThemeToggleButton(isDarkMode) {
    const themeToggle = document.getElementById('theme-toggle');
    if (!themeToggle) return;

    if (isDarkMode) {
        themeToggle.classList.add('dark-mode');
    } else {
        themeToggle.classList.remove('dark-mode');
    }
}

// 更新图表主题
export function updateChartsTheme() {
    // 获取当前主题是否为深色模式
    const isDarkMode = document.body.classList.contains('dark-mode');

    // 如果地图实例存在，更新地图主题
    if (window.geoMapChart) {
        const theme = isDarkMode ? {
            visualMap: {
                inRange: { color: ['#2a5769', '#7eb9ff'] }
            }
        } : {
            visualMap: {
                inRange: { color: ['#e0ffff', '#006edd'] }
            }
        };

        window.geoMapChart.setOption(theme, false);
    }
}