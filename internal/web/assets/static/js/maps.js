import {
    fectchLocationStats,
} from './api.js';

import {
    updateChartsTheme,
} from './theme.js';

// 初始化地图实例
let geoMapChart = null;
let currentMapView = 'china'; // 默认显示中国地图
let currentWebsiteId = '';
let range = 'today';

let zhWrodNameMap = { "Afghanistan": "阿富汗", "Singapore": "新加坡", "Angola": "安哥拉", "Albania": "阿尔巴尼亚", "United Arab Emirates": "阿联酋", "Argentina": "阿根廷", "Armenia": "亚美尼亚", "French Southern and Antarctic Lands": "法属南半球和南极领地", "Australia": "澳大利亚", "Austria": "奥地利", "Azerbaijan": "阿塞拜疆", "Burundi": "布隆迪", "Belgium": "比利时", "Benin": "贝宁", "Burkina Faso": "布基纳法索", "Bangladesh": "孟加拉国", "Bulgaria": "保加利亚", "The Bahamas": "巴哈马", "Bosnia and Herzegovina": "波斯尼亚和黑塞哥维那", "Belarus": "白俄罗斯", "Belize": "伯利兹", "Bermuda": "百慕大", "Bolivia": "玻利维亚", "Brazil": "巴西", "Brunei": "文莱", "Bhutan": "不丹", "Botswana": "博茨瓦纳", "Central African Republic": "中非共和国", "Canada": "加拿大", "Switzerland": "瑞士", "Chile": "智利", "China": "中国", "Ivory Coast": "象牙海岸", "Cameroon": "喀麦隆", "Democratic Republic of the Congo": "刚果民主共和国", "Republic of the Congo": "刚果共和国", "Colombia": "哥伦比亚", "Costa Rica": "哥斯达黎加", "Cuba": "古巴", "Northern Cyprus": "北塞浦路斯", "Cyprus": "塞浦路斯", "Czech Republic": "捷克共和国", "Germany": "德国", "Djibouti": "吉布提", "Denmark": "丹麦", "Dominican Republic": "多明尼加共和国", "Algeria": "阿尔及利亚", "Ecuador": "厄瓜多尔", "Egypt": "埃及", "Eritrea": "厄立特里亚", "Spain": "西班牙", "Estonia": "爱沙尼亚", "Ethiopia": "埃塞俄比亚", "Finland": "芬兰", "Fiji": "斐", "Falkland Islands": "福克兰群岛", "France": "法国", "Gabon": "加蓬", "United Kingdom": "英国", "Georgia": "格鲁吉亚", "Ghana": "加纳", "Guinea": "几内亚", "Gambia": "冈比亚", "Guinea Bissau": "几内亚比绍", "Greece": "希腊", "Greenland": "格陵兰", "Guatemala": "危地马拉", "French Guiana": "法属圭亚那", "Guyana": "圭亚那", "Honduras": "洪都拉斯", "Croatia": "克罗地亚", "Haiti": "海地", "Hungary": "匈牙利", "Indonesia": "印度尼西亚", "India": "印度", "Ireland": "爱尔兰", "Iran": "伊朗", "Iraq": "伊拉克", "Iceland": "冰岛", "Israel": "以色列", "Italy": "意大利", "Jamaica": "牙买加", "Jordan": "约旦", "Japan": "日本", "Kazakhstan": "哈萨克斯坦", "Kenya": "肯尼亚", "Kyrgyzstan": "吉尔吉斯斯坦", "Cambodia": "柬埔寨", "Kosovo": "科索沃", "Kuwait": "科威特", "Laos": "老挝", "Lebanon": "黎巴嫩", "Liberia": "利比里亚", "Libya": "利比亚", "Sri Lanka": "斯里兰卡", "Lesotho": "莱索托", "Lithuania": "立陶宛", "Luxembourg": "卢森堡", "Latvia": "拉脱维亚", "Morocco": "摩洛哥", "Moldova": "摩尔多瓦", "Madagascar": "马达加斯加", "Mexico": "墨西哥", "Macedonia": "马其顿", "Mali": "马里", "Myanmar": "缅甸", "Montenegro": "黑山", "Mongolia": "蒙古", "Mozambique": "莫桑比克", "Mauritania": "毛里塔尼亚", "Malawi": "马拉维", "Malaysia": "马来西亚", "Namibia": "纳米比亚", "New Caledonia": "新喀里多尼亚", "Niger": "尼日尔", "Nigeria": "尼日利亚", "Nicaragua": "尼加拉瓜", "Netherlands": "荷兰", "Norway": "挪威", "Nepal": "尼泊尔", "New Zealand": "新西兰", "Oman": "阿曼", "Pakistan": "巴基斯坦", "Panama": "巴拿马", "Peru": "秘鲁", "Philippines": "菲律宾", "Papua New Guinea": "巴布亚新几内亚", "Poland": "波兰", "Puerto Rico": "波多黎各", "North Korea": "北朝鲜", "Portugal": "葡萄牙", "Paraguay": "巴拉圭", "Qatar": "卡塔尔", "Romania": "罗马尼亚", "Russia": "俄罗斯", "Rwanda": "卢旺达", "Western Sahara": "西撒哈拉", "Saudi Arabia": "沙特阿拉伯", "Sudan": "苏丹", "South Sudan": "南苏丹", "Senegal": "塞内加尔", "Solomon Islands": "所罗门群岛", "Sierra Leone": "塞拉利昂", "El Salvador": "萨尔瓦多", "Somaliland": "索马里兰", "Somalia": "索马里", "Republic of Serbia": "塞尔维亚", "Suriname": "苏里南", "Slovakia": "斯洛伐克", "Slovenia": "斯洛文尼亚", "Sweden": "瑞典", "Swaziland": "斯威士兰", "Syria": "叙利亚", "Chad": "乍得", "Togo": "多哥", "Thailand": "泰国", "Tajikistan": "塔吉克斯坦", "Turkmenistan": "土库曼斯坦", "East Timor": "东帝汶", "Trinidad and Tobago": "特里尼达和多巴哥", "Tunisia": "突尼斯", "Turkey": "土耳其", "United Republic of Tanzania": "坦桑尼亚", "Uganda": "乌干达", "Ukraine": "乌克兰", "Uruguay": "乌拉圭", "United States": "美国", "Uzbekistan": "乌兹别克斯坦", "Venezuela": "委内瑞拉", "Vietnam": "越南", "Vanuatu": "瓦努阿图", "West Bank": "西岸", "Yemen": "也门", "South Africa": "南非", "Zambia": "赞比亚", "Korea": "韩国", "Tanzania": "坦桑尼亚", "Zimbabwe": "津巴布韦", "Congo": "刚果", "Central African Rep.": "中非", "Serbia": "塞尔维亚", "Bosnia and Herz.": "波斯尼亚和黑塞哥维那", "Czech Rep.": "捷克", "W. Sahara": "西撒哈拉", "Lao PDR": "老挝", "Dem.Rep.Korea": "朝鲜", "Falkland Is.": "福克兰群岛", "Timor-Leste": "东帝汶", "Solomon Is.": "所罗门群岛", "Palestine": "巴勒斯坦", "N. Cyprus": "北塞浦路斯", "Aland": "奥兰群岛", "Fr. S. Antarctic Lands": "法属南半球和南极陆地", "Mauritius": "毛里求斯", "Comoros": "科摩罗", "Eq. Guinea": "赤道几内亚", "Guinea-Bissau": "几内亚比绍", "Dominican Rep.": "多米尼加", "Saint Lucia": "圣卢西亚", "Dominica": "多米尼克", "Antigua and Barb.": "安提瓜和巴布达", "U.S. Virgin Is.": "美国原始岛屿", "Montserrat": "蒙塞拉特", "Grenada": "格林纳达", "Barbados": "巴巴多斯", "Samoa": "萨摩亚", "Bahamas": "巴哈马", "Cayman Is.": "开曼群岛", "Faeroe Is.": "法罗群岛", "IsIe of Man": "马恩岛", "Malta": "马耳他共和国", "Jersey": "泽西", "Cape Verde": "佛得角共和国", "Turks and Caicos Is.": "特克斯和凯科斯群岛", "St. Vin. and Gren.": "圣文森特和格林纳丁斯", "Singapore Rep.": "新加坡", "Côte d'Ivoire": "科特迪瓦", "Siachen Glacier": "锡亚琴冰川", "Br. Indian Ocean Ter.": "英属印度洋领土", "Dem. Rep. Congo": "刚果民主共和国", "Dem. Rep. Korea": "朝鲜", "S. Sudan": "南苏丹" }


