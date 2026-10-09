// SPDX-License-Identifier: Apache-2.0
// Package crypto provides bounded native cryptographic capabilities. See
// specs/runtime/capabilities and the SDK capability contract for ownership.
package crypto

import (
	stdcrypto "crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"sync"
)

// MaxBytes is the largest byte input most operations accept (64MiB).
const MaxBytes = 64 << 20

// MaxPEMBytes is the largest PEM input ImportPEM accepts (64KiB).
const MaxPEMBytes = 64 << 10

var invalid = errors.New("crypto: invalid input or key")
var closed = errors.New("crypto: key is closed")

// Key is an opaque RSA (2048, 3072 or 4096-bit) or NIST P-256, P-384 or P-521 key. Aliases share Close state. Native
// operations retain a snapshot under a read lock (ECDH snapshots its private
// input before reading its public input); Close waits for reads, drops
// references, and is idempotent. It cannot guarantee erasure of host copies.
type Key struct {
	mu    sync.RWMutex
	value any
}

// Close releases the key and invalidates every alias of it; it accepts nil and is idempotent.
func (k *Key) Close() {
	if k != nil {
		k.mu.Lock()
		k.value = nil
		k.mu.Unlock()
	}
}
func (k *Key) read() (any, func(), error) {
	if k == nil {
		return nil, func() {}, invalid
	}
	k.mu.RLock()
	if k.value == nil {
		k.mu.RUnlock()
		return nil, func() {}, closed
	}
	return k.value, k.mu.RUnlock, nil
}
func bounded(xs ...[]byte) bool {
	for _, x := range xs {
		if len(x) > MaxBytes {
			return false
		}
	}
	return true
}

// Random returns n bytes from the host cryptographically secure random generator.
func Random(n int) ([]byte, error) {
	if n < 0 || n > MaxBytes {
		return nil, invalid
	}
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// SHA256 returns the 32-byte SHA-256 digest of data.
func SHA256(data []byte) ([]byte, error) {
	if !bounded(data) {
		return nil, invalid
	}
	h := sha256.Sum256(data)
	return h[:], nil
}

// HMACSHA256 returns the 32-byte HMAC-SHA256 of data under key.
func HMACSHA256(key, data []byte) ([]byte, error) {
	if !bounded(key, data) {
		return nil, invalid
	}
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil), nil
}

// HMACSHA256Verify verifies a 32-byte SHA-256 MAC using constant-time equality.
// Invalid sizes/bounds return false,error; a well-formed mismatch is false,nil.
func HMACSHA256Verify(key, data, mac []byte) (bool, error) {
	if !bounded(key, data) || len(mac) != sha256.Size {
		return false, invalid
	}
	digest, err := HMACSHA256(key, data)
	if err != nil {
		return false, err
	}
	return hmac.Equal(digest, mac), nil
}

