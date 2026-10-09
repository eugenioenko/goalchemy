// Package time measures and displays time. Times are always UTC and carry no
// monotonic reading: Now reads the host wall clock, and Format and Parse
// accept only the RFC3339 and RFC3339Nano layouts.
package time

import (
	"github.com/eugenioenko/goalchemy/lib/clock"
	"github.com/eugenioenko/goalchemy/lib/errors"
)

// A Duration is the elapsed time between two instants as an int64 nanosecond
// count. It is distinct from lib/time.Duration, which drives the scheduler
// clock; convert between them explicitly.
type Duration int64

const (
	minDuration Duration = -1 << 63
	maxDuration Duration = 1<<63 - 1
)

// Common durations.
const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// Layouts accepted by Format and Parse.
const (
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
)

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour

	unixToInternal int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay
	internalToUnix int64 = -unixToInternal
)

// A Month specifies a month of the year (January = 1, ...).
type Month int

// Months of the year.
const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

var monthNames = []string{"January", "February", "March", "April", "May", "June", "July",
	"August", "September", "October", "November", "December"}

// String returns the English name of the month ("January", "February", ...).
func (m Month) String() string {
	if January <= m && m <= December {
		return monthNames[m-1]
	}
	return "%!Month(" + formatUint(uint64(m)) + ")"
}

// A Weekday specifies a day of the week (Sunday = 0, ...).
type Weekday int

// Days of the week.
const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// String returns the English name of the day ("Sunday", "Monday", ...).
func (d Weekday) String() string {
	if Sunday <= d && d <= Saturday {
		return dayNames[d]
	}
	return "%!Weekday(" + formatUint(uint64(d)) + ")"
}

// A Location names a time zone. Only UTC exists.
type Location struct {
	name string
}

var utcLoc = Location{name: "UTC"}

// UTC is Coordinated Universal Time, the location of every Time.
var UTC *Location = &utcLoc

// String returns the location name.
func (l *Location) String() string { return "UTC" }

// A Time is an instant with nanosecond precision, always in UTC. The zero
// value is January 1, year 1, 00:00:00 UTC.
type Time struct {
	sec  int64
	nsec int64
}

// Now returns the current host wall-clock time. It has no monotonic reading,
// so Since and Sub follow host clock adjustments.
func Now() Time { return Unix(0, clock.UnixNano()) }

// Unix returns the Time corresponding to sec seconds and nsec nanoseconds
// since January 1, 1970 UTC. nsec may be outside [0, 999999999].
func Unix(sec int64, nsec int64) Time {
	if nsec < 0 || nsec >= 1e9 {
		n := nsec / 1e9
		sec += n
		nsec -= n * 1e9
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
	}
	return Time{sec: sec + unixToInternal, nsec: nsec}
}

// UnixMilli returns the Time corresponding to msec milliseconds since the
// Unix epoch.
func UnixMilli(msec int64) Time { return Unix(msec/1e3, (msec%1e3)*1e6) }

// UnixMicro returns the Time corresponding to usec microseconds since the
// Unix epoch.
func UnixMicro(usec int64) Time { return Unix(usec/1e6, (usec%1e6)*1e3) }

// Date returns the Time for the given date and time of day. Values outside
// their usual ranges are normalized, as in Go; loc must be non-nil.
func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
	if loc == nil {
		panic("time: missing Location in call to Date")
	}
	m := int(month) - 1
	year, m = norm(year, m, 12)
	sec, nsec = norm(sec, nsec, 1e9)
	min, sec = norm(min, sec, 60)
	hour, min = norm(hour, min, 60)
	day, hour = norm(day, hour, 24)
	days := daysFromCivil(int64(year), int64(m+1), 1) + int64(day-1)
	unix := days*secondsPerDay + int64(hour*secondsPerHour+min*secondsPerMinute+sec)
	return Time{sec: unix + unixToInternal, nsec: int64(nsec)}
}

func norm(hi, lo, base int) (int, int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}

// Since returns the time elapsed since t; it is Now().Sub(t).
func Since(t Time) Duration { return Now().Sub(t) }

// Until returns the duration until t; it is t.Sub(Now()).
func Until(t Time) Duration { return t.Sub(Now()) }

// IsZero reports whether t is the zero time, January 1, year 1, 00:00:00 UTC.
func (t Time) IsZero() bool { return t.sec == 0 && t.nsec == 0 }

// After reports whether t is after u.
func (t Time) After(u Time) bool { return t.sec > u.sec || t.sec == u.sec && t.nsec > u.nsec }

