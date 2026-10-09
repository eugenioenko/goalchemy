package main

import (
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/time"
)

func show(t time.Time) string {
	return t.Format(time.RFC3339Nano) + " " + strconv.FormatInt(t.Unix(), 10) + "." + strconv.Itoa(t.Nanosecond())
}

func main() {
	inputs := []string{
		"2006-01-02T15:04:05Z", "2006-01-02T15:04:05.1Z", "2006-01-02T15:04:05.123456789+07:00",
		"2006-01-02T15:04:05.1234567891-07:30", "2006-01-02T15:04:05,5Z", "2006-01-02T5:04:05Z",
		"2024-02-29T23:59:59.999999999Z", "2023-02-29T00:00:00Z", "1900-02-29T00:00:00Z", "2000-02-29T00:00:00Z",
		"0000-01-01T00:00:00Z", "9999-12-31T23:59:59.999999999-24:00", "1970-01-01T00:00:00+25:00",
		"1970-01-01T00:00:00+23:61", "2006-13-01T00:00:00Z", "2006-01-02T24:00:00Z", "2006-01-02T23:59:60Z",
		"2006-01-02T15:04:05", "2006-01-02T15:04:05Zjunk", "2006-01-02 15:04:05Z", "20a6-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z\xff", "",
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		for _, s := range inputs {
			t, err := time.Parse(layout, s)
			if err != nil {
				println("err", err.Error())
				continue
			}
			println("ok", show(t), t.Format(time.RFC3339))
		}
	}
	_, err := time.Parse("2006-01-02", "2006-01-02")
	println(err.Error())

	secs := []int64{0, 1, -1, 951782400, 1709251199, -62135596800, 253402300799, -86400 * 366, 4102444800, 1 << 35}
	nanos := []int64{0, 1, 500000000, 999999999, 120000000, 1500000000, -1}
	for i, s := range secs {
		t := time.Unix(s, nanos[i%len(nanos)])
		y, m, d := t.Date()
		h, mi, sec := t.Clock()
		println(show(t), t.String(), y, m.String(), d, h, mi, sec, t.Weekday().String(), t.YearDay())
		println(t.UnixMilli(), t.UnixMicro(), t.IsZero())
		for _, dur := range []time.Duration{time.Nanosecond, 7 * time.Millisecond, time.Second, 90 * time.Minute, 24 * time.Hour, 1234567891} {
			println(show(t.Truncate(dur)), show(t.Round(dur)), show(t.Add(dur)), show(t.Add(-dur)))
		}
		println(show(t.AddDate(1, -14, 400)), show(t.AddDate(-3, 2, -31)))
		u := time.Unix(secs[(i+3)%len(secs)], 250)
		println(t.Sub(u).String(), int64(u.Sub(t)), t.Before(u), t.After(u), t.Equal(u), t.Compare(u))
	}
	println(show(time.UnixMilli(-1500)), show(time.UnixMicro(1700000000123456)))
	println(show(time.Date(2024, 2, 30, 25, 61, 61, 1000000005, time.UTC)), show(time.Date(2024, -15, -40, -3, -70, -90, -5, time.UTC)))
	println(show(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), time.Time{}.IsZero(), time.Time{}.String())
	println(time.Month(13).String(), time.Weekday(-1).String(), time.UTC.String())

	durations := []time.Duration{0, 1, -1, 999, 1000, 1500, 1000000, 1500000, time.Second, -61 * time.Second,
		3*time.Hour + 4*time.Minute + 5*time.Second + 6, 1<<63 - 1, -1 << 63}
	for _, d := range durations {
		println(d.String(), d.Seconds(), d.Minutes(), d.Hours(), d.Milliseconds(), d.Microseconds(), d.Abs().String())
		println(d.Truncate(time.Millisecond).String(), d.Round(time.Second).String(), d.Round(7).String())
	}
	for _, s := range []string{"1h2m3.5s", "-1.5h", "300ms", "1µs", "1us", ".5m", "1", "1x", "", "-", "9223372036854775808ns",
		"-9223372036854775808ns", "0.3333333333333333333h", "+0", "1.s"} {
		d, err := time.ParseDuration(s)
		if err != nil {
			println("err", err.Error())
			continue
		}
		println("ok", d.String(), int64(d))
	}

	now := time.Now()
	println(now.Unix() > 1700000000, now.Year() >= 2024, time.Since(now) >= 0, time.Until(now) <= 0, now.Location().String())
}
