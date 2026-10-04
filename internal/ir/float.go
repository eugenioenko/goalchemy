package ir

import (
	"go/constant"
	"strconv"
	"strings"
)

// FloatLiteral rounds an exact Go constant directly to its declared IEEE width.
// In particular a binary32 constant never passes through binary64 rounding.
// Emit the exactly widened rounded value: hosts parsing literals as binary64
// must not interpret a shortest binary32 spelling as a different binary64 value.
func FloatLiteral(c *Const) string {
	bits := c.Type.U().FloatBits
	var v float64
	if bits == 32 {
		f, _ := constant.Float32Val(c.Val)
		v = float64(f)
	} else {
		v, _ = constant.Float64Val(c.Val)
	}
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
