package slog

import (
	"bytes"
	goslog "log/slog"
	"math"
	"strings"
	"testing"
	gotime "time"

	"github.com/eugenioenko/goalchemy/lib/context"
	hostlog "github.com/eugenioenko/goalchemy/lib/log"
	"github.com/eugenioenko/goalchemy/std/time"
)

type grp struct {
	key  string
	args []any
}
type dur int64
type tm int64
type valuer struct{ v any }
type secret string
type failErr string
type text string
type badText string

func (e failErr) Error() string                { return string(e) }
func (t text) MarshalText() ([]byte, error)    { return []byte("T:" + string(t)), nil }
func (t badText) MarshalText() ([]byte, error) { return nil, failErr("no text") }

type ourValuer struct{ v any }
type goValuer struct{ v any }

func (v ourValuer) LogValue() Value       { return AnyValue(ours(v.v)) }
func (v goValuer) LogValue() goslog.Value { return goslog.AnyValue(theirs(v.v)) }

func ours(a any) any {
	switch a := a.(type) {
	case grp:
		return Group(a.key, oursAll(a.args)...)
	case dur:
		return time.Duration(a)
	case tm:
		return time.Unix(0, int64(a)).UTC()
	case valuer:
		return ourValuer{a.v}
	case lvlv:
		return Level(a)
	}
	return a
}

func theirs(a any) any {
	switch a := a.(type) {
	case grp:
		return goslog.Group(a.key, theirsAll(a.args)...)
	case dur:
		return gotime.Duration(a)
	case tm:
		return gotime.Unix(0, int64(a)).UTC()
	case valuer:
		return goValuer{a.v}
	case lvlv:
		return goslog.Level(a)
	}
	return a
}

func oursAll(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = ours(a)
	}
	return out
}

func theirsAll(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = theirs(a)
	}
	return out
}

type op struct {
	group string
	with  []any
}

type scenario struct {
	name    string
	ops     []op
	level   int
	msg     string
	args    []any
	replace bool
	min     *int
}

func scenarios() []scenario {
	debug := -4
	return []scenario{
		{name: "plain", level: 4, msg: "retry", args: []any{"kas", "https://kas.example/v2", "attempt", 2}},
		{name: "quoting", level: 8, msg: "has space", args: []any{"empty", "", "eq", "a=b", "quote", `say "hi"`, "nl", "a\nb", "tab", "a\tb", "bs", `a\b`, "uni", "héllo", "nbsp", "a b", "bad", "a\xffb", "k y", 1}},
		{name: "kinds", level: 4, msg: "kinds", args: []any{"i", -3, "i8", int8(-8), "u", uint(7), "u8", uint8(255), "u64", uint64(math.MaxUint64), "f", 1.5, "f32", float32(0.1), "big", 1e21, "nan", math.NaN(), "inf", math.Inf(-1), "b", true, "d", dur(90 * gotime.Second), "t", tm(1704164645006789000), "zt", tm(0), "bytes", []byte("a b"), "err", failErr("boom"), "nilv", nil, "text", text("x y"), "badtext", badText("z"), "lvl", lvlv(6)}},
		{name: "groups", level: 4, msg: "groups", args: []any{grp{"req", []any{"id", 7, "path", "/a"}}, grp{"empty", nil}, grp{"", []any{"inline", 1}}, grp{"nested", []any{grp{"deep", []any{"x", 1}}}}}},
		{name: "bad keys", level: 4, msg: "bad", args: []any{1, "dangling"}},
		{name: "with chains", ops: []op{{with: []any{"svc", "kas"}}, {group: "g"}, {with: []any{"a", 1}}, {group: "h"}}, level: 4, msg: "chain", args: []any{"b", 2}},
		{name: "group without attrs", ops: []op{{group: "g"}}, level: 4, msg: "none"},
		{name: "with empty group", ops: []op{{with: []any{grp{"e", nil}}}}, level: 4, msg: "e", args: []any{"x", 1}},
		{name: "valuer", level: 4, msg: "valuer", args: []any{"v", valuer{"hidden"}, "g", valuer{grp{"in", []any{"a", 1}}}}},
		{name: "replace", ops: []op{{group: "g"}, {with: []any{"secret", "p"}}}, level: 4, msg: "rep", args: []any{"secret", "q", "keep", 1}, replace: true},
		{name: "levels", level: 6, msg: "between", args: nil},
		{name: "debug", level: -4, msg: "dbg", args: []any{"x", 1}, min: &debug},
		{name: "info+1", level: 1, msg: "info", min: &debug},
	}
}

