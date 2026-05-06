package wspadron

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoaquinTavella/go-afip-kit/soap"
	"github.com/JoaquinTavella/go-afip-kit/wsaa"
)

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <dummyResponse xmlns="http://impl.service.ws_padron.afip.gov/">
      <return>
        <appserver>OK</appserver>
        <dbserver>OK</dbserver>
        <authserver>OK</authserver>
      </return>
    </dummyResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := &Client{soap: soap.NewClient(), production: false}

	req := dummyRequest{XMLNS: padronNS}
	var resp dummyResponse
	err := client.soap.Call(context.Background(), server.URL, actionDummy, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Return.AppServer != "OK" {
		t.Errorf("expected AppServer=OK, got %s", resp.Return.AppServer)
	}
	if resp.Return.DbServer != "OK" {
		t.Errorf("expected DbServer=OK, got %s", resp.Return.DbServer)
	}
	if resp.Return.AuthServer != "OK" {
		t.Errorf("expected AuthServer=OK, got %s", resp.Return.AuthServer)
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

func TestGetPersona_Juridica(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getPersonaResponse xmlns="http://impl.service.ws_padron.afip.gov/">
      <personaReturn>
        <datosGenerales>
          <tipoPersona>JURIDICA</tipoPersona>
          <tipoClave>CUIT</tipoClave>
          <idPersona>30123456789</idPersona>
          <estadoClave>ACTIVO</estadoClave>
          <esSucesion>N</esSucesion>
          <razonSocial>EMPRESA EJEMPLO SRL</razonSocial>
          <domicilioFiscal>
            <direccion>AV CORRIENTES 1234</direccion>
            <localidad>CABA</localidad>
            <idProvincia>2</idProvincia>
            <codPostal>1043</codPostal>
          </domicilioFiscal>
        </datosGenerales>
        <datosMonotributo>
          <impuesto>
            <idImpuesto>20</idImpuesto>
            <estado>ACTIVO</estado>
          </impuesto>
          <categoriaMonotributo>
            <idCategoria>1</idCategoria>
            <descripcionCategoria>Categoria A</descripcionCategoria>
          </categoriaMonotributo>
        </datosMonotributo>
        <datosRegimenGeneral>
          <impuesto>
            <idImpuesto>30</idImpuesto>
            <estado>ACTIVO</estado>
          </impuesto>
          <actividad>
            <idActividad>62020</idActividad>
          </actividad>
        </datosRegimenGeneral>
      </personaReturn>
    </getPersonaResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := &Client{soap: soap.NewClient(), production: false}
	token := &wsaa.TokenAcceso{Token: "t", Sign: "s"}

	req := getPersonaRequest{
		XMLNS:            padronNS,
		Token:            token.Token,
		Sign:             token.Sign,
		CuitRepresentada: 20123456789,
		IDPersona:        30123456789,
	}
	var resp getPersonaResponse
	err := client.soap.Call(context.Background(), server.URL, actionGetPersona, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Return == nil {
		t.Fatal("expected personaReturn")
	}

	persona := resp.Return.toPersona()
	if persona == nil {
		t.Fatal("expected persona")
	}

	if persona.TipoPersona != PersonaJuridica {
		t.Errorf("expected TipoPersona=JURIDICA, got %s", persona.TipoPersona)
	}
	if persona.IDPersona != 30123456789 {
		t.Errorf("expected IDPersona=30123456789, got %d", persona.IDPersona)
	}
	if persona.EstadoClave != EstadoActivo {
		t.Errorf("expected EstadoClave=ACTIVO, got %s", persona.EstadoClave)
	}
	if persona.Denominacion() != "EMPRESA EJEMPLO SRL" {
		t.Errorf("expected Denominacion='EMPRESA EJEMPLO SRL', got %s", persona.Denominacion())
	}
	if persona.DomicilioFiscal.Direccion != "AV CORRIENTES 1234" {
		t.Errorf("expected Direccion='AV CORRIENTES 1234', got %s", persona.DomicilioFiscal.Direccion)
	}
	if persona.DomicilioFiscal.Provincia != "CABA" {
		t.Errorf("expected Provincia='CABA', got %s", persona.DomicilioFiscal.Provincia)
	}
	if persona.DomicilioFiscal.CodPostal != "1043" {
		t.Errorf("expected CodPostal='1043', got %s", persona.DomicilioFiscal.CodPostal)
	}
	if !persona.HasImpuesto(ImpuestoIVA) {
		t.Error("expected HasImpuesto(IVA)=true")
	}
	if !persona.HasImpuesto(ImpuestoMonotributo) {
		t.Error("expected HasImpuesto(Monotributo)=true")
	}
	if persona.DatosMonotributo.CategoriaMonotributo == nil {
		t.Fatal("expected CategoriaMonotributo")
	}
	if persona.DatosMonotributo.CategoriaMonotributo.Descripcion != "Categoria A" {
		t.Errorf("expected Categoria='Categoria A', got %s", persona.DatosMonotributo.CategoriaMonotributo.Descripcion)
	}

	actividades := persona.Actividades()
	if len(actividades) != 1 || actividades[0] != 62020 {
		t.Errorf("expected Actividades=[62020], got %v", actividades)
	}
}

func TestGetPersona_Fisica(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getPersonaResponse xmlns="http://impl.service.ws_padron.afip.gov/">
      <personaReturn>
        <datosGenerales>
          <tipoPersona>FISICA</tipoPersona>
          <tipoClave>CUIL</tipoClave>
          <idPersona>20123456789</idPersona>
          <estadoClave>ACTIVO</estadoClave>
          <esSucesion>N</esSucesion>
          <apellido>PEREZ</apellido>
          <nombre>JUAN CARLOS</nombre>
          <domicilioFiscal>
            <direccion>CALLE FALSA 123</direccion>
            <localidad>ROSARIO</localidad>
            <idProvincia>13</idProvincia>
            <codPostal>2000</codPostal>
          </domicilioFiscal>
        </datosGenerales>
        <datosRegimenGeneral>
          <impuesto>
            <idImpuesto>30</idImpuesto>
            <estado>ACTIVO</estado>
          </impuesto>
          <impuesto>
            <idImpuesto>10</idImpuesto>
            <estado>ACTIVO</estado>
          </impuesto>
        </datosRegimenGeneral>
      </personaReturn>
    </getPersonaResponse>
  </soap:Body>
</soap:Envelope>`))
	}))
	defer server.Close()

	client := &Client{soap: soap.NewClient(), production: false}
	token := &wsaa.TokenAcceso{Token: "t", Sign: "s"}

	req := getPersonaRequest{
		XMLNS:            padronNS,
		Token:            token.Token,
		Sign:             token.Sign,
		CuitRepresentada: 20123456789,
		IDPersona:        20123456789,
	}
	var resp getPersonaResponse
	err := client.soap.Call(context.Background(), server.URL, actionGetPersona, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	persona := resp.Return.toPersona()
	if persona == nil {
		t.Fatal("expected persona")
	}

	if persona.TipoPersona != PersonaFisica {
		t.Errorf("expected TipoPersona=FISICA, got %s", persona.TipoPersona)
	}
	if persona.Denominacion() != "PEREZ, JUAN CARLOS" {
		t.Errorf("expected Denominacion='PEREZ, JUAN CARLOS', got %s", persona.Denominacion())
	}
	if persona.DomicilioFiscal.Provincia != "Santa Fe" {
		t.Errorf("expected Provincia='Santa Fe', got %s", persona.DomicilioFiscal.Provincia)
	}
	if !persona.HasImpuesto(ImpuestoIVA) {
		t.Error("expected HasImpuesto(IVA)=true")
	}
	if !persona.HasImpuesto(ImpuestoGanancias) {
		t.Error("expected HasImpuesto(Ganancias)=true")
	}
}

func TestPersona_ImpuestosActivos(t *testing.T) {
	pr := &personaReturn{
		DatosGenerales: &datosGenerales{
			TipoPersona: "JURIDICA",
			IDPersona:   30123456789,
		},
		DatosMonotributo: &datosMonotributoXML{
			Impuestos: []impuestoXML{
				{ID: 20, Estado: "ACTIVO"},
				{ID: 21, Estado: "BAJA"},
			},
		},
		DatosRegimenGeneral: &datosRegimenGeneralXML{
			Impuestos: []impuestoXML{
				{ID: 30, Estado: "ACTIVO"},
				{ID: 10, Estado: "ACTIVO"},
				{ID: 301, Estado: "BAJA"},
			},
		},
	}

	persona := pr.toPersona()
	activos := persona.ImpuestosActivos()

	expected := []int{20, 30, 10}
	if len(activos) != len(expected) {
		t.Fatalf("expected %d active taxes, got %d", len(expected), len(activos))
	}

	for _, exp := range expected {
		found := false
		for _, act := range activos {
			if act == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tax %d to be active", exp)
		}
	}
}

func TestPersona_Errores(t *testing.T) {
	pr := &personaReturn{
		DatosGenerales: &datosGenerales{
			TipoPersona: "FISICA",
			IDPersona:   20123456789,
		},
		ErrorConstancia:  []errorItem{{Error: "Error en constancia"}},
		ErrorMonotributo: []errorItem{{Error: "Error en monotributo"}},
	}

	persona := pr.toPersona()
	if len(persona.Errores) != 2 {
		t.Errorf("expected 2 errors, got %d", len(persona.Errores))
	}
	if persona.Errores[0] != "Error en constancia" {
		t.Errorf("expected first error 'Error en constancia', got %s", persona.Errores[0])
	}
}

func TestProvinciaNombre(t *testing.T) {
	tests := []struct {
		id       int
		expected string
	}{
		{1, "Buenos Aires"},
		{2, "CABA"},
		{13, "Santa Fe"},
		{24, "Tierra del Fuego"},
		{99, "Desconocida (99)"},
	}

	for _, tt := range tests {
		got := provinciaNombre(tt.id)
		if got != tt.expected {
			t.Errorf("provinciaNombre(%d) = %q, want %q", tt.id, got, tt.expected)
		}
	}
}