// 更新 WebsiteId 和 range
export function updateGeoMapWebsiteIdAndRange(websiteId, newRange) {
    currentWebsiteId = websiteId;
    range = newRange;
    updateGeoMap();
}

// 初始化地图
export function initGeoMap() {
    if (!document.getElementById('geo-map')) {
        console.error('找不到地图容器元素');
        return;
    }

    // 初始化ECharts实例
    geoMapChart = echarts.init(document.getElementById('geo-map'));
    window.geoMapChart = geoMapChart; // 方便调试时使用

    // 绑定地图视图切换事件
    bindMapViewToggle();
}

// 绑定地图视图切换事件
function bindMapViewToggle() {
    const mapToggleBtns = document.querySelectorAll('.data-map-toggle-btn');

    mapToggleBtns.forEach(btn => {
        btn.addEventListener('click', function () {
            currentMapView = this.dataset.mapView;

            mapToggleBtns.forEach(b => b.classList.remove('active'));
            this.classList.add('active');

            updateGeoMap();
        });
    });
}

// 渲染中国地图
function renderChinaMap(geoData) {
    if (!geoData || geoData.length === 0) {
        return;
    }

    const maxValue = geoData[0].value;
    const option = {
        tooltip: {
            trigger: 'item',
            formatter: function (params) {
                let value = 0;
                if (params.value !== undefined && params.value !== null && !isNaN(params.value)) {
                    value = params.value;
                }
                return `${params.name}<br/>访问量: ${value.toLocaleString()}`;
            }
        },
        visualMap: {
            backgroundColor: 'transparent',
            min: -5,
            max: maxValue,
            left: 'left',
            bottom: '10%',
            calculable: false
        },
        geo: {
            map: 'china',
            roam: true,
            label: {
                show: false
            },
            regions: [{
                name: '南海诸岛',
                selected: false,
                itemStyle: {
                    areaColor: 'transparent',
                    opacity: 0
                }
            }]
        },
        series: [{
            name: '访问量',
            type: 'map',
            map: 'china',
            geoIndex: 0,
            data: geoData
        }]

    };

    geoMapChart.setOption(option, true);
}

