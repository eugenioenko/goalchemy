package strconv

const fnParseFloat = "ParseFloat"

func commonPrefixLenIgnoreCase(s, prefix string) int {
	n := min(len(prefix), len(s))
	for i := 0; i < n; i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != prefix[i] {
			return i
		}
	}
	return n
}

func pow2(k int) float64 {
	f := 1.0
	for k >= 64 {
		f *= 0x1p64
		k -= 64
	}
	for k <= -64 {
		f *= 0x1p-64
		k += 64
	}
	for k > 0 {
		f *= 2
		k--
	}
	for k < 0 {
		f *= 0.5
		k++
	}
	return f
}

func inf(sign int) float64 {
	f := pow2(1023) * 2
	if sign < 0 {
		return -f
	}
	return f
}

func nan() float64 {
	f := inf(1)
	return f - f
}

func float64frombits(b uint64) float64 {
	e := int(b>>52) & 0x7FF
	m := b & (1<<52 - 1)
	var f float64
	switch {
	case e == 0x7FF && m != 0:
		return nan()
	case e == 0x7FF:
		f = inf(1)
	case e == 0:
		f = float64(m) * pow2(-1074)
	default:
		f = float64(m|1<<52) * pow2(e-1075)
	}
	if b>>63 != 0 {
		return -f
	}
	return f
}

func float32frombits(b uint32) float32 {
	e := int(b>>23) & 0xFF
	m := uint64(b & (1<<23 - 1))
	var f float64
	switch {
	case e == 0xFF && m != 0:
		return float32(nan())
	case e == 0xFF:
		f = inf(1)
	case e == 0:
		f = float64(m) * pow2(-149)
	default:
		f = float64(m|1<<23) * pow2(e-150)
	}
	if b>>31 != 0 {
		f = -f
	}
	return float32(f)
}

func special(s string) (float64, int, bool) {
	if len(s) == 0 {
		return 0, 0, false
	}
	sign := 1
	nsign := 0
	c := s[0]
	if c == '+' || c == '-' {
		if c == '-' {
			sign = -1
		}
		nsign = 1
		s = s[1:]
		c = 'i'
	}
	switch c {
	case 'i', 'I':
		n := commonPrefixLenIgnoreCase(s, "infinity")
		if 3 < n && n < 8 {
			n = 3
		}
		if n == 3 || n == 8 {
			return inf(sign), nsign + n, true
		}
	case 'n', 'N':
		if commonPrefixLenIgnoreCase(s, "nan") == 3 {
			return nan(), 3, true
		}
	}
	return 0, 0, false
}

