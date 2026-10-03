// Package strings manipulates UTF-8 encoded strings.
package strings

import (
	"github.com/eugenioenko/goalchemy/std/unicode"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

// Builder builds a string with appends.
type Builder struct {
	buf []byte
}

// String returns the accumulated string.
func (b *Builder) String() string { return string(b.buf) }

// Len returns the number of accumulated bytes.
func (b *Builder) Len() int { return len(b.buf) }

// Cap returns the capacity of the builder's buffer.
func (b *Builder) Cap() int { return cap(b.buf) }

// Reset empties the builder.
func (b *Builder) Reset() { b.buf = nil }

// Grow ensures space for another n bytes.
func (b *Builder) Grow(n int) {
	if n < 0 {
		panic("strings.Builder.Grow: negative count")
	}
	if cap(b.buf)-len(b.buf) < n {
		buf := make([]byte, len(b.buf), 2*cap(b.buf)+n)
		copy(buf, b.buf)
		b.buf = buf
	}
}

// Write appends p and always returns len(p), nil.
func (b *Builder) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteByte appends c and always returns nil.
func (b *Builder) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}

// WriteRune appends the UTF-8 encoding of r and returns its length.
func (b *Builder) WriteRune(r rune) (int, error) {
	n := len(b.buf)
	b.buf = utf8.AppendRune(b.buf, r)
	return len(b.buf) - n, nil
}

// WriteString appends s and always returns len(s), nil.
func (b *Builder) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}

// Compare returns 0 if a == b, -1 if a < b, and +1 if a > b.
func Compare(a, b string) int {
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}

