package fiscal

import (
	"testing"
)

func TestValidarCuadratura_OK(t *testing.T) {
	imp := Importes{
		ImpNeto:      100000.00,
		ImpIVA:       21000.00,
		ImpTributos:  0,
		ImpNoGravado: 0,
		ImpExento:    0,
		ImpTotal:     121000.00,
	}
	if err := ValidarCuadratura(imp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidarCuadratura_WithTributos(t *testing.T) {
	imp := Importes{
		ImpNeto:      100000.00,
		ImpIVA:       21000.00,
		ImpTributos:  3500.00,
		ImpNoGravado: 0,
		ImpExento:    0,
		ImpTotal:     124500.00,
	}
	if err := ValidarCuadratura(imp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidarCuadratura_Mismatch(t *testing.T) {
	imp := Importes{
		ImpNeto:  100000.00,
		ImpIVA:   21000.00,
		ImpTotal: 999999.00, // wrong
	}
	if err := ValidarCuadratura(imp); err == nil {
		t.Fatal("expected error for mismatched amounts")
	}
}

func TestValidarCuadratura_PreciseMatch(t *testing.T) {
	// This tests that integer-based comparison works correctly
	// and doesn't have float tolerance issues
	imp := Importes{
		ImpNeto:  100000.00,
		ImpIVA:   21000.00,
		ImpTotal: 121000.01, // off by 1 centavo — should FAIL with integer math
	}
	if err := ValidarCuadratura(imp); err == nil {
		t.Fatal("expected error for 1 centavo mismatch")
	}
}

func TestValidarAlicuotasIVA_OK(t *testing.T) {
	alicuotas := []AlicuotaIVA{
		{ID: 5, BaseImp: 100000.00, Importe: 21000.00},
	}
	err := ValidarAlicuotasIVA(alicuotas, 100000.00, 21000.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidarAlicuotasIVA_MultipleRates(t *testing.T) {
	alicuotas := []AlicuotaIVA{
		{ID: 5, BaseImp: 80000.00, Importe: 16800.00},
		{ID: 4, BaseImp: 20000.00, Importe: 2100.00},
	}
	err := ValidarAlicuotasIVA(alicuotas, 100000.00, 18900.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidarAlicuotasIVA_BaseMismatch(t *testing.T) {
	alicuotas := []AlicuotaIVA{
		{ID: 5, BaseImp: 50000.00, Importe: 10500.00},
	}
	err := ValidarAlicuotasIVA(alicuotas, 100000.00, 10500.00)
	if err == nil {
		t.Fatal("expected error for base mismatch")
	}
}

func TestValidarAlicuotasIVA_NoAlicuotasWithIVA(t *testing.T) {
	err := ValidarAlicuotasIVA(nil, 100000.00, 21000.00)
	if err == nil {
		t.Fatal("expected error when ImpIVA > 0 but no alicuotas")
	}
}

func TestCalcularIVA(t *testing.T) {
	tests := []struct {
		base     float64
		alicuota int
		expected float64
		wantErr  bool
	}{
		{100000, 5, 21000, false},  // 21%
		{100000, 4, 10500, false},  // 10.5%
		{100000, 6, 27000, false},  // 27%
		{100000, 8, 5000, false},   // 5%
		{100000, 9, 2500, false},   // 2.5%
		{100000, 3, 0, false},      // 0%
		{100000, 99, 0, true},      // unknown — error
	}
	for _, tt := range tests {
		got, err := CalcularIVA(tt.base, tt.alicuota)
		if (err != nil) != tt.wantErr {
			t.Errorf("CalcularIVA(%.0f, %d): err=%v, wantErr=%v", tt.base, tt.alicuota, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("CalcularIVA(%.0f, %d) = %.2f, want %.2f", tt.base, tt.alicuota, got, tt.expected)
		}
	}
}

func TestValidarConcepto_Servicios(t *testing.T) {
	err := ValidarConcepto(2, "", "", "")
	if err == nil {
		t.Fatal("expected error for concepto 2 without dates")
	}

	err = ValidarConcepto(2, "20260301", "20260331", "20260418")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidarConcepto_Productos(t *testing.T) {
	err := ValidarConcepto(1, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error for concepto 1: %v", err)
	}
}

func TestValidarConcepto_MissingPartial(t *testing.T) {
	err := ValidarConcepto(2, "20260301", "20260331", "")
	if err == nil {
		t.Fatal("expected error for missing FchVtoPago")
	}
}

func TestValidarConcepto_InvalidDate(t *testing.T) {
	err := ValidarConcepto(2, "20260301", "20260332", "20260418") // day 32
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestValidarFechaAFIP(t *testing.T) {
	// Valid
	for _, d := range []string{"20260101", "20261231", "20260228"} {
		if err := ValidarFechaAFIP(d); err != nil {
			t.Errorf("ValidarFechaAFIP(%q) unexpected error: %v", d, err)
		}
	}

	// Invalid format
	for _, d := range []string{"2026-01-01", "abcdefgh", "2026011", "202601011"} {
		if err := ValidarFechaAFIP(d); err == nil {
			t.Errorf("ValidarFechaAFIP(%q) expected error", d)
		}
	}

	// Invalid date (Feb 30)
	if err := ValidarFechaAFIP("20260230"); err == nil {
		t.Error("expected error for Feb 30")
	}
}

func TestFormatFechaAFIP(t *testing.T) {
	d, _ := ParseFechaAFIP("20260318")
	got := FormatFechaAFIP(d)
	if got != "20260318" {
		t.Errorf("got %q, want 20260318", got)
	}
}

func TestParseFechaAFIP(t *testing.T) {
	d, err := ParseFechaAFIP("20260318")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2026 || d.Month() != 3 || d.Day() != 18 {
		t.Errorf("got %v", d)
	}

	_, err = ParseFechaAFIP("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}
