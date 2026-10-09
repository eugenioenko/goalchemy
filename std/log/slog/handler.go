package slog

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	hostlog "github.com/eugenioenko/goalchemy/lib/log"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/time"
	"github.com/eugenioenko/goalchemy/std/unicode"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

// A Handler handles log records produced by a Logger.
type Handler interface {
	Enabled(context.Context, Level) bool
	Handle(context.Context, Record) error
	WithAttrs(attrs []Attr) Handler
	WithGroup(name string) Handler
}

// HandlerOptions are options for the host handler.
type HandlerOptions struct {
	// AddSource is ignored: source locations are not recorded.
	AddSource bool
	// Level is the minimum level to handle. If nil, the host application's
	// minimum level alone decides.
	Level Leveler
	// ReplaceAttr is called to rewrite each non-group attribute before it is
	// rendered, as in log/slog.
	ReplaceAttr func(groups []string, a Attr) Attr
}

// Keys for the built-in attributes.
const (
	TimeKey    = "time"
	LevelKey   = "level"
	MessageKey = "msg"
	SourceKey  = "source"
)

// DiscardHandler discards all log output.
var DiscardHandler Handler = discardHandler{}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, Level) bool  { return false }
func (discardHandler) Handle(context.Context, Record) error { return nil }
func (dh discardHandler) WithAttrs([]Attr) Handler          { return dh }
func (dh discardHandler) WithGroup(string) Handler          { return dh }

// HostHandler sends records to the host application's log sink. It renders
// attributes like log/slog's TextHandler: group names prefix keys with ".",
// and the sink receives the key/value pairs and the rendered line, which
// omits the time.
type HostHandler struct {
	opts        HandlerOptions
	preText     []byte
	prePairs    []string
	groupPrefix string
	groups      []string
	nOpenGroups int
}