func (b *decimal) set(s string) bool {
	i := 0
	b.neg = false
	b.trunc = false
	if i >= len(s) {
		return false
	}
	switch s[i] {
	case '+':
		i++
	case '-':
		i++
		b.neg = true
	}
	sawdot := false
	sawdigits := false
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			continue
		}
		if c == '.' {
			if sawdot {
				return false
			}
			sawdot = true
			b.dp = b.nd
			continue
		}
		if '0' <= c && c <= '9' {
			sawdigits = true
			if c == '0' && b.nd == 0 {
				b.dp--
				continue
			}
			if b.nd < len(b.d) {
				b.d[b.nd] = c
				b.nd++
			} else if c != '0' {
				b.trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return false
	}
	if !sawdot {
		b.dp = b.nd
	}
	if i < len(s) && lower(s[i]) == 'e' {
		i++
		if i >= len(s) {
			return false
		}
		esign := 1
		switch s[i] {
		case '+':
			i++
		case '-':
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return false
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		b.dp += e * esign
	}
	return i == len(s)
}

type floatParts struct {
	mantissa uint64
	exp      int
	neg      bool
	trunc    bool
	hex      bool
	n        int
}

func readFloat(s string) (floatParts, bool) {
	var p floatParts
	underscores := false
	i := 0
	if i >= len(s) {
		return p, false
	}
	switch s[i] {
	case '+':
		i++
	case '-':
		i++
		p.neg = true
	}
	base := uint64(10)
	maxMantDigits := 19
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base = 16
		maxMantDigits = 16
		i += 2
		expChar = 'p'
		p.hex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			underscores = true
			continue
		}
		if c == '.' {
			if sawdot {
				break
			}
			sawdot = true
			dp = nd
			continue
		}
		if '0' <= c && c <= '9' {
			sawdigits = true
			if c == '0' && nd == 0 {
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				p.mantissa *= base
				p.mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				p.trunc = true
			}
			continue
		}
		if base == 16 && 'a' <= lower(c) && lower(c) <= 'f' {
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				p.mantissa *= 16
				p.mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				p.trunc = true
			}
			continue
		}
		break
	}
	p.n = i
	if !sawdigits {
		return p, false
	}
	if !sawdot {
		dp = nd
	}
	if base == 16 {
		dp *= 4
		ndMant *= 4
	}
	if i < len(s) && lower(s[i]) == expChar {
		i++
		p.n = i
		if i >= len(s) {
			return p, false
		}
		esign := 1
		switch s[i] {
		case '+':
			i++
		case '-':
			i++
			esign = -1
		}
		p.n = i
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return p, false
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		p.n = i
		dp += e * esign
	} else if base == 16 {
		return p, false
	}
	if p.mantissa != 0 {
		p.exp = dp - ndMant
	}
	if underscores && !underscoreOK(s[:i]) {
		return p, false
	}
	return p, true
}

var powtab = []int{1, 3, 6, 9, 13, 16, 19, 23, 26}

