package json

import "github.com/eugenioenko/goalchemy/std/bytes"

// HTMLEscape appends to dst the JSON-encoded src with <, >, &, U+2028 and U+2029
// characters inside string literals changed to \u003c, \u003e, \u0026, \u2028, \u2029
// so that the JSON will be safe to embed inside HTML <script> tags.
// For historical reasons, web browsers don't honor standard HTML
// escaping within <script> tags, so an alternative JSON encoding must be used.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	dst.Write(appendHTMLEscape(nil, src))
}

func appendHTMLEscape(dst, src []byte) []byte {
	start := 0
	for i, c := range src {
		if c == '<' || c == '>' || c == '&' {
			dst = append(dst, src[start:i]...)
			dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xF])
			start = i + 1
		}

		if c == 0xE2 && i+2 < len(src) && src[i+1] == 0x80 && src[i+2]&^1 == 0xA8 {
			dst = append(dst, src[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[src[i+2]&0xF])
			start = i + len("\u2029")
		}
	}
	return append(dst, src[start:]...)
}

// Compact appends to dst the JSON-encoded src with
// insignificant space characters elided.
func Compact(dst *bytes.Buffer, src []byte) error {
	b, err := appendCompact(nil, src, false)
	dst.Write(b)
	return err
}

func appendCompact(dst, src []byte, escape bool) ([]byte, error) {
	return compactValue(dst, src, escape, 0, 1)
}

func compactValue(dst, src []byte, escape bool, base, depth int) ([]byte, error) {
	if err := checkTopLevel(src, base, depth); err != nil {
		return dst, err
	}
	origLen := len(dst)
	scan := newScanner()
	start := 0
	for i, c := range src {
		if escape && (c == '<' || c == '>' || c == '&') {
			if start < i {
				dst = append(dst, src[start:i]...)
			}
			dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xF])
			start = i + 1
		}

		if escape && c == 0xE2 && i+2 < len(src) && src[i+1] == 0x80 && src[i+2]&^1 == 0xA8 {
			if start < i {
				dst = append(dst, src[start:i]...)
			}
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[src[i+2]&0xF])
			start = i + 3
		}
		v := scan.step(scan, c)
		if v >= scanSkipSpace {
			if v == scanError {
				break
			}
			if start < i {
				dst = append(dst, src[start:i]...)
			}
			start = i + 1
		}
	}
	if scan.eof() == scanError {
		return dst[:origLen], scan.err
	}
	if start < len(src) {
		dst = append(dst, src[start:]...)
	}
	return dst, nil
}

func appendNewline(dst []byte, prefix, indent string, depth int) []byte {
	dst = append(dst, '\n')
	dst = append(dst, prefix...)
	for i := 0; i < depth; i++ {
		dst = append(dst, indent...)
	}
	return dst
}

// Indent appends to dst an indented form of the JSON-encoded src.
// Each element in a JSON object or array begins on a new,
// indented line beginning with prefix followed by one or more
// copies of indent according to the indentation nesting.
// The data appended to dst does not begin with the prefix nor
// any indentation, to make it easier to embed inside other formatted JSON data.
// Although leading space characters (space, tab, carriage return, newline)
// at the beginning of src are dropped, trailing space characters
// at the end of src are preserved and copied to dst.
// For example, if src has no trailing spaces, neither will dst;
// if src ends in a trailing newline, so will dst.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	b, err := appendIndent(nil, src, prefix, indent)
	dst.Write(b)
	return err
}

func appendIndent(dst, src []byte, prefix, indent string) ([]byte, error) {
	if err := checkTopLevel(src, 0, 1); err != nil {
		return dst, err
	}
	origLen := len(dst)
	scan := newScanner()
	needIndent := false
	depth := 0
	for _, c := range src {
		scan.bytes++
		v := scan.step(scan, c)
		if v == scanSkipSpace {
			continue
		}
		if v == scanError {
			break
		}
		if needIndent && v != scanEndObject && v != scanEndArray {
			needIndent = false
			depth++
			dst = appendNewline(dst, prefix, indent, depth)
		}

		if v == scanContinue {
			dst = append(dst, c)
			continue
		}

		switch c {
		case '{', '[':

			needIndent = true
			dst = append(dst, c)
		case ',':
			dst = append(dst, c)
			dst = appendNewline(dst, prefix, indent, depth)
		case ':':
			dst = append(dst, c, ' ')
		case '}', ']':
			if needIndent {
				needIndent = false
			} else {
				depth--
				dst = appendNewline(dst, prefix, indent, depth)
			}
			dst = append(dst, c)
		default:
			dst = append(dst, c)
		}
	}
	if scan.eof() == scanError {
		return dst[:origLen], scan.err
	}
	return dst, nil
}