type lvlv int

func replaceOurs(groups []string, a Attr) Attr {
	if a.Key == "secret" {
		return Attr{}
	}
	if a.Key == LevelKey {
		return String("severity", a.Value.String())
	}
	if len(groups) > 0 {
		a.Key = strings.Join(groups, "/") + ":" + a.Key
	}
	return a
}

func replaceTheirs(groups []string, a goslog.Attr) goslog.Attr {
	if a.Key == goslog.TimeKey && len(groups) == 0 {
		return goslog.Attr{}
	}
	if a.Key == "secret" {
		return goslog.Attr{}
	}
	if a.Key == goslog.LevelKey {
		return goslog.String("severity", a.Value.String())
	}
	if len(groups) > 0 {
		a.Key = strings.Join(groups, "/") + ":" + a.Key
	}
	return a
}

func dropTime(groups []string, a goslog.Attr) goslog.Attr {
	if a.Key == goslog.TimeKey && len(groups) == 0 {
		return goslog.Attr{}
	}
	return a
}

func TestHostHandlerMatchesTextHandler(t *testing.T) {
	defer hostlog.SetHandler(nil, hostlog.DefaultLevel)
	for _, sc := range scenarios() {
		t.Run(sc.name, func(t *testing.T) {
			var got []hostlog.Record
			hostlog.SetHandler(func(r hostlog.Record) { got = append(got, r) }, -4)
			var buf bytes.Buffer
			gopts := &goslog.HandlerOptions{Level: goslog.LevelDebug, ReplaceAttr: dropTime}
			var opts *HandlerOptions
			if sc.replace {
				gopts.ReplaceAttr = replaceTheirs
				opts = &HandlerOptions{ReplaceAttr: replaceOurs}
			}
			want := goslog.New(goslog.NewTextHandler(&buf, gopts))
			l := New(NewHostHandler(opts))
			if sc.min != nil {
				hostlog.SetHandler(func(r hostlog.Record) { got = append(got, r) }, *sc.min)
			}
			for _, o := range sc.ops {
				if o.group != "" {
					want, l = want.WithGroup(o.group), l.WithGroup(o.group)
				} else {
					want, l = want.With(theirsAll(o.with)...), l.With(oursAll(o.with)...)
				}
			}
			want.Log(nil, goslog.Level(sc.level), sc.msg, theirsAll(sc.args)...)
			before := gotime.Now()
			l.Log(context.Background(), Level(sc.level), sc.msg, oursAll(sc.args)...)
			if len(got) != 1 {
				t.Fatalf("records %d", len(got))
			}
			r := got[0]
			if r.Text+"\n" != buf.String() {
				t.Fatalf("text\n got %q\nwant %q", r.Text, buf.String())
			}
			if r.Level != sc.level || r.Message != sc.msg || r.Time.Before(before.Add(-gotime.Second)) || len(r.Attrs)%2 != 0 {
				t.Fatalf("record %+v", r)
			}
			var pairs []string
			for i := 0; i < len(r.Attrs); i += 2 {
				k, v := r.Attrs[i], r.Attrs[i+1]
				pairs = append(pairs, k+"="+v)
				if !strings.Contains(r.Text, k) {
					t.Fatalf("pair key %q not in %q", k, r.Text)
				}
			}
			t.Log(strings.Join(pairs, " | "))
		})
	}
}

