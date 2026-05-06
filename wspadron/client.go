package wspadron

import (
	"context"
	"fmt"

	afipkit "github.com/JoaquinTavella/go-afip-kit"
	"github.com/JoaquinTavella/go-afip-kit/soap"
	"github.com/JoaquinTavella/go-afip-kit/wsaa"
)

const (
	homoURL = "https://awshomo.afip.gov.ar/sr-padron/webservices/personaServiceA5"
	prodURL = "https://aws.afip.gov.ar/sr-padron/webservices/personaServiceA5"

	actionGetPersona = padronNS + "getPersona"
	actionDummy      = padronNS + "dummy"
)

// HealthStatus represents the state of AFIP's Padron servers.
type HealthStatus struct {
	AppServer  string
	DbServer   string
	AuthServer string
}

// IsOK returns true if all three servers report "OK".
func (h *HealthStatus) IsOK() bool {
	return h.AppServer == "OK" && h.DbServer == "OK" && h.AuthServer == "OK"
}

// Client communicates with WS-SR-PADRON A5 for taxpayer data queries.
type Client struct {
	soap       *soap.Client
	production bool
}

// NewClient creates a WS-SR-PADRON client. Set production=true for live AFIP.
func NewClient(production bool, opts ...soap.Option) *Client {
	return &Client{
		soap:       soap.NewClient(opts...),
		production: production,
	}
}

func (c *Client) url() string {
	if c.production {
		return prodURL
	}
	return homoURL
}

// HealthCheck verifies AFIP Padron server status (dummy). Does not require authentication.
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	req := dummyRequest{XMLNS: padronNS}
	var resp dummyResponse

	if err := c.soap.Call(ctx, c.url(), actionDummy, &req, &resp); err != nil {
		return nil, &afipkit.ErrSOAP{Err: err}
	}

	return &HealthStatus{
		AppServer:  resp.Return.AppServer,
		DbServer:   resp.Return.DbServer,
		AuthServer: resp.Return.AuthServer,
	}, nil
}

// GetPersona queries taxpayer data by CUIT from Padron Alcance 5 (Constancia de Inscripción).
// Returns the Persona record and any non-fatal errors returned by AFIP.
func (c *Client) GetPersona(ctx context.Context, token *wsaa.TokenAcceso, cuit int64, idPersona int64) (*Persona, error) {
	req := getPersonaRequest{
		XMLNS:            padronNS,
		Token:            token.Token,
		Sign:             token.Sign,
		CuitRepresentada: cuit,
		IDPersona:        idPersona,
	}

	var resp getPersonaResponse
	if err := c.soap.Call(ctx, c.url(), actionGetPersona, &req, &resp); err != nil {
		return nil, &afipkit.ErrSOAP{Err: err}
	}

	if resp.Return == nil {
		return nil, fmt.Errorf("wspadron: empty response from AFIP")
	}

	if resp.Return.DatosGenerales == nil {
		// Check if there are errors explaining why
		var errs []string
		for _, e := range resp.Return.ErrorConstancia {
			errs = append(errs, e.Error)
		}
		for _, e := range resp.Return.ErrorMonotributo {
			errs = append(errs, e.Error)
		}
		for _, e := range resp.Return.ErrorRegimenGeneral {
			errs = append(errs, e.Error)
		}
		if len(errs) > 0 {
			return nil, afipkit.NewAFIPError(nil)
		}
		return nil, fmt.Errorf("wspadron: no persona data found")
	}

	persona := resp.Return.toPersona()

	// Return persona even with errors (some errors are non-fatal)
	if len(persona.Errores) > 0 {
		return persona, fmt.Errorf("wspadron: persona retrieved with warnings: %v", persona.Errores)
	}

	return persona, nil
}

// --- Dummy types ---

type dummyRequest struct {
	XMLName struct{} `xml:"dummy"`
	XMLNS   string   `xml:"xmlns,attr"`
}

type dummyResponse struct {
	XMLName struct{}    `xml:"dummyResponse"`
	Return  dummyReturn `xml:"return"`
}

type dummyReturn struct {
	AppServer  string `xml:"appserver"`
	DbServer   string `xml:"dbserver"`
	AuthServer string `xml:"authserver"`
}
