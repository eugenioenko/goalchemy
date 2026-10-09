package time_test

import (
	"math"
	"math/rand"
	"testing"
	stdtime "time"

	"github.com/eugenioenko/goalchemy/std/time"
)

var parseInputs = []string{
	"2006-01-02T15:04:05Z", "2006-01-02T15:04:05+07:00", "2006-01-02T15:04:05-07:30", "2006-01-02T15:04:05.1Z",
	"2006-01-02T15:04:05.12Z", "2006-01-02T15:04:05.123Z", "2006-01-02T15:04:05.1234Z", "2006-01-02T15:04:05.12345Z",
	"2006-01-02T15:04:05.123456Z", "2006-01-02T15:04:05.1234567Z", "2006-01-02T15:04:05.12345678Z",
	"2006-01-02T15:04:05.123456789Z", "2006-01-02T15:04:05.1234567891Z", "2006-01-02T15:04:05.000000000Z",
	"2006-01-02T15:04:05,5Z", "2006-01-02T15:04:05.Z", "2006-01-02T15:04:05.5", "2024-02-29T00:00:00Z",
	"2023-02-29T00:00:00Z", "2000-02-29T23:59:59Z", "1900-02-29T00:00:00Z", "2006-04-31T00:00:00Z",
	"2006-13-01T00:00:00Z", "2006-00-01T00:00:00Z", "2006-01-00T00:00:00Z", "2006-01-32T00:00:00Z",
	"2006-01-02T24:00:00Z", "2006-01-02T23:60:00Z", "2006-01-02T23:59:60Z", "2006-01-02T5:04:05Z",
	"2006-01-02T15:4:05Z", "2006-01-02T15:04:5Z", "2006-1-02T15:04:05Z", "06-01-02T15:04:05Z",
	"0000-01-01T00:00:00Z", "9999-12-31T23:59:59.999999999Z", "0001-01-01T00:00:00Z",
	"1970-01-01T00:00:00+24:00", "1970-01-01T00:00:00+25:00", "1970-01-01T00:00:00+23:60",
	"1970-01-01T00:00:00+23:61", "1970-01-01T00:00:00*01:00", "1970-01-01T00:00:00+0100",
	"1970-01-01T00:00:00+01:0", "1970-01-01T00:00:00+0a:00", "1970-01-01T00:00:00z",
	"1970-01-01t00:00:00Z", "1970-01-01 00:00:00Z", "1970-01-01T00:00:00Zjunk", "1970-01-01T00:00:00",
	"", "x", "2006", "2006-01-02", "20a6-01-02T15:04:05Z", "2006-01-02T15:04:05.123+01:00",
	"2006-01-02T15:04:05,123-01:00", "1969-12-31T23:59:59.999999999Z", "2262-04-11T23:47:16.854775807Z",
	"2006-01-02T15:04:05Z\xff", "\"2006-01-02T15:04:05Z\"", "2006-01-02T15:04:05é",
}

func TestParseMatchesStd(t *testing.T) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		for _, s := range parseInputs {
			got, err := time.Parse(layout, s)
			want, werr := stdtime.Parse(layout, s)
			if (err == nil) != (werr == nil) {
				t.Fatalf("Parse(%q, %q) error %v, want %v", layout, s, err, werr)
			}
			if err != nil {
				if err.Error() != werr.Error() {
					t.Fatalf("Parse(%q, %q) error\n%s\nwant\n%s", layout, s, err, werr)
				}
				continue
			}
			if got.Unix() != want.Unix() || got.Nanosecond() != want.Nanosecond() {
				t.Fatalf("Parse(%q, %q) = %s, want %s", layout, s, got, want.UTC())
			}
		}
	}
	if _, err := time.Parse("2006-01-02", "2006-01-02"); err == nil {
		t.Fatal("unsupported layout accepted")
	}
}