func TestLevelsAndValues(t *testing.T) {
	for l := -10; l <= 14; l++ {
		if got, want := Level(l).String(), goslog.Level(l).String(); got != want {
			t.Fatalf("Level(%d) = %q, want %q", l, got, want)
		}
		var ours Level
		var theirs goslog.Level
		s := goslog.Level(l).String()
		if err := ours.UnmarshalText([]byte(strings.ToLower(s))); err != nil || theirs.UnmarshalText([]byte(strings.ToLower(s))) != nil || int(ours) != int(theirs) {
			t.Fatalf("parse %q: %d %v", s, ours, err)
		}
	}
	for _, bad := range []string{"", "LOUD", "INFO+x", "WARN+"} {
		var ours Level
		var theirs goslog.Level
		err, goErr := ours.UnmarshalText([]byte(bad)), theirs.UnmarshalText([]byte(bad))
		if (err == nil) != (goErr == nil) || err != nil && err.Error() != goErr.Error() {
			t.Fatalf("parse %q: %v, want %v", bad, err, goErr)
		}
	}
	values := []any{"s", 1, int64(-2), uint64(3), 1.25, true, dur(1500 * gotime.Millisecond), tm(1704164645006789000), nil, []byte("b"), failErr("e"), grp{"g", []any{"a", 1, "b", "x y"}}}
	for _, v := range values {
		got, want := AnyValue(ours(v)), goslog.AnyValue(theirs(v))
		if want.Kind() == goslog.KindGroup {
			got, want = ours(v).(Attr).Value, theirs(v).(goslog.Attr).Value
		}
		if got.Kind().String() != want.Kind().String() || got.String() != want.String() {
			t.Fatalf("%v: %s %q, want %s %q", v, got.Kind(), got.String(), want.Kind(), want.String())
		}
	}
	if !Int("a", 1).Equal(Int64("a", 1)) || Int("a", 1).Equal(Uint64("a", 1)) || !Group("g", "a", 1).Equal(Group("g", "a", 1)) {
		t.Fatal("Equal")
	}
	if Int("a", 1).String() != "a=1" || Group("g", "a", 1, "b", "c").String() != goslog.Group("g", "a", 1, "b", "c").String() {
		t.Fatal(Group("g", "a", 1, "b", "c").String())
	}
	var lv LevelVar
	lv.Set(LevelWarn)
	if lv.String() != "LevelVar(WARN)" || lv.Level() != LevelWarn {
		t.Fatal(lv.String())
	}
}

func TestEnabledAndDefault(t *testing.T) {
	defer hostlog.SetHandler(nil, hostlog.DefaultLevel)
	var got []hostlog.Record
	hostlog.SetHandler(func(r hostlog.Record) { got = append(got, r) }, hostlog.DefaultLevel)
	Info("hidden")
	Debug("hidden")
	Warn("shown", "n", 1)
	Error("shown")
	if len(got) != 2 || got[0].Text != "level=WARN msg=shown n=1" || got[1].Text != "level=ERROR msg=shown" {
		t.Fatalf("%+v", got)
	}
	got = nil
	hostlog.SetHandler(func(r hostlog.Record) { got = append(got, r) }, -4)
	l := New(NewHostHandler(&HandlerOptions{Level: LevelInfo}))
	if l.Enabled(nil, LevelDebug) || !l.Enabled(nil, LevelInfo) {
		t.Fatal("handler level")
	}
	l.Debug("no")
	l.Info("yes")
	SetDefault(l.With("sdk", "go"))
	InfoContext(context.Background(), "default")
	Default().Debug("no")
	if len(got) != 2 || got[0].Text != "level=INFO msg=yes" || got[1].Text != "level=INFO msg=default sdk=go" {
		t.Fatalf("%+v", got)
	}
	New(DiscardHandler).Error("discarded")
	r := NewRecord(time.Time{}, LevelInfo, "m", 0)
	r.Add("a", 1, Int("b", 2), grp{"e", nil}.key)
	r2 := r.Clone()
	r2.AddAttrs(String("c", "3"))
	if r.NumAttrs() != 3 || r2.NumAttrs() != 4 {
		t.Fatal(r.NumAttrs(), r2.NumAttrs())
	}
	got = nil
	_ = NewHostHandler(nil).Handle(nil, r)
	if len(got) != 1 || !got[0].Time.Equal(gotime.Unix(0, 0)) || got[0].Text != `level=INFO msg=m a=1 b=2 !BADKEY=e` {
		t.Fatalf("%+v", got)
	}
}