// Before reports whether t is before u.
func (t Time) Before(u Time) bool { return t.sec < u.sec || t.sec == u.sec && t.nsec < u.nsec }

// Equal reports whether t and u are the same instant.
func (t Time) Equal(u Time) bool { return t.sec == u.sec && t.nsec == u.nsec }

// Compare returns -1 if t is before u, 0 if they are equal, and +1 if t is
// after u.
func (t Time) Compare(u Time) int {
	switch {
	case t.Before(u):
		return -1
	case t.After(u):
		return 1
	}
	return 0
}

// UTC returns t; every Time is already in UTC.
func (t Time) UTC() Time { return t }

// In returns t; only UTC exists. It panics if loc is nil.
func (t Time) In(loc *Location) Time {
	if loc == nil {
		panic("time: missing Location in call to Time.In")
	}
	return t
}

// Location returns UTC.
func (t Time) Location() *Location { return UTC }

// Unix returns t as seconds since January 1, 1970 UTC.
func (t Time) Unix() int64 { return t.sec + internalToUnix }

// UnixMilli returns t as milliseconds since January 1, 1970 UTC.
func (t Time) UnixMilli() int64 { return t.Unix()*1e3 + t.nsec/1e6 }

// UnixMicro returns t as microseconds since January 1, 1970 UTC.
func (t Time) UnixMicro() int64 { return t.Unix()*1e6 + t.nsec/1e3 }

// UnixNano returns t as nanoseconds since January 1, 1970 UTC. The result
// overflows outside the years 1678 to 2262.
func (t Time) UnixNano() int64 { return t.Unix()*1e9 + t.nsec }

// Nanosecond returns the nanosecond offset within the second, in [0, 999999999].
func (t Time) Nanosecond() int { return int(t.nsec) }

func (t Time) days() (int64, int64) {
	unix := t.sec + internalToUnix
	days := unix / secondsPerDay
	rem := unix % secondsPerDay
	if rem < 0 {
		days--
		rem += secondsPerDay
	}
	return days, rem
}

// Date returns the year, month and day of t.
func (t Time) Date() (int, Month, int) {
	days, _ := t.days()
	y, m, d := civilFromDays(days)
	return int(y), Month(m), int(d)
}

// Clock returns the hour, minute and second within the day of t.
func (t Time) Clock() (int, int, int) {
	_, rem := t.days()
	return int(rem / secondsPerHour), int(rem % secondsPerHour / secondsPerMinute), int(rem % secondsPerMinute)
}

// Year returns the year of t.
func (t Time) Year() int {
	y, _, _ := t.Date()
	return y
}

// Month returns the month of the year of t.
func (t Time) Month() Month {
	_, m, _ := t.Date()
	return m
}

// Day returns the day of the month of t.
func (t Time) Day() int {
	_, _, d := t.Date()
	return d
}

// Hour returns the hour within the day of t, in [0, 23].
func (t Time) Hour() int {
	h, _, _ := t.Clock()
	return h
}

// Minute returns the minute within the hour of t, in [0, 59].
func (t Time) Minute() int {
	_, m, _ := t.Clock()
	return m
}

// Second returns the second within the minute of t, in [0, 59].
func (t Time) Second() int {
	_, _, s := t.Clock()
	return s
}

// Weekday returns the day of the week of t.
func (t Time) Weekday() Weekday {
	days, _ := t.days()
	w := (days + 4) % 7
	if w < 0 {
		w += 7
	}
	return Weekday(w)
}

// YearDay returns the day of the year of t, in [1, 365] or [1, 366] in leap
// years.
func (t Time) YearDay() int {
	days, _ := t.days()
	y, _, _ := civilFromDays(days)
	return int(days-daysFromCivil(y, 1, 1)) + 1
}

// Add returns t+d. Results beyond the representable range saturate.
func (t Time) Add(d Duration) Time {
	dsec := int64(d / 1e9)
	nsec := t.nsec + int64(d%1e9)
	if nsec >= 1e9 {
		dsec++
		nsec -= 1e9
	} else if nsec < 0 {
		dsec--
		nsec += 1e9
	}
	t.nsec = nsec
	sum := t.sec + dsec
	if (sum > t.sec) == (dsec > 0) {
		t.sec = sum
	} else if dsec > 0 {
		t.sec = 1<<63 - 1
	} else {
		t.sec = -(1<<63 - 1)
	}
	return t
}

