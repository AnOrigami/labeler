package util

import (
	"database/sql"
	"time"
)

func init() {
	time.Local = time.FixedZone("CST", 8*3600)
}

const (
	TimeLayoutDatetimeN = "2006-01-02 15:04:05.9"
	TimeLayoutDatetime  = "2006-01-02 15:04:05"
)

type Datetime time.Time

func (d *Datetime) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}

func (d *Datetime) SqlNullTime() sql.NullTime {
	t := d.Time()
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{
		Time:  *t,
		Valid: true,
	}
}

func (d Datetime) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	if t.IsZero() {
		return []byte("null"), nil
	}
	s := t.Format(TimeLayoutDatetime)
	return append([]byte(`"`), append([]byte(s), '"')...), nil
}

func (d *Datetime) UnmarshalJSON(data []byte) error {
	// 双引号或者null
	if len(data) <= 4 {
		return nil
	}
	t, err := ParseDatetime(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*d = Datetime(t)
	return nil
}

func ParseDatetime(s string) (time.Time, error) {
	return time.ParseInLocation(TimeLayoutDatetimeN, s, time.Local)
}

func DurationSecs(d time.Duration) int64 {
	return int64(d / time.Second)
}

func SqlNullTimeToTimeFormat(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}
