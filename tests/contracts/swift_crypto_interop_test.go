package contracts

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"hash/crc32"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// TestSwiftCryptoInterop exercises emitted source and the actual native
// adapters. Independent Go keys/signatures/ciphertexts enter Swift; native Go
// verifies Swift output. A native self-round-trip alone cannot pass this test.
func TestSwiftCryptoInterop(t *testing.T) {
	fixture, _ := javaCryptoFixtures(t)
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	properties := map[string][]byte{}
	for _, line := range strings.Split(string(read("native.properties")), "\n") {
		if line == "" {
			continue
		}
		name, encoded, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatal("malformed native fixture")
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		properties[name] = data
	}
	parsePrivate := func(name string) any {
		t.Helper()
		block, _ := pem.Decode(read(name + "-private.pem"))
		if block == nil {
			t.Fatal("missing private PEM")
		}
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	rsaKey := parsePrivate("rsa").(*rsa.PrivateKey)
	ecKey := parsePrivate("ec").(*ecdsa.PrivateKey)
	var nativeSig struct{ R, S *big.Int }
	if rest, err := asn1.Unmarshal(properties["ec-signature"], &nativeSig); err != nil || len(rest) != 0 {
		t.Fatalf("Go ECDSA fixture: %v", err)
	}
	esSignature := append(nativeSig.R.FillBytes(make([]byte, 32)), nativeSig.S.FillBytes(make([]byte, 32))...)
	key, nonce, aad := make([]byte, 32), make([]byte, 12), []byte{255, 0, 128}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	plain := properties["data"]
	aesCipher := gcm.Seal(nil, nonce, plain, aad)
	source := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var program strings.Builder
	program.WriteString(`package main
import (
 "github.com/eugenioenko/goalchemy/lib/crypto"
 "github.com/eugenioenko/goalchemy/lib/checksum"
 "github.com/eugenioenko/goalchemy/std/encoding/hex"
)
func need(err error) { if err != nil { panic(err.Error()) } }
func show(name string, data []byte, err error) { need(err); println(name, hex.EncodeToString(data)) }
`)
	for name, data := range map[string][]byte{
		"rsaPrivate": read("rsa-private.pem"), "rsaPublic": read("rsa-cert.pem"),
		"ecPrivate": read("ec-private.pem"), "ecPublic": read("ec-cert.pem"),
		"ecPeer": read("ec-peer-public.pem"),
	} {
		fmt.Fprintf(&program, "const %s = %s\n", name, strconv.Quote(string(data)))
	}
	for name, data := range map[string][]byte{
		"data": plain, "goRS": properties["rsa-signature"], "goES": esSignature,
		"goCipher": properties["rsa-cipher"], "key": key, "nonce": nonce,
		"aad": aad, "goAES": aesCipher, "secret": properties["secret"],
		"salt": properties["salt"], "info": properties["info"],
	} {
		fmt.Fprintf(&program, "var %s = %#v\n", name, data)
	}
	program.WriteString(`
func main() {
 r, err := crypto.ImportPEM(rsaPrivate); need(err); defer r.Close()
 rp, err := crypto.ImportPEM(rsaPublic); need(err); defer rp.Close()
 e, err := crypto.ImportPEM(ecPrivate); need(err); defer e.Close()
 ep, err := crypto.ImportPEM(ecPublic); need(err); defer ep.Close()
 peer, err := crypto.ImportPEM(ecPeer); need(err); defer peer.Close()
 ok, err := crypto.RS256Verify(rp, data, goRS); need(err); if !ok { panic("Go RSA rejected") }
 ok, err = crypto.ES256Verify(ep, data, goES); need(err); if !ok { panic("Go EC rejected") }
 got, err := crypto.RSAOAEPDecrypt(r, goCipher); show("GO_RSA_PLAIN", got, err)
 got, err = crypto.AES256GCMDecrypt(key, nonce, goAES, aad); show("GO_AES_PLAIN", got, err)
 got, err = crypto.RS256Sign(r, data); show("RS", got, err)
 got, err = crypto.ES256Sign(e, data); show("ES", got, err)
 got, err = crypto.RSAOAEPEncrypt(rp, data); show("OAEP", got, err)
 got, err = crypto.AES256GCMEncrypt(key, nonce, data, aad); show("AES", got, err)
 got, err = crypto.ECDH(e, peer); show("ECDH", got, err)
 got, err = crypto.HKDFSHA256(secret, salt, info, 42); show("HKDF", got, err)
 got, err = crypto.SHA256(data); show("SHA", got, err)
 got, err = crypto.HMACSHA256(secret, data); show("HMAC", got, err)
 ok, err = crypto.HMACSHA256Verify(secret, data, got); need(err); if !ok { panic("HMAC rejected") }
 got[0] ^= 1
 ok, err = crypto.HMACSHA256Verify(secret, data, got); need(err); if ok { panic("changed HMAC accepted") }
 jwk, err := r.PublicJWK(); need(err)
 if len(jwk) != 6 || jwk[0] != "RSA" || jwk[1] != "" || jwk[4] != "" || jwk[5] != "" { panic("RSA JWK shape") }
 show("RSA_N", []byte(jwk[2]), nil); show("RSA_E", []byte(jwk[3]), nil)
 jwk, err = e.PublicJWK(); need(err)
 if len(jwk) != 6 || jwk[0] != "EC" || jwk[1] != "P-256" || jwk[2] != "" || jwk[3] != "" { panic("EC JWK shape") }
 show("EC_X", []byte(jwk[4]), nil); show("EC_Y", []byte(jwk[5]), nil)
 generated, err := crypto.GenerateRSA2048(); need(err); defer generated.Close()
 private, err := generated.PrivatePEM(); show("GENERATED_PRIVATE", []byte(private), err)
 public, err := generated.PublicPEM(); show("GENERATED_PUBLIC", []byte(public), err)
 got, err = crypto.RS256Sign(generated, data); show("GENERATED_SIG", got, err)
 got, err = crypto.Random(32); show("RANDOM", got, err)
 println("CRC", checksum.CRC32IEEE(data))
 alias := r; r.Close(); _, err = crypto.RS256Sign(alias, data)
 if err == nil { panic("closed alias accepted") }
 println("CHECK", "ok")
}
`)
	for name, data := range map[string]string{
		"main.go": program.String(),
		"go.mod":  fmt.Sprintf("module swiftcryptoprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", strconv.Quote(root)),
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	if ds := testutil.CompileGate(source, "swift", out, "cooperative"); len(ds) != 0 {
		t.Fatal(ds)
	}
	observation, err := testutil.Runners["swift"](out)
	if err != nil || observation.Exit != 0 {
		t.Fatalf("Swift native source: %v\n%s", err, observation)
	}
	values := map[string][]byte{}
	labels := map[string]bool{
		"GO_RSA_PLAIN": true, "GO_AES_PLAIN": true, "RS": true, "ES": true,
		"OAEP": true, "AES": true, "ECDH": true, "HKDF": true, "SHA": true, "HMAC": true,
		"RSA_N": true, "RSA_E": true, "EC_X": true, "EC_Y": true,
		"GENERATED_PRIVATE": true, "GENERATED_PUBLIC": true, "GENERATED_SIG": true, "RANDOM": true,
	}
	var crc uint64
	checked := false
	for _, line := range strings.Split(observation.Stderr, "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "CHECK":
			checked = parts[1] == "ok"
		case "CRC":
			crc, err = strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				t.Fatal(err)
			}
		default:
			if !labels[parts[0]] {
				continue
			}
			data, err := hex.DecodeString(parts[1])
			if err != nil {
				t.Fatal(err)
			}
			values[parts[0]] = data
		}
	}
	if !checked || len(values) != len(labels) {
		t.Fatalf("missing Swift output checks: %s", observation)
	}
	hash := sha256.Sum256(plain)
	privateBlock, _ := pem.Decode(values["GENERATED_PRIVATE"])
	publicBlock, _ := pem.Decode(values["GENERATED_PUBLIC"])
	if privateBlock == nil || privateBlock.Type != "PRIVATE KEY" || publicBlock == nil || publicBlock.Type != "PUBLIC KEY" {
		t.Fatal("generated RSA key exports must be PKCS#8 and SPKI")
	}
	privateValue, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	publicValue, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	generatedPrivate, privateOK := privateValue.(*rsa.PrivateKey)
	generatedPublic, publicOK := publicValue.(*rsa.PublicKey)
	if !privateOK || !publicOK || generatedPublic.N.BitLen() != 2048 || generatedPublic.E != 65537 || generatedPrivate.N.Cmp(generatedPublic.N) != 0 {
		t.Fatal("generated RSA key shape or exported key identity differs")
	}
	if err := generatedPrivate.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := rsa.VerifyPKCS1v15(generatedPublic, crypto.SHA256, hash[:], values["GENERATED_SIG"]); err != nil {
		t.Fatalf("generated Swift RSA signature rejected by Go: %v", err)
	}
	if len(values["RANDOM"]) != 32 || hmac.Equal(values["RANDOM"], make([]byte, 32)) {
		t.Fatal("native random bytes have incorrect length or are all zero")
	}
	for name, expected := range map[string][]byte{
		"RSA_N": rsaKey.N.Bytes(), "RSA_E": big.NewInt(int64(rsaKey.E)).Bytes(),
		"EC_X": ecKey.X.FillBytes(make([]byte, 32)), "EC_Y": ecKey.Y.FillBytes(make([]byte, 32)),
	} {
		decoded, err := base64.RawURLEncoding.DecodeString(string(values[name]))
		if err != nil || !hmac.Equal(decoded, expected) {
			t.Fatalf("%s JWK differs from native Go: %v", name, err)
		}
	}
	if err := rsa.VerifyPKCS1v15(&rsaKey.PublicKey, crypto.SHA256, hash[:], values["RS"]); err != nil {
		t.Fatalf("Swift RSA signature rejected by Go: %v", err)
	}
	sig := values["ES"]
	if len(sig) != 64 || !ecdsa.Verify(&ecKey.PublicKey, hash[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("Swift ES256 signature rejected by Go")
	}
	decrypted, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, rsaKey, values["OAEP"], nil)
	if err != nil || !hmac.Equal(decrypted, plain) {
		t.Fatalf("Swift OAEP ciphertext rejected by Go: %v", err)
	}
	mac := hmac.New(sha256.New, properties["secret"])
	mac.Write(plain)
	for name, expected := range map[string][]byte{
		"GO_RSA_PLAIN": plain, "GO_AES_PLAIN": plain, "AES": aesCipher,
		"ECDH": properties["ecdh"], "HKDF": properties["hkdf"],
		"SHA": hash[:], "HMAC": mac.Sum(nil),
	} {
		if !hmac.Equal(values[name], expected) {
			t.Errorf("%s differs from native Go", name)
		}
	}
	if crc != uint64(crc32.ChecksumIEEE(plain)) {
		t.Error("CRC differs from native Go")
	}
}
