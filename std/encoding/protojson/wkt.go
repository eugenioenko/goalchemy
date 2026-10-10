package protojson

import (
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/time"
)

const (
	maxDurationSeconds  = 315576000000
	maxNanos            = 999999999
	minTimestampSeconds = -62135596800
	maxTimestampSeconds = 253402300799
)

// Timestamp is google.protobuf.Timestamp, encoded as an RFC 3339 string.
type Timestamp struct {
	Seconds int64
	Nanos   int32
}

// TimestampOf returns the Timestamp for t.
func TimestampOf(t time.Time) *Timestamp {
	return &Timestamp{Seconds: t.Unix(), Nanos: int32(t.Nanosecond())}
}

// AsTime returns the instant as a UTC time.Time.
func (t *Timestamp) AsTime() time.Time { return time.Unix(t.Seconds, int64(t.Nanos)) }

func fraction(nanos int64) string {
	if nanos == 0 {
		return ""
	}
	s := strconv.FormatInt(nanos+1000000000, 10)[1:]
	s = strings.TrimSuffix(s, "000")
	s = strings.TrimSuffix(s, "000")
	return "." + s
}

// EncodeProtoJSON implements Message.
func (t *Timestamp) EncodeProtoJSON(e *Encoder) {
	if t.Seconds < minTimestampSeconds || t.Seconds > maxTimestampSeconds {
		e.Fail("google.protobuf.Timestamp: seconds out of range " + strconv.FormatInt(t.Seconds, 10))
		return
	}
	if t.Nanos < 0 || t.Nanos > maxNanos {
		e.Fail("google.protobuf.Timestamp: nanos out of range " + strconv.FormatInt(int64(t.Nanos), 10))
		return
	}
	s := time.Unix(t.Seconds, 0).UTC().Format(time.RFC3339)
	e.String(strings.TrimSuffix(s, "Z") + fraction(int64(t.Nanos)) + "Z")
}

// DecodeProtoJSON implements Message.
func (t *Timestamp) DecodeProtoJSON(d *Decoder, v jsonvalue.Value) {
	s, ok := v.AsString()
	if !ok {
		d.invalid("message", "google.protobuf.Timestamp", v)
		return
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		d.Fail("invalid google.protobuf.Timestamp value " + strconv.Quote(s))
		return
	}
	secs := parsed.Unix()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		d.Fail("google.protobuf.Timestamp value out of range: " + strconv.Quote(s))
		return
	}
	i := strings.LastIndexByte(s, '.')
	j := strings.LastIndexAny(s, "Z-+")
	if i >= 0 && j >= i && j-i > len(".999999999") {
		d.Fail("invalid google.protobuf.Timestamp value " + strconv.Quote(s))
		return
	}
	t.Seconds = secs
	t.Nanos = int32(parsed.Nanosecond())
}

// MarshalJSON implements json.Marshaler.
func (t *Timestamp) MarshalJSON() ([]byte, error) { return Marshal(t) }

// UnmarshalJSON implements json.Unmarshaler.
func (t *Timestamp) UnmarshalJSON(b []byte) error { return Unmarshal(b, t) }

// Duration is google.protobuf.Duration, encoded as decimal seconds with an
// "s" suffix.
type Duration struct {
	Seconds int64
	Nanos   int32
}

// DurationOf returns the Duration for d.
func DurationOf(d time.Duration) *Duration {
	n := int64(d)
	return &Duration{Seconds: n / 1000000000, Nanos: int32(n % 1000000000)}
}

// AsDuration returns the span as a time.Duration, saturating on overflow.
func (x *Duration) AsDuration() time.Duration {
	const maxSeconds = 9223372036
	if x.Seconds > maxSeconds || x.Seconds == maxSeconds && x.Nanos > 854775807 {
		return time.Duration(9223372036854775807)
	}
	if x.Seconds < -maxSeconds || x.Seconds == -maxSeconds && x.Nanos < -854775808 {
		return time.Duration(-9223372036854775807 - 1)
	}
	return time.Duration(x.Seconds*1000000000 + int64(x.Nanos))
}

