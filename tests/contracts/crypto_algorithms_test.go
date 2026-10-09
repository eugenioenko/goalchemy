package contracts

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// TestCryptoAlgorithms compiles one probe program to every target. Go-made
// keys, signatures, ciphertexts and ECDH secrets go into the target; Go then
// checks what the target produced. The probe also runs the vendored
// Wycheproof vectors, so every adapter must reject the same malformed
// signatures and invalid public keys that Go rejects.
func TestCryptoAlgorithms(t *testing.T) {
	fx := newCryptoFixtures(t)
	vectors, wycheproof := wycheproofRecords(t)
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	files := map[string]string{
		"main.go": cryptoProbe(fx, vectors),
		"go.mod":  fmt.Sprintf("module cryptoprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", strconv.Quote(root)),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	targets := []string{"go", "typescript", "python", "java", "csharp", "rust", "c", "swift"}
	if only := os.Getenv("GOALCHEMY_TEST_TARGETS"); only != "" {
		targets = strings.Split(only, ",")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			out := t.TempDir()
			if ds := testutil.CompileGate(source, target, out, "cooperative"); len(ds) != 0 {
				t.Fatal(ds)
			}
			if target == "java" {
				javaCryptoClasspath(t, out)
			}
			observation, err := testutil.Runners[target](out)
			if err != nil || observation.Exit != 0 {
				t.Fatalf("%s probe: %v\n%s", target, err, observation)
			}
			fx.verify(t, observation.Stderr, wycheproof)
		})
	}
}

// javaCryptoClasspath adds the pinned Bouncy Castle jar to a generated Java
// executable. Executables do not bundle it, but EC private key imports use it
// to derive the public point.
func javaCryptoClasspath(t *testing.T, out string) {
	t.Helper()
	jar, err := exec.Command("bash", filepath.Join(root, "targets/java/tests/crypto-dependencies.sh")).Output()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(out, "run.sh")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	fixed := strings.Replace(string(data), "-cp classes ", "-cp classes:"+strings.TrimSpace(string(jar))+" ", 1)
	if fixed == string(data) {
		t.Fatal("generated Java run.sh has no classpath to extend")
	}
	if err := os.WriteFile(script, []byte(fixed), 0700); err != nil {
		t.Fatal(err)
	}
}

type cryptoFixtures struct {
	data                         []byte
	rsa3072, rsa4096             *rsa.PrivateKey
	p384, p521, peer384, peer521 *ecdsa.PrivateKey
	pems                         map[string]string
	bytes                        map[string][]byte
}

