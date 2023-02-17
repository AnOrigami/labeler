package database

import (
	"database/sql"
	"time"
)

func SqlNullTime2TimeStamp(t sql.NullTime) int64 {
	if t.Valid {
		return t.Time.UnixMilli()
	}
	return 0
}

func SqlNullTimeToKitChen(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.Kitchen)
}