func randomTime(r *rand.Rand) (time.Time, stdtime.Time) {
	var sec int64
	switch r.Intn(4) {
	case 0:
		sec = r.Int63n(1 << 32)
	case 1:
		sec = r.Int63n(253402300800+62135596800) - 62135596800
	case 2:
		sec = r.Int63n(1<<40) - 1<<39
	default:
		sec = r.Int63n(1<<36) - 1<<35
	}
	nsec := r.Int63n(1e9)
	if r.Intn(4) == 0 {
		nsec = r.Int63n(1000) * 1e6
	}
	return time.Unix(sec, nsec), stdtime.Unix(sec, nsec).UTC()
}

func same(t *testing.T, label string, got time.Time, want stdtime.Time) {
	t.Helper()
	if got.Unix() != want.Unix() || got.Nanosecond() != want.Nanosecond() {
		t.Fatalf("%s: got %d.%09d, want %d.%09d", label, got.Unix(), got.Nanosecond(), want.Unix(), want.Nanosecond())
	}
}

func TestTimeMatchesStd(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	durations := []time.Duration{1, 7, 1000, 250 * time.Millisecond, time.Second, 7 * time.Second, time.Minute,
		90 * time.Minute, time.Hour, 24 * time.Hour, 1234567891, time.Duration(math.MaxInt64), -1}
	for i := 0; i < 20000; i++ {
		g, w := randomTime(r)
		if g.Format(time.RFC3339) != w.Format(stdtime.RFC3339) || g.Format(time.RFC3339Nano) != w.Format(stdtime.RFC3339Nano) {
			t.Fatalf("Format %s: %s %s", w, g.Format(time.RFC3339Nano), w.Format(stdtime.RFC3339Nano))
		}
		if g.String() != w.String() {
			t.Fatalf("String: %s, want %s", g, w)
		}
		y, m, d := g.Date()
		wy, wm, wd := w.Date()
		h, mi, s := g.Clock()
		wh, wmi, ws := w.Clock()
		if y != wy || int(m) != int(wm) || d != wd || h != wh || mi != wmi || s != ws || g.YearDay() != w.YearDay() ||
			int(g.Weekday()) != int(w.Weekday()) || g.Month().String() != w.Month().String() || g.Weekday().String() != w.Weekday().String() {
			t.Fatalf("fields of %s", w)
		}
		if g.UnixMilli() != w.UnixMilli() || g.UnixMicro() != w.UnixMicro() || g.UnixNano() != w.UnixNano() {
			t.Fatalf("Unix units of %s", w)
		}
		if w.Year() >= 0 && w.Year() <= 9999 {
			p, err := time.Parse(time.RFC3339Nano, g.Format(time.RFC3339Nano))
			if err != nil || !p.Equal(g) {
				t.Fatalf("round trip %s: %v", g, err)
			}
		}
		dur := durations[r.Intn(len(durations))]
		same(t, "Truncate", g.Truncate(dur), w.Truncate(stdtime.Duration(dur)))
		same(t, "Round", g.Round(dur), w.Round(stdtime.Duration(dur)))
		same(t, "Add", g.Add(dur), w.Add(stdtime.Duration(dur)))
		same(t, "Add negative", g.Add(-dur), w.Add(-stdtime.Duration(dur)))
		years, months, days := r.Intn(41)-20, r.Intn(41)-20, r.Intn(800)-400
		same(t, "AddDate", g.AddDate(years, months, days), w.AddDate(years, months, days))
		g2, w2 := randomTime(r)
		if int64(g.Sub(g2)) != int64(w.Sub(w2)) || g.Compare(g2) != w.Compare(w2) || g.Before(g2) != w.Before(w2) || g.After(g2) != w.After(w2) {
			t.Fatalf("Sub/Compare %s %s", w, w2)
		}
	}
	for _, c := range [][8]int{{2024, 13, 1, 0, 0, 0, 0}, {2024, 0, 0, 0, 0, 0, 0}, {2024, 2, 30, 25, 61, 61, 1e9 + 5},
		{2024, -15, -40, -3, -70, -90, -5}, {1, 1, 1, 0, 0, 0, 0}, {-400, 3, 1, 0, 0, 0, 0}, {100000, 6, 15, 12, 0, 0, 0}} {
		same(t, "Date", time.Date(c[0], time.Month(c[1]), c[2], c[3], c[4], c[5], c[6], time.UTC),
			stdtime.Date(c[0], stdtime.Month(c[1]), c[2], c[3], c[4], c[5], c[6], stdtime.UTC))
	}
	var zero time.Time
	if !zero.IsZero() || zero.String() != (stdtime.Time{}).String() || zero.Unix() != (stdtime.Time{}).Unix() || zero.Weekday() != time.Monday {
		t.Fatalf("zero time %s", zero)
	}
	if time.Month(13).String() != stdtime.Month(13).String() || time.Weekday(-1).String() != stdtime.Weekday(-1).String() {
		t.Fatal("out-of-range names")
	}
}