func newCryptoFixtures(t *testing.T) *cryptoFixtures {
	t.Helper()
	fx := &cryptoFixtures{data: []byte{0, 255, 128, 1, 2, 3, 'g', 'o'}, pems: map[string]string{}, bytes: map[string][]byte{}}
	var err error
	if fx.rsa3072, err = rsa.GenerateKey(rand.Reader, 3072); err != nil {
		t.Fatal(err)
	}
	if fx.rsa4096, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
		t.Fatal(err)
	}
	for _, k := range []**ecdsa.PrivateKey{&fx.p384, &fx.peer384} {
		if *k, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader); err != nil {
			t.Fatal(err)
		}
	}
	for _, k := range []**ecdsa.PrivateKey{&fx.p521, &fx.peer521} {
		if *k, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader); err != nil {
			t.Fatal(err)
		}
	}
	block := func(label string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: label, Bytes: der}))
	}
	pkcs8 := func(k any) string {
		der, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			t.Fatal(err)
		}
		return block("PRIVATE KEY", der)
	}
	spki := func(k any) string {
		der, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			t.Fatal(err)
		}
		return block("PUBLIC KEY", der)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Goalchemy crypto fixture"}, NotBefore: time.Unix(0, 0), NotAfter: time.Unix(4102444800, 0), KeyUsage: x509.KeyUsageDigitalSignature}
	cert, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &fx.p384.PublicKey, fx.p384)
	if err != nil {
		t.Fatal(err)
	}
	weak, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	k1, err := hex.DecodeString("308184020100301006072a8648ce3d020106052b8104000a046d306b02010104201610d53f54bd50fbfa1fbe474a7d19a2a356fec331fd487b0cb3e9c1d66c50a3a1440342000441acc6c10fb0070c29eb644b730e6c056a2c002f14043f1d6dd07f78e883ce021fddef96872cd865dddc321bcd66c0a8937d09ae62c71d591fd3472c122fe2cf")
	if err != nil {
		t.Fatal(err)
	}
	fx.pems = map[string]string{
		"rsa3072Private": pkcs8(fx.rsa3072), "rsa3072Public": spki(&fx.rsa3072.PublicKey),
		"rsa4096Private": block("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(fx.rsa4096)),
		"rsa4096Public":  block("RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(&fx.rsa4096.PublicKey)),
		"p384Private":    pkcs8(fx.p384), "p384Certificate": block("CERTIFICATE", cert),
		"p521Private": pkcs8(fx.p521), "p521Public": spki(&fx.p521.PublicKey),
		"peer384Public": spki(&fx.peer384.PublicKey), "peer521Public": spki(&fx.peer521.PublicKey),
		"rsa1024Public": spki(&weak.PublicKey), "secp256k1Private": block("PRIVATE KEY", k1),
	}
	sign := func(name string, sig []byte, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		fx.bytes[name] = sig
	}
	h256, h384, h512 := sha256.Sum256(fx.data), sha512.Sum384(fx.data), sha512.Sum512(fx.data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, fx.rsa3072, crypto.SHA256, h256[:])
	sign("goRS256", sig, err)
	sig, err = rsa.SignPKCS1v15(rand.Reader, fx.rsa4096, crypto.SHA384, h384[:])
	sign("goRS384", sig, err)
	sig, err = rsa.SignPKCS1v15(rand.Reader, fx.rsa3072, crypto.SHA512, h512[:])
	sign("goRS512", sig, err)
	sign("goES384", rawECDSA(t, fx.p384, h384[:]), nil)
	sign("goES512", rawECDSA(t, fx.p521, h512[:]), nil)
	sig, err = rsa.EncryptOAEP(sha1.New(), rand.Reader, &fx.rsa3072.PublicKey, fx.data, nil)
	sign("goOAEP3072", sig, err)
	sig, err = rsa.EncryptOAEP(sha1.New(), rand.Reader, &fx.rsa4096.PublicKey, fx.data, nil)
	sign("goOAEP4096", sig, err)
	fx.bytes["data"] = fx.data
	return fx
}

func rawECDSA(t *testing.T, k *ecdsa.PrivateKey, digest []byte) []byte {
	t.Helper()
	r, s, err := ecdsa.Sign(rand.Reader, k, digest)
	if err != nil {
		t.Fatal(err)
	}
	n := (k.Curve.Params().BitSize + 7) / 8
	return append(r.FillBytes(make([]byte, n)), s.FillBytes(make([]byte, n))...)
}

func sharedSecret(t *testing.T, a, b *ecdsa.PrivateKey) []byte {
	t.Helper()
	p, err := a.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	q, err := b.PublicKey.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.ECDH(q)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type wycheproofFile struct {
	TestGroups []struct {
		PublicKeyDer string `json:"publicKeyDer"`
		Sha          string `json:"sha"`
		Curve        string `json:"curve"`
		Tests        []struct {
			TcID    int    `json:"tcId"`
			Msg     string `json:"msg"`
			Sig     string `json:"sig"`
			Public  string `json:"public"`
			Private string `json:"private"`
			Shared  string `json:"shared"`
			Result  string `json:"result"`
		} `json:"tests"`
	} `json:"testGroups"`
}

// wycheproofRecords encodes the vendored vectors as binary probe records
// and returns how many cases the probe must pass. "acceptable" cases, which
// Wycheproof lets implementations either accept or reject, are left out.
//
//	F n name                      vector file name for failure reports
//	K alg u16 der                 SPKI public key for the following S records
//	P u16 der                     PKCS#8 private key for the following E records
//	S u16 id valid u16 msg u16 sig
//	E u16 id valid u16 spki u16 shared
func wycheproofRecords(t *testing.T) ([][]byte, int) {
	t.Helper()
	dir := filepath.Join(root, "tests/contracts/testdata/wycheproof")
	names, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(names) == 0 {
		t.Fatalf("Wycheproof vectors: %v", err)
	}
	sort.Strings(names)
	var records [][]byte
	cases := 0
	field := func(r []byte, h string) []byte {
		b, err := hex.DecodeString(h)
		if err != nil || len(b) > 0xffff {
			t.Fatalf("Wycheproof field %q: %v", h, err)
		}
		return append(append(r, byte(len(b)>>8), byte(len(b))), b...)
	}
	test := func(kind byte, id int, result string) []byte {
		valid := byte(0)
		if result == "valid" {
			valid = 1
		}
		return []byte{kind, byte(id >> 8), byte(id), valid}
	}
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		var file wycheproofFile
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		base := strings.TrimSuffix(filepath.Base(name), "_test.json")
		records = append(records, append([]byte{'F', byte(len(base))}, base...))
		for _, g := range file.TestGroups {
			if strings.HasPrefix(base, "ecdh_") {
				curve := map[string]ecdh.Curve{"secp384r1": ecdh.P384(), "secp521r1": ecdh.P521()}[g.Curve]
				size := map[string]int{"secp384r1": 48, "secp521r1": 66}[g.Curve]
				tests := g.Tests
				sort.SliceStable(tests, func(i, j int) bool { return tests[i].Private < tests[j].Private })
				last := ""
				for _, c := range tests {
					if c.Result == "acceptable" {
						continue
					}
					if c.Private != last {
						scalar, _ := new(big.Int).SetString(c.Private, 16)
						key, err := curve.NewPrivateKey(scalar.FillBytes(make([]byte, size)))
						if err != nil {
							t.Fatalf("%s %d: private scalar: %v", base, c.TcID, err)
						}
						der, err := x509.MarshalPKCS8PrivateKey(key)
						if err != nil {
							t.Fatal(err)
						}
						records = append(records, field([]byte{'P'}, hex.EncodeToString(der)))
						last = c.Private
					}
					records = append(records, field(field(test('E', c.TcID, c.Result), c.Public), c.Shared))
					cases++
				}
				continue
			}
			alg := map[string]byte{"SHA-256": 0, "SHA-384": 1, "SHA-512": 2}[g.Sha]
			if strings.HasPrefix(base, "ecdsa_") {
				alg += 3
			}
			records = append(records, field([]byte{'K', alg}, g.PublicKeyDer))
			for _, c := range g.Tests {
				if c.Result == "acceptable" {
					continue
				}
				records = append(records, field(field(test('S', c.TcID, c.Result), c.Msg), c.Sig))
				cases++
			}
		}
	}
	return records, cases
}

// cryptoProbe returns a Goalchemy program. Vector records are packed into
// base64 string literals of at most 32KiB, within per-literal limits such as
// Java's 64KiB constants; the probe decodes them with the native capability.
func cryptoProbe(fx *cryptoFixtures, records [][]byte) string {
	var b strings.Builder
	b.WriteString(`package main

import (
	"github.com/eugenioenko/goalchemy/lib/crypto"
	"github.com/eugenioenko/goalchemy/lib/encoding"
	"github.com/eugenioenko/goalchemy/std/bytes"
	"github.com/eugenioenko/goalchemy/std/encoding/hex"
)

`)
	keys := make([]string, 0, len(fx.pems))
	for k := range fx.pems {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "const %s = %s\n", k, strconv.Quote(fx.pems[k]))
	}
	keys = keys[:0]
	for k := range fx.bytes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "const %sHex = %q\n", k, hex.EncodeToString(fx.bytes[k]))
	}
	b.WriteString("\nvar vectors = []string{\n")
	var packed []byte
	for _, r := range records {
		if len(packed)+len(r) > 24<<10 {
			fmt.Fprintf(&b, "\t%q,\n", base64.StdEncoding.EncodeToString(packed))
			packed = packed[:0]
		}
		packed = append(packed, r...)
	}
	fmt.Fprintf(&b, "\t%q,\n}\n", base64.StdEncoding.EncodeToString(packed))
	b.WriteString(cryptoProbeBody)
	return b.String()
}

