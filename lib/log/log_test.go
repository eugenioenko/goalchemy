package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestDefaultLevelAndHandler(t *testing.T) {
	defer SetHandler(nil, DefaultLevel)
	if Enabled(0) || !Enabled(4) || !Enabled(8) {
		t.Fatal("default level is not warn")
	}
	var got []Record
	SetHandler(func(r Record) { got = append(got, r) }, -4)
	if !Enabled(-4) || Enabled(-5) {
		t.Fatal("SetHandler level")
	}
	attrs := []string{"a", "1", "g.b", "x y"}
	Emit(-4, 1700000000123456789, "hello", attrs, `level=DEBUG msg=hello a=1 g.b="x y"`)
	attrs[0] = "changed"
	if len(got) != 1 || got[0].Level != -4 || got[0].Message != "hello" || got[0].Attrs[0] != "a" || got[0].Attrs[3] != "x y" ||
		!got[0].Time.Equal(time.Unix(0, 1700000000123456789)) || got[0].Text != `level=DEBUG msg=hello a=1 g.b="x y"` {
		t.Fatalf("%+v", got)
	}
	SetHandler(func(Record) { panic("sink") }, 0)
	Emit(0, 0, "ignored", nil, "level=INFO msg=ignored")
}

func TestSlogHandler(t *testing.T) {
	defer SetHandler(nil, DefaultLevel)
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	SetHandler(Slog(l), -4)
	Emit(-4, 0, "dropped", nil, "")
	Emit(4, time.Date(2024, 1, 2, 3, 4, 5, 6000000, time.UTC).UnixNano(), "retry", []string{"kas.url", "https://kas", "attempt", "2"}, "")
	if got := buf.String(); got != "time=2024-01-02T03:04:05.006Z level=WARN msg=retry kas.url=https://kas attempt=2\n" {
		t.Fatalf("%q", got)
	}
	if strings.Contains(buf.String(), "dropped") {
		t.Fatal("slog level ignored")
	}
}