// HKDFSHA256 implements RFC 5869 extract then expand, including empty salt/info.
func HKDFSHA256(secret, salt, info []byte, n int) ([]byte, error) {
	if !bounded(secret, salt, info) || n < 0 || n > 255*sha256.Size {
		return nil, invalid
	}
	prk, _ := HMACSHA256(salt, secret)
	out := make([]byte, 0, n)
	var prev []byte
	for counter := 1; len(out) < n; counter++ {
		h := hmac.New(sha256.New, prk)
		h.Write(prev)
		h.Write(info)
		h.Write([]byte{byte(counter)})
		prev = h.Sum(nil)
		take := n - len(out)
		if take > len(prev) {
			take = len(prev)
		}
		out = append(out, prev[:take]...)
	}
	return out, nil
}
func gcm(key, nonce, data, aad []byte) (cipher.AEAD, error) {
	if len(key) != 32 || len(nonce) != 12 || len(aad) > MaxBytes {
		return nil, invalid
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// AES256GCMEncrypt returns ciphertext followed by the 16-byte tag, no nonce.
func AES256GCMEncrypt(key, nonce, data, aad []byte) ([]byte, error) {
	if len(data) > MaxBytes {
		return nil, invalid
	}
	g, err := gcm(key, nonce, data, aad)
	if err != nil {
		return nil, err
	}
	return g.Seal(nil, nonce, data, aad), nil
}

// AES256GCMDecrypt authenticates and decrypts ciphertext followed by its 16-byte tag.
func AES256GCMDecrypt(key, nonce, data, aad []byte) ([]byte, error) {
	if len(data) > MaxBytes+16 {
		return nil, invalid
	}
	g, err := gcm(key, nonce, data, aad)
	if err != nil {
		return nil, err
	}
	if len(data) < 16 {
		return nil, invalid
	}
	out, err := g.Open(nil, nonce, data, aad)
	if err != nil {
		return nil, errors.New("crypto: authentication failed")
	}
	return out, nil
}

func generateRSA(bits int) (*Key, error) {
	k, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return &Key{value: k}, nil
}

// GenerateRSA2048 returns a new random RSA-2048 private key.
func GenerateRSA2048() (*Key, error) { return generateRSA(2048) }

// GenerateRSA4096 returns a new random RSA-4096 private key.
func GenerateRSA4096() (*Key, error) { return generateRSA(4096) }

func generateEC(c elliptic.Curve) (*Key, error) {
	k, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Key{value: k}, nil
}

// GenerateP256 returns a new random P-256 private key.
func GenerateP256() (*Key, error) { return generateEC(elliptic.P256()) }

// GenerateP384 returns a new random P-384 private key.
func GenerateP384() (*Key, error) { return generateEC(elliptic.P384()) }

// GenerateP521 returns a new random P-521 private key.
func GenerateP521() (*Key, error) { return generateEC(elliptic.P521()) }

// coordinate returns the fixed field-element width of a supported curve, or 0.
func coordinate(c elliptic.Curve) int {
	switch c {
	case elliptic.P256():
		return 32
	case elliptic.P384():
		return 48
	case elliptic.P521():
		return 66
	}
	return 0
}
func public(v any) any {
	switch k := v.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	}
	return v
}
func valid(v any) bool {
	switch k := v.(type) {
	case *rsa.PrivateKey:
		return k.Validate() == nil && valid(&k.PublicKey)
	case *rsa.PublicKey:
		if k.N == nil || k.N.Sign() <= 0 || k.E < 3 || k.E%2 != 1 {
			return false
		}
		bits := k.N.BitLen()
		return bits == 2048 || bits == 3072 || bits == 4096
	case *ecdsa.PrivateKey:
		if !valid(&k.PublicKey) || k.D == nil || k.D.Sign() <= 0 || k.D.Cmp(k.Curve.Params().N) >= 0 {
			return false
		}
		x, y := k.Curve.ScalarBaseMult(k.D.Bytes())
		return x.Cmp(k.X) == 0 && y.Cmp(k.Y) == 0
	case *ecdsa.PublicKey:
		return coordinate(k.Curve) != 0 && k.X != nil && k.Y != nil && k.Curve.IsOnCurve(k.X, k.Y)
	}
	return false
}

// ImportPEM accepts exactly one unencrypted SPKI, PKCS8, certificate or PKCS1
// PEM block; certificates provide only their public key, no trust assertion.
func ImportPEM(data string) (*Key, error) {
	if len(data) > MaxPEMBytes {
		return nil, invalid
	}
	data = strings.TrimSpace(data)
	if !strings.HasPrefix(data, "-----BEGIN ") {
		return nil, invalid
	}
	b, rest := pem.Decode([]byte(data))
	if b == nil || len(b.Headers) != 0 || strings.TrimSpace(string(rest)) != "" {
		return nil, invalid
	}
	// pem.Decode may skip malformed leading blocks. Require the exact first
	// BEGIN line to match the decoded block and forbid additional BEGIN markers.
	if !strings.HasPrefix(data, "-----BEGIN "+b.Type+"-----\n") && !strings.HasPrefix(data, "-----BEGIN "+b.Type+"-----\r\n") {
		return nil, invalid
	}
	if strings.Count(data, "-----BEGIN ") != 1 {
		return nil, invalid
	}
	var v any
	var err error
	switch b.Type {
	case "PUBLIC KEY":
		v, err = x509.ParsePKIXPublicKey(b.Bytes)
	case "PRIVATE KEY":
		v, err = x509.ParsePKCS8PrivateKey(b.Bytes)
	case "RSA PUBLIC KEY":
		v, err = x509.ParsePKCS1PublicKey(b.Bytes)
	case "RSA PRIVATE KEY":
		v, err = x509.ParsePKCS1PrivateKey(b.Bytes)
	case "CERTIFICATE":
		var c *x509.Certificate
		c, err = x509.ParseCertificate(b.Bytes)
		if err == nil {
			v = c.PublicKey
		}
	default:
		return nil, invalid
	}
	if err != nil || !valid(v) {
		return nil, invalid
	}
	return &Key{value: v}, nil
}

// PublicPEM returns the public key as a PKIX (SPKI) PUBLIC KEY PEM block.
func (k *Key) PublicPEM() (string, error) {
	v, done, err := k.read()
	if err != nil {
		return "", err
	}
	defer done()
	b, err := x509.MarshalPKIXPublicKey(public(v))
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})), nil
}