const cryptoProbeBody = `
func need(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func must(ok bool, what string) {
	if !ok {
		panic(what)
	}
}

func unhex(s string) []byte {
	if s == "-" {
		return []byte{}
	}
	b, err := hex.DecodeString(s)
	need(err)
	return b
}

func load(text string) *crypto.Key {
	k, err := crypto.ImportPEM(text)
	need(err)
	return k
}

func pemOf(label string, der []byte) string {
	text, err := encoding.Base64Encode(der)
	need(err)
	out := "-----BEGIN " + label + "-----\n"
	for len(text) > 64 {
		out += text[:64] + "\n"
		text = text[64:]
	}
	return out + text + "\n-----END " + label + "-----\n"
}

func u16(b []byte, i int) int { return int(b[i])<<8 | int(b[i+1]) }

func field(b []byte, i int) ([]byte, int) {
	n := u16(b, i)
	return b[i+2 : i+2+n], i + 2 + n
}

var algorithms = []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}

func sign(alg string, k *crypto.Key, data []byte) ([]byte, error) {
	switch alg {
	case "RS256":
		return crypto.RS256Sign(k, data)
	case "RS384":
		return crypto.RS384Sign(k, data)
	case "RS512":
		return crypto.RS512Sign(k, data)
	case "ES256":
		return crypto.ES256Sign(k, data)
	case "ES384":
		return crypto.ES384Sign(k, data)
	}
	return crypto.ES512Sign(k, data)
}

func verify(alg string, k *crypto.Key, data, sig []byte) (bool, error) {
	switch alg {
	case "RS256":
		return crypto.RS256Verify(k, data, sig)
	case "RS384":
		return crypto.RS384Verify(k, data, sig)
	case "RS512":
		return crypto.RS512Verify(k, data, sig)
	case "ES256":
		return crypto.ES256Verify(k, data, sig)
	case "ES384":
		return crypto.ES384Verify(k, data, sig)
	}
	return crypto.ES512Verify(k, data, sig)
}

func show(label string, data []byte, err error) {
	need(err)
	println(label, hex.EncodeToString(data))
}

func showJWK(name string, k *crypto.Key) {
	fields, err := k.PublicJWK()
	need(err)
	must(len(fields) == 6, "JWK shape")
	line := "JWK " + name
	for _, f := range fields {
		if f == "" {
			f = "-"
		}
		line += " " + f
	}
	println(line)
}

func rejected(what string, err error) {
	must(err != nil, what+" accepted")
}

func main() {
	data := unhex(dataHex)
	r3, r3pub, r4, r4pub := load(rsa3072Private), load(rsa3072Public), load(rsa4096Private), load(rsa4096Public)
	e3, e3cert, e5, e5pub := load(p384Private), load(p384Certificate), load(p521Private), load(p521Public)
	peer3, peer5 := load(peer384Public), load(peer521Public)
	for _, c := range []struct {
		alg string
		key *crypto.Key
		sig string
	}{{"RS256", r3pub, goRS256Hex}, {"RS384", r4pub, goRS384Hex}, {"RS512", r3, goRS512Hex}, {"ES384", e3cert, goES384Hex}, {"ES512", e5pub, goES512Hex}} {
		ok, err := verify(c.alg, c.key, data, unhex(c.sig))
		need(err)
		must(ok, "Go "+c.alg+" signature rejected")
		sig := unhex(c.sig)
		sig[len(sig)-1] ^= 1
		ok, err = verify(c.alg, c.key, data, sig)
		need(err)
		must(!ok, "changed Go "+c.alg+" signature accepted")
	}
	for _, c := range []struct {
		key    *crypto.Key
		cipher string
	}{{r3, goOAEP3072Hex}, {r4, goOAEP4096Hex}} {
		plain, err := crypto.RSAOAEPDecrypt(c.key, unhex(c.cipher))
		need(err)
		must(hex.EncodeToString(plain) == dataHex, "Go OAEP plaintext")
	}
	for _, c := range []struct {
		label, alg string
		key        *crypto.Key
	}{{"SIG RS256 rsa3072", "RS256", r3}, {"SIG RS384 rsa4096", "RS384", r4}, {"SIG RS512 rsa3072", "RS512", r3}, {"SIG RS512 rsa4096", "RS512", r4}, {"SIG ES384 p384", "ES384", e3}, {"SIG ES512 p521", "ES512", e5}} {
		sig, err := sign(c.alg, c.key, data)
		show(c.label, sig, err)
	}
	got, err := crypto.RSAOAEPEncrypt(r3pub, data)
	show("OAEP rsa3072", got, err)
	got, err = crypto.RSAOAEPEncrypt(r4pub, data)
	show("OAEP rsa4096", got, err)
	got, err = crypto.ECDH(e3, peer3)
	show("ECDH p384", got, err)
	got, err = crypto.ECDH(e5, peer5)
	show("ECDH p521", got, err)
	showJWK("rsa3072", r3)
	showJWK("rsa4096", r4pub)
	showJWK("p384", e3cert)
	showJWK("p521", e5)

	_, err = crypto.ES384Sign(e5, data)
	rejected("ES384 with a P-521 key", err)
	_, err = crypto.ES256Sign(e3, data)
	rejected("ES256 with a P-384 key", err)
	_, err = crypto.ES512Sign(r4, data)
	rejected("ES512 with an RSA key", err)
	_, err = crypto.RS384Sign(e3, data)
	rejected("RS384 with an EC key", err)
	_, err = crypto.ES512Sign(e5pub, data)
	rejected("ES512 with a public key", err)
	_, err = crypto.ES512Verify(e5, data, make([]byte, 131))
	rejected("short ES512 signature", err)
	_, err = crypto.RS384Verify(r4pub, data, make([]byte, 256))
	rejected("RS384 signature of the wrong length", err)
	_, err = crypto.ECDH(e3, peer5)
	rejected("ECDH across curves", err)
	_, err = crypto.RSAOAEPEncrypt(r4pub, make([]byte, 471))
	rejected("OAEP plaintext over 470 bytes", err)
	_, err = crypto.RSAOAEPEncrypt(r4pub, make([]byte, 470))
	need(err)
	_, err = crypto.RSAOAEPDecrypt(r4, make([]byte, 256))
	rejected("OAEP ciphertext of the wrong length", err)
	_, err = crypto.ImportPEM(rsa1024Public)
	rejected("RSA-1024 import", err)
	_, err = crypto.ImportPEM(secp256k1Private)
	rejected("secp256k1 import", err)

	for _, c := range []struct{ kind, alg string }{{"RSA4096", "RS512"}, {"P384", "ES384"}, {"P521", "ES512"}} {
		var k *crypto.Key
		switch c.kind {
		case "RSA4096":
			k, err = crypto.GenerateRSA4096()
		case "P384":
			k, err = crypto.GenerateP384()
		default:
			k, err = crypto.GenerateP521()
		}
		need(err)
		public, err := k.PublicPEM()
		need(err)
		private, err := k.PrivatePEM()
		need(err)
		reloaded := load(private)
		sig, err := sign(c.alg, reloaded, data)
		need(err)
		ok, err := verify(c.alg, k, data, sig)
		need(err)
		must(ok, "generated "+c.kind+" signature")
		println("GEN", c.kind, c.alg, hex.EncodeToString([]byte(public)), hex.EncodeToString(sig))
		k.Close()
		reloaded.Close()
		_, err = sign(c.alg, k, data)
		rejected("closed key", err)
	}

	file, alg := "", ""
	var key, private *crypto.Key
	pass, fail := 0, 0
	result := func(id int, accepted, valid bool) {
		if accepted == valid {
			pass++
			return
		}
		println("FAIL", file, id)
		fail++
	}
	for _, chunk := range vectors {
		b, err := encoding.Base64Decode(chunk)
		need(err)
		for i := 0; i < len(b); {
			kind := b[i]
			switch kind {
			case 'F':
				n := int(b[i+1])
				file = string(b[i+2 : i+2+n])
				i += 2 + n
			case 'K':
				if key != nil {
					key.Close()
				}
				alg = algorithms[b[i+1]]
				var der []byte
				der, i = field(b, i+2)
				key, err = crypto.ImportPEM(pemOf("PUBLIC KEY", der))
				if err != nil {
					println("FAIL", file, "key", err.Error())
					fail++
				}
			case 'P':
				if private != nil {
					private.Close()
				}
				var der []byte
				der, i = field(b, i+1)
				private, err = crypto.ImportPEM(pemOf("PRIVATE KEY", der))
				if err != nil {
					println("FAIL", file, "private", err.Error())
					fail++
				}
			case 'S', 'E':
				id, valid := u16(b, i+1), b[i+3] == 1
				first, next := field(b, i+4)
				second, end := field(b, next)
				i = end
				if kind == 'S' {
					ok, err := verify(alg, key, first, second)
					result(id, err == nil && ok, valid)
					continue
				}
				accepted := false
				public, err := crypto.ImportPEM(pemOf("PUBLIC KEY", first))
				if err == nil {
					secret, err := crypto.ECDH(private, public)
					public.Close()
					accepted = err == nil && (!valid || bytes.Equal(secret, second))
				}
				result(id, accepted, valid)
			default:
				panic("malformed vector record")
			}
		}
	}
	println("WYCHEPROOF", pass, fail)
	println("CHECK ok")
}
`

