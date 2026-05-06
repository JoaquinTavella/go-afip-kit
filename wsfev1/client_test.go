package wsfev1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	afipkit "github.com/JoaquinTavella/go-afip-kit"
	"github.com/JoaquinTavella/go-afip-kit/soap"
	"github.com/JoaquinTavella/go-afip-kit/wsaa"
)

func newTestServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(response))
	}))
}

func newTestClient(serverURL string) *Client {
	return &Client{
		soap:       soap.NewClient(),
		production: false,
	}
}

// Override url for testing
func newTestClientWithURL(serverURL string) *testableClient {
	return &testableClient{
		Client: Client{
			soap:       soap.NewClient(),
			production: false,
		},
		overrideURL: serverURL,
	}
}

type testableClient struct {
	Client
	overrideURL string
}

func TestHealthCheck(t *testing.T) {
	server := newTestServer(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FEDummyResponse xmlns="http://ar.gov.afip.dif.FEV1/">
      <FEDummyResult>
        <AppServer>OK</AppServer>
        <DbServer>OK</DbServer>
        <AuthServer>OK</AuthServer>
      </FEDummyResult>
    </FEDummyResponse>
  </soap:Body>
</soap:Envelope>`)
	defer server.Close()

	// Create client with test SOAP
	client := &Client{
		soap:       soap.NewClient(),
		production: false,
	}

	// Use the server URL directly via soap.Call
	req := FEDummyRequest{XMLNS: wsfeNS}
	var resp FEDummyResponse
	err := client.soap.Call(context.Background(), server.URL, actionDummy, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FEDummyResult.AppServer != "OK" {
		t.Errorf("expected AppServer=OK, got %s", resp.FEDummyResult.AppServer)
	}
	if resp.FEDummyResult.DbServer != "OK" {
		t.Errorf("expected DbServer=OK, got %s", resp.FEDummyResult.DbServer)
	}
}

func TestUltimoAutorizado_Success(t *testing.T) {
	server := newTestServer(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FECompUltimoAutorizadoResponse xmlns="http://ar.gov.afip.dif.FEV1/">
      <FECompUltimoAutorizadoResult>
        <PtoVta>1</PtoVta>
        <CbteTipo>6</CbteTipo>
        <CbteNro>42</CbteNro>
      </FECompUltimoAutorizadoResult>
    </FECompUltimoAutorizadoResponse>
  </soap:Body>
</soap:Envelope>`)
	defer server.Close()

	client := &Client{soap: soap.NewClient(), production: false}
	token := &wsaa.TokenAcceso{Token: "t", Sign: "s"}

	req := FECompUltimoAutorizadoRequest{
		XMLNS:    wsfeNS,
		Auth:     Auth{Token: token.Token, Sign: token.Sign, Cuit: 20123456789},
		PtoVta:   1,
		CbteTipo: 6,
	}
	var resp FECompUltimoAutorizadoResponse
	err := client.soap.Call(context.Background(), server.URL, actionUltimoAut, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := resp.FECompUltimoAutorizadoResult
	if result.CbteNro != 42 {
		t.Errorf("expected CbteNro=42, got %d", result.CbteNro)
	}
}

func TestUltimoAutorizado_AFIPError(t *testing.T) {
	server := newTestServer(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FECompUltimoAutorizadoResponse xmlns="http://ar.gov.afip.dif.FEV1/">
      <FECompUltimoAutorizadoResult>
        <PtoVta>1</PtoVta>
        <CbteTipo>6</CbteTipo>
        <CbteNro>0</CbteNro>
        <Errors>
          <Err><Code>10005</Code><Msg>Punto de venta invalido</Msg></Err>
        </Errors>
      </FECompUltimoAutorizadoResult>
    </FECompUltimoAutorizadoResponse>
  </soap:Body>
</soap:Envelope>`)
	defer server.Close()

	client := &Client{soap: soap.NewClient(), production: false}
	token := &wsaa.TokenAcceso{Token: "t", Sign: "s"}

	req := FECompUltimoAutorizadoRequest{
		XMLNS:    wsfeNS,
		Auth:     Auth{Token: token.Token, Sign: token.Sign, Cuit: 20123456789},
		PtoVta:   1,
		CbteTipo: 6,
	}
	var resp FECompUltimoAutorizadoResponse
	err := client.soap.Call(context.Background(), server.URL, actionUltimoAut, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that we can parse errors properly
	afipErr := errorsToAFIP(resp.FECompUltimoAutorizadoResult.Errors)
	if afipErr == nil {
		t.Fatal("expected AFIP error")
	}

	e, ok := afipkit.IsAFIPError(afipErr)
	if !ok {
		t.Fatal("expected AFIPError type")
	}
	if !e.IsInvalidPtoVta() {
		t.Errorf("expected IsInvalidPtoVta, got code %d", e.FirstCode())
	}
}

func TestCAESolicitar_Approved(t *testing.T) {
	server := newTestServer(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FECAESolicitarResponse xmlns="http://ar.gov.afip.dif.FEV1/">
      <FECAESolicitarResult>
        <FeCabResp>
          <Cuit>20123456789</Cuit>
          <PtoVta>1</PtoVta>
          <CbteTipo>6</CbteTipo>
          <FchProceso>20260318153000</FchProceso>
          <CantReg>1</CantReg>
          <Resultado>A</Resultado>
          <Reproceso>N</Reproceso>
        </FeCabResp>
        <FeDetResp>
          <FECAEDetResponse>
            <Concepto>2</Concepto>
            <DocTipo>80</DocTipo>
            <DocNro>30123456789</DocNro>
            <CbteDesde>1</CbteDesde>
            <CbteHasta>1</CbteHasta>
            <CbteFch>20260318</CbteFch>
            <Resultado>A</Resultado>
            <CAE>71234567890123</CAE>
            <CAEFchVto>20260328</CAEFchVto>
          </FECAEDetResponse>
        </FeDetResp>
      </FECAESolicitarResult>
    </FECAESolicitarResponse>
  </soap:Body>
</soap:Envelope>`)
	defer server.Close()

	// Parse response directly
	client := &Client{soap: soap.NewClient(), production: false}

	req := &FECAESolicitarRequest{
		XMLNS: wsfeNS,
		Auth:  Auth{Token: "t", Sign: "s", Cuit: 20123456789},
		FeCAEReq: FeCAEReq{
			FeCabReq: FeCabReq{CantReg: 1, PtoVta: 1, CbteTipo: 6},
			FeDetReq: []FECAEDetRequest{{
				Concepto: 2, DocTipo: 80, DocNro: 30123456789,
				CbteDesde: 1, CbteHasta: 1, CbteFch: "20260318",
				ImpTotal: "121000.00", ImpTotConc: "0", ImpNeto: "100000.00",
				ImpOpEx: "0", ImpIVA: "21000.00", ImpTrib: "0",
				MonId: "PES", MonCotiz: "1",
			}},
		},
	}

	var resp FECAESolicitarResponse
	err := client.soap.Call(context.Background(), server.URL, actionCAESolicitar, req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := resp.FECAESolicitarResult
	if result.FeCabResp.Resultado != "A" {
		t.Errorf("expected Resultado=A, got %s", result.FeCabResp.Resultado)
	}
	if len(result.FeDetResp) == 0 {
		t.Fatal("expected at least one detail response")
	}
	det := result.FeDetResp[0]
	if det.CAE != "71234567890123" {
		t.Errorf("expected CAE=71234567890123, got %s", det.CAE)
	}
	if det.Resultado != "A" {
		t.Errorf("expected detail Resultado=A, got %s", det.Resultado)
	}
}

func TestHealthStatus_IsOK(t *testing.T) {
	ok := &HealthStatus{AppServer: "OK", DbServer: "OK", AuthServer: "OK"}
	if !ok.IsOK() {
		t.Error("expected IsOK=true")
	}

	partial := &HealthStatus{AppServer: "OK", DbServer: "OK", AuthServer: ""}
	if partial.IsOK() {
		t.Error("expected IsOK=false when AuthServer is empty")
	}
}
