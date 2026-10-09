package main

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/log/slog"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/time"
)

type token string

func (t token) LogValue() slog.Value { return slog.StringValue("REDACTED") }

type endpoint struct {
	host string
	port int
}

func (e endpoint) String() string { return fmt.Sprintf("%s:%d", e.host, e.port) }

type request struct {
	id   int
	path string
}

func (r request) LogValue() slog.Value {
	return slog.GroupValue(slog.Int("id", r.id), slog.String("path", r.path))
}

type printHandler struct {
	level  slog.Leveler
	prefix string
	attrs  []slog.Attr
}

func (h *printHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *printHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Level.String() + " " + r.Message)
	for _, a := range h.attrs {
		b.WriteString(" " + a.String())
	}
	r.Attrs(func(a slog.Attr) bool {
		b.WriteString(" " + h.prefix + a.Key + "=" + a.Value.Resolve().String())
		return true
	})
	fmt := " time=set"
	if r.Time.IsZero() {
		fmt = " time=zero"
	}
	println(b.String() + fmt)
	return nil
}

func (h *printHandler) WithAttrs(as []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = append(append([]slog.Attr(nil), h.attrs...), as...)
	return &h2
}

func (h *printHandler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.prefix = h.prefix + name + "."
	return &h2
}

func main() {
	slog.Info("hidden by the host's default warn level")
	slog.Debug("hidden")
	slog.Warn("retry", "kas", "https://kas.example.com/v2", "attempt", 2, "wait", 1500*time.Millisecond)
	slog.Error("rewrap failed", "err", errors.New("kas: 403 forbidden"), "allowed", false)

	at := time.Date(2024, time.January, 2, 3, 4, 5, 6789000, time.UTC)
	slog.Warn("kinds",
		"int", -3, "uint", uint64(18446744073709551615), "float", 0.1, "big", 1e21, "f32", float32(0.1),
		"bool", true, "time", at, "zero", time.Time{}, "dur", 90*time.Second,
		"bytes", []byte("a b"), "nil", nil, "stringer", endpoint{"kas", 8443}, "level", slog.LevelWarn+2,
		"token", token("secret"), "req", request{7, "/v2/rewrap"})
	slog.Warn("quoting", "empty", "", "eq", "a=b", "quote", `say "hi"`, "nl", "a\nb", "tab", "a\tb",
		"backslash", `a\b`, "unicode", "héllo wörld", "nbsp", "a b", "invalid", "a\xffb", "key with space", 1)
	slog.Warn("bad keys", 42, "dangling")
	slog.Warn("groups",
		slog.Group("req", "id", 7, slog.String("path", "/a")), slog.Group("empty"),
		slog.Group("", "inline", 1), slog.Group("outer", slog.Group("inner", "x", 1)))
	slog.LogAttrs(context.Background(), slog.LevelError, "attrs", slog.Int("n", 1), slog.Bool("ok", true))

	l := slog.Default().With("sdk", "goalchemy").WithGroup("kas").With("url", "https://kas").WithGroup("rewrap")
	l.Warn("chained", "attempt", 3)
	l.Warn("no record attrs")
	slog.Default().WithGroup("unused").Warn("empty group omitted")

	redact := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "password" {
			return slog.Attr{}
		}
		if a.Key == slog.LevelKey {
			return slog.String("severity", strings.ToLower(a.Value.String()))
		}
		if len(groups) > 0 {
			a.Key = strings.Join(groups, "/") + ":" + a.Key
		}
		return a
	}
	r := slog.New(slog.NewHostHandler(&slog.HandlerOptions{Level: slog.LevelError, ReplaceAttr: redact}))
	r.Warn("filtered by handler level")
	r.WithGroup("auth").Error("login", "user", "alice", "password", "hunter2")

	var lv slog.LevelVar
	lv.Set(slog.LevelInfo)
	p := slog.New(&printHandler{level: &lv})
	p.Debug("debug hidden")
	p.Info("info", "n", 1)
	lv.Set(slog.LevelDebug)
	p.With("svc", "kas").WithGroup("g").Debug("debug shown", "x", 2, "tok", token("t"))
	println("enabled", p.Enabled(context.Background(), slog.LevelDebug), slog.Default().Enabled(context.Background(), slog.LevelInfo), slog.Default().Enabled(nil, slog.LevelWarn))

	old := slog.Default()
	slog.SetDefault(p)
	slog.Info("through custom default", "k", "v")
	slog.SetDefault(old)
	slog.Warn("restored")

	rec := slog.NewRecord(time.Time{}, slog.LevelWarn, "manual", 0)
	rec.Add("a", 1, slog.String("b", "2"), slog.Group("empty"), "tail")
	clone := rec.Clone()
	clone.AddAttrs(slog.Int("c", 3))
	println("attrs", rec.NumAttrs(), clone.NumAttrs())
	_ = slog.Default().Handler().Handle(context.Background(), clone)
	_ = p.Handler().Handle(context.Background(), rec)

	var parsed slog.Level
	for _, s := range []string{"debug", "INFO+2", "Warn-1", "error+10", "loud"} {
		err := parsed.UnmarshalText([]byte(s))
		println(s, parsed.String(), err != nil)
	}
	println(slog.Level(-9).String(), slog.Level(3).String(), slog.KindGroup.String(), slog.AnyValue(at).Kind().String())
	println(slog.Int("a", 1).Equal(slog.Int64("a", 1)), slog.Group("g", "a", 1, "b", "x").String(), slog.AnyValue(90*time.Second).String())
	slog.New(slog.DiscardHandler).Error("discarded")
}
