package crypto

import (
	stdcrypto "crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func hx(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
func equal(t *testing.T, b []byte, err error, want string) {
	t.Helper()
	if err != nil || hex.EncodeToString(b) != want {
		t.Fatalf("got %x, %v; want %s", b, err, want)
	}
}
func TestKnownAnswers(t *testing.T) {
	b, e := SHA256([]byte("abc"))
	equal(t, b, e, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	b, e = HMACSHA256(hx(strings.Repeat("0b", 20)), []byte("Hi There"))
	equal(t, b, e, "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7")
	b, e = HKDFSHA256(hx(strings.Repeat("0b", 22)), hx("000102030405060708090a0b0c"), hx("f0f1f2f3f4f5f6f7f8f9"), 42)
	equal(t, b, e, "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865")
	// NIST AES-256 GCM all-zero key/IV/plaintext, independent published vector.
	b, e = AES256GCMEncrypt(make([]byte, 32), make([]byte, 12), make([]byte, 16), nil)
	equal(t, b, e, "cea7403d4d606b6e074ec5d3baf39d18d0d1c8a799996bf0265b98b5d48ab919")
	b, e = AES256GCMDecrypt(make([]byte, 32), make([]byte, 12), hx("cea7403d4d606b6e074ec5d3baf39d18d0d1c8a799996bf0265b98b5d48ab919"), nil)
	equal(t, b, e, strings.Repeat("00", 16))
	// RFC 5869 case 3 empty salt/info.
	b, e = HKDFSHA256(hx(strings.Repeat("0b", 22)), nil, nil, 42)
	equal(t, b, e, "8da4e775a563c18f715f802a063c5a31b8a11f5c5ee1879ec3454e5f3c738d2d9d201395faa4b61a96c8")
}
func TestInvalidInputs(t *testing.T) {
	if _, e := Random(-1); e == nil {
		t.Fatal("negative random")
	}
	if _, e := Random(MaxBytes + 1); e == nil {
		t.Fatal("unbounded random")
	}
	for _, n := range []int{-1, 8161} {
		if _, e := HKDFSHA256(nil, nil, nil, n); e == nil {
			t.Fatal("HKDF length")
		}
	}
	for _, n := range []int{0, 16, 31, 33} {
		if _, e := AES256GCMEncrypt(make([]byte, n), make([]byte, 12), nil, nil); e == nil {
			t.Fatal("AES key length")
		}
	}
	if _, e := AES256GCMEncrypt(make([]byte, 32), nil, nil, nil); e == nil {
		t.Fatal("nonce")
	}
	if b, e := AES256GCMDecrypt(make([]byte, 32), make([]byte, 12), make([]byte, 16), nil); e == nil || b != nil {
		t.Fatal("unauthenticated plaintext")
	}
	for _, s := range []string{"garbage", "-----BEGIN PRIVATE KEY-----\nAA==\n-----END PRIVATE KEY-----", strings.Repeat("x", MaxPEMBytes+1)} {
		if _, e := ImportPEM(s); e == nil {
			t.Fatal("malformed PEM")
		}
	}
	if _, e := RS256Sign(nil, nil); e == nil {
		t.Fatal("nil key")
	}
}
func TestRSAInterchangeAndParameters(t *testing.T) {
	k, e := GenerateRSA2048()
	if e != nil {
		t.Fatal(e)
	}
	defer k.Close()
	privatePEM, e := k.PrivatePEM()
	if e != nil {
		t.Fatal(e)
	}
	block, _ := pem.Decode([]byte(privatePEM))
	native, e := x509.ParsePKCS8PrivateKey(block.Bytes)
	if e != nil {
		t.Fatal(e)
	}
	p := native.(*rsa.PrivateKey)
	publicPEM, e := k.PublicPEM()
	if e != nil {
		t.Fatal(e)
	}
	pub, e := ImportPEM(publicPEM)
	if e != nil {
		t.Fatal(e)
	}
	defer pub.Close()
	msg := []byte("OAEP SHA1 + MGF1 SHA1 empty label")
	encrypted, e := RSAOAEPEncrypt(pub, msg)
	if e != nil {
		t.Fatal(e)
	}
	out, e := rsa.DecryptOAEP(sha1.New(), rand.Reader, p, encrypted, nil)
	if e != nil || string(out) != string(msg) {
		t.Fatal(e)
	}
	if _, e = rsa.DecryptOAEP(sha256.New(), rand.Reader, p, encrypted, nil); e == nil {
		t.Fatal("SHA256 accepted")
	}
	if _, e = rsa.DecryptOAEP(sha1.New(), rand.Reader, p, encrypted, []byte("label")); e == nil {
		t.Fatal("nonempty label accepted")
	}
	encrypted, e = rsa.EncryptOAEP(sha1.New(), rand.Reader, &p.PublicKey, msg, nil)
	if e != nil {
		t.Fatal(e)
	}
	out, e = RSAOAEPDecrypt(k, encrypted)
	if e != nil || string(out) != string(msg) {
		t.Fatal(e)
	}
	sig, e := RS256Sign(k, msg)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256(msg)
	if e = rsa.VerifyPKCS1v15(&p.PublicKey, stdcrypto.SHA256, h[:], sig); e != nil {
		t.Fatal(e)
	}
	sig, e = rsa.SignPKCS1v15(rand.Reader, p, stdcrypto.SHA256, h[:])
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := RS256Verify(pub, msg, sig); e != nil || !ok {
		t.Fatal(e)
	}
	sig[0] ^= 1
	if ok, e := RS256Verify(pub, msg, sig); e != nil || ok {
		t.Fatal("tamper", e)
	}
	if _, e = RSAOAEPEncrypt(pub, make([]byte, 215)); e == nil {
		t.Fatal("OAEP size")
	}
	if _, e = RSAOAEPDecrypt(pub, make([]byte, 256)); e == nil {
		t.Fatal("public decrypt")
	}
	if _, e = pub.PrivatePEM(); e == nil {
		t.Fatal("private export")
	}
	jwk, e := pub.PublicJWK()
	if e != nil || len(jwk) != 6 || jwk[0] != "RSA" || jwk[3] != "AQAB" || jwk[2] != base64.RawURLEncoding.EncodeToString(p.N.Bytes()) {
		t.Fatal(jwk, e)
	}
	for _, fixture := range []string{string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(p)})), string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&p.PublicKey)}))} {
		imported, e := ImportPEM(fixture)
		if e != nil {
			t.Fatal(e)
		}
		imported.Close()
	}
	cert, e := x509.CreateCertificate(rand.Reader, &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Unix(0, 0), NotAfter: time.Unix(1, 0)}, &x509.Certificate{SerialNumber: big.NewInt(1)}, &p.PublicKey, p)
	if e != nil {
		t.Fatal(e)
	}
	imported, e := ImportPEM(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})))
	if e != nil {
		t.Fatal(e)
	}
	imported.Close()
	for _, fixture := range []string{publicPEM + publicPEM, "junk\n" + publicPEM, "-----BEGIN BROKEN-----\njunk\n" + publicPEM, publicPEM + "junk"} {
		if _, e = ImportPEM(fixture); e == nil {
			t.Fatal("PEM trailing/leading data")
		}
	}
	alias := k
	k.Close()
	k.Close()
	if _, e = alias.PublicPEM(); e == nil {
		t.Fatal("closed alias")
	}
	if _, e = RS256Sign(k, msg); e == nil {
		t.Fatal("closed sign")
	}
}
func TestP256InterchangeAndECDH(t *testing.T) {
	k, e := GenerateP256()
	if e != nil {
		t.Fatal(e)
	}
	defer k.Close()
	s, e := k.PrivatePEM()
	if e != nil {
		t.Fatal(e)
	}
	block, _ := pem.Decode([]byte(s))
	v, e := x509.ParsePKCS8PrivateKey(block.Bytes)
	if e != nil {
		t.Fatal(e)
	}
	p := v.(*ecdsa.PrivateKey)
	msg := []byte("JOSE raw R || S")
	h := sha256.Sum256(msg)
	sig, e := ES256Sign(k, msg)
	if e != nil || len(sig) != 64 {
		t.Fatal(e)
	}
	if !ecdsa.Verify(&p.PublicKey, h[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("raw verify")
	}
	r, sigS, e := ecdsa.Sign(rand.Reader, p, h[:])
	if e != nil {
		t.Fatal(e)
	}
	r.FillBytes(sig[:32])
	sigS.FillBytes(sig[32:])
	if ok, e := ES256Verify(k, msg, sig); e != nil || !ok {
		t.Fatal(e)
	}
	if ok, e := ES256Verify(k, []byte("changed"), sig); e != nil || ok {
		t.Fatal("tampered", e)
	}
	if _, e := ES256Verify(k, msg, sig[:63]); e == nil {
		t.Fatal("wrong signature size")
	}
	jwk, e := k.PublicJWK()
	if e != nil || jwk[0] != "EC" || jwk[1] != "P-256" {
		t.Fatal(jwk, e)
	}
	for _, i := range []int{4, 5} {
		b, e := base64.RawURLEncoding.DecodeString(jwk[i])
		if e != nil || len(b) != 32 {
			t.Fatal("coordinate", e)
		}
	}
	// Fixed scalars d=1 and d=2: expected secret is the published P-256 2G X.
	scalar := func(d int64) *Key {
		n := big.NewInt(d)
		x, y := elliptic.P256().ScalarBaseMult(n.Bytes())
		return &Key{value: &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, D: n}}
	}
	a, b := scalar(1), scalar(2)
	defer a.Close()
	defer b.Close()
	secret, e := ECDH(a, b)
	equal(t, secret, e, "7cf27b188d034f7e8a52380304b51ac3c08969e277f21b35a60b48fc47669978")
	priv, _ := p.ECDH()
	other := b.value.(*ecdsa.PrivateKey)
	pub, _ := other.PublicKey.ECDH()
	expected, e := priv.ECDH(pub)
	if e != nil {
		t.Fatal(e)
	}
	actual, e := ECDH(k, b)
	if e != nil || hex.EncodeToString(actual) != hex.EncodeToString(expected) {
		t.Fatal(e)
	}
	imported, e := ImportPEM(s)
	if e != nil {
		t.Fatal(e)
	}
	imported.Close()
	if _, e := RS256Sign(k, msg); e == nil {
		t.Fatal("wrong key type")
	}
	bigger, e := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	der, _ := x509.MarshalPKIXPublicKey(&bigger.PublicKey)
	if _, e := ImportPEM(string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))); e == nil {
		t.Fatal("P384 accepted")
	}
}
func TestRandomAndOwnership(t *testing.T) {
	a, e := Random(32)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Random(32)
	if e != nil || string(a) == string(b) {
		t.Fatal("random", e)
	}
	src := []byte("abc")
	digest, e := SHA256(src)
	if e != nil {
		t.Fatal(e)
	}
	digest[0] ^= 1
	if string(src) != "abc" {
		t.Fatal("input mutation")
	}
}

func TestAESMaximumBoundary(t *testing.T) {
	key, nonce := make([]byte, 32), make([]byte, 12)
	data := make([]byte, MaxBytes)
	ct, e := AES256GCMEncrypt(key, nonce, data, nil)
	if e != nil || len(ct) != MaxBytes+16 {
		t.Fatal(len(ct), e)
	}
	pt, e := AES256GCMDecrypt(key, nonce, ct, nil)
	if e != nil || len(pt) != MaxBytes {
		t.Fatal(len(pt), e)
	}
	if _, e = AES256GCMDecrypt(key, nonce, make([]byte, MaxBytes+17), nil); e == nil {
		t.Fatal("oversize ciphertext")
	}
}

func TestConcurrentKeyClose(t *testing.T) {
	key, err := GenerateP256()
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for i := 0; i < 100; i++ {
			_, _ = key.PublicPEM()
			_, _ = ES256Sign(key, []byte("in-flight"))
		}
	}()
	key.Close()
	key.Close()
	<-finished
	if _, err := key.PublicPEM(); err != closed {
		t.Fatal(err)
	}
}
