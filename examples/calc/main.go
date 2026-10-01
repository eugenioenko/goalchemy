// Command calc parses and evaluates integer arithmetic expressions.
package main

type tokenKind int

const (
	tNum tokenKind = iota
	tOp
	tLParen
	tRParen
)

type token struct {
	kind tokenKind
	val  int64
	op   byte
}

type ParseError struct {
	Pos int
	Msg string
}

func (e ParseError) Error() string { return e.Msg }

func tokenize(s string) ([]token, error) {
	var toks []token
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
		case c >= '0' && c <= '9':
			var n int64
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				n = n*10 + int64(s[i]-'0')
				i++
			}
			i--
			toks = append(toks, token{kind: tNum, val: n})
		case c == '+' || c == '-' || c == '*' || c == '/' || c == '%':
			toks = append(toks, token{kind: tOp, op: c})
		case c == '(':
			toks = append(toks, token{kind: tLParen})
		case c == ')':
			toks = append(toks, token{kind: tRParen})
		default:
			return nil, ParseError{i, "unexpected character " + string(rune(c))}
		}
	}
	return toks, nil
}

type Node interface{ Eval() int64 }

type Num int64
type BinOp struct {
	Op   byte
	L, R Node
}

func (n Num) Eval() int64 { return int64(n) }

func (b BinOp) Eval() int64 {
	l, r := b.L.Eval(), b.R.Eval()
	switch b.Op {
	case '+':
		return l + r
	case '-':
		return l - r
	case '*':
		return l * r
	case '/':
		return l / r
	}
	return l % r
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() (token, bool) {
	if p.pos < len(p.toks) {
		return p.toks[p.pos], true
	}
	return token{}, false
}

func (p *parser) expr() Node {
	n := p.term()
	for {
		t, ok := p.peek()
		if !ok || t.kind != tOp || (t.op != '+' && t.op != '-') {
			return n
		}
		p.pos++
		n = BinOp{t.op, n, p.term()}
	}
}

func (p *parser) term() Node {
	n := p.factor()
	for {
		t, ok := p.peek()
		if !ok || t.kind != tOp || (t.op != '*' && t.op != '/' && t.op != '%') {
			return n
		}
		p.pos++
		n = BinOp{t.op, n, p.factor()}
	}
}

func (p *parser) factor() Node {
	t, ok := p.peek()
	if !ok {
		panic(ParseError{p.pos, "unexpected end of input"})
	}
	p.pos++
	switch t.kind {
	case tNum:
		return Num(t.val)
	case tLParen:
		n := p.expr()
		if c, ok := p.peek(); !ok || c.kind != tRParen {
			panic(ParseError{p.pos, "missing )"})
		}
		p.pos++
		return n
	case tOp:
		if t.op == '-' {
			return BinOp{'-', Num(0), p.factor()}
		}
	}
	panic(ParseError{p.pos, "unexpected token"})
}

func evaluate(src string) (result int64, err error) {
	toks, err := tokenize(src)
	if err != nil {
		return 0, err
	}
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(ParseError); ok {
				err = pe
				return
			}
			err = r.(error)
		}
	}()
	p := &parser{toks: toks}
	n := p.expr()
	if p.pos != len(toks) {
		return 0, ParseError{p.pos, "trailing input"}
	}
	return n.Eval(), nil
}

func main() {
	for _, src := range []string{"1 + 2 * 3", "(1 + 2) * 3", "-(4 - 10) % 4", "9223372036854775807 + 1", "8 / (3 - 3)", "2 * (3 + ", "7 $ 2"} {
		v, err := evaluate(src)
		if err != nil {
			println(src, "=> error:", err.Error())
			continue
		}
		println(src, "=>", v)
	}
}