// Sub returns t-u, saturating at the minimum or maximum Duration.
func (t Time) Sub(u Time) Duration {
	d := Duration(t.sec-u.sec)*Second + Duration(t.nsec-u.nsec)
	switch {
	case u.Add(d).Equal(t):
		return d
	case t.Before(u):
		return minDuration
	}
	return maxDuration
}

// AddDate returns t with the given numbers of years, months and days added,
// normalized like Date.
func (t Time) AddDate(years int, months int, days int) Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return Date(year+years, month+Month(months), day+days, hour, min, sec, int(t.nsec), UTC)
}

// Truncate returns t rounded down to a multiple of d since the zero time.
func (t Time) Truncate(d Duration) Time {
	if d <= 0 {
		return t
	}
	return t.Add(-t.mod(d))
}

// Round returns t rounded to the nearest multiple of d since the zero time,
// rounding halfway values up.
func (t Time) Round(d Duration) Time {
	if d <= 0 {
		return t
	}
	r := t.mod(d)
	if lessThanHalf(r, d) {
		return t.Add(-r)
	}
	return t.Add(d - r)
}

func (t Time) mod(d Duration) Duration {
	neg := false
	sec := t.sec
	nsec := t.nsec
	if sec < 0 {
		neg = true
		sec = -sec
		nsec = -nsec
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
	}
	var r Duration
	switch {
	case d < Second && Second%(d+d) == 0:
		r = Duration(nsec % int64(d))
	case d%Second == 0:
		d1 := int64(d / Second)
		r = Duration(sec%d1)*Second + Duration(nsec)
	default:
		s := uint64(sec)
		tmp := (s >> 32) * 1e9
		u1 := tmp >> 32
		u0 := tmp << 32
		tmp = (s & 0xFFFFFFFF) * 1e9
		prev := u0
		u0 += tmp
		if u0 < prev {
			u1++
		}
		prev = u0
		u0 += uint64(nsec)
		if u0 < prev {
			u1++
		}
		d1 := uint64(d)
		for d1>>63 != 1 {
			d1 <<= 1
		}
		d0 := uint64(0)
		for {
			if u1 > d1 || u1 == d1 && u0 >= d0 {
				prev = u0
				u0 -= d0
				if u0 > prev {
					u1--
				}
				u1 -= d1
			}
			if d1 == 0 && d0 == uint64(d) {
				break
			}
			d0 >>= 1
			d0 |= (d1 & 1) << 63
			d1 >>= 1
		}
		r = Duration(u0)
	}
	if neg && r != 0 {
		r = d - r
	}
	return r
}

func lessThanHalf(x, y Duration) bool { return uint64(x)+uint64(x) < uint64(y) }