func (d *decimal) floatBits(flt *floatInfo) (uint64, bool) {
	exp := 0
	var mant uint64
	overflow := false
	switch {
	case d.nd == 0 || d.dp < -330:
		exp = flt.bias
	case d.dp > 310:
		overflow = true
	default:
		for d.dp > 0 {
			n := 27
			if d.dp < len(powtab) {
				n = powtab[d.dp]
			}
			d.Shift(-n)
			exp += n
		}
		for d.dp < 0 || d.dp == 0 && d.d[0] < '5' {
			n := 27
			if -d.dp < len(powtab) {
				n = powtab[-d.dp]
			}
			d.Shift(n)
			exp -= n
		}
		exp--
		if exp < flt.bias+1 {
			n := flt.bias + 1 - exp
			d.Shift(-n)
			exp += n
		}
		if exp-flt.bias >= 1<<flt.expbits-1 {
			overflow = true
			break
		}
		d.Shift(int(1 + flt.mantbits))
		mant = d.RoundedInteger()
		if mant == 2<<flt.mantbits {
			mant >>= 1
			exp++
			if exp-flt.bias >= 1<<flt.expbits-1 {
				overflow = true
				break
			}
		}
		if mant&(1<<flt.mantbits) == 0 {
			exp = flt.bias
		}
	}
	if overflow {
		mant = 0
		exp = 1<<flt.expbits - 1 + flt.bias
	}
	bits := mant & (uint64(1)<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if d.neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	return bits, overflow
}

var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

var float32pow10 = []float32{1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10}

func atof64exact(mantissa uint64, exp int, neg bool) (float64, bool) {
	if mantissa>>float64info.mantbits != 0 {
		return 0, false
	}
	f := float64(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		return f, true
	case exp > 0 && exp <= 15+22:
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f > 1e15 || f < -1e15 {
			return 0, false
		}
		return f * float64pow10[exp], true
	case exp < 0 && exp >= -22:
		return f / float64pow10[-exp], true
	}
	return 0, false
}

func atof32exact(mantissa uint64, exp int, neg bool) (float32, bool) {
	if mantissa>>float32info.mantbits != 0 {
		return 0, false
	}
	f := float32(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		return f, true
	case exp > 0 && exp <= 7+10:
		if exp > 10 {
			f *= float32pow10[exp-10]
			exp = 10
		}
		if f > 1e7 || f < -1e7 {
			return 0, false
		}
		return f * float32pow10[exp], true
	case exp < 0 && exp >= -10:
		return f / float32pow10[-exp], true
	}
	return 0, false
}

func atofHex(s string, flt *floatInfo, mantissa uint64, exp int, neg, trunc bool) (float64, error) {
	maxExp := 1<<flt.expbits + flt.bias - 2
	minExp := flt.bias + 1
	exp += int(flt.mantbits)
	for mantissa != 0 && mantissa>>(flt.mantbits+2) == 0 {
		mantissa <<= 1
		exp--
	}
	if trunc {
		mantissa |= 1
	}
	for mantissa>>(1+flt.mantbits+2) != 0 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	for mantissa > 1 && exp < minExp-2 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	round := mantissa & 3
	mantissa >>= 2
	round |= mantissa & 1
	exp += 2
	if round == 3 {
		mantissa++
		if mantissa == 1<<(1+flt.mantbits) {
			mantissa >>= 1
			exp++
		}
	}
	if mantissa>>flt.mantbits == 0 {
		exp = flt.bias
	}
	var err error
	if exp > maxExp {
		mantissa = 1 << flt.mantbits
		exp = maxExp + 1
		err = rangeError(fnParseFloat, s)
	}
	bits := mantissa & (1<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	if flt.mantbits == float32info.mantbits {
		return float64(float32frombits(uint32(bits))), err
	}
	return float64frombits(bits), err
}

func atof32(s string) (float32, int, error) {
	if val, n, ok := special(s); ok {
		return float32(val), n, nil
	}
	p, ok := readFloat(s)
	if !ok {
		return 0, p.n, syntaxError(fnParseFloat, s)
	}
	if p.hex {
		f, err := atofHex(s[:p.n], &float32info, p.mantissa, p.exp, p.neg, p.trunc)
		return float32(f), p.n, err
	}
	if !p.trunc {
		if f, ok := atof32exact(p.mantissa, p.exp, p.neg); ok {
			return f, p.n, nil
		}
	}
	d := newDecimal()
	if !d.set(s[:p.n]) {
		return 0, p.n, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float32info)
	f := float32frombits(uint32(b))
	if ovf {
		return f, p.n, rangeError(fnParseFloat, s)
	}
	return f, p.n, nil
}

func atof64(s string) (float64, int, error) {
	if val, n, ok := special(s); ok {
		return val, n, nil
	}
	p, ok := readFloat(s)
	if !ok {
		return 0, p.n, syntaxError(fnParseFloat, s)
	}
	if p.hex {
		f, err := atofHex(s[:p.n], &float64info, p.mantissa, p.exp, p.neg, p.trunc)
		return f, p.n, err
	}
	if !p.trunc {
		if f, ok := atof64exact(p.mantissa, p.exp, p.neg); ok {
			return f, p.n, nil
		}
	}
	d := newDecimal()
	if !d.set(s[:p.n]) {
		return 0, p.n, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float64info)
	f := float64frombits(b)
	if ovf {
		return f, p.n, rangeError(fnParseFloat, s)
	}
	return f, p.n, nil
}

// ParseFloat converts the string s to a floating-point number with the
// precision specified by bitSize: 32 for float32, or 64 for float64. It
// accepts Go's decimal and hexadecimal floating-point syntax, with "NaN",
// "Inf" and "Infinity" in any case, and returns the nearest value under
// IEEE 754 round-half-to-even. Errors are *NumError values wrapping
// ErrSyntax or ErrRange; on ErrRange the result is ±Inf.
func ParseFloat(s string, bitSize int) (float64, error) {
	var f float64
	var n int
	var err error
	if bitSize == 32 {
		var f32 float32
		f32, n, err = atof32(s)
		f = float64(f32)
	} else {
		f, n, err = atof64(s)
	}
	if n != len(s) && (err == nil || err.(*NumError).Err != ErrSyntax) {
		return 0, syntaxError(fnParseFloat, s)
	}
	return f, err
}
