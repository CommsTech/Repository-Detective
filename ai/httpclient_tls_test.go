package ai

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newSelfSignedHostCert returns a PEM certificate + key for a self-signed CA
// and a server certificate signed by it, for TLS regression tests.
func newSelfSignedHostCert(t *testing.T, host string) (caPEM, certPEM, keyPEM []byte) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "rd-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("server key: %v", err)
	}
	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     []string{host},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("server cert: %v", err)
	}

	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	keyDERServer, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDERServer})
	return caPEM, certPEM, keyPEM
}

func TestHTTPClientCustomCATrustsPrivateEndpoint(t *testing.T) {
	caPEM, certPEM, keyPEM := newSelfSignedHostCert(t, "127.0.0.1")
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("key pair: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`ok`))
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()

	// System-roots-only client must fail against the private CA.
	sysClient := NewHTTPClient(false, "")
	resp, err := sysClient.Get(server.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("system-root client unexpectedly trusted private-CA endpoint")
	}

	// Custom CA client must succeed with full verification (no skip).
	caClient := NewHTTPClient(false, string(caPEM))
	resp, err = caClient.Get(server.URL)
	if err != nil {
		t.Fatalf("custom-CA client failed: %v", err)
	}
	resp.Body.Close()

	// Wrong CA bundle must fail (verification stays on).
	otherCA, _, _ := newSelfSignedHostCert(t, "127.0.0.1")
	wrongClient := NewHTTPClient(false, string(otherCA))
	if resp, err := wrongClient.Get(server.URL); err == nil {
		resp.Body.Close()
		t.Fatalf("client with wrong CA unexpectedly connected")
	}
}

func TestHTTPClientCACertWinsOverInsecureSkip(t *testing.T) {
	caPEM, certPEM, keyPEM := newSelfSignedHostCert(t, "127.0.0.1")
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("key pair: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()

	// CACertPEM set: verification must remain enabled even if the legacy
	// insecure flag is also true — a wrong CA must fail, not pass.
	client := NewHTTPClient(true, string(caPEM))
	wrongCA, _, _ := newSelfSignedHostCert(t, "127.0.0.1")
	wrongClient := NewHTTPClient(true, string(wrongCA))
	if _, werr := wrongClient.Get(server.URL); werr == nil {
		t.Fatalf("CACertPEM did not disable InsecureSkipVerify")
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("correct CA should verify: %v", err)
	}
	resp.Body.Close()
}

func TestHTTPClientInvalidCABundleFallsBackToSystemRoots(t *testing.T) {
	client := NewHTTPClient(false, "not a pem bundle")
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport")
	}
	if transport.TLSClientConfig.RootCAs != nil {
		t.Fatalf("invalid CA bundle should not set RootCAs")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("verification must stay on after invalid CA bundle")
	}
}
