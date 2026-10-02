package contracts

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"io/fs"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJavaNativeCrypto(t *testing.T) {
	env := driver.ToolEnv()
	bin := ""
	for _, v := range env {
		if strings.HasPrefix(v, "JAVA_HOME=") {
			bin = filepath.Join(strings.TrimPrefix(v, "JAVA_HOME="), "bin")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	deps := exec.CommandContext(ctx, "bash", filepath.Join(root, "targets/java/tests/crypto-dependencies.sh"))
	deps.Env = env
	output, err := deps.Output()
	if err != nil {
		t.Fatal(err)
	}
	jar := strings.TrimSpace(string(output))
	out := t.TempDir()
	sources := []string{}
	for _, dir := range []string{"targets/java/types", "targets/java/runtime"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() && strings.HasSuffix(p, ".java") {
				sources = append(sources, p)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	sources = append(sources, filepath.Join(root, "targets/java/tests/CryptoTest.java"))
	cmd := exec.CommandContext(ctx, filepath.Join(bin, "javac"), append([]string{"-d", out}, sources...)...)
	cmd.Env = env
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("native Java compile: %v\n%s", e, data)
	}
	fixture, verify := javaCryptoFixtures(t)
	cmd = exec.CommandContext(ctx, filepath.Join(bin, "java"), "-cp", out+":"+jar, "rt.CryptoTest", fixture)
	cmd.Env = env
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("JCA/BC vectors: %v\n%s", e, data)
	} else {
		t.Log(string(data))
	}
	verify()
}

// Go standard crypto supplies independent PEM/certificate/JWK/primitive fixtures;
// outputs from Java are subsequently verified with the same native Go keys.
func javaCryptoFixtures(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, b []byte) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(dir, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	data := []byte{0, 255, 128, 1, 2, 3}
	properties := map[string][]byte{"data": data, "secret": {11, 22, 0, 255}, "salt": {0, 128}, "info": {255, 1}}
	rsaKey, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	ecKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	url := base64.RawURLEncoding.EncodeToString
	for name, key := range map[string]any{"rsa": rsaKey, "ec": ecKey} {
		var pub any
		var jwk map[string]string
		if name == "rsa" {
			pub = &rsaKey.PublicKey
			jwk = map[string]string{"kty": "RSA", "n": url(rsaKey.N.Bytes()), "e": url(big.NewInt(int64(rsaKey.E)).Bytes())}
		} else {
			pub = &ecKey.PublicKey
			jwk = map[string]string{"kty": "EC", "crv": "P-256", "x": url(ecKey.X.FillBytes(make([]byte, 32))), "y": url(ecKey.Y.FillBytes(make([]byte, 32)))}
		}
		der, e := x509.MarshalPKCS8PrivateKey(key)
		if e != nil {
			t.Fatal(e)
		}
		write(name+"-private.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
		tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Java native fixture"}, NotBefore: time.Unix(0, 0), NotAfter: time.Unix(4102444800, 0), KeyUsage: x509.KeyUsageDigitalSignature}
		cert, e := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, key)
		if e != nil {
			t.Fatal(e)
		}
		write(name+"-cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}))
		properties[name+"-jwk"], e = json.Marshal(jwk)
		if e != nil {
			t.Fatal(e)
		}
	}
	hash := sha256.Sum256(data)
	properties["rsa-signature"], e = rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
	if e != nil {
		t.Fatal(e)
	}
	properties["ec-signature"], e = ecdsa.SignASN1(rand.Reader, ecKey, hash[:])
	if e != nil {
		t.Fatal(e)
	}
	// TDF3 OAEP uses the accepted SHA1 framing rather than modern SHA256 defaults.
	properties["rsa-cipher"], e = rsa.EncryptOAEP(sha1.New(), rand.Reader, &rsaKey.PublicKey, data, nil)
	if e != nil {
		t.Fatal(e)
	}
	peer, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	peerDer, e := x509.MarshalPKIXPublicKey(&peer.PublicKey)
	if e != nil {
		t.Fatal(e)
	}
	write("ec-peer-public.pem", pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: peerDer}))
	native, e := ecKey.ECDH()
	if e != nil {
		t.Fatal(e)
	}
	peerNative, e := peer.PublicKey.ECDH()
	if e != nil {
		t.Fatal(e)
	}
	properties["ecdh"], e = native.ECDH(peerNative)
	if e != nil {
		t.Fatal(e)
	}
	properties["hkdf"], e = hkdf.Key(sha256.New, properties["secret"], properties["salt"], string(properties["info"]), 42)
	if e != nil {
		t.Fatal(e)
	}
	var lines strings.Builder
	for k, v := range properties {
		lines.WriteString(k + "=" + base64.StdEncoding.EncodeToString(v) + "\n")
	}
	write("native.properties", []byte(lines.String()))
	return dir, func() {
		read := func(name string) []byte {
			t.Helper()
			b, e := os.ReadFile(filepath.Join(dir, name))
			if e != nil {
				t.Fatal(e)
			}
			return b
		}
		if e := rsa.VerifyPKCS1v15(&rsaKey.PublicKey, crypto.SHA256, hash[:], read("rsa-java-signature.bin")); e != nil {
			t.Fatal(e)
		}
		if !ecdsa.VerifyASN1(&ecKey.PublicKey, hash[:], read("ec-java-signature.bin")) {
			t.Fatal("Java ES256 signature rejected by Go")
		}
		plain, e := rsa.DecryptOAEP(sha1.New(), rand.Reader, rsaKey, read("rsa-java-cipher.bin"), nil)
		if e != nil || !bytes.Equal(plain, data) {
			t.Fatalf("Java OAEP to Go: %v", e)
		}
		for _, name := range []string{"ecdh", "hkdf"} {
			if !bytes.Equal(properties[name], read(name+"-java.bin")) {
				t.Fatal("Java to Go " + name)
			}
		}
		t.Log("native Go/Java signatures, OAEP, ECDH, HKDF, canonical JWK and RSA/P256 certificates both directions")
	}
}