// EncodeProtoJSON implements Message.
func (x *Duration) EncodeProtoJSON(e *Encoder) {
	secs, nanos := x.Seconds, int64(x.Nanos)
	if secs < -maxDurationSeconds || secs > maxDurationSeconds {
		e.Fail("google.protobuf.Duration: seconds out of range " + strconv.FormatInt(secs, 10))
		return
	}
	if nanos < -maxNanos || nanos > maxNanos {
		e.Fail("google.protobuf.Duration: nanos out of range " + strconv.FormatInt(nanos, 10))
		return
	}
	if secs > 0 && nanos < 0 || secs < 0 && nanos > 0 {
		e.Fail("google.protobuf.Duration: signs of seconds and nanos do not match")
		return
	}
	sign := ""
	if secs < 0 || nanos < 0 {
		sign, secs, nanos = "-", -secs, -nanos
	}
	e.String(sign + strconv.FormatInt(secs, 10) + fraction(nanos) + "s")
}

// DecodeProtoJSON implements Message.
func (x *Duration) DecodeProtoJSON(d *Decoder, v jsonvalue.Value) {
	s, ok := v.AsString()
	if !ok {
		d.invalid("message", "google.protobuf.Duration", v)
		return
	}
	secs, nanos, ok := parseDuration(s)
	if !ok {
		d.Fail("invalid google.protobuf.Duration value " + strconv.Quote(s))
		return
	}
	if secs < -maxDurationSeconds || secs > maxDurationSeconds {
		d.Fail("google.protobuf.Duration value out of range: " + strconv.Quote(s))
		return
	}
	x.Seconds, x.Nanos = secs, nanos
}

// MarshalJSON implements json.Marshaler.
func (x *Duration) MarshalJSON() ([]byte, error) { return Marshal(x) }

// UnmarshalJSON implements json.Unmarshaler.
func (x *Duration) UnmarshalJSON(b []byte) error { return Unmarshal(b, x) }

func parseDuration(input string) (int64, int32, bool) {
	b := input
	if len(b) < 2 || b[len(b)-1] != 's' {
		return 0, 0, false
	}
	b = b[:len(b)-1]
	neg := false
	switch b[0] {
	case '-':
		neg = true
		b = b[1:]
	case '+':
		b = b[1:]
	}
	if b == "" {
		return 0, 0, false
	}
	intp := ""
	switch {
	case b[0] == '0':
		b = b[1:]
	case b[0] >= '1' && b[0] <= '9':
		n := 1
		for n < len(b) && isDigit(b[n]) {
			n++
		}
		intp, b = b[:n], b[n:]
	case b[0] == '.':
	default:
		return 0, 0, false
	}
	hasFrac := false
	frac := ""
	if b != "" {
		if b[0] != '.' {
			return 0, 0, false
		}
		b = b[1:]
		n := 0
		for n < len(b) && n < 9 && isDigit(b[n]) {
			n++
		}
		frac, b = b[:n], b[n:]
		if b != "" {
			return 0, 0, false
		}
		for len(frac) < 9 {
			frac += "0"
		}
		hasFrac = true
	}
	var secs int64
	if intp != "" {
		n, err := strconv.ParseInt(intp, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		secs = n
	}
	var nanos int64
	if hasFrac {
		trimmed := strings.TrimLeft(frac, "0")
		if trimmed != "" {
			n, err := strconv.ParseInt(trimmed, 10, 32)
			if err != nil {
				return 0, 0, false
			}
			nanos = n
		}
	}
	if neg {
		secs, nanos = -secs, -nanos
	}
	return secs, int32(nanos), true
}

// Empty is google.protobuf.Empty, encoded as {}.
type Empty struct{}

// EncodeProtoJSON implements Message.
func (x *Empty) EncodeProtoJSON(e *Encoder) {
	e.BeginObject()
	e.EndObject()
}

// DecodeProtoJSON implements Message.
func (x *Empty) DecodeProtoJSON(d *Decoder, v jsonvalue.Value) {
	d.Object(v, "google.protobuf.Empty")
}

// MarshalJSON implements json.Marshaler.
func (x *Empty) MarshalJSON() ([]byte, error) { return Marshal(x) }

// UnmarshalJSON implements json.Unmarshaler.
func (x *Empty) UnmarshalJSON(b []byte) error { return Unmarshal(b, x) }

// FieldMask is google.protobuf.FieldMask, encoded as comma-separated
// lowerCamelCase paths.
type FieldMask struct {
	Paths []string
}

func camelCase(s string) string {
	var b []byte
	upper := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_':
			upper = true
		case upper && c >= 'a' && c <= 'z':
			b = append(b, c-'a'+'A')
			upper = false
		default:
			b = append(b, c)
			upper = false
		}
	}
	return string(b)
}

func snakeCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b = append(b, '_', c-'A'+'a')
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}

func validPath(s string) bool {
	for _, part := range strings.Split(s, ".") {
		if part == "" {
			return false
		}
		for i := 0; i < len(part); i++ {
			c := part[i]
			if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && isDigit(c)) {
				return false
			}
		}
	}
	return true
}

// EncodeProtoJSON implements Message.
func (x *FieldMask) EncodeProtoJSON(e *Encoder) {
	paths := make([]string, 0, len(x.Paths))
	for _, s := range x.Paths {
		if !validPath(s) {
			e.Fail("google.protobuf.FieldMask.paths contains invalid path: " + strconv.Quote(s))
			return
		}
		cc := camelCase(s)
		if s != snakeCase(cc) {
			e.Fail("google.protobuf.FieldMask.paths contains irreversible value " + strconv.Quote(s))
			return
		}
		paths = append(paths, cc)
	}
	e.String(strings.Join(paths, ","))
}

// DecodeProtoJSON implements Message.
func (x *FieldMask) DecodeProtoJSON(d *Decoder, v jsonvalue.Value) {
	s, ok := v.AsString()
	if !ok {
		d.invalid("message", "google.protobuf.FieldMask", v)
		return
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	for _, p := range strings.Split(s, ",") {
		snake := snakeCase(p)
		if strings.Contains(p, "_") || !validPath(snake) {
			d.Fail("google.protobuf.FieldMask.paths contains invalid path: " + strconv.Quote(p))
			return
		}
		x.Paths = append(x.Paths, snake)
	}
}

// MarshalJSON implements json.Marshaler.
func (x *FieldMask) MarshalJSON() ([]byte, error) { return Marshal(x) }

// UnmarshalJSON implements json.Unmarshaler.
func (x *FieldMask) UnmarshalJSON(b []byte) error { return Unmarshal(b, x) }

// Any is google.protobuf.Any. Fields holds the JSON members that follow
// "@type": the embedded message's fields, or the single member "value" for a
// well-known type. Payloads of registered generated messages and of the
// well-known types are kept in canonical form; other payloads are kept as
// received. Use NewAny and UnmarshalTo to convert typed messages.
type Any struct {
	TypeURL string
	Fields  jsonvalue.Value
}

var registry = map[string]func(string) Message{}

// Register makes generated messages known to Any by full name. Generated
// packages call it from init.
func Register(names []string, newMessage func(name string) Message) {
	for _, n := range names {
		registry[n] = newMessage
	}
}

func typeName(url string) string {
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			return url[i+1:]
		}
	}
	return url
}

func wellKnownMessage(name string) Message {
	switch name {
	case "google.protobuf.Timestamp":
		return &Timestamp{}
	case "google.protobuf.Duration":
		return &Duration{}
	case "google.protobuf.Empty":
		return &Empty{}
	case "google.protobuf.FieldMask":
		return &FieldMask{}
	case "google.protobuf.Any":
		return &Any{}
	}
	return nil
}

func isWellKnownJSON(name string) bool {
	switch name {
	case "google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue":
		return true
	}
	return false
}

func isWellKnown(m Message) bool {
	switch m.(type) {
	case *Timestamp, *Duration, *Empty, *FieldMask, *Any:
		return true
	}
	return false
}

func reparse(e *Encoder) (jsonvalue.Value, error) {
	if e.err != nil {
		return jsonvalue.Value{}, e.err
	}
	return jsonvalue.Parse(e.buf)
}