// NewHostHandler creates a HostHandler with opts, or with default options
// if opts is nil.
func NewHostHandler(opts *HandlerOptions) *HostHandler {
	h := &HostHandler{}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

func (h *HostHandler) clone() *HostHandler {
	return &HostHandler{
		opts:        h.opts,
		preText:     append([]byte(nil), h.preText...),
		prePairs:    append([]string(nil), h.prePairs...),
		groupPrefix: h.groupPrefix,
		groups:      append([]string(nil), h.groups...),
		nOpenGroups: h.nOpenGroups,
	}
}

// Enabled reports whether both the handler's level and the host sink
// accept level.
func (h *HostHandler) Enabled(_ context.Context, level Level) bool {
	if h.opts.Level != nil && level < h.opts.Level.Level() {
		return false
	}
	return hostlog.Enabled(int(level))
}

// WithAttrs returns a HostHandler whose records include attrs.
func (h *HostHandler) WithAttrs(attrs []Attr) Handler {
	if countEmptyGroups(attrs) == len(attrs) {
		return h
	}
	h2 := h.clone()
	s := h2.newState("")
	s.buf = h2.preText
	s.pairs = h2.prePairs
	s.prefix = []byte(h.groupPrefix)
	if len(h2.preText) > 0 {
		s.sep = " "
	}
	pos, npairs := len(s.buf), len(s.pairs)
	s.openGroups()
	if !s.appendAttrs(attrs) {
		h2.preText, h2.prePairs = s.buf[:pos], s.pairs[:npairs]
		return h2
	}
	h2.preText, h2.prePairs = s.buf, s.pairs
	h2.groupPrefix = string(s.prefix)
	h2.nOpenGroups = len(h2.groups)
	return h2
}

// WithGroup returns a HostHandler that qualifies later keys with name.
func (h *HostHandler) WithGroup(name string) Handler {
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

// Handle renders r and sends it to the host sink.
func (h *HostHandler) Handle(_ context.Context, r Record) error {
	s := h.newState("")
	stateGroups := s.groups
	s.groups = nil
	rep := h.opts.ReplaceAttr
	if rep == nil {
		s.appendKey(LevelKey)
		s.appendString(r.Level.String())
	} else {
		s.appendAttr(Any(LevelKey, r.Level), false)
	}
	if rep == nil {
		s.appendKey(MessageKey)
		s.appendString(r.Message)
	} else {
		s.appendAttr(String(MessageKey, r.Message), false)
	}
	s.groups = stateGroups
	if len(h.preText) > 0 {
		s.buf = append(s.buf, s.sep...)
		s.buf = append(s.buf, h.preText...)
		s.pairs = append(s.pairs, h.prePairs...)
		s.sep = " "
	}
	if r.NumAttrs() > 0 {
		s.prefix = append(s.prefix, h.groupPrefix...)
		pos, npairs := len(s.buf), len(s.pairs)
		s.openGroups()
		empty := true
		r.Attrs(func(a Attr) bool {
			if s.appendAttr(a, true) {
				empty = false
			}
			return true
		})
		if empty {
			s.buf, s.pairs = s.buf[:pos], s.pairs[:npairs]
		}
	}
	var unixNano int64
	if !r.Time.IsZero() {
		unixNano = r.Time.UnixNano()
	}
	hostlog.Emit(int(r.Level), unixNano, r.Message, s.pairs, string(s.buf))
	return nil
}

type handleState struct {
	h      *HostHandler
	buf    []byte
	pairs  []string
	sep    string
	prefix []byte
	groups []string
}

func (h *HostHandler) newState(sep string) *handleState {
	s := &handleState{h: h, sep: sep}
	if h.opts.ReplaceAttr != nil {
		s.groups = append(make([]string, 0, len(h.groups)), h.groups[:h.nOpenGroups]...)
	}
	return s
}

func (s *handleState) openGroups() {
	for _, n := range s.h.groups[s.h.nOpenGroups:] {
		s.openGroup(n)
	}
}

func (s *handleState) openGroup(name string) {
	s.prefix = append(s.prefix, name...)
	s.prefix = append(s.prefix, '.')
	if s.groups != nil {
		s.groups = append(s.groups, name)
	}
}

func (s *handleState) closeGroup(name string) {
	s.prefix = s.prefix[:len(s.prefix)-len(name)-1]
	s.sep = " "
	if s.groups != nil {
		s.groups = s.groups[:len(s.groups)-1]
	}
}

func (s *handleState) appendAttrs(as []Attr) bool {
	nonEmpty := false
	for _, a := range as {
		if s.appendAttr(a, true) {
			nonEmpty = true
		}
	}
	return nonEmpty
}

func (s *handleState) appendAttr(a Attr, pair bool) bool {
	a.Value = a.Value.Resolve()
	if rep := s.h.opts.ReplaceAttr; rep != nil && a.Value.Kind() != KindGroup {
		a = rep(s.groups, a)
		a.Value = a.Value.Resolve()
	}
	if a.isEmpty() {
		return false
	}
	if a.Value.Kind() == KindGroup {
		attrs := a.Value.Group()
		if len(attrs) > 0 {
			pos, npairs := len(s.buf), len(s.pairs)
			if a.Key != "" {
				s.openGroup(a.Key)
			}
			if !s.appendAttrs(attrs) {
				s.buf, s.pairs = s.buf[:pos], s.pairs[:npairs]
				return false
			}
			if a.Key != "" {
				s.closeGroup(a.Key)
			}
		}
		return true
	}
	key := string(s.prefix) + a.Key
	s.appendKey(a.Key)
	text := s.appendValue(a.Value)
	if pair {
		s.pairs = append(s.pairs, key, text)
	}
	return true
}

func (s *handleState) appendKey(key string) {
	s.buf = append(s.buf, s.sep...)
	if len(s.prefix) > 0 {
		x := string(s.prefix)
		if !needsQuoting(x) && !needsQuoting(key) {
			s.buf = append(s.buf, x...)
			s.buf = append(s.buf, key...)
		} else {
			s.buf = strconv.AppendQuote(s.buf, x+key)
		}
	} else {
		s.appendString(key)
	}
	s.buf = append(s.buf, '=')
	s.sep = " "
}

func (s *handleState) appendString(str string) {
	if needsQuoting(str) {
		s.buf = strconv.AppendQuote(s.buf, str)
	} else {
		s.buf = append(s.buf, str...)
	}
}

func (s *handleState) appendValue(v Value) (text string) {
	defer func() {
		if r := recover(); r != nil {
			text = fmt.Sprintf("!PANIC: %v", r)
			if strings.Contains(text, "nil pointer dereference") {
				text = "<nil>"
			}
			s.appendString(text)
		}
	}()
	text, quote := textValue(v)
	if quote {
		s.buf = strconv.AppendQuote(s.buf, text)
	} else if v.Kind() == KindString || v.Kind() == KindAny {
		s.appendString(text)
	} else {
		s.buf = append(s.buf, text...)
	}
	return text
}

type textMarshaler interface {
	MarshalText() ([]byte, error)
}

func textValue(v Value) (string, bool) {
	switch v.Kind() {
	case KindString:
		return v.str, false
	case KindTime:
		return rfc3339Millis(v.t), false
	case KindAny:
		if tm, ok := v.any.(textMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				return fmt.Sprintf("!ERROR:%v", err), false
			}
			return string(data), false
		}
		if bs, ok := v.any.([]byte); ok {
			return string(bs), true
		}
		return fmt.Sprintf("%+v", v.Any()), false
	default:
		return string(v.append(nil)), false
	}
}

func rfc3339Millis(t time.Time) string {
	const prefixLen = len("2006-01-02T15:04:05.000")
	t = t.Truncate(time.Millisecond).Add(time.Millisecond / 10)
	b := t.AppendFormat(nil, time.RFC3339Nano)
	return string(b[:prefixLen]) + string(b[prefixLen+1:])
}

func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if b != '\\' && (b == ' ' || b == '=' || b == '"' || b < 0x20 || b == 0x7f) {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}