func daysFromCivil(y, m, d int64) int64 {
	if m <= 2 {
		y--
	}
	era := y
	if era < 0 {
		era -= 399
	}
	era /= 400
	yoe := y - era*400
	mp := m - 3
	if m <= 2 {
		mp = m + 9
	}
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func civilFromDays(z int64) (int64, int64, int64) {
	z += 719468
	era := z
	if era < 0 {
		era -= 146096
	}
	era /= 146097
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if mp >= 10 {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

func daysIn(m Month, year int) int {
	switch m {
	case February:
		if isLeap(year) {
			return 29
		}
		return 28
	case April, June, September, November:
		return 30
	}
	return 31
}

// String returns t formatted as "2006-01-02 15:04:05.999999999 +0000 UTC".
func (t Time) String() string {
	b := t.appendDate(nil, ' ')
	b = appendFraction(b, t.nsec)
	return string(b) + " +0000 UTC"
}

// Format returns t in layout, which must be RFC3339 or RFC3339Nano; any other
// layout panics. The offset is always written as Z.
func (t Time) Format(layout string) string {
	return string(t.AppendFormat(nil, layout))
}

// AppendFormat is like Format but appends to b.
func (t Time) AppendFormat(b []byte, layout string) []byte {
	if layout != RFC3339 && layout != RFC3339Nano {
		panic("time: layout " + quote(layout) + " is not supported; use RFC3339 or RFC3339Nano")
	}
	b = t.appendDate(b, 'T')
	if layout == RFC3339Nano {
		b = appendFraction(b, t.nsec)
	}
	return append(b, 'Z')
}

func (t Time) appendDate(b []byte, sep byte) []byte {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	b = appendInt(b, year, 4)
	b = append(b, '-')
	b = appendInt(b, int(month), 2)
	b = append(b, '-')
	b = appendInt(b, day, 2)
	b = append(b, sep)
	b = appendInt(b, hour, 2)
	b = append(b, ':')
	b = appendInt(b, min, 2)
	b = append(b, ':')
	return appendInt(b, sec, 2)
}

func appendInt(b []byte, x int, width int) []byte {
	u := uint64(x)
	if x < 0 {
		b = append(b, '-')
		u = uint64(-x)
	}
	digits := formatUint(u)
	for i := len(digits); i < width; i++ {
		b = append(b, '0')
	}
	for i := 0; i < len(digits); i++ {
		b = append(b, digits[i])
	}
	return b
}

func appendFraction(b []byte, nsec int64) []byte {
	if nsec == 0 {
		return b
	}
	digits := make([]byte, 9)
	for i := 8; i >= 0; i-- {
		digits[i] = byte(nsec%10) + '0'
		nsec /= 10
	}
	n := 9
	for n > 0 && digits[n-1] == '0' {
		n--
	}
	b = append(b, '.')
	return append(b, digits[:n]...)
}

func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 20)
	w := len(buf)
	for v > 0 {
		w--
		buf[w] = byte(v%10) + '0'
		v /= 10
	}
	return string(buf[w:])
}

// ParseError describes a problem parsing a time string.
type ParseError struct {
	Layout     string
	Value      string
	LayoutElem string
	ValueElem  string
	Message    string
}

// Error returns the string representation of a ParseError.
func (e *ParseError) Error() string {
	if e.Message == "" {
		return "parsing time " + quote(e.Value) + " as " + quote(e.Layout) + ": cannot parse " +
			quote(e.ValueElem) + " as " + quote(e.LayoutElem)
	}
	return "parsing time " + quote(e.Value) + e.Message
}

type chunk struct {
	prefix string
	elem   string
}

var rfc3339Chunks = []chunk{{"", "2006"}, {"-", "01"}, {"-", "02"}, {"T", "15"}, {":", "04"}, {":", "05"}, {"", "Z07:00"}}

var rfc3339NanoChunks = []chunk{{"", "2006"}, {"-", "01"}, {"-", "02"}, {"T", "15"}, {":", "04"}, {":", "05"},
	{"", ".999999999"}, {"", "Z07:00"}}

// Parse parses value in layout, which must be RFC3339 or RFC3339Nano, and
// returns the instant in UTC: a numeric offset is applied, then dropped. Both
// layouts accept and reject the same inputs as Go, with the same errors.
func Parse(layout, value string) (Time, error) {
	var chunks []chunk
	switch layout {
	case RFC3339:
		chunks = rfc3339Chunks
	case RFC3339Nano:
		chunks = rfc3339NanoChunks
	default:
		return Time{}, &ParseError{Layout: layout, Value: value,
			Message: ": layout " + quote(layout) + " is not supported; use RFC3339 or RFC3339Nano"}
	}
	avalue := value
	var year, month, day, hour, min, sec, nsec int
	offset := 0
	for i, c := range chunks {
		rest, ok := skip(value, c.prefix)
		if !ok {
			return Time{}, &ParseError{Layout: layout, Value: avalue, LayoutElem: c.prefix, ValueElem: rest}
		}
		value = rest
		hold := value
		bad := false
		rangeErr := ""
		switch c.elem {
		case "2006":
			if len(value) < 4 || !isDigit(value, 0) {
				bad = true
				break
			}
			year, bad = atoi(value[0:4])
			value = value[4:]
		case "01":
			month, value, bad = getnum(value, true)
			if !bad && (month <= 0 || 12 < month) {
				rangeErr = "month"
			}
		case "02":
			day, value, bad = getnum(value, true)
		case "15":
			hour, value, bad = getnum(value, false)
			if hour < 0 || 24 <= hour {
				rangeErr = "hour"
			}
		case "04":
			min, value, bad = getnum(value, true)
			if min < 0 || 60 <= min {
				rangeErr = "minute"
			}
		case "05":
			sec, value, bad = getnum(value, true)
			if bad {
				break
			}
			if sec < 0 || 60 <= sec {
				rangeErr = "second"
				break
			}
			if len(value) >= 2 && isFractionMark(value[0]) && isDigit(value, 1) && chunks[i+1].elem != ".999999999" {
				n := 2
				for n < len(value) && isDigit(value, n) {
					n++
				}
				nsec, bad = parseNanoseconds(value, n)
				value = value[n:]
			}
		case ".999999999":
			if len(value) < 2 || !isFractionMark(value[0]) || value[1] < '0' || '9' < value[1] {
				break
			}
			n := 1
			for n < len(value) && '0' <= value[n] && value[n] <= '9' {
				n++
			}
			nsec, bad = parseNanoseconds(value, n)
			value = value[n:]
		case "Z07:00":
			if len(value) >= 1 && value[0] == 'Z' {
				value = value[1:]
				break
			}
			if len(value) < 6 || value[3] != ':' {
				bad = true
				break
			}
			sign := value[0]
			hh, mm := value[1:3], value[4:6]
			value = value[6:]
			var hr, mi int
			hr, _, bad = getnum(hh, true)
			if !bad {
				mi, _, bad = getnum(mm, true)
			}
			if hr > 24 {
				rangeErr = "time zone offset hour"
			}
			if mi > 60 {
				rangeErr = "time zone offset minute"
			}
			offset = (hr*60 + mi) * 60
			switch sign {
			case '+':
			case '-':
				offset = -offset
			default:
				bad = true
			}
		}
		if rangeErr != "" {
			return Time{}, &ParseError{Layout: layout, Value: avalue, LayoutElem: c.elem, ValueElem: value,
				Message: ": " + rangeErr + " out of range"}
		}
		if bad {
			return Time{}, &ParseError{Layout: layout, Value: avalue, LayoutElem: c.elem, ValueElem: hold}
		}
	}
	if len(value) != 0 {
		return Time{}, &ParseError{Layout: layout, Value: avalue, ValueElem: value, Message: ": extra text: " + quote(value)}
	}
	if day < 1 || day > daysIn(Month(month), year) {
		return Time{}, &ParseError{Layout: layout, Value: avalue, ValueElem: value, Message: ": day out of range"}
	}
	return Date(year, Month(month), day, hour, min, sec, nsec, UTC).Add(Duration(-offset) * Second), nil
}

func skip(value, prefix string) (string, bool) {
	for len(prefix) > 0 {
		if len(value) == 0 || value[0] != prefix[0] {
			return value, false
		}
		prefix = prefix[1:]
		value = value[1:]
	}
	return value, true
}

func isDigit(s string, i int) bool {
	if len(s) <= i {
		return false
	}
	return '0' <= s[i] && s[i] <= '9'
}

func isFractionMark(c byte) bool { return c == '.' || c == ',' }

func getnum(s string, fixed bool) (int, string, bool) {
	if !isDigit(s, 0) {
		return 0, s, true
	}
	if !isDigit(s, 1) {
		if fixed {
			return 0, s, true
		}
		return int(s[0] - '0'), s[1:], false
	}
	return int(s[0]-'0')*10 + int(s[1]-'0'), s[2:], false
}

func atoi(s string) (int, bool) {
	x := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, true
		}
		x = x*10 + int(s[i]-'0')
	}
	return x, false
}