func TestNow(t *testing.T) {
	before := stdtime.Now().UnixNano()
	n := time.Now().UnixNano()
	after := stdtime.Now().UnixNano()
	if n < before || n > after {
		t.Fatal(before, n, after)
	}
	if time.Since(time.Unix(0, before)) < 0 || time.Until(time.Unix(0, before)) > 0 {
		t.Fatal("Since/Until")
	}
}

var durationInputs = []string{
	"0", "-0", "+0", "", "-", "1", "1s", "-1s", "+1s", "1.5h", "1.h", ".5m", ".s", "-.s", "2h45m30.5s", "300ms",
	"1us", "1µs", "1μs", "1ns", "1x", "1sm", "1.0000000000000000000001s", "9223372036854775807ns",
	"9223372036854775808ns", "-9223372036854775808ns", "-9223372036854775809ns", "2562047h47m16.854775807s",
	"2562047h47m16.854775808s", "99999999999999999999h", "0.3333333333333333333h", "1h1h1h", "1e3s", "1..5s",
	"1 s", "\xffs", "3.000000001s",
}

func TestDurationMatchesStd(t *testing.T) {
	for _, s := range durationInputs {
		got, err := time.ParseDuration(s)
		want, werr := stdtime.ParseDuration(s)
		if (err == nil) != (werr == nil) || err != nil && err.Error() != werr.Error() || int64(got) != int64(want) {
			t.Fatalf("ParseDuration(%q) = %d, %v; want %d, %v", s, got, err, want, werr)
		}
	}
	r := rand.New(rand.NewSource(2))
	values := []int64{0, 1, -1, 999, 1000, 1001, 999999, 1000000, 1500000, 1e9, -1e9, 61e9, 3600e9 + 1,
		math.MaxInt64, math.MinInt64, math.MinInt64 + 1}
	for i := 0; i < 5000; i++ {
		values = append(values, r.Int63()>>uint(r.Intn(63))*int64(1-2*r.Intn(2)))
	}
	ms := []int64{0, -1, 1, 7, 1000, 1e6, 1e9, 60e9, 3600e9, math.MaxInt64}
	for _, v := range values {
		g, w := time.Duration(v), stdtime.Duration(v)
		if g.String() != w.String() || g.Seconds() != w.Seconds() || g.Minutes() != w.Minutes() || g.Hours() != w.Hours() ||
			g.Milliseconds() != w.Milliseconds() || g.Microseconds() != w.Microseconds() || int64(g.Abs()) != int64(w.Abs()) {
			t.Fatalf("Duration %d: %s want %s", v, g, w)
		}
		if v != math.MinInt64 {
			p, err := time.ParseDuration(g.String())
			if err != nil || p != g {
				t.Fatalf("round trip %s: %d %v", g, p, err)
			}
		}
		for _, m := range ms {
			if int64(g.Truncate(time.Duration(m))) != int64(w.Truncate(stdtime.Duration(m))) ||
				int64(g.Round(time.Duration(m))) != int64(w.Round(stdtime.Duration(m))) {
				t.Fatalf("Truncate/Round %d by %d", v, m)
			}
		}
	}
}
