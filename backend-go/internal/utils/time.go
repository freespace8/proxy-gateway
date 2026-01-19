package utils

import "time"

// NextLocalMidnight 返回“下一次本地时区的 00:00:00”。
// 例如：今天 23:59 -> 明天 00:00；今天 00:00 -> 明天 00:00（始终取未来的午夜）。
func NextLocalMidnight(now time.Time) time.Time {
	loc := now.Location()
	y, m, d := now.In(loc).Date()
	todayMidnight := time.Date(y, m, d, 0, 0, 0, 0, loc)
	return todayMidnight.Add(24 * time.Hour)
}