func (fx *cryptoFixtures) verify(t *testing.T, stderr string, wycheproof int) {
	t.Helper()
	seen := map[string]bool{}
	checked := false
	hashes := map[string]crypto.Hash{"RS256": crypto.SHA256, "RS384": crypto.SHA384, "RS512": crypto.SHA512, "ES384": crypto.SHA384, "ES512": crypto.SHA512}
	digest := func(alg string) []byte {
		h := hashes[alg].New()
		h.Write(fx.data)
		return h.Sum(nil)
	}
	rsaKeys := map[string]*rsa.PrivateKey{"rsa3072": fx.rsa3072, "rsa4096": fx.rsa4096}
	ecKeys := map[string]*ecdsa.PrivateKey{"p384": fx.p384, "p521": fx.p521}
	url := base64.RawURLEncoding.EncodeToString
	decode := func(s string) []byte {
		t.Helper()
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	verifyEC := func(k *ecdsa.PublicKey, alg string, sig []byte) bool {
		n := (k.Curve.Params().BitSize + 7) / 8
		return len(sig) == 2*n && ecdsa.Verify(k, digest(alg), new(big.Int).SetBytes(sig[:n]), new(big.Int).SetBytes(sig[n:]))
	}
	for _, line := range strings.Split(stderr, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "SIG", "GEN", "JWK":
			seen[strings.Join(f[:min(3, len(f))], " ")] = true
		case "OAEP", "ECDH":
			seen[strings.Join(f[:min(2, len(f))], " ")] = true
		}
		switch {
		case f[0] == "SIG" && len(f) == 4:
			sig := decode(f[3])
			if k := rsaKeys[f[2]]; k != nil {
				if err := rsa.VerifyPKCS1v15(&k.PublicKey, hashes[f[1]], digest(f[1]), sig); err != nil {
					t.Errorf("%s from %s rejected by Go: %v", f[1], f[2], err)
				}
			} else if !verifyEC(&ecKeys[f[2]].PublicKey, f[1], sig) {
				t.Errorf("%s from %s rejected by Go", f[1], f[2])
			}
		case f[0] == "OAEP" && len(f) == 3:
			plain, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, rsaKeys[f[1]], decode(f[2]), nil)
			if err != nil || !bytes.Equal(plain, fx.data) {
				t.Errorf("OAEP %s to Go: %v", f[1], err)
			}
		case f[0] == "ECDH" && len(f) == 3:
			want := map[string][]byte{"p384": sharedSecret(t, fx.p384, fx.peer384), "p521": sharedSecret(t, fx.p521, fx.peer521)}[f[1]]
			if !bytes.Equal(decode(f[2]), want) {
				t.Errorf("ECDH %s differs from Go", f[1])
			}
		case f[0] == "JWK" && len(f) == 8:
			var want []string
			if k := rsaKeys[f[1]]; k != nil {
				want = []string{"RSA", "-", url(k.N.Bytes()), url(big.NewInt(int64(k.E)).Bytes()), "-", "-"}
			} else {
				k := ecKeys[f[1]]
				n := (k.Curve.Params().BitSize + 7) / 8
				want = []string{"EC", k.Curve.Params().Name, "-", "-", url(k.X.FillBytes(make([]byte, n))), url(k.Y.FillBytes(make([]byte, n)))}
			}
			if strings.Join(f[2:], " ") != strings.Join(want, " ") {
				t.Errorf("JWK %s = %v, want %v", f[1], f[2:], want)
			}
		case f[0] == "GEN" && len(f) == 5:
			block, _ := pem.Decode(decode(f[3]))
			if block == nil || block.Type != "PUBLIC KEY" {
				t.Errorf("generated %s public PEM", f[1])
				continue
			}
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				t.Errorf("generated %s: %v", f[1], err)
				continue
			}
			sig := decode(f[4])
			switch k := pub.(type) {
			case *rsa.PublicKey:
				if f[1] != "RSA4096" || k.N.BitLen() != 4096 || rsa.VerifyPKCS1v15(k, hashes[f[2]], digest(f[2]), sig) != nil {
					t.Errorf("generated %s: %d-bit RSA key or its %s signature rejected by Go", f[1], k.N.BitLen(), f[2])
				}
			case *ecdsa.PublicKey:
				if "P"+strconv.Itoa(k.Curve.Params().BitSize) != f[1] || !verifyEC(k, f[2], sig) {
					t.Errorf("generated %s: %s key or its %s signature rejected by Go", f[1], k.Curve.Params().Name, f[2])
				}
			default:
				t.Errorf("generated %s: unexpected %T", f[1], pub)
			}
		case f[0] == "FAIL":
			t.Errorf("Wycheproof %s", strings.Join(f[1:], " "))
		case f[0] == "WYCHEPROOF" && len(f) == 3:
			seen["WYCHEPROOF"] = true
			t.Logf("Wycheproof: %s of %d cases passed", f[1], wycheproof)
			if f[1] != strconv.Itoa(wycheproof) || f[2] != "0" {
				t.Errorf("Wycheproof passed %s and failed %s of %d cases", f[1], f[2], wycheproof)
			}
		case f[0] == "CHECK":
			checked = f[1] == "ok"
		}
	}
	for _, want := range []string{"SIG RS256 rsa3072", "SIG RS384 rsa4096", "SIG RS512 rsa3072", "SIG RS512 rsa4096", "SIG ES384 p384", "SIG ES512 p521",
		"OAEP rsa3072", "OAEP rsa4096", "ECDH p384", "ECDH p521", "JWK rsa3072 RSA", "JWK rsa4096 RSA", "JWK p384 EC", "JWK p521 EC",
		"GEN RSA4096 RS512", "GEN P384 ES384", "GEN P521 ES512", "WYCHEPROOF"} {
		if !seen[want] {
			t.Errorf("probe did not report %q", want)
		}
	}
	if !checked {
		t.Error("probe did not finish")
	}
}
