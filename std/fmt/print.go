// Package fmt formats values into strings and errors with Go's verbs, flags,
// widths, precisions and argument indexes.
//
// Without reflection, operands are formatted when they are nil, a boolean,
// integer, float or string of a predeclared type, a []byte, or a value whose
// type implements Formatter, GoStringer (for %#v), error or Stringer. Other
// operands, such as structs, maps and named basic types without methods,
// print as %!v(unsupported); the compiler rejects them when their static type
// is known. %T and the type names in bad-verb and extra-argument notes are
// known only for the supported basic types and print as ? otherwise. %p is
// unsupported. A nil pointer whose Error or String method panics prints as
// <nil> when the panic is a nil dereference.
package fmt

import (
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

const (
	commaSpaceString  = ", "
	nilAngleString    = "<nil>"
	percentBangString = "%!"
	missingString     = "(MISSING)"
	badIndexString    = "(BADINDEX)"
	panicString       = "(PANIC="
	extraString       = "%!(EXTRA "
	badWidthString    = "%!(BADWIDTH)"
	badPrecString     = "%!(BADPREC)"
	noVerbString      = "%!(NOVERB)"
	unsupportedString = "(unsupported)"
)

// State represents the printer state passed to custom formatters. It
// provides access to the output buffer and the format options.
type State interface {
	// Write is the function to call to emit formatted output to be printed.
	Write(b []byte) (n int, err error)
	// Width returns the value of the width option and whether it has been set.
	Width() (wid int, ok bool)
	// Precision returns the value of the precision option and whether it has been set.
	Precision() (prec int, ok bool)
	// Flag reports whether the flag c, a character, has been set.
	Flag(c int) bool
}

// Formatter is implemented by any value that has a Format method. The
// implementation controls how State and rune are interpreted.
type Formatter interface {
	Format(f State, verb rune)
}

// Stringer is implemented by any value that has a String method, which
// defines the native format for that value.
type Stringer interface {
	String() string
}

// GoStringer is implemented by any value that has a GoString method, which
// defines the Go syntax for that value used by %#v.
type GoStringer interface {
	GoString() string
}

// FormatString returns a string representing the fully qualified formatting
// directive captured by the State, followed by the argument verb.
func FormatString(state State, verb rune) string {
	b := []byte{'%'}
	for _, c := range " +-#0" {
		if state.Flag(int(c)) {
			b = append(b, byte(c))
		}
	}
	if w, ok := state.Width(); ok {
		b = strconv.AppendInt(b, int64(w), 10)
	}
	if p, ok := state.Precision(); ok {
		b = append(b, '.')
		b = strconv.AppendInt(b, int64(p), 10)
	}
	b = utf8.AppendRune(b, verb)
	return string(b)
}

type buffer []byte

func (b *buffer) write(p []byte) {
	*b = append(*b, p...)
}

func (b *buffer) writeString(s string) {
	*b = append(*b, s...)
}

func (b *buffer) writeByte(c byte) {
	*b = append(*b, c)
}

func (b *buffer) writeRune(r rune) {
	*b = utf8.AppendRune(*b, r)
}

// pp is used to store a printer's state.
type pp struct {
	buf buffer

	fmt formatter

	reordered   bool
	goodArgNum  bool
	panicking   bool
	erroring    bool
	wrapErrs    bool
	wrappedErrs []int
}

func newPrinter() *pp {
	p := &pp{}
	p.fmt.init(&p.buf)
	return p
}

func (p *pp) Width() (wid int, ok bool) { return p.fmt.wid, p.fmt.widPresent }

func (p *pp) Precision() (prec int, ok bool) { return p.fmt.prec, p.fmt.precPresent }

func (p *pp) Flag(b int) bool {
	switch b {
	case '-':
		return p.fmt.minus
	case '+':
		return p.fmt.plus || p.fmt.plusV
	case '#':
		return p.fmt.sharp || p.fmt.sharpV
	case ' ':
		return p.fmt.space
	case '0':
		return p.fmt.zero
	}
	return false
}

func (p *pp) Write(b []byte) (ret int, err error) {
	p.buf.write(b)
	return len(b), nil
}

// Sprintf formats according to a format specifier and returns the resulting string.
func Sprintf(format string, a ...any) string {
	p := newPrinter()
	p.doPrintf(format, a)
	return string(p.buf)
}

// Appendf formats according to a format specifier, appends the result to
// the byte slice, and returns the updated slice.
func Appendf(b []byte, format string, a ...any) []byte {
	p := newPrinter()
	p.doPrintf(format, a)
	return append(b, p.buf...)
}

// Sprint formats using the default formats for its operands and returns the
// resulting string. Spaces are added between operands when neither is a string.
func Sprint(a ...any) string {
	p := newPrinter()
	p.doPrint(a)
	return string(p.buf)
}

// Append formats using the default formats for its operands, appends the
// result to the byte slice, and returns the updated slice.
func Append(b []byte, a ...any) []byte {
	p := newPrinter()
	p.doPrint(a)
	return append(b, p.buf...)
}

// Sprintln formats using the default formats for its operands and returns
// the resulting string. Spaces are always added between operands and a
// newline is appended.
func Sprintln(a ...any) string {
	p := newPrinter()
	p.doPrintln(a)
	return string(p.buf)
}

// Appendln formats using the default formats for its operands, appends the
// result to the byte slice, and returns the updated slice.
func Appendln(b []byte, a ...any) []byte {
	p := newPrinter()
	p.doPrintln(a)
	return append(b, p.buf...)
}

func tooLarge(x int) bool {
	const max int = 1e6
	return x > max || x < -max
}

func parsenum(s string, start, end int) (num int, isnum bool, newi int) {
	if start >= end {
		return 0, false, end
	}
	for newi = start; newi < end && '0' <= s[newi] && s[newi] <= '9'; newi++ {
		if tooLarge(num) {
			return 0, false, end
		}
		num = num*10 + int(s[newi]-'0')
		isnum = true
	}
	return
}

// typeString returns the Go type name of a supported basic operand.
func typeString(arg any) (string, bool) {
	switch arg.(type) {
	case bool:
		return "bool", true
	case int:
		return "int", true
	case int8:
		return "int8", true
	case int16:
		return "int16", true
	case int32:
		return "int32", true
	case int64:
		return "int64", true
	case uint:
		return "uint", true
	case uint8:
		return "uint8", true
	case uint16:
		return "uint16", true
	case uint32:
		return "uint32", true
	case uint64:
		return "uint64", true
	case float32:
		return "float32", true
	case float64:
		return "float64", true
	case string:
		return "string", true
	case []byte:
		return "[]uint8", true
	}
	return "?", false
}

func (p *pp) badVerb(arg any, verb rune) {
	p.erroring = true
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeByte('(')
	if arg != nil {
		name, _ := typeString(arg)
		p.buf.writeString(name)
		p.buf.writeByte('=')
		p.printArg(arg, 'v')
	} else {
		p.buf.writeString(nilAngleString)
	}
	p.buf.writeByte(')')
	p.erroring = false
}

func (p *pp) unsupported(verb rune) {
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeString(unsupportedString)
}

func (p *pp) fmtBool(arg any, v bool, verb rune) {
	switch verb {
	case 't', 'v':
		p.fmt.fmtBoolean(v)
	default:
		p.badVerb(arg, verb)
	}
}

func (p *pp) fmt0x64(v uint64, leading0x bool) {
	sharp := p.fmt.sharp
	p.fmt.sharp = leading0x
	p.fmt.fmtInteger(v, 16, unsigned, 'v', ldigits)
	p.fmt.sharp = sharp
}

func (p *pp) fmtInteger(arg any, v uint64, isSigned bool, verb rune) {
	switch verb {
	case 'v':
		if p.fmt.sharpV && !isSigned {
			p.fmt0x64(v, true)
		} else {
			p.fmt.fmtInteger(v, 10, isSigned, verb, ldigits)
		}
	case 'd':
		p.fmt.fmtInteger(v, 10, isSigned, verb, ldigits)
	case 'b':
		p.fmt.fmtInteger(v, 2, isSigned, verb, ldigits)
	case 'o', 'O':
		p.fmt.fmtInteger(v, 8, isSigned, verb, ldigits)
	case 'x':
		p.fmt.fmtInteger(v, 16, isSigned, verb, ldigits)
	case 'X':
		p.fmt.fmtInteger(v, 16, isSigned, verb, udigits)
	case 'c':
		p.fmt.fmtC(v)
	case 'q':
		p.fmt.fmtQc(v)
	case 'U':
		p.fmt.fmtUnicode(v)
	default:
		p.badVerb(arg, verb)
	}
}

func (p *pp) fmtFloat(arg any, v float64, size int, verb rune) {
	switch verb {
	case 'v':
		p.fmt.fmtFloat(v, size, 'g', -1)
	case 'b', 'g', 'G', 'x', 'X':
		p.fmt.fmtFloat(v, size, verb, -1)
	case 'f', 'e', 'E':
		p.fmt.fmtFloat(v, size, verb, 6)
	case 'F':
		p.fmt.fmtFloat(v, size, 'f', 6)
	default:
		p.badVerb(arg, verb)
	}
}

func (p *pp) fmtString(arg any, v string, verb rune) {
	switch verb {
	case 'v':
		if p.fmt.sharpV {
			p.fmt.fmtQ(v)
		} else {
			p.fmt.fmtS(v)
		}
	case 's':
		p.fmt.fmtS(v)
	case 'x':
		p.fmt.fmtSx(v, ldigits)
	case 'X':
		p.fmt.fmtSx(v, udigits)
	case 'q':
		p.fmt.fmtQ(v)
	default:
		p.badVerb(arg, verb)
	}
}

func (p *pp) fmtBytes(v []byte, verb rune) {
	switch verb {
	case 'v', 'd':
		if p.fmt.sharpV {
			p.buf.writeString("[]byte")
			if v == nil {
				p.buf.writeString("(nil)")
				return
			}
			p.buf.writeByte('{')
			for i, c := range v {
				if i > 0 {
					p.buf.writeString(commaSpaceString)
				}
				p.fmt0x64(uint64(c), true)
			}
			p.buf.writeByte('}')
		} else {
			p.buf.writeByte('[')
			for i, c := range v {
				if i > 0 {
					p.buf.writeByte(' ')
				}
				p.fmt.fmtInteger(uint64(c), 10, unsigned, verb, ldigits)
			}
			p.buf.writeByte(']')
		}
	case 's':
		p.fmt.fmtBs(v)
	case 'x':
		p.fmt.fmtBx(v, ldigits)
	case 'X':
		p.fmt.fmtBx(v, udigits)
	case 'q':
		p.fmt.fmtQ(string(v))
	default:
		if p.fmt.sharpV {
			p.buf.writeString("[]uint8")
			if v == nil {
				p.buf.writeString("(nil)")
				return
			}
			p.buf.writeByte('{')
			for i, c := range v {
				if i > 0 {
					p.buf.writeString(commaSpaceString)
				}
				p.fmtInteger(c, uint64(c), unsigned, verb)
			}
			p.buf.writeByte('}')
			return
		}
		p.buf.writeByte('[')
		for i, c := range v {
			if i > 0 {
				p.buf.writeByte(' ')
			}
			p.fmtInteger(c, uint64(c), unsigned, verb)
		}
		p.buf.writeByte(']')
	}
}

func (p *pp) catchPanic(verb rune, method string) {
	if err := recover(); err != nil {
		msg := Sprint(err)
		if strings.Contains(msg, "nil pointer dereference") {
			p.buf.writeString(nilAngleString)
			return
		}
		if p.panicking {
			panic(err)
		}
		oldFlags := p.fmt.fmtFlags
		p.fmt.clearflags()
		p.buf.writeString(percentBangString)
		p.buf.writeRune(verb)
		p.buf.writeString(panicString)
		p.buf.writeString(method)
		p.buf.writeString(" method: ")
		p.panicking = true
		p.printArg(err, 'v')
		p.panicking = false
		p.buf.writeByte(')')
		p.fmt.fmtFlags = oldFlags
	}
}

func (p *pp) handleMethods(arg any, verb rune) (handled bool) {
	if p.erroring {
		return
	}
	if verb == 'w' {
		_, ok := arg.(error)
		if !ok || !p.wrapErrs {
			p.badVerb(arg, verb)
			return true
		}
		verb = 'v'
	}
	if formatter, ok := arg.(Formatter); ok {
		handled = true
		defer p.catchPanic(verb, "Format")
		formatter.Format(p, verb)
		return
	}
	if p.fmt.sharpV {
		if stringer, ok := arg.(GoStringer); ok {
			handled = true
			defer p.catchPanic(verb, "GoString")
			p.fmt.fmtS(stringer.GoString())
			return
		}
	} else {
		switch verb {
		case 'v', 's', 'x', 'X', 'q':
			switch v := arg.(type) {
			case error:
				handled = true
				defer p.catchPanic(verb, "Error")
				p.fmtString(arg, v.Error(), verb)
				return
			case Stringer:
				handled = true
				defer p.catchPanic(verb, "String")
				p.fmtString(arg, v.String(), verb)
				return
			}
		}
	}
	return false
}

func (p *pp) printArg(arg any, verb rune) {
	if arg == nil {
		switch verb {
		case 'T', 'v':
			p.fmt.padString(nilAngleString)
		default:
			p.badVerb(arg, verb)
		}
		return
	}
	switch verb {
	case 'T':
		if name, ok := typeString(arg); ok {
			p.fmt.fmtS(name)
		} else {
			p.unsupported(verb)
		}
		return
	case 'p':
		p.unsupported(verb)
		return
	}
	switch f := arg.(type) {
	case bool:
		p.fmtBool(arg, f, verb)
	case float32:
		p.fmtFloat(arg, float64(f), 32, verb)
	case float64:
		p.fmtFloat(arg, f, 64, verb)
	case int:
		p.fmtInteger(arg, uint64(f), signed, verb)
	case int8:
		p.fmtInteger(arg, uint64(f), signed, verb)
	case int16:
		p.fmtInteger(arg, uint64(f), signed, verb)
	case int32:
		p.fmtInteger(arg, uint64(f), signed, verb)
	case int64:
		p.fmtInteger(arg, uint64(f), signed, verb)
	case uint:
		p.fmtInteger(arg, uint64(f), unsigned, verb)
	case uint8:
		p.fmtInteger(arg, uint64(f), unsigned, verb)
	case uint16:
		p.fmtInteger(arg, uint64(f), unsigned, verb)
	case uint32:
		p.fmtInteger(arg, uint64(f), unsigned, verb)
	case uint64:
		p.fmtInteger(arg, f, unsigned, verb)
	case string:
		p.fmtString(arg, f, verb)
	case []byte:
		p.fmtBytes(f, verb)
	default:
		if !p.handleMethods(arg, verb) {
			p.unsupported(verb)
		}
	}
}

func intFromArg(a []any, argNum int) (num int, isInt bool, newArgNum int) {
	newArgNum = argNum
	if argNum < len(a) {
		switch v := a[argNum].(type) {
		case int:
			num, isInt = v, true
		case int8:
			num, isInt = int(v), true
		case int16:
			num, isInt = int(v), true
		case int32:
			num, isInt = int(v), true
		case int64:
			num, isInt = int(v), true
		case uint:
			num, isInt = int(v), int(v) >= 0
		case uint8:
			num, isInt = int(v), true
		case uint16:
			num, isInt = int(v), true
		case uint32:
			num, isInt = int(v), true
		case uint64:
			num, isInt = int(v), int(v) >= 0
		}
		if !isInt {
			num = 0
		}
		newArgNum = argNum + 1
		if tooLarge(num) {
			num = 0
			isInt = false
		}
	}
	return
}

func parseArgNumber(format string) (index int, wid int, ok bool) {
	if len(format) < 3 {
		return 0, 1, false
	}
	for i := 1; i < len(format); i++ {
		if format[i] == ']' {
			width, ok, newi := parsenum(format, 1, i)
			if !ok || newi != i {
				return 0, i + 1, false
			}
			return width - 1, i + 1, true
		}
	}
	return 0, 1, false
}

func (p *pp) argNumber(argNum int, format string, i int, numArgs int) (newArgNum, newi int, found bool) {
	if len(format) <= i || format[i] != '[' {
		return argNum, i, false
	}
	p.reordered = true
	index, wid, ok := parseArgNumber(format[i:])
	if ok && 0 <= index && index < numArgs {
		return index, i + wid, true
	}
	p.goodArgNum = false
	return argNum, i + wid, ok
}

func (p *pp) badArgNum(verb rune) {
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeString(badIndexString)
}

func (p *pp) missingArg(verb rune) {
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeString(missingString)
}

func (p *pp) doPrintf(format string, a []any) {
	end := len(format)
	argNum := 0
	afterIndex := false
	p.reordered = false
formatLoop:
	for i := 0; i < end; {
		p.goodArgNum = true
		lasti := i
		for i < end && format[i] != '%' {
			i++
		}
		if i > lasti {
			p.buf.writeString(format[lasti:i])
		}
		if i >= end {
			break
		}
		i++
		p.fmt.clearflags()
	simpleFormat:
		for ; i < end; i++ {
			c := format[i]
			switch c {
			case '#':
				p.fmt.sharp = true
			case '0':
				p.fmt.zero = true
			case '+':
				p.fmt.plus = true
			case '-':
				p.fmt.minus = true
			case ' ':
				p.fmt.space = true
			default:
				if 'a' <= c && c <= 'z' && argNum < len(a) {
					switch c {
					case 'w':
						p.wrappedErrs = append(p.wrappedErrs, argNum)
						p.fmt.sharpV = p.fmt.sharp
						p.fmt.sharp = false
						p.fmt.plusV = p.fmt.plus
						p.fmt.plus = false
					case 'v':
						p.fmt.sharpV = p.fmt.sharp
						p.fmt.sharp = false
						p.fmt.plusV = p.fmt.plus
						p.fmt.plus = false
					}
					p.printArg(a[argNum], rune(c))
					argNum++
					i++
					continue formatLoop
				}
				break simpleFormat
			}
		}

		argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))

		if i < end && format[i] == '*' {
			i++
			p.fmt.wid, p.fmt.widPresent, argNum = intFromArg(a, argNum)
			if !p.fmt.widPresent {
				p.buf.writeString(badWidthString)
			}
			if p.fmt.wid < 0 {
				p.fmt.wid = -p.fmt.wid
				p.fmt.minus = true
				p.fmt.zero = false
			}
			afterIndex = false
		} else {
			p.fmt.wid, p.fmt.widPresent, i = parsenum(format, i, end)
			if afterIndex && p.fmt.widPresent {
				p.goodArgNum = false
			}
		}

		if i+1 < end && format[i] == '.' {
			i++
			if afterIndex {
				p.goodArgNum = false
			}
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))
			if i < end && format[i] == '*' {
				i++
				p.fmt.prec, p.fmt.precPresent, argNum = intFromArg(a, argNum)
				if p.fmt.prec < 0 {
					p.fmt.prec = 0
					p.fmt.precPresent = false
				}
				if !p.fmt.precPresent {
					p.buf.writeString(badPrecString)
				}
				afterIndex = false
			} else {
				p.fmt.prec, p.fmt.precPresent, i = parsenum(format, i, end)
				if !p.fmt.precPresent {
					p.fmt.prec = 0
					p.fmt.precPresent = true
				}
			}
		}

		if !afterIndex {
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))
		}

		if i >= end {
			p.buf.writeString(noVerbString)
			break
		}

		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size

		switch {
		case verb == '%':
			p.buf.writeByte('%')
		case !p.goodArgNum:
			p.badArgNum(verb)
		case argNum >= len(a):
			p.missingArg(verb)
		case verb == 'w' || verb == 'v':
			if verb == 'w' {
				p.wrappedErrs = append(p.wrappedErrs, argNum)
			}
			p.fmt.sharpV = p.fmt.sharp
			p.fmt.sharp = false
			p.fmt.plusV = p.fmt.plus
			p.fmt.plus = false
			p.printArg(a[argNum], verb)
			argNum++
		default:
			p.printArg(a[argNum], verb)
			argNum++
		}
	}

	if !p.reordered && argNum < len(a) {
		p.fmt.clearflags()
		p.buf.writeString(extraString)
		for i, arg := range a[argNum:] {
			if i > 0 {
				p.buf.writeString(commaSpaceString)
			}
			if arg == nil {
				p.buf.writeString(nilAngleString)
			} else {
				name, _ := typeString(arg)
				p.buf.writeString(name)
				p.buf.writeByte('=')
				p.printArg(arg, 'v')
			}
		}
		p.buf.writeByte(')')
	}
}

func (p *pp) doPrint(a []any) {
	prevString := false
	for argNum, arg := range a {
		_, isString := arg.(string)
		if argNum > 0 && !isString && !prevString {
			p.buf.writeByte(' ')
		}
		p.printArg(arg, 'v')
		prevString = isString
	}
}

func (p *pp) doPrintln(a []any) {
	for argNum, arg := range a {
		if argNum > 0 {
			p.buf.writeByte(' ')
		}
		p.printArg(arg, 'v')
	}
	p.buf.writeByte('\n')
}
