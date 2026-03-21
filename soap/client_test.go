package soap

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testRequest struct {
	XMLName xml.Name `xml:"TestRequest"`
	XMLNS   string   `xml:"xmlns,attr"`
	Value   string   `xml:"Value"`
}

type testResponse struct {
	XMLName xml.Name `xml:"TestResponse"`
	Result  string   `xml:"Result"`
}

func TestCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/xml; charset=utf-8" {
			t.Errorf("expected content-type text/xml, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("SOAPAction") != "TestAction" {
			t.Errorf("expected SOAPAction TestAction, got %s", r.Header.Get("SOAPAction"))
		}

		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <TestResponse><Result>OK</Result></TestResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := NewClient(WithTimeout(5 * time.Second))
	req := testRequest{XMLNS: "http://test.example.com/", Value: "hello"}
	var resp testResponse

	err := client.Call(context.Background(), server.URL, "TestAction", &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Result != "OK" {
		t.Errorf("expected Result=OK, got %s", resp.Result)
	}
}

func TestCall_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewClient()
	req := testRequest{Value: "test"}
	var resp testResponse

	err := client.Call(context.Background(), server.URL, "", &req, &resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCall_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := NewClient(WithTimeout(10 * time.Second))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := testRequest{Value: "test"}
	var resp testResponse

	err := client.Call(ctx, server.URL, "", &req, &resp)
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
}
