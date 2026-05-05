# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Go library for Argentina's AFIP (ARCA) electronic invoicing web services. Covers WSAA authentication, WSFEv1 invoicing, WS-SR-PADRON taxpayer lookup, certificate management, and fiscal rules. Pure library — no framework, no database, no CLI.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./soap/
go test ./wsaa/
go test ./wsfev1/
go test ./wspadron/
go test ./fiscal/
go test ./cert/

# Run a single test
go test ./fiscal/ -run TestValidarCuadratura

# Build (verify compilation)
go build ./...

# Vet
go vet ./...
```

## Architecture

The module is `github.com/lukcba-developers/go-afip-kit`. Only external dependency is `go.mozilla.org/pkcs7` (for CMS signing).

**Package layering (bottom-up):**

1. **`soap/`** — Minimal SOAP 1.1 HTTP client. Handles envelope wrapping/unwrapping, fault detection. No WSDL — callers provide typed request/response structs via `xml` tags. All AFIP communication goes through `soap.Client.Call()`.

2. **`wsaa/`** — WSAA authentication (LoginCms). Signs a TRA (Ticket de Requerimiento de Acceso) with PKCS#7/CMS, sends it to WSAA, gets back a `TokenAcceso` (token + sign + expiration). `CachedClient` wraps `Client` with a `TokenStore` interface for automatic token caching/renewal. `MemoryStore` is the built-in in-memory implementation; users implement `TokenStore` for Redis etc.

3. **`wsfev1/`** — WSFEv1 electronic invoicing client. `SolicitarCAE` emits comprobantes, `SolicitarCAEConReproceso` adds idempotent retry (handles error 10016 by recovering the CAE via `ConsultarComprobante`). Also exposes parameter queries (tipos de comprobante, IVA, monedas, puntos de venta, cotizacion).

4. **`wspadron/`** — WS-SR-PADRON A5 taxpayer lookup client. `GetPersona` queries taxpayer registration data by CUIT (name, address, tax status, activities, monotributo category). `Persona` struct has helpers like `Denominacion()`, `ImpuestosActivos()`, `HasImpuesto()`. Uses same auth pattern as WSFEv1 (Token/Sign from WSAA).

5. **`cert/`** — Certificate utilities: PEM parsing (PKCS#1 and PKCS#8), validation (expiry, CUIT extraction, TLS pairing), and `KeyVault` for AES-256-GCM encryption of private keys at rest.

6. **`fiscal/`** — Argentine fiscal rules: `DeterminarTipoComprobante` (RI→RI=A, RI→CF=B, Mono/Exento→C), amount validation (`ValidarCuadratura` uses integer centavo arithmetic to avoid float issues), IVA calculation, AFIP date format helpers (YYYYMMDD).

7. **Root package (`afipkit`)** — Constants (comprobante types, document types, IVA codes, currencies, environments) and shared error types (`AFIPError` with well-known codes like 10016, `ErrSOAP`, `ErrWSAAAuth`, `ErrCertificate`).

**Key patterns:**
- All SOAP types use `encoding/xml` struct tags matching AFIP's WSDL exactly
- Monetary amounts in WSFEv1 requests are `string` (not float) for XML precision
- Errors from AFIP are parsed into `AFIPError` with code-checking helpers (`IsAlreadyRegistered`, `IsInvalidPtoVta`, etc.)
- Tests use `httptest.Server` with canned XML responses — no live AFIP calls
- Functional options pattern (`soap.WithTimeout`, `wsaa.WithRenewalMargin`)
- All public API methods take `context.Context` as first parameter
- Environment switching: `production bool` flag on clients selects homo vs prod URLs

## Domain Notes

- AFIP dates are YYYYMMDD strings, not `time.Time`
- Comprobante = invoice/receipt. CAE = authorization code returned by AFIP
- Error 10016 = comprobante already registered (idempotent — recover CAE with `FECompConsultar`)
- `CondicionIVA` determines invoice letter: RI→RI = A, RI→consumer = B, Monotributo = C
- Homologacion = AFIP's testing/sandbox environment