// canonical decodes payload as the message named name and re-encodes it.
// ok is false when the type is unknown.
func canonical(d *Decoder, name string, payload jsonvalue.Value) (v jsonvalue.Value, ok bool) {
	if m := wellKnownMessage(name); m != nil || isWellKnownJSON(name) {
		value, found := payload.Lookup("value")
		for _, k := range payload.Keys() {
			if k != "value" {
				d.Fail("google.protobuf.Any: unexpected field " + strconv.Quote(k) + " for " + name)
				return jsonvalue.Value{}, true
			}
		}
		if !found {
			d.Fail("google.protobuf.Any: missing \"value\" field for " + name)
			return jsonvalue.Value{}, true
		}
		e := &Encoder{}
		switch name {
		case "google.protobuf.Struct":
			e.Struct(d.Struct(value, "value"))
		case "google.protobuf.ListValue":
			e.ListValue(d.ListValue(value, "value"))
		case "google.protobuf.Value":
			e.Value(d.Value(value, "value"))
		default:
			m.DecodeProtoJSON(d, value)
			m.EncodeProtoJSON(e)
		}
		if d.err != nil {
			return jsonvalue.Value{}, true
		}
		out, err := reparse(e)
		if err != nil {
			d.Fail(err.Error())
		}
		return jsonvalue.Object("value", out), true
	}
	newMessage, found := registry[name]
	if !found {
		return jsonvalue.Value{}, false
	}
	m := newMessage(name)
	m.DecodeProtoJSON(d, payload)
	if d.err != nil {
		return jsonvalue.Value{}, true
	}
	e := &Encoder{}
	m.EncodeProtoJSON(e)
	out, err := reparse(e)
	if err != nil {
		d.Fail(err.Error())
	}
	return out, true
}

// NewAny packs m under typeURL, for example
// "type.googleapis.com/policy.Attribute".
func NewAny(typeURL string, m Message) (*Any, error) {
	b, err := Marshal(m)
	if err != nil {
		return nil, err
	}
	v, err := jsonvalue.Parse(b)
	if err != nil {
		return nil, err
	}
	if isWellKnown(m) {
		v = jsonvalue.Object("value", v)
	}
	return &Any{TypeURL: typeURL, Fields: v}, nil
}

// UnmarshalTo decodes the embedded message into m.
func (x *Any) UnmarshalTo(m Message) error {
	if isWellKnown(m) {
		return UnmarshalValue(x.Fields.Get("value"), m)
	}
	fields := x.Fields
	if !fields.Exists() {
		fields = jsonvalue.Object()
	}
	return UnmarshalValue(fields, m)
}

// EncodeProtoJSON implements Message.
func (x *Any) EncodeProtoJSON(e *Encoder) {
	if x.TypeURL == "" {
		if x.Fields.Len() > 0 {
			e.Fail("google.protobuf.Any: type_url is not set")
			return
		}
		e.BeginObject()
		e.EndObject()
		return
	}
	e.BeginObject()
	e.Key("@type")
	e.String(x.TypeURL)
	for _, k := range x.Fields.Keys() {
		e.Key(k)
		e.raw(x.Fields.Get(k))
	}
	e.EndObject()
}

// DecodeProtoJSON implements Message.
func (x *Any) DecodeProtoJSON(d *Decoder, v jsonvalue.Value) {
	if !d.Object(v, "google.protobuf.Any") {
		return
	}
	t, found := v.Lookup("@type")
	if !found {
		return
	}
	url, ok := t.AsString()
	if !ok {
		d.invalid("string", "@type", t)
		return
	}
	if url == "" {
		d.Fail("google.protobuf.Any: empty @type")
		return
	}
	var members []any
	for _, k := range v.Keys() {
		if k != "@type" {
			members = append(members, k, v.Get(k))
		}
	}
	payload := jsonvalue.Object(members...)
	if c, known := canonical(d, typeName(url), payload); known {
		payload = c
	}
	x.TypeURL = url
	x.Fields = payload
}

// MarshalJSON implements json.Marshaler.
func (x *Any) MarshalJSON() ([]byte, error) { return Marshal(x) }

// UnmarshalJSON implements json.Unmarshaler.
func (x *Any) UnmarshalJSON(b []byte) error { return Unmarshal(b, x) }

