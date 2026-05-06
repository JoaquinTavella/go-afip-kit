// Package wsaa implements the AFIP WSAA (Web Service de Autenticacion y Autorizacion)
// client for obtaining access tokens via CMS/PKCS#7 signed login tickets.
package wsaa

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"

	afipkit "github.com/JoaquinTavella/go-afip-kit"
	"github.com/JoaquinTavella/go-afip-kit/soap"
)

const (
	homoURL = "https://wsaahomo.afip.gov.ar/ws/services/LoginCms"
	prodURL = "https://wsaa.afip.gov.ar/ws/services/LoginCms"

	wsaaNamespace  = "http://wsaa.view.sua.dvadac.desein.afip.gov"
	actionLoginCms = `""`
)

// Client communicates with WSAA to obtain access tokens.
type Client struct {
	soap *soap.Client
}

// NewClient creates a WSAA client.
func NewClient(opts ...soap.Option) *Client {
	return &Client{
		soap: soap.NewClient(opts...),
	}
}

// Authenticate obtains a TokenAcceso from WSAA by signing a TRA with the
// provided certificate and private key.
//
// The service parameter identifies the target web service (e.g. "wsfe" for WSFEv1).
// Set production=true to use the production WSAA endpoint.
func (c *Client) Authenticate(ctx context.Context, cert *x509.Certificate, key *rsa.PrivateKey, service string, production bool) (*TokenAcceso, error) {
	// 1. Generate TRA (Ticket de Requerimiento de Acceso)
	now := time.Now()
	tra := buildLoginTicketRequest(now, service)

	traXML, err := xml.Marshal(tra)
	if err != nil {
		return nil, &afipkit.ErrWSAAAuth{Err: fmt.Errorf("marshal TRA: %w", err)}
	}

	// Prepend XML declaration (AFIP requires it)
	traXML = append([]byte(xml.Header), traXML...)

	// 2. Sign TRA with CMS/PKCS#7
	cms, err := signCMS(traXML, cert, key)
	if err != nil {
		return nil, &afipkit.ErrWSAAAuth{Err: fmt.Errorf("sign CMS: %w", err)}
	}
	cmsBase64 := base64.StdEncoding.EncodeToString(cms)

	// 3. Call LoginCms via SOAP
	url := homoURL
	if production {
		url = prodURL
	}

	req := loginCmsRequest{
		XMLNS: wsaaNamespace,
		In0:   cmsBase64,
	}

	var resp loginCmsResponse
	if err := c.soap.Call(ctx, url, actionLoginCms, &req, &resp); err != nil {
		return nil, &afipkit.ErrWSAAAuth{Err: fmt.Errorf("LoginCms: %w", err)}
	}

	// 4. Parse the TA (Ticket de Acceso) from the response string
	token, err := parseLoginTicketResponseXML(resp.LoginCmsReturn)
	if err != nil {
		return nil, &afipkit.ErrWSAAAuth{Err: fmt.Errorf("parse TA: %w", err)}
	}

	return token, nil
}

func buildLoginTicketRequest(now time.Time, service string) loginTicketRequest {
	return loginTicketRequest{
		Version: "1.0",
		Header: traHeader{
			UniqueID:       now.Unix(),
			GenerationTime: now.Add(-10 * time.Minute).Format(time.RFC3339),
			ExpirationTime: now.Add(10 * time.Minute).Format(time.RFC3339),
		},
		Service: service,
	}
}

// parseLoginTicketResponseXML parses the XML string returned inside loginCmsReturn.
func parseLoginTicketResponseXML(xmlStr string) (*TokenAcceso, error) {
	var ta loginTicketResponse
	if err := xml.Unmarshal([]byte(xmlStr), &ta); err != nil {
		return nil, fmt.Errorf("unmarshal TA: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, ta.Header.ExpirationTime)
	if err != nil {
		return nil, fmt.Errorf("parse expiration time: %w", err)
	}

	return &TokenAcceso{
		Token:      ta.Credentials.Token,
		Sign:       ta.Credentials.Sign,
		Expiration: expiration,
	}, nil
}
