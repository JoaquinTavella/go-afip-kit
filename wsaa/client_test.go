package wsaa

import (
	"context"
	"testing"
	"time"
)

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
