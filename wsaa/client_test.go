package wsaa

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JoaquinTavella/go-afip-kit/soap"
)

func TestBuildLoginTicketRequestUsesUnsignedIntUniqueID(t *testing.T) {
	now := time.Date(2026, 5, 6, 17, 45, 0, 0, time.FixedZone("ART", -3*60*60))

	tra := buildLoginTicketRequest(now, "wsfe")

	if tra.Header.UniqueID != 1778100300 {
		t.Fatalf("expected Unix seconds uniqueId, got %d", tra.Header.UniqueID)
	}
}

func TestParseLoginTicketResponseXML(t *testing.T) {
	xmlStr := `<loginTicketResponse version="1.0">
  <header>
    <source>CN=wsaahomo</source>
    <destination>SERIALNUMBER=CUIT 20123456789</destination>
    <uniqueId>1710720000</uniqueId>
    <generationTime>2026-03-18T10:00:00-03:00</generationTime>
    <expirationTime>2026-03-18T22:00:00-03:00</expirationTime>
  </header>
  <credentials>
    <token>PD94bWwgdmVyc2lvbj0iMS4wIj8+token_data_here</token>
    <sign>XYZ123signdata</sign>
  </credentials>
</loginTicketResponse>`

	token, err := parseLoginTicketResponseXML(xmlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.Token != "PD94bWwgdmVyc2lvbj0iMS4wIj8+token_data_here" {
		t.Errorf("unexpected token: %s", token.Token)
	}
	if token.Sign != "XYZ123signdata" {
		t.Errorf("unexpected sign: %s", token.Sign)
	}

	loc := time.FixedZone("ART", -3*60*60)
	expected := time.Date(2026, 3, 18, 22, 0, 0, 0, loc)
	if !token.Expiration.Equal(expected) {
		t.Errorf("expected expiration %v, got %v", expected, token.Expiration)
	}
}

func TestAuthenticateSendsLoginCmsSOAPAction(t *testing.T) {
	cert, key := generateTestCertificate(t)
	var gotSOAPAction string
	client := NewClient(soap.WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotSOAPAction = req.Header.Get("SOAPAction")
			body := `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Body><loginCmsResponse><loginCmsReturn>&lt;loginTicketResponse version=&#34;1.0&#34;&gt;&lt;header&gt;&lt;source&gt;CN=wsaahomo&lt;/source&gt;&lt;destination&gt;SERIALNUMBER=CUIT 20123456789&lt;/destination&gt;&lt;uniqueId&gt;1&lt;/uniqueId&gt;&lt;generationTime&gt;2026-03-18T10:00:00-03:00&lt;/generationTime&gt;&lt;expirationTime&gt;2026-03-18T22:00:00-03:00&lt;/expirationTime&gt;&lt;/header&gt;&lt;credentials&gt;&lt;token&gt;token&lt;/token&gt;&lt;sign&gt;sign&lt;/sign&gt;&lt;/credentials&gt;&lt;/loginTicketResponse&gt;</loginCmsReturn></loginCmsResponse></soapenv:Body></soapenv:Envelope>`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/xml"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	}))

	if _, err := client.Authenticate(context.Background(), cert, key, "wsfe", false); err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if gotSOAPAction != `""` {
		t.Fatalf(`expected SOAPAction "", got %q`, gotSOAPAction)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func generateTestCertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test",
			SerialNumber: "CUIT 20123456789",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

func TestTokenAcceso_IsExpired(t *testing.T) {
	future := &TokenAcceso{
		Token:      "t",
		Sign:       "s",
		Expiration: time.Now().Add(2 * time.Hour),
	}
	if future.IsExpired(0) {
		t.Error("token should not be expired")
	}
	if future.IsExpired(30 * time.Minute) {
		t.Error("token should not be expired with 30min margin (2h remaining)")
	}

	almost := &TokenAcceso{
		Token:      "t",
		Sign:       "s",
		Expiration: time.Now().Add(20 * time.Minute),
	}
	if almost.IsExpired(0) {
		t.Error("token should not be expired with zero margin")
	}
	if !almost.IsExpired(30 * time.Minute) {
		t.Error("token should be expired with 30min margin (20min remaining)")
	}

	past := &TokenAcceso{
		Token:      "t",
		Sign:       "s",
		Expiration: time.Now().Add(-1 * time.Hour),
	}
	if !past.IsExpired(0) {
		t.Error("token should be expired")
	}
}

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Get from empty store
	token, err := store.Get(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != nil {
		t.Fatal("expected nil token from empty store")
	}

	// Set and get
	ta := &TokenAcceso{
		Token:      "mytoken",
		Sign:       "mysign",
		Expiration: time.Now().Add(12 * time.Hour),
	}
	if err := store.Set(ctx, "test", ta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token, err = store.Get(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil || token.Token != "mytoken" {
		t.Fatal("expected stored token")
	}

	// Delete
	if err := store.Delete(ctx, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token, _ = store.Get(ctx, "test")
	if token != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestMemoryStore_ExpiredToken(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	ta := &TokenAcceso{
		Token:      "expired",
		Sign:       "s",
		Expiration: time.Now().Add(-1 * time.Hour),
	}
	_ = store.Set(ctx, "key", ta)

	token, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != nil {
		t.Error("expected nil for expired token")
	}
}
