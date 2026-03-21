# Code Review: Hallazgos y correcciones aplicadas

> Fecha: 2026-03-21
> Scope: Review completa de los 18 archivos fuente de go-afip-kit

## Bugs corregidos

### 1. Data race en MemoryStore (CRITICO)

**Archivo:** `wsaa/token_store.go`
**Problema:** El mapa `tokens` se accedia sin sincronizacion. Multiples goroutines llamando Get/Set/Delete causarian data race.
**Fix:** Se agrego `sync.RWMutex`. `Get` usa `RLock`, `Set`/`Delete` usan `Lock`.

### 2. CUIT substring match en lugar de exact match (CRITICO)

**Archivo:** `cert/validator.go`
**Problema:** `strings.Contains()` permitia que "123456" matcheara contra "20123456789". Un CUIT parcial pasaria la validacion.
**Fix:** Se reemplazo `normalizeCUIT()` por `extractCUIT()` que:
- Extrae exactamente 11 digitos
- Hace comparacion exacta (no substring)
- Valida que el resultado sean 11 digitos con regex

### 3. ParseCondicionIVA retornaba zero-value silenciosamente (CRITICO)

**Archivo:** `fiscal/types.go`
**Problema:** Para inputs invalidos retornaba `CondicionIVA(0)` sin error. Los callers no podian distinguir entre "desconocido" y "no seteado".
**Fix:**
- Se definio `CondicionDesconocida = 0` como sentinel explicito
- `ParseCondicionIVA` ahora retorna `(CondicionIVA, error)`
- Se agrego `IsValid()` para verificar valores

### 4. Precision float en calculos monetarios (CRITICO)

**Archivo:** `fiscal/validation.go`
**Problema:** `ValidarCuadratura` usaba aritmetica float64 con tolerancia magica de 0.01. `CalcularIVA` tenia formula de redondeo fragil.
**Fix:**
- `ValidarCuadratura`: convierte a **centavos (int64)** internamente. Comparacion exacta, sin tolerancia.
- `CalcularIVA`: usa rates en milesimasx100 (`21% = 2100`) y aritmetica de centavos.
- `CalcularIVA` ahora retorna `(float64, error)` para alicuotas desconocidas.

### 5. IVAAlicuotaPorcentaje retornaba 0 silenciosamente

**Archivo:** `afip.go`
**Problema:** Para IDs desconocidos retornaba 0 sin error.
**Fix:** Ahora retorna `(float64, error)`.

## Mejoras de robustez

### 6. SOAP Fault detection

**Archivo:** `soap/client.go`
**Problema:** Si AFIP retornaba un SOAP Fault (status 200 con `<Fault>`), el codigo intentaba unmarshalearlo como respuesta normal y fallaba con error generico.
**Fix:** Se agrego `extractFault()` que detecta `<Fault>` en la respuesta y retorna un `*soap.Fault` tipado. Se chequea tanto en HTTP 200 como en HTTP 500.

### 7. UniqueID del TRA podia colisionar

**Archivo:** `wsaa/client.go`
**Problema:** `now.Unix()` genera el mismo ID si dos requests se hacen en el mismo segundo.
**Fix:** Cambiado a `now.UnixNano()` para precision de nanosegundos.

### 8. Errores retryable vs permanente

**Archivo:** `errors.go`
**Problema:** No habia distincion entre errores que se pueden reintentar y errores permanentes.
**Fix:** Se agrego `IsRetryable()` a `AFIPError`. Los errores de config/calculo (10005, 10000, 10048, 10015) son permanentes. Los demas (10016, 10013) son retryables.

### 9. Validacion de CUIT (mod-11)

**Archivo:** `afip.go`
**Problema:** No existia validacion de formato/checksum de CUIT.
**Fix:** Se agrego `ValidarCUIT()` que verifica 11 digitos + digito verificador con algoritmo modulo 11.

### 10. Validacion de fechas AFIP (YYYYMMDD)

**Archivo:** `fiscal/validation.go`
**Problema:** Los campos de fecha se pasaban como string sin validar formato ni existencia de la fecha.
**Fix:** Se agrego `ValidarFechaAFIP()`, `ParseFechaAFIP()`, `FormatFechaAFIP()`. `ValidarConcepto()` ahora valida el formato de las fechas ademas de su presencia.

## Resumen de impacto

| # | Severidad | Tipo | Archivos afectados |
|---|-----------|------|-------------------|
| 1 | Critico | Bug (data race) | `wsaa/token_store.go` |
| 2 | Critico | Bug (logica) | `cert/validator.go` |
| 3 | Critico | Bug (API) | `fiscal/types.go` |
| 4 | Critico | Bug (precision) | `fiscal/validation.go` |
| 5 | Alto | Bug (API) | `afip.go` |
| 6 | Alto | Mejora | `soap/client.go` |
| 7 | Medio | Mejora | `wsaa/client.go` |
| 8 | Alto | Feature | `errors.go` |
| 9 | Alto | Feature | `afip.go` |
| 10 | Alto | Feature | `fiscal/validation.go` |

Todos los cambios mantienen backward compatibility excepto:
- `ParseCondicionIVA` ahora retorna 2 valores
- `CalcularIVA` ahora retorna 2 valores
- `IVAAlicuotaPorcentaje` ahora retorna 2 valores
- `ValidarCuadratura` ahora es estricta (sin tolerancia de 0.01)