// 渲染世界地图
function renderWorldMap(geoData) {
    if (!geoData || geoData.length === 0) {
        return;
    }
    const maxValue = geoData.length > 0 ? geoData[0].value : 10;

    const option = {
        tooltip: {
            trigger: 'item',
            formatter: function (params) {
                let value = 0;
                if (params.value !== undefined && params.value !== null && !isNaN(params.value)) {
                    value = params.value;
                }
                return `${params.name}<br/>访问量: ${value.toLocaleString()}`;
            }
        },
        visualMap: {
            backgroundColor: 'transparent',
            min: -5,
            max: maxValue,
            left: 'left',
            bottom: '10%',
            calculable: false
        },
        series: [{
            name: '访问量',
            type: 'map',
            map: 'world',
            nameMap: zhWrodNameMap,
            roam: true,
            label: {
                show: false
            },
            data: geoData
        }]
    };

    geoMapChart.setOption(option, true);
}

// 更新地区排名表格
function updateGeoRankingTable(data) {
    const tableBody = document.querySelector('#geo-ranking-table tbody');

    // 清空表格内容
    tableBody.innerHTML = '';

    if (!data || data.length === 0) {
        const row = document.createElement('tr');
        row.classList.add('loading-row');
        row.innerHTML = '<td colspan="2">暂无数据</td>';
        tableBody.appendChild(row);
        return;
    }

    // 填充表格数据
    data.forEach((item) => {
        const row = document.createElement('tr');
        const percentage = item.percentage || 0;
        row.innerHTML = `
            <td class="item-path" title="${item.name}">${item.name}</td>
            <td class="item-count">
                <div class="bar-container">
                    <span class="bar-label">${item.value.toLocaleString()}</span>
                    <div class="bar">
                        <div class="bar-fill" style="width: ${percentage}%;"></div>
                        <span class="bar-percentage">${percentage}%</span>
                    </div>
                </div>
            </td>`;

        tableBody.appendChild(row);
    });
}

// 修改 updateGeoMap 函数来同时更新地图和排名表
async function updateGeoMap() {
    let geoData;

    if (currentMapView === 'china') {
        const statsData = await fectchLocationStats(currentWebsiteId, range, "domestic", 99)
        geoData = statsData.key.map((location, index) => ({
            name: location,
            value: statsData.uv[index],
            percentage: statsData.uv_percent[index]
        })).filter(item => item.name !== '国外' && item.name !== '未知');

        renderChinaMap(geoData);
    } else {
        const statsData = await fectchLocationStats(currentWebsiteId, range, "global", 99)
        geoData = statsData.key.map((location, index) => ({
            name: location,
            value: statsData.uv[index],
            percentage: statsData.uv_percent[index]
        })).filter(item => item.name !== '国外' && item.name !== '未知');

        renderWorldMap(geoData);
    }
    updateChartsTheme();

    // 更新地区排名表格,去前10
    geoData = geoData.slice(0, 10);
    updateGeoRankingTable(geoData);
}