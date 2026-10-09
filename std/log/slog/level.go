package slog

import (
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
)

// A Level is the importance or severity of a log event.
type Level int

// Names for common levels.
const (
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
)

func levelString(base string, val Level) string {
	if val == 0 {
		return base
	}
	sval := strconv.Itoa(int(val))
	if val > 0 {
		sval = "+" + sval
	}
	return base + sval
}

// String returns a name for the level, such as "WARN" or "INFO+2".
func (l Level) String() string {
	switch {
	case l < LevelInfo:
		return levelString("DEBUG", l-LevelDebug)
	case l < LevelWarn:
		return levelString("INFO", l-LevelInfo)
	case l < LevelError:
		return levelString("WARN", l-LevelWarn)
	default:
		return levelString("ERROR", l-LevelError)
	}
}

// AppendText appends the level's name to b.
func (l Level) AppendText(b []byte) ([]byte, error) {
	return append(b, l.String()...), nil
}

// MarshalText returns the level's name.
func (l Level) MarshalText() ([]byte, error) {
	return l.AppendText(nil)
}

// UnmarshalText parses a level name produced by String, ignoring case.
func (l *Level) UnmarshalText(data []byte) error {
	return l.parse(string(data))
}

func (l *Level) parse(s string) error {
	name := s
	offset := 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		n, err := strconv.Atoi(s[i:])
		if err != nil {
			return fmt.Errorf("slog: level string %q: %w", s, err)
		}
		offset = n
	}
	switch strings.ToUpper(name) {
	case "DEBUG":
		*l = LevelDebug
	case "INFO":
		*l = LevelInfo
	case "WARN":
		*l = LevelWarn
	case "ERROR":
		*l = LevelError
	default:
		return fmt.Errorf("slog: level string %q: %w", s, errors.New("unknown name"))
	}
	*l += Level(offset)
	return nil
}

// Level returns l, so a Level is a Leveler.
func (l Level) Level() Level { return l }

// A Leveler provides a Level value.
type Leveler interface {
	Level() Level
}

// A LevelVar is a Level variable that can change while handlers use it.
type LevelVar struct {
	val Level
}

// Level returns v's level.
func (v *LevelVar) Level() Level { return v.val }

// Set sets v's level to l.
func (v *LevelVar) Set(l Level) { v.val = l }

// String returns "LevelVar(" + the level name + ")".
func (v *LevelVar) String() string {
	return "LevelVar(" + v.Level().String() + ")"
}

// MarshalText returns the name of v's level.
func (v *LevelVar) MarshalText() ([]byte, error) {
	return v.Level().MarshalText()
}

// UnmarshalText sets v's level from a level name.
func (v *LevelVar) UnmarshalText(data []byte) error {
	var l Level
	if err := l.UnmarshalText(data); err != nil {
		return err
	}
	v.Set(l)
	return nil
}