// Index returns the index of the first instance of substr in s, or -1.
func Index(s, substr string) int {
	n := len(substr)
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// LastIndex returns the index of the last instance of substr in s, or -1.
func LastIndex(s, substr string) int {
	n := len(substr)
	for i := len(s) - n; i >= 0; i-- {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// IndexByte returns the index of the first instance of c in s, or -1.
func IndexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// LastIndexByte returns the index of the last instance of c in s, or -1.
func LastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// IndexRune returns the index of the first instance of r in s, or -1. For
// utf8.RuneError it matches the first invalid byte or encoded U+FFFD.
func IndexRune(s string, r rune) int {
	if 0 <= r && r < utf8.RuneSelf {
		return IndexByte(s, byte(r))
	}
	if r == utf8.RuneError {
		for i := 0; i < len(s); {
			c, size := utf8.DecodeRuneInString(s[i:])
			if c == utf8.RuneError {
				return i
			}
			i += size
		}
		return -1
	}
	if !utf8.ValidRune(r) {
		return -1
	}
	return Index(s, string(utf8.AppendRune(nil, r)))
}

// IndexFunc returns the index of the first rune satisfying f, or -1.
func IndexFunc(s string, f func(rune) bool) int {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if f(r) {
			return i
		}
		i += size
	}
	return -1
}

// LastIndexFunc returns the index of the last rune satisfying f, or -1.
func LastIndexFunc(s string, f func(rune) bool) int {
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		if f(r) {
			return i
		}
	}
	return -1
}

// IndexAny returns the index of the first rune of s that is in chars, or -1.
func IndexAny(s, chars string) int {
	if chars == "" {
		return -1
	}
	return IndexFunc(s, func(r rune) bool { return ContainsRune(chars, r) })
}

// LastIndexAny returns the index of the last rune of s that is in chars, or -1.
func LastIndexAny(s, chars string) int {
	if chars == "" {
		return -1
	}
	return LastIndexFunc(s, func(r rune) bool { return ContainsRune(chars, r) })
}

// Contains reports whether substr is within s.
func Contains(s, substr string) bool { return Index(s, substr) >= 0 }

// ContainsRune reports whether r is within s.
func ContainsRune(s string, r rune) bool { return IndexRune(s, r) >= 0 }

// ContainsAny reports whether any rune of chars is within s.
func ContainsAny(s, chars string) bool { return IndexAny(s, chars) >= 0 }

// ContainsFunc reports whether any rune of s satisfies f.
func ContainsFunc(s string, f func(rune) bool) bool { return IndexFunc(s, f) >= 0 }

// HasPrefix reports whether s begins with prefix.
func HasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

// HasSuffix reports whether s ends with suffix.
func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Count counts the non-overlapping instances of substr in s. An empty
// substr counts as one more than the number of runes in s.
func Count(s, substr string) int {
	if substr == "" {
		return utf8.RuneCountInString(s) + 1
	}
	n := 0
	for {
		i := Index(s, substr)
		if i < 0 {
			return n
		}
		n++
		s = s[i+len(substr):]
	}
}

// Cut slices s around the first instance of sep.
func Cut(s, sep string) (before, after string, found bool) {
	if i := Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// CutPrefix returns s without prefix and whether it was present.
func CutPrefix(s, prefix string) (after string, found bool) {
	if !HasPrefix(s, prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

// CutSuffix returns s without suffix and whether it was present.
func CutSuffix(s, suffix string) (before string, found bool) {
	if !HasSuffix(s, suffix) {
		return s, false
	}
	return s[:len(s)-len(suffix)], true
}

func explode(s string, n int) []string {
	l := utf8.RuneCountInString(s)
	if n < 0 || n > l {
		n = l
	}
	a := make([]string, n)
	for i := 0; i < n-1; i++ {
		_, size := utf8.DecodeRuneInString(s)
		a[i] = s[:size]
		s = s[size:]
	}
	if n > 0 {
		a[n-1] = s
	}
	return a
}

func genSplit(s, sep string, sepSave, n int) []string {
	if n == 0 {
		return nil
	}
	if sep == "" {
		return explode(s, n)
	}
	if n < 0 {
		n = Count(s, sep) + 1
	}
	if n > len(s)+1 {
		n = len(s) + 1
	}
	a := make([]string, n)
	n--
	i := 0
	for i < n {
		m := Index(s, sep)
		if m < 0 {
			break
		}
		a[i] = s[:m+sepSave]
		s = s[m+len(sep):]
		i++
	}
	a[i] = s
	return a[:i+1]
}

// Split slices s into all substrings separated by sep.
func Split(s, sep string) []string { return genSplit(s, sep, 0, -1) }

// SplitN slices s into at most n substrings separated by sep; n < 0 means all.
func SplitN(s, sep string, n int) []string { return genSplit(s, sep, 0, n) }

// SplitAfter slices s after each instance of sep.
func SplitAfter(s, sep string) []string { return genSplit(s, sep, len(sep), -1) }

// SplitAfterN slices s after each instance of sep into at most n substrings.
func SplitAfterN(s, sep string, n int) []string { return genSplit(s, sep, len(sep), n) }

// Fields splits s around runs of white space.
func Fields(s string) []string { return FieldsFunc(s, unicode.IsSpace) }

// FieldsFunc splits s around runs of runes satisfying f.
func FieldsFunc(s string, f func(rune) bool) []string {
	var out []string
	start := -1
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if f(r) {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
		i += size
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// Join concatenates elems with sep between them.
func Join(elems []string, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	var b Builder
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}

// Repeat returns count copies of s. It panics if count is negative.
func Repeat(s string, count int) string {
	if count < 0 {
		panic("strings: negative Repeat count")
	}
	var b Builder
	b.Grow(len(s) * count)
	for i := 0; i < count; i++ {
		b.WriteString(s)
	}
	return b.String()
}

// Replace returns s with the first n non-overlapping instances of old
// replaced by new; n < 0 replaces all. An empty old matches before every
// rune and at the end.
func Replace(s, old, new string, n int) string {
	if old == new || n == 0 {
		return s
	}
	if m := Count(s, old); m == 0 {
		return s
	} else if n < 0 || m < n {
		n = m
	}
	var b Builder
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if old == "" {
			if i > 0 {
				_, size := utf8.DecodeRuneInString(s[start:])
				j += size
			}
		} else {
			j += Index(s[start:], old)
		}
		b.WriteString(s[start:j])
		b.WriteString(new)
		start = j + len(old)
	}
	b.WriteString(s[start:])
	return b.String()
}

// ReplaceAll replaces every non-overlapping instance of old with new.
func ReplaceAll(s, old, new string) string { return Replace(s, old, new, -1) }

// Map returns s with every rune mapped by mapping; runes mapped to a
// negative value are dropped. Invalid UTF-8 bytes are mapped as RuneError.
func Map(mapping func(rune) rune, s string) string {
	var b Builder
	changed := false
	for i := 0; i < len(s); {
		c, size := utf8.DecodeRuneInString(s[i:])
		r := mapping(c)
		if r != c || c == utf8.RuneError && size == 1 {
			changed = true
		}
		if r >= 0 {
			b.WriteRune(r)
		}
		i += size
	}
	if !changed {
		return s
	}
	return b.String()
}

// ToUpper maps every rune of s to upper case.
func ToUpper(s string) string { return Map(unicode.ToUpper, s) }

// ToLower maps every rune of s to lower case.
func ToLower(s string) string { return Map(unicode.ToLower, s) }

// ToTitle maps every rune of s to title case.
func ToTitle(s string) string { return Map(unicode.ToTitle, s) }

// ToValidUTF8 replaces each run of invalid UTF-8 bytes with replacement.
func ToValidUTF8(s, replacement string) string {
	var b Builder
	invalid := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if !invalid {
				b.WriteString(replacement)
				invalid = true
			}
		} else {
			b.WriteString(s[i : i+size])
			invalid = false
		}
		i += size
	}
	return b.String()
}

// EqualFold reports whether s and t are equal under simple Unicode case
// folding.
func EqualFold(s, t string) bool {
	for s != "" && t != "" {
		sr, ss := utf8.DecodeRuneInString(s)
		tr, ts := utf8.DecodeRuneInString(t)
		s, t = s[ss:], t[ts:]
		if sr == tr {
			continue
		}
		if tr < sr {
			tr, sr = sr, tr
		}
		r := unicode.SimpleFold(sr)
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r != tr {
			return false
		}
	}
	return s == t
}

// TrimLeftFunc removes leading runes satisfying f.
func TrimLeftFunc(s string, f func(rune) bool) string {
	i := IndexFunc(s, func(r rune) bool { return !f(r) })
	if i < 0 {
		return ""
	}
	return s[i:]
}

// TrimRightFunc removes trailing runes satisfying f.
func TrimRightFunc(s string, f func(rune) bool) string {
	i := LastIndexFunc(s, func(r rune) bool { return !f(r) })
	if i < 0 {
		return ""
	}
	_, size := utf8.DecodeRuneInString(s[i:])
	return s[:i+size]
}

// TrimFunc removes leading and trailing runes satisfying f.
func TrimFunc(s string, f func(rune) bool) string { return TrimRightFunc(TrimLeftFunc(s, f), f) }

// TrimSpace removes leading and trailing white space.
func TrimSpace(s string) string { return TrimFunc(s, unicode.IsSpace) }

// Trim removes leading and trailing runes contained in cutset.
func Trim(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	return TrimFunc(s, func(r rune) bool { return ContainsRune(cutset, r) })
}

// TrimLeft removes leading runes contained in cutset.
func TrimLeft(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	return TrimLeftFunc(s, func(r rune) bool { return ContainsRune(cutset, r) })
}

// TrimRight removes trailing runes contained in cutset.
func TrimRight(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	return TrimRightFunc(s, func(r rune) bool { return ContainsRune(cutset, r) })
}

// TrimPrefix removes prefix from s if present.
func TrimPrefix(s, prefix string) string {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix removes suffix from s if present.
func TrimSuffix(s, suffix string) string {
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// Replacer replaces a list of strings with replacements.
type Replacer struct {
	oldnew []string
}

// NewReplacer returns a Replacer from old, new string pairs. Replacements
// happen in target string order without overlapping; at the same position,
// earlier pairs win. It panics on an odd argument count.
func NewReplacer(oldnew ...string) *Replacer {
	if len(oldnew)%2 == 1 {
		panic("strings.NewReplacer: odd argument count")
	}
	return &Replacer{oldnew: append([]string(nil), oldnew...)}
}

// Replace returns s with all replacements performed.
func (r *Replacer) Replace(s string) string {
	var b Builder
	prevEmpty := false
	for i := 0; i <= len(s); {
		matched := false
		for k := 0; k < len(r.oldnew); k += 2 {
			old := r.oldnew[k]
			if old == "" && prevEmpty {
				continue
			}
			if HasPrefix(s[i:], old) {
				b.WriteString(r.oldnew[k+1])
				i += len(old)
				prevEmpty = old == ""
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		prevEmpty = false
		if i == len(s) {
			break
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
