// Command jwt signs and verifies HS256 JSON Web Tokens.
package main

import (
	"github.com/eugenioenko/goalchemy/lib/crypto"
	"github.com/eugenioenko/goalchemy/lib/encoding"
	"github.com/eugenioenko/goalchemy/lib/errors"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
)

const secret = "goalchemy-example-secret"

const now int64 = 1700000600

var (
	ErrMalformed = errors.New("malformed token")
	ErrSignature = errors.New("invalid signature")
	ErrExpired   = errors.New("token expired")
)

type Claims struct {
	Subject string
	Name    string
	Admin   bool
	Issued  int64
	Expires int64
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func claimsJSON(c Claims) string {
	var b strings.Builder
	b.WriteString(`{"sub":`)
	b.WriteString(quote(c.Subject))
	b.WriteString(`,"name":`)
	b.WriteString(quote(c.Name))
	b.WriteString(`,"admin":`)
	b.WriteString(strconv.FormatBool(c.Admin))
	b.WriteString(`,"iat":`)
	b.WriteString(strconv.FormatInt(c.Issued, 10))
	b.WriteString(`,"exp":`)
	b.WriteString(strconv.FormatInt(c.Expires, 10))
	b.WriteString("}")
	return b.String()
}

func sign(c Claims) (string, error) {
	header, err := encoding.Base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	if err != nil {
		return "", err
	}
	payload, err := encoding.Base64URLEncode([]byte(claimsJSON(c)))
	if err != nil {
		return "", err
	}
	input := header + "." + payload
	mac, err := crypto.HMACSHA256([]byte(secret), []byte(input))
	if err != nil {
		return "", err
	}
	sig, err := encoding.Base64URLEncode(mac)
	if err != nil {
		return "", err
	}
	return input + "." + sig, nil
}

func field(json, key string) (string, bool) {
	_, rest, ok := strings.Cut(json, `"`+key+`":`)
	if !ok {
		return "", false
	}
	if strings.HasPrefix(rest, `"`) {
		var b strings.Builder
		for i := 1; i < len(rest); i++ {
			c := rest[i]
			if c == '"' {
				return b.String(), true
			}
			if c == '\\' && i+1 < len(rest) {
				i++
				switch rest[i] {
				case 'n':
					c = '\n'
				case 't':
					c = '\t'
				default:
					c = rest[i]
				}
			}
			b.WriteByte(c)
		}
		return "", false
	}
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func parseClaims(json string) (Claims, error) {
	var c Claims
	var ok bool
	if c.Subject, ok = field(json, "sub"); !ok {
		return c, ErrMalformed
	}
	if c.Name, ok = field(json, "name"); !ok {
		return c, ErrMalformed
	}
	admin, ok := field(json, "admin")
	if !ok {
		return c, ErrMalformed
	}
	var err error
	if c.Admin, err = strconv.ParseBool(admin); err != nil {
		return c, ErrMalformed
	}
	iat, ok := field(json, "iat")
	if !ok {
		return c, ErrMalformed
	}
	if c.Issued, err = strconv.ParseInt(iat, 10, 64); err != nil {
		return c, ErrMalformed
	}
	exp, ok := field(json, "exp")
	if !ok {
		return c, ErrMalformed
	}
	if c.Expires, err = strconv.ParseInt(exp, 10, 64); err != nil {
		return c, ErrMalformed
	}
	return c, nil
}

func verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformed
	}
	header, err := encoding.Base64URLDecode(parts[0])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	if alg, ok := field(string(header), "alg"); !ok || alg != "HS256" {
		return Claims{}, ErrMalformed
	}
	mac, err := encoding.Base64URLDecode(parts[2])
	if err != nil || len(mac) != 32 {
		return Claims{}, ErrMalformed
	}
	valid, err := crypto.HMACSHA256Verify([]byte(secret), []byte(parts[0]+"."+parts[1]), mac)
	if err != nil {
		return Claims{}, err
	}
	if !valid {
		return Claims{}, ErrSignature
	}
	payload, err := encoding.Base64URLDecode(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	c, err := parseClaims(string(payload))
	if err != nil {
		return c, err
	}
	if now >= c.Expires {
		return c, ErrExpired
	}
	return c, nil
}

func check(label, token string) {
	c, err := verify(token)
	if err != nil {
		println(label+":", "rejected:", err.Error())
		return
	}
	println(label+":", "ok sub="+c.Subject, "name="+c.Name, "admin="+strconv.FormatBool(c.Admin), "expires in", strconv.FormatInt(c.Expires-now, 10)+"s")
}

func main() {
	token, err := sign(Claims{Subject: "alice", Name: `Alice "Al" O\Neil`, Admin: true, Issued: 1700000000, Expires: 1700003600})
	if err != nil {
		println("sign failed:", err.Error())
		return
	}
	println("token:", token)
	check("valid", token)

	parts := strings.Split(token, ".")
	forged, _ := encoding.Base64URLEncode([]byte(`{"sub":"mallory","name":"Mallory","admin":true,"iat":1700000000,"exp":1700003600}`))
	check("tampered", parts[0]+"."+forged+"."+parts[2])

	expired, err := sign(Claims{Subject: "bob", Name: "Bob", Admin: false, Issued: 1699990000, Expires: 1699993600})
	if err != nil {
		println("sign failed:", err.Error())
		return
	}
	check("expired", expired)
	check("malformed", "not-a-token")
}
