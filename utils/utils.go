package utils

import (
	"fmt"
	"time"
)

func FormatCleanupTime(duration time.Duration) string {
	days := duration / (time.Hour * 24)
	hours := (duration % (time.Hour * 24)) / time.Hour
	minutes := (duration % time.Hour) / time.Minute
	seconds := (duration % time.Minute) / time.Second
	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}