package util

import (
	"fmt"
	"time"
)

// TimePeriod 根据时间范围字符串计算开始和结束时间
func TimePeriod(timeRange string) (time.Time, time.Time, error) {

	now := time.Now()
	endTime := setTime(now, 23, 59, 59) // 设置为当天最后一秒

	var startTime time.Time
	switch timeRange {
	case "today":
		startTime = setTime(now, 0, 0, 0)
	case "yesterday":
		startTime = setTime(now.AddDate(0, 0, -1), 0, 0, 0)
		endTime = setTime(now.AddDate(0, 0, -1), 23, 59, 59)
	case "week":
		startTime, endTime = weekBounds(now)
	case "last7days":
		startTime = setTime(now.AddDate(0, 0, -6), 0, 0, 0)
	case "month":
		startTime, endTime = monthBounds(now)
	case "last30days":
		startTime = setTime(now.AddDate(0, 0, -29), 0, 0, 0)
	default:
		startTime = setTime(now, 0, 0, 0)
	}

	return startTime, endTime, nil
}

// TimePointsAndLabels 根据时间范围类型和视图类型直接返回时间点数组和标签数组
func TimePointsAndLabels(
	timeRangeType string, viewType string) ([]time.Time, []string) {
	now := time.Now()

	var timePoints []time.Time
	var labels []string

	if timeRangeType == "today" {
		for hour := 0; hour <= 23; hour++ {
			hourTime := setTime(now, hour, 0, 0)
			timePoints = append(timePoints, hourTime)
			hourLabel := fmt.Sprintf("%d:00", hour)
			labels = append(labels, hourLabel)
		}
		return timePoints, labels
	} else if timeRangeType == "yesterday" {
		for hour := 0; hour <= 23; hour++ {
			hourTime := setTime(now.AddDate(0, 0, -1), hour, 0, 0)
			timePoints = append(timePoints, hourTime)
			hourLabel := fmt.Sprintf("%d:00", hour)
			labels = append(labels, hourLabel)
		}
		return timePoints, labels
	}

	var startDay, endDay time.Time
	switch timeRangeType {
	case "week":
		startDay, endDay = weekBounds(now)
	case "last7days":
		startDay = setTime(now.AddDate(0, 0, -6), 0, 0, 0)
		endDay = setTime(now, 23, 0, 0)
	case "month":
		startDay, endDay = monthBounds(now)
	case "last30days":
		startDay = setTime(now.AddDate(0, 0, -29), 0, 0, 0)
		endDay = setTime(now, 23, 0, 0)
	}

	includeWeekday := (viewType == "daily" && timeRangeType == "last7days") ||
		(viewType == "daily" && timeRangeType == "week")
	hourly := viewType == "hourly"

	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		dayLabel := FormatDateWithWeekday(day, includeWeekday)

		if hourly {
			for hour := range 24 {
				hourTime := setTime(day, hour, 0, 0)
				timePoints = append(timePoints, hourTime)
				labels = append(labels, dayLabel)
			}
		} else {
			timePoints = append(timePoints, day)
			labels = append(labels, dayLabel)
		}
	}

	return timePoints, labels
}

// FormatDateWithWeekday 返回格式化的日期字符串，可选是否包含星期
// 格式：M.D 或 M.D 周X
func FormatDateWithWeekday(date time.Time, includeWeekday bool) string {
	monthDay := fmt.Sprintf("%d.%d", date.Month(), date.Day())

	if includeWeekday {
		dayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
		return fmt.Sprintf("%s %s", monthDay, dayNames[date.Weekday()])
	}

	return monthDay
}

// weekBounds 返回包含指定日期的那一周的开始和结束时间
func weekBounds(t time.Time) (time.Time, time.Time) {
	// 获取包含 t 的那一周的周一
	weekday := t.Weekday()
	daysToMonday := 0

	if weekday == 0 { // 周日
		daysToMonday = 6
	} else {
		daysToMonday = int(weekday) - 1
	}

	monday := t.AddDate(0, 0, -daysToMonday)
	weekStart := setTime(monday, 0, 0, 0)
	sunday := monday.AddDate(0, 0, 6)
	weekEnd := setTime(sunday, 23, 0, 0)

	return weekStart, weekEnd
}

// monthBounds 返回指定日期所在月份的第一天和最后一天
func monthBounds(t time.Time) (time.Time, time.Time) {
	firstDay := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	nextMonth := firstDay.AddDate(0, 1, 0)
	lastDay := nextMonth.AddDate(0, 0, -1)
	lastDayEnd := setTime(lastDay, 23, 0, 0)

	return firstDay, lastDayEnd
}

// setTime 设置指定时间的时、分、秒，保留原日期
func setTime(t time.Time, hour, min, sec int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, min, sec, 0, t.Location())
}
