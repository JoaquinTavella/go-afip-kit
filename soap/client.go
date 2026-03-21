// Package soap provides a minimal SOAP 1.1 client for AFIP web services.
//
// It handles envelope wrapping/unwrapping, HTTP transport, SOAP fault detection,
// and configurable timeouts. No WSDL parsing — callers provide typed request/response structs.
package soap

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a SOAP 1.1 HTTP client.
type Client struct {
	httpClient *http.Client
}

// Option configures the SOAP client.
type Option func(*Client)

// WithTimeout sets the HTTP client timeout. Default is 30s.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient replaces the underlying HTTP client entirely.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a SOAP client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Fault represents a SOAP Fault element returned by the server.
type Fault struct {
	Code   string `xml:"faultcode"`
	String string `xml:"faultstring"`
	Detail string `xml:"detail"`
}

func (f *Fault) Error() string {
	return fmt.Sprintf("soap fault: [%s] %s", f.Code, f.String)
}

// faultEnvelope is used to detect SOAP faults in the response body.
type faultEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Fault *Fault `xml:"Fault"`
	} `xml:"Body"`
}

// Call performs a SOAP request: marshals request into a SOAP envelope, sends it
// via HTTP POST, and unmarshals the response body into the provided struct.
//
// It detects and returns SOAP faults as *Fault errors.
// The soapAction is sent as the SOAPAction HTTP header.
func (c *Client) Call(ctx context.Context, url, soapAction string, request, response any) error {
	// 1. Marshal request body
	bodyXML, err := xml.Marshal(request)
	if err != nil {
		return fmt.Errorf("soap: marshal request: %w", err)
	}

	// 2. Wrap in SOAP envelope
	envelope := NewEnvelope(bodyXML)
	envXML, err := xml.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("soap: marshal envelope: %w", err)
	}

	// Prepend XML declaration
	fullXML := append([]byte(xml.Header), envXML...)

	// 3. HTTP POST
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(fullXML))
	if err != nil {
		return fmt.Errorf("soap: create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	if soapAction != "" {
		req.Header.Set("SOAPAction", soapAction)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("soap: http call: %w", err)
	}
	defer resp.Body.Close()

	// 4. Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("soap: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to extract SOAP fault from error responses
		if fault := extractFault(respBody); fault != nil {
			return fault
		}
		return fmt.Errorf("soap: http status %d: %s", resp.StatusCode, truncate(respBody, 500))
	}

	// 5. Check for SOAP fault in 200 response
	if fault := extractFault(respBody); fault != nil {
		return fault
	}

	// 6. Unwrap SOAP envelope
	var respEnvelope ResponseEnvelope
	if err := xml.Unmarshal(respBody, &respEnvelope); err != nil {
		return fmt.Errorf("soap: unmarshal envelope: %w (body: %s)", err, truncate(respBody, 300))
	}

	// 7. Unmarshal inner body into caller's response struct
	if err := xml.Unmarshal(respEnvelope.Body.Content, response); err != nil {
		return fmt.Errorf("soap: unmarshal response body: %w", err)
	}

	return nil
}

// extractFault attempts to parse a SOAP Fault from response bytes.
// Returns nil if no fault is found.
func extractFault(body []byte) *Fault {
	// Quick check to avoid unnecessary parsing
	if !strings.Contains(string(body), "Fault") {
		return nil
	}
	var fe faultEnvelope
	if err := xml.Unmarshal(body, &fe); err != nil {
		return nil
	}
	return fe.Body.Fault
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
