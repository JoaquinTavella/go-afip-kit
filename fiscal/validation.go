package fiscal

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
)

// Importes represents the breakdown of amounts in a comprobante.
// All values should use 2 decimal places (AFIP requirement).
type Importes struct {
	ImpNeto      float64 // Neto gravado (taxable base)
	ImpIVA       float64 // Total IVA
	ImpTributos  float64 // Total tributos (IIBB, municipal, etc.)
	ImpNoGravado float64 // No gravado (non-taxable)
	ImpExento    float64 // Exento (exempt)
	ImpTotal     float64 // Total
}

// ValidarCuadratura checks that ImpTotal equals the sum of all parts.
// AFIP rejects comprobantes where amounts don't add up (error 10048).
//
// Uses integer arithmetic (centavos) to avoid floating-point precision issues.
func ValidarCuadratura(imp Importes) error {
	// Convert to centavos (integer) to avoid float precision issues
	total := toCentavos(imp.ImpTotal)
	expected := toCentavos(imp.ImpNeto) + toCentavos(imp.ImpIVA) +
		toCentavos(imp.ImpTributos) + toCentavos(imp.ImpNoGravado) + toCentavos(imp.ImpExento)

	if total != expected {
		return fmt.Errorf("fiscal: importes no cuadran: total=%.2f, expected=%.2f (neto=%.2f + iva=%.2f + trib=%.2f + nograv=%.2f + exento=%.2f)",
			imp.ImpTotal, fromCentavos(expected), imp.ImpNeto, imp.ImpIVA, imp.ImpTributos, imp.ImpNoGravado, imp.ImpExento)
	}
	return nil
}

// toCentavos converts a float amount to integer centavos, rounding to nearest.
func toCentavos(f float64) int64 {
	return int64(math.Round(f * 100))
}

// fromCentavos converts integer centavos back to float for display.
func fromCentavos(c int64) float64 {
	return float64(c) / 100
}

// AlicuotaIVA represents an IVA rate entry for validation.
type AlicuotaIVA struct {
	ID      int     // AFIP code (3=0%, 4=10.5%, 5=21%, 6=27%, 8=5%, 9=2.5%)
	BaseImp float64 // Taxable base for this rate
	Importe float64 // Calculated IVA amount
}

// ValidarAlicuotasIVA verifies that the IVA breakdown matches ImpIVA and ImpNeto.
//
// Checks:
//  1. Sum of BaseImp == ImpNeto
//  2. Sum of Importe == ImpIVA
func ValidarAlicuotasIVA(alicuotas []AlicuotaIVA, impNeto, impIVA float64) error {
	if len(alicuotas) == 0 && toCentavos(impIVA) > 0 {
		return fmt.Errorf("fiscal: ImpIVA=%.2f pero no hay alicuotas", impIVA)
	}

	var sumBase, sumImporte int64
	for _, a := range alicuotas {
		sumBase += toCentavos(a.BaseImp)
		sumImporte += toCentavos(a.Importe)
	}

	if sumBase != toCentavos(impNeto) {
		return fmt.Errorf("fiscal: suma BaseImp alicuotas (%.2f) != ImpNeto (%.2f)", fromCentavos(sumBase), impNeto)
	}
	if sumImporte != toCentavos(impIVA) {
		return fmt.Errorf("fiscal: suma Importe alicuotas (%.2f) != ImpIVA (%.2f)", fromCentavos(sumImporte), impIVA)
	}

	return nil
}

// ivaRates maps AFIP alicuota IDs to their percentage * 100 (integer math).
// E.g., 21% = 2100, 10.5% = 1050.
var ivaRates = map[int]int64{
	3: 0,    // 0%
	4: 1050, // 10.5%
	5: 2100, // 21%
	6: 2700, // 27%
	8: 500,  // 5%
	9: 250,  // 2.5%
}

// CalcularIVA computes the IVA amount for a given base and AFIP alicuota code.
// Returns the amount rounded to 2 decimals.
// Returns an error if the alicuota ID is not a valid AFIP code.
func CalcularIVA(baseImp float64, alicuotaID int) (float64, error) {
	rate, ok := ivaRates[alicuotaID]
	if !ok {
		return 0, fmt.Errorf("fiscal: alicuota IVA desconocida: %d", alicuotaID)
	}
	if rate == 0 {
		return 0, nil
	}
	// Use integer math: (base * 100) * rate / 10000 / 100
	baseCentavos := toCentavos(baseImp)
	ivaCentavos := (baseCentavos * rate) / 10000
	return fromCentavos(ivaCentavos), nil
}

// ValidarConcepto checks that required date fields are present for servicios (concepto 2, 3).
// For concepto 1 (productos), service dates are not required.
func ValidarConcepto(concepto int, fchServDesde, fchServHasta, fchVtoPago string) error {
	if concepto != 2 && concepto != 3 {
		return nil
	}

	if fchServDesde == "" {
		return fmt.Errorf("fiscal: FchServDesde obligatorio para concepto %d (servicios)", concepto)
	}
	if fchServHasta == "" {
		return fmt.Errorf("fiscal: FchServHasta obligatorio para concepto %d (servicios)", concepto)
	}
	if fchVtoPago == "" {
		return fmt.Errorf("fiscal: FchVtoPago obligatorio para concepto %d (servicios)", concepto)
	}

	// Validate date formats
	for _, d := range []struct{ name, val string }{
		{"FchServDesde", fchServDesde},
		{"FchServHasta", fchServHasta},
		{"FchVtoPago", fchVtoPago},
	} {
		if err := ValidarFechaAFIP(d.val); err != nil {
			return fmt.Errorf("fiscal: %s: %w", d.name, err)
		}
	}

	return nil
}

// dateRegex matches YYYYMMDD format (8 digits).
var dateRegex = regexp.MustCompile(`^\d{8}$`)

// ValidarFechaAFIP validates that a string is a valid AFIP date (YYYYMMDD format).
func ValidarFechaAFIP(fecha string) error {
	if !dateRegex.MatchString(fecha) {
		return fmt.Errorf("formato invalido %q, esperado YYYYMMDD (8 digitos)", fecha)
	}

	year, _ := strconv.Atoi(fecha[:4])
	month, _ := strconv.Atoi(fecha[4:6])
	day, _ := strconv.Atoi(fecha[6:8])

	if month < 1 || month > 12 {
		return fmt.Errorf("mes invalido %d en fecha %q", month, fecha)
	}
	if day < 1 || day > 31 {
		return fmt.Errorf("dia invalido %d en fecha %q", day, fecha)
	}

	// Verify it's a real date
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day || t.Month() != time.Month(month) || t.Year() != year {
		return fmt.Errorf("fecha invalida %q (dia no existe)", fecha)
	}

	return nil
}

// FormatFechaAFIP formats a time.Time to AFIP's YYYYMMDD string format.
func FormatFechaAFIP(t time.Time) string {
	return t.Format("20060102")
}

// ParseFechaAFIP parses an AFIP YYYYMMDD date string to time.Time.
func ParseFechaAFIP(fecha string) (time.Time, error) {
	if err := ValidarFechaAFIP(fecha); err != nil {
		return time.Time{}, err
	}
	return time.Parse("20060102", fecha)
}