// PrivatePEM returns the private key as a PKCS#8 PRIVATE KEY PEM block.
func (k *Key) PrivatePEM() (string, error) {
	v, done, err := k.read()
	if err != nil {
		return "", err
	}
	defer done()
	switch v.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey:
	default:
		return "", invalid
	}
	b, err := x509.MarshalPKCS8PrivateKey(v)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})), nil
}

// PublicJWK returns [kty,crv,n,e,x,y], base64url fields, no JSON or private data.
func (k *Key) PublicJWK() ([]string, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	enc := base64.RawURLEncoding.EncodeToString
	switch p := public(v).(type) {
	case *rsa.PublicKey:
		return []string{"RSA", "", enc(p.N.Bytes()), enc(big.NewInt(int64(p.E)).Bytes()), "", ""}, nil
	case *ecdsa.PublicKey:
		n := coordinate(p.Curve)
		return []string{"EC", p.Curve.Params().Name, "", "", enc(p.X.FillBytes(make([]byte, n))), enc(p.Y.FillBytes(make([]byte, n)))}, nil
	}
	return nil, invalid
}

// RSAOAEPEncrypt encrypts with RSA OAEP using SHA-1, MGF1-SHA1 and an empty label.
func RSAOAEPEncrypt(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := public(v).(*rsa.PublicKey)
	if !ok || len(data) > p.Size()-2*sha1.Size-2 {
		return nil, invalid
	}
	return rsa.EncryptOAEP(sha1.New(), rand.Reader, p, data, nil)
}

// RSAOAEPDecrypt decrypts RSA OAEP using SHA-1, MGF1-SHA1 and an empty label.
func RSAOAEPDecrypt(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*rsa.PrivateKey)
	if !ok || len(data) != p.Size() {
		return nil, invalid
	}
	out, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, p, data, nil)
	if err != nil {
		return nil, errors.New("crypto: decryption failed")
	}
	return out, nil
}

func digest(h stdcrypto.Hash, data []byte) []byte {
	switch h {
	case stdcrypto.SHA384:
		d := sha512.Sum384(data)
		return d[:]
	case stdcrypto.SHA512:
		d := sha512.Sum512(data)
		return d[:]
	}
	d := sha256.Sum256(data)
	return d[:]
}

func rsSign(k *Key, h stdcrypto.Hash, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*rsa.PrivateKey)
	if !ok || !bounded(data) {
		return nil, invalid
	}
	return rsa.SignPKCS1v15(rand.Reader, p, h, digest(h, data))
}

func rsVerify(k *Key, h stdcrypto.Hash, data, sig []byte) (bool, error) {
	v, done, err := k.read()
	if err != nil {
		return false, err
	}
	defer done()
	p, ok := public(v).(*rsa.PublicKey)
	if !ok || !bounded(data) || len(sig) != p.Size() {
		return false, invalid
	}
	return rsa.VerifyPKCS1v15(p, h, digest(h, data), sig) == nil, nil
}

// RS256Sign returns an RSASSA-PKCS1-v1_5 SHA-256 signature over data.
func RS256Sign(k *Key, data []byte) ([]byte, error) { return rsSign(k, stdcrypto.SHA256, data) }

// RS256Verify reports whether sig is a valid RS256 signature of data.
func RS256Verify(k *Key, data, sig []byte) (bool, error) {
	return rsVerify(k, stdcrypto.SHA256, data, sig)
}

// RS384Sign returns an RSASSA-PKCS1-v1_5 SHA-384 signature over data.
func RS384Sign(k *Key, data []byte) ([]byte, error) { return rsSign(k, stdcrypto.SHA384, data) }