func parseNanoseconds(value string, n int) (int, bool) {
	if n > 10 {
		n = 10
	}
	ns, bad := atoi(value[1:n])
	if bad {
		return 0, true
	}
	for i := 0; i < 10-n; i++ {
		ns *= 10
	}
	return ns, false
}

const lowerhex = "0123456789abcdef"

func quote(s string) string {
	b := make([]byte, 1, len(s)+2)
	b[0] = '"'
	for i, c := range s {
		if c >= 0x80 || c < ' ' {
			width := 1
			if c == 0xFFFD {
				if i+2 < len(s) && s[i:i+3] == "�" {
					width = 3
				}
			} else {
				width = len(string(c))
			}
			for j := 0; j < width; j++ {
				b = append(b, '\\', 'x', lowerhex[s[i+j]>>4], lowerhex[s[i+j]&0xF])
			}
			continue
		}
		if c == '"' || c == '\\' {
			b = append(b, '\\')
		}
		b = append(b, byte(c))
	}
	return string(append(b, '"'))
}

// Nanoseconds returns the duration as an integer nanosecond count.
func (d Duration) Nanoseconds() int64 { return int64(d) }

// Microseconds returns the duration as an integer microsecond count.
func (d Duration) Microseconds() int64 { return int64(d) / 1e3 }

// Milliseconds returns the duration as an integer millisecond count.
func (d Duration) Milliseconds() int64 { return int64(d) / 1e6 }

// Seconds returns the duration as a floating-point number of seconds.
func (d Duration) Seconds() float64 {
	sec := d / Second
	nsec := d % Second
	return float64(sec) + float64(nsec)/1e9
}