func (e *Encoder) raw(v jsonvalue.Value) {
	switch v.Kind() {
	case jsonvalue.KindNull:
		e.Null()
	case jsonvalue.KindBool:
		e.Bool(v.Bool())
	case jsonvalue.KindNumber:
		e.value()
		e.buf = append(e.buf, v.Number()...)
	case jsonvalue.KindString:
		e.String(v.String())
	case jsonvalue.KindArray:
		e.BeginArray()
		for i := 0; i < v.Len(); i++ {
			e.raw(v.Index(i))
		}
		e.EndArray()
	case jsonvalue.KindObject:
		e.BeginObject()
		for _, k := range v.Keys() {
			e.Key(k)
			e.raw(v.Get(k))
		}
		e.EndObject()
	default:
		e.Fail("invalid JSON value")
	}
}

// NullValue is google.protobuf.NullValue, encoded as null.
type NullValue int32

// NullValue_NULL_VALUE is the only NullValue.
const NullValue_NULL_VALUE NullValue = 0

// Value writes v as google.protobuf.Value: objects as Structs with sorted
// keys and numbers as doubles.
func (e *Encoder) Value(v jsonvalue.Value) {
	switch v.Kind() {
	case jsonvalue.KindNull:
		e.Null()
	case jsonvalue.KindBool:
		e.Bool(v.Bool())
	case jsonvalue.KindNumber:
		f, err := strconv.ParseFloat(v.Number(), 64)
		if err != nil || isNaN(f) || isInf(f, 0) {
			e.Fail("google.protobuf.Value.number_value: invalid " + v.Number() + " value")
			return
		}
		e.Float64(f)
	case jsonvalue.KindString:
		e.String(v.String())
	case jsonvalue.KindArray:
		e.ListValue(v)
	case jsonvalue.KindObject:
		e.Struct(v)
	default:
		e.Fail("google.protobuf.Value: none of the oneof fields is set")
	}
}

// Struct writes v, which must be an object, as google.protobuf.Struct.
func (e *Encoder) Struct(v jsonvalue.Value) {
	if v.Kind() != jsonvalue.KindObject {
		e.Fail("google.protobuf.Struct: value is not a JSON object")
		return
	}
	e.BeginObject()
	for _, k := range SortedKeys(v.Keys()) {
		e.Key(k)
		e.Value(v.Get(k))
	}
	e.EndObject()
}

// ListValue writes v, which must be an array, as google.protobuf.ListValue.
func (e *Encoder) ListValue(v jsonvalue.Value) {
	if v.Kind() != jsonvalue.KindArray {
		e.Fail("google.protobuf.ListValue: value is not a JSON array")
		return
	}
	e.BeginArray()
	for i := 0; i < v.Len(); i++ {
		e.Value(v.Index(i))
	}
	e.EndArray()
}

func (d *Decoder) checkValue(v jsonvalue.Value) {
	switch v.Kind() {
	case jsonvalue.KindNumber:
		if _, err := strconv.ParseFloat(v.Number(), 64); err != nil {
			d.Fail("invalid google.protobuf.Value: " + v.Number())
		}
	case jsonvalue.KindArray:
		for i := 0; i < v.Len(); i++ {
			d.checkValue(v.Index(i))
		}
	case jsonvalue.KindObject:
		for _, k := range v.Keys() {
			d.checkValue(v.Get(k))
		}
	}
}

// Value decodes a google.protobuf.Value field; JSON null is a set value.
func (d *Decoder) Value(v jsonvalue.Value, field string) jsonvalue.Value {
	d.checkValue(v)
	return v
}

// Struct decodes a google.protobuf.Struct field.
func (d *Decoder) Struct(v jsonvalue.Value, field string) jsonvalue.Value {
	if v.Kind() != jsonvalue.KindObject {
		d.invalid("message", field, v)
		return jsonvalue.Value{}
	}
	d.checkValue(v)
	return v
}

// ListValue decodes a google.protobuf.ListValue field.
func (d *Decoder) ListValue(v jsonvalue.Value, field string) jsonvalue.Value {
	if v.Kind() != jsonvalue.KindArray {
		d.invalid("message", field, v)
		return jsonvalue.Value{}
	}
	d.checkValue(v)
	return v
}