// RS384Verify reports whether sig is a valid RS384 signature of data.
func RS384Verify(k *Key, data, sig []byte) (bool, error) {
	return rsVerify(k, stdcrypto.SHA384, data, sig)
}

// RS512Sign returns an RSASSA-PKCS1-v1_5 SHA-512 signature over data.
func RS512Sign(k *Key, data []byte) ([]byte, error) { return rsSign(k, stdcrypto.SHA512, data) }

// RS512Verify reports whether sig is a valid RS512 signature of data.
func RS512Verify(k *Key, data, sig []byte) (bool, error) {
	return rsVerify(k, stdcrypto.SHA512, data, sig)
}

func esSign(k *Key, c elliptic.Curve, h stdcrypto.Hash, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*ecdsa.PrivateKey)
	if !ok || p.Curve != c || !bounded(data) {
		return nil, invalid
	}
	r, s, err := ecdsa.Sign(rand.Reader, p, digest(h, data))
	if err != nil {
		return nil, err
	}
	n := coordinate(c)
	out := make([]byte, 2*n)
	r.FillBytes(out[:n])
	s.FillBytes(out[n:])
	return out, nil
}

func esVerify(k *Key, c elliptic.Curve, h stdcrypto.Hash, data, sig []byte) (bool, error) {
	v, done, err := k.read()
	if err != nil {
		return false, err
	}
	defer done()
	p, ok := public(v).(*ecdsa.PublicKey)
	n := coordinate(c)
	if !ok || p.Curve != c || !bounded(data) || len(sig) != 2*n {
		return false, invalid
	}
	return ecdsa.Verify(p, digest(h, data), new(big.Int).SetBytes(sig[:n]), new(big.Int).SetBytes(sig[n:])), nil
}

// ES256Sign returns an ECDSA P-256 SHA-256 signature over data as raw 32-byte R and S.
func ES256Sign(k *Key, data []byte) ([]byte, error) {
	return esSign(k, elliptic.P256(), stdcrypto.SHA256, data)
}

// ES256Verify reports whether sig is a valid raw R||S ES256 signature of data.
func ES256Verify(k *Key, data, sig []byte) (bool, error) {
	return esVerify(k, elliptic.P256(), stdcrypto.SHA256, data, sig)
}

// ES384Sign returns an ECDSA P-384 SHA-384 signature over data as raw 48-byte R and S.
func ES384Sign(k *Key, data []byte) ([]byte, error) {
	return esSign(k, elliptic.P384(), stdcrypto.SHA384, data)
}

// ES384Verify reports whether sig is a valid raw R||S ES384 signature of data.
func ES384Verify(k *Key, data, sig []byte) (bool, error) {
	return esVerify(k, elliptic.P384(), stdcrypto.SHA384, data, sig)
}

// ES512Sign returns an ECDSA P-521 SHA-512 signature over data as raw 66-byte R and S.
func ES512Sign(k *Key, data []byte) ([]byte, error) {
	return esSign(k, elliptic.P521(), stdcrypto.SHA512, data)
}

// ES512Verify reports whether sig is a valid raw R||S ES512 signature of data.
func ES512Verify(k *Key, data, sig []byte) (bool, error) {
	return esVerify(k, elliptic.P521(), stdcrypto.SHA512, data, sig)
}

// ECDH returns the x-coordinate shared secret of two keys on the same NIST
// curve before any KDF: 32 bytes for P-256, 48 for P-384 and 66 for P-521.
func ECDH(privateKey, publicKey *Key) ([]byte, error) {
	// Acquire and convert independently, avoiding double-lock/deadlock when aliased.
	v, done, err := privateKey.read()
	if err != nil {
		return nil, err
	}
	p, ok := v.(*ecdsa.PrivateKey)
	if !ok {
		done()
		return nil, invalid
	}
	priv, err := p.ECDH()
	done()
	if err != nil {
		return nil, err
	}
	v, done, err = publicKey.read()
	if err != nil {
		return nil, err
	}
	defer done()
	q, ok := public(v).(*ecdsa.PublicKey)
	if !ok || q.Curve != p.Curve {
		return nil, invalid
	}
	pub, err := q.ECDH()
	if err != nil {
		return nil, err
	}
	return derive(priv, pub)
}
func derive(p *ecdh.PrivateKey, q *ecdh.PublicKey) ([]byte, error) { return p.ECDH(q) }
