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
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"sync"
)

const MaxBytes = 64 << 20
const MaxPEMBytes = 64 << 10

var invalid = errors.New("crypto: invalid input or key")
var closed = errors.New("crypto: key is closed")

// Key is an opaque RSA-2048 or P-256 key. Aliases share Close state. Native
// operations retain a snapshot under a read lock (ECDH snapshots its private
// input before reading its public input); Close waits for reads, drops
// references, and is idempotent. It cannot guarantee erasure of host copies.
type Key struct {
	mu    sync.RWMutex
	value any
}

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
func SHA256(data []byte) ([]byte, error) {
	if !bounded(data) {
		return nil, invalid
	}
	h := sha256.Sum256(data)
	return h[:], nil
}
func HMACSHA256(key, data []byte) ([]byte, error) {
	if !bounded(key, data) {
		return nil, invalid
	}
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil), nil
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
func GenerateRSA2048() (*Key, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &Key{value: k}, nil
}
func GenerateP256() (*Key, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Key{value: k}, nil
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
		return k.N != nil && k.N.Sign() > 0 && k.N.BitLen() == 2048 && k.E >= 3 && k.E%2 == 1
	case *ecdsa.PrivateKey:
		if k.D == nil || k.D.Sign() <= 0 || k.D.Cmp(elliptic.P256().Params().N) >= 0 || !valid(&k.PublicKey) {
			return false
		}
		x, y := elliptic.P256().ScalarBaseMult(k.D.Bytes())
		return x.Cmp(k.X) == 0 && y.Cmp(k.Y) == 0
	case *ecdsa.PublicKey:
		return k.Curve == elliptic.P256() && k.X != nil && k.Y != nil && k.Curve.IsOnCurve(k.X, k.Y)
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
		return []string{"EC", "P-256", "", "", enc(p.X.FillBytes(make([]byte, 32))), enc(p.Y.FillBytes(make([]byte, 32)))}, nil
	}
	return nil, invalid
}

// RSAOAEP uses SHA-1 for both OAEP and MGF1, with an empty label.
func RSAOAEPEncrypt(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := public(v).(*rsa.PublicKey)
	if !ok || len(data) > 214 {
		return nil, invalid
	}
	return rsa.EncryptOAEP(sha1.New(), rand.Reader, p, data, nil)
}
func RSAOAEPDecrypt(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*rsa.PrivateKey)
	if !ok || len(data) != 256 {
		return nil, invalid
	}
	out, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, p, data, nil)
	if err != nil {
		return nil, errors.New("crypto: decryption failed")
	}
	return out, nil
}
func RS256Sign(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*rsa.PrivateKey)
	if !ok || !bounded(data) {
		return nil, invalid
	}
	h := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, p, stdcrypto.SHA256, h[:])
}
func RS256Verify(k *Key, data, sig []byte) (bool, error) {
	v, done, err := k.read()
	if err != nil {
		return false, err
	}
	defer done()
	p, ok := public(v).(*rsa.PublicKey)
	if !ok || !bounded(data) || len(sig) != 256 {
		return false, invalid
	}
	h := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(p, stdcrypto.SHA256, h[:], sig) == nil, nil
}
func ES256Sign(k *Key, data []byte) ([]byte, error) {
	v, done, err := k.read()
	if err != nil {
		return nil, err
	}
	defer done()
	p, ok := v.(*ecdsa.PrivateKey)
	if !ok || !bounded(data) {
		return nil, invalid
	}
	h := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, p, h[:])
	if err != nil {
		return nil, err
	}
	out := make([]byte, 64)
	r.FillBytes(out[:32])
	s.FillBytes(out[32:])
	return out, nil
}
func ES256Verify(k *Key, data, sig []byte) (bool, error) {
	v, done, err := k.read()
	if err != nil {
		return false, err
	}
	defer done()
	p, ok := public(v).(*ecdsa.PublicKey)
	if !ok || !bounded(data) || len(sig) != 64 {
		return false, invalid
	}
	h := sha256.Sum256(data)
	return ecdsa.Verify(p, h[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])), nil
}

// ECDH returns the 32-byte P-256 x-coordinate shared secret before any KDF.
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
	if !ok {
		return nil, invalid
	}
	pub, err := q.ECDH()
	if err != nil {
		return nil, err
	}
	return derive(priv, pub)
}
func derive(p *ecdh.PrivateKey, q *ecdh.PublicKey) ([]byte, error) { return p.ECDH(q) }
