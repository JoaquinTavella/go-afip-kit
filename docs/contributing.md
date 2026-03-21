# Guia para contribuir a go-afip-kit

## Como agregar un nuevo web service de AFIP

La libreria esta disenada para que agregar nuevos web services sea mecanico.
Sigue estos pasos para agregar, por ejemplo, WSFEX (facturas de exportacion).

### Paso 1: Crear el paquete

```
go-afip-kit/
└── wsfex/                     ← nuevo paquete
    ├── client.go              ← metodos publicos
    ├── types_request.go       ← structs XML de request
    ├── types_response.go      ← structs XML de response
    └── client_test.go         ← tests
```

### Paso 2: Definir types (request/response)

Seguir el patron de `wsfev1/types_request.go`:

```go
package wsfex

import "encoding/xml"

type FEXAuthorizeRequest struct {
    XMLName xml.Name `xml:"FEXAuthorize"`
    XMLNS   string   `xml:"xmlns,attr"`
    Auth    Auth     `xml:"ClsFEXAuthRequest"` // WSFEX usa nombres diferentes
    Cmp     CmpData  `xml:"Cmp"`
}
```

**Reglas para structs XML:**

1. Los nombres de campos XML deben coincidir EXACTAMENTE con el WSDL/manual de AFIP
2. Importes como `string` (no `float64`) para evitar perdida de precision
3. Fechas como `string` en formato YYYYMMDD
4. Usar `omitempty` solo para campos opcionales
5. El campo `XMLNS` debe usar el namespace del servicio

### Paso 3: Implementar el client

Seguir el patron de `wsfev1/client.go`:

```go
package wsfex

import (
    afipkit "github.com/lukcba-developers/go-afip-kit"
    "github.com/lukcba-developers/go-afip-kit/soap"
    "github.com/lukcba-developers/go-afip-kit/wsaa"
)

const (
    homoURL = "https://wswhomo.afip.gov.ar/wsfexv1/service.asmx"
    prodURL = "https://servicios1.afip.gov.ar/wsfexv1/service.asmx"
    ns      = "http://ar.gov.afip.dif.fexv1/"
)

type Client struct {
    soap       *soap.Client
    production bool
}

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
```

**Patron clave:** Cada metodo publico:
1. Construye el struct de request con Auth
2. Llama `c.soap.Call(ctx, url, soapAction, &req, &resp)`
3. Chequea errores AFIP con `errorsToAFIP(resp.Errors)`
4. Retorna el resultado tipado o error

### Paso 4: Manejar errores

Reutilizar `afipkit.AFIPError` y `afipkit.ErrSOAP`:

```go
func errorsToAFIP(ea *ErrorsArray) *afipkit.AFIPError {
    if ea == nil || len(ea.Err) == 0 {
        return nil
    }
    details := make([]afipkit.AFIPErrorDetail, len(ea.Err))
    for i, e := range ea.Err {
        details[i] = afipkit.AFIPErrorDetail{Code: e.Code, Msg: e.Msg}
    }
    return afipkit.NewAFIPError(details)
}
```

Si el nuevo servicio tiene codigos de error especificos, agregarlos a `errors.go`:

```go
// En afip.go o errors.go
const ErrCodeFEXInvalidDestination = 1234 // ejemplo
```

### Paso 5: Agregar constantes

Si el servicio usa codigos nuevos (tipos de comprobante, etc.), agregarlos a `afip.go`:

```go
// En afip.go
const CbteFacturaE = 19  // ya existe
```

### Paso 6: Tests

**Tests unitarios** (obligatorios, corren en CI):
- Mock del server SOAP con `httptest.NewServer`
- Verificar serializacion/deserializacion XML correcta
- Cubrir errores AFIP tipados

```go
func TestFEXAuthorize_Approved(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/xml")
        _, _ = w.Write([]byte(`<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FEXAuthorizeResponse>...</FEXAuthorizeResponse>
  </soap:Body>
</soap:Envelope>`))
    }))
    defer server.Close()
    // ...
}
```

**Tests de integracion** (opcionales, con build tag):

```go
//go:build integration

func TestFEXAuthorize_Integration(t *testing.T) {
    // Requiere certificado de homologacion en testdata/
    // Se ejecutan manualmente: go test -tags=integration ./wsfex/
}
```

## Agregar reglas fiscales

Nuevas reglas van en `fiscal/`:

1. **Tipos nuevos** → `fiscal/types.go` (si aplica)
2. **Reglas de determinacion** → `fiscal/rules.go`
3. **Validaciones** → `fiscal/validation.go`
4. **Tests** → archivo `_test.go` correspondiente

## Checklist para PR

- [ ] `go vet ./...` sin errores
- [ ] `go test ./... -count=1` todos pasan
- [ ] Sin dependencias nuevas (o justificacion fuerte si se agrega una)
- [ ] Funciones publicas tienen godoc comment
- [ ] Errores usan los tipos de `afipkit` (AFIPError, ErrSOAP, etc.)
- [ ] Importes como `string` en structs XML, nunca `float64`
- [ ] Tests cubren happy path + al menos 1 error path
- [ ] Sin estado global ni variables de paquete mutables

## Referencia de pyafipws

El gold standard para implementaciones AFIP es `reingart/pyafipws` (Python).
Cuando implementes un nuevo servicio, consulta el mapeo:

| pyafipws | go-afip-kit | Descripcion |
|---|---|---|
| `wsaa.py` | `wsaa/` | Autenticacion |
| `wsfev1.py` | `wsfev1/` | Facturacion nacional |
| `wsfexv1.py` | `wsfex/` (pendiente) | Facturacion exportacion |
| `wsmtxca.py` | `wsmtxca/` (pendiente) | Facturacion con detalle |

## Servicios pendientes de implementar

| Paquete | Servicio AFIP | Prioridad | Notas |
|---|---|---|---|
| `wsfex/` | WSFEXV1 | Media | Facturas de exportacion (tipo E) |
| `wsmtxca/` | WSMTXCA | Baja | Facturas con detalle de items |
| `wsfecred/` | WSFECred | Media | FCE MiPyME ciclo de vida |
| `wscpe/` | WSCPE | Alta | Carta de Porte Electronica |
| `padron/` | ws_sr_padron | Baja | Consulta de contribuyentes |