// Minutes returns the duration as a floating-point number of minutes.
func (d Duration) Minutes() float64 {
	min := d / Minute
	nsec := d % Minute
	return float64(min) + float64(nsec)/(60*1e9)
}

// Hours returns the duration as a floating-point number of hours.
func (d Duration) Hours() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)/(60*60*1e9)
}

// Truncate returns d rounded toward zero to a multiple of m. If m <= 0, it
// returns d unchanged.
func (d Duration) Truncate(m Duration) Duration {
	if m <= 0 {
		return d
	}
	return d - d%m
}

// Round returns d rounded to the nearest multiple of m, halfway values away
// from zero, saturating on overflow. If m <= 0, it returns d unchanged.
func (d Duration) Round(m Duration) Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if lessThanHalf(r, m) {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return minDuration
	}
	if lessThanHalf(r, m) {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return maxDuration
}

// Abs returns the absolute value of d, mapping the minimum Duration to the
// maximum.
func (d Duration) Abs() Duration {
	switch {
	case d >= 0:
		return d
	case d == minDuration:
		return maxDuration
	}
	return -d
}

// String returns the duration in the form "72h3m0.5s", with leading zero
// units omitted and sub-second durations in smaller units such as "1.2ms".
func (d Duration) String() string {
	buf := make([]byte, 32)
	w := len(buf)
	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}
	if u < uint64(Second) {
		prec := 0
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			buf[w] = '0'
			return string(buf[w:])
		case u < uint64(Microsecond):
			buf[w] = 'n'
		case u < uint64(Millisecond):
			prec = 3
			buf[w] = 0xB5
			w--
			buf[w] = 0xC2
		default:
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)
	} else {
		w--
		buf[w] = 's'
		w, u = fmtFrac(buf[:w], u, 9)
		w = fmtInt(buf[:w], u%60)
		u /= 60
		if u > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], u%60)
			u /= 60
			if u > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], u)
			}
		}
	}
	if neg {
		w--
		buf[w] = '-'
	}
	return string(buf[w:])
}

func fmtFrac(buf []byte, v uint64, prec int) (int, uint64) {
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
		return w
	}
	for v > 0 {
		w--
		buf[w] = byte(v%10) + '0'
		v /= 10
	}
	return w
}

// ParseDuration parses a duration string such as "300ms", "-1.5h" or
// "2h45m". Valid units are "ns", "us" (or "µs"), "ms", "s", "m" and "h".
func ParseDuration(s string) (Duration, error) {
	orig := s
	var d uint64
	neg := false
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("time: invalid duration " + quote(orig))
	}
	for s != "" {
		var v, f uint64
		scale := 1.0
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		pl := len(s)
		var ok bool
		v, s, ok = leadingInt(s)
		if !ok {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		pre := pl != len(s)
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, errors.New("time: missing unit in duration " + quote(orig))
		}
		u := s[:i]
		s = s[i:]
		unit := unitNanos(u)
		if unit == 0 {
			return 0, errors.New("time: unknown unit " + quote(u) + " in duration " + quote(orig))
		}
		if v > 1<<63/unit {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		v *= unit
		if f > 0 {
			v += uint64(float64(f) * (float64(unit) / scale))
			if v > 1<<63 {
				return 0, errors.New("time: invalid duration " + quote(orig))
			}
		}
		d += v
		if d > 1<<63 {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
	}
	if neg {
		return -Duration(d), nil
	}
	if d > 1<<63-1 {
		return 0, errors.New("time: invalid duration " + quote(orig))
	}
	return Duration(d), nil
}

func unitNanos(u string) uint64 {
	switch u {
	case "ns":
		return uint64(Nanosecond)
	case "us", "µs", "μs":
		return uint64(Microsecond)
	case "ms":
		return uint64(Millisecond)
	case "s":
		return uint64(Second)
	case "m":
		return uint64(Minute)
	case "h":
		return uint64(Hour)
	}
	return 0
}

func leadingInt(s string) (uint64, string, bool) {
	var x uint64
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > 1<<63/10 {
			return 0, s, false
		}
		x = x*10 + uint64(c) - '0'
		if x > 1<<63 {
			return 0, s, false
		}
	}
	return x, s[i:], true
}

func leadingFraction(s string) (uint64, float64, string) {
	var x uint64
	scale := 1.0
	overflow := false
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			overflow = true
			continue
		}
		y := x*10 + uint64(c) - '0'
		if y > 1<<63 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}
