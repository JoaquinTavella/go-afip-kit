package afipkit

import "testing"

func TestCbteDescripcion(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{CbteFacturaA, "Factura A"},
		{CbteFacturaB, "Factura B"},
		{CbteFacturaC, "Factura C"},
		{CbteNotaCreditoA, "Nota de Credito A"},
		{CbteFCEFacturaA, "FCE Factura A"},
		{999, "Desconocido"},
	}
	for _, tt := range tests {
		got := CbteDescripcion(tt.code)
		if got != tt.want {
			t.Errorf("CbteDescripcion(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestCbteLetra(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{CbteFacturaA, "A"},
		{CbteNotaCreditoA, "A"},
		{CbteFacturaB, "B"},
		{CbteFacturaC, "C"},
		{CbteFacturaE, "E"},
		{CbteFacturaM, "M"},
		{CbteFCEFacturaA, "A"},
		{999, ""},
	}
	for _, tt := range tests {
		got := CbteLetra(tt.code)
		if got != tt.want {
			t.Errorf("CbteLetra(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestNotaCreditoPara(t *testing.T) {
	if got := NotaCreditoPara(CbteFacturaA); got != CbteNotaCreditoA {
		t.Errorf("got %d, want %d", got, CbteNotaCreditoA)
	}
	if got := NotaCreditoPara(CbteFacturaB); got != CbteNotaCreditoB {
		t.Errorf("got %d, want %d", got, CbteNotaCreditoB)
	}
	if got := NotaCreditoPara(999); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestIVAAlicuotaPorcentaje(t *testing.T) {
	got, err := IVAAlicuotaPorcentaje(IVA21)
	if err != nil {
		t.Fatal(err)
	}
	if got != 21 {
		t.Errorf("got %f, want 21", got)
	}

	got, err = IVAAlicuotaPorcentaje(IVA105)
	if err != nil {
		t.Fatal(err)
	}
	if got != 10.5 {
		t.Errorf("got %f, want 10.5", got)
	}

	_, err = IVAAlicuotaPorcentaje(999)
	if err == nil {
		t.Fatal("expected error for unknown alicuota")
	}
}

func TestValidarCUIT(t *testing.T) {
	// Valid CUITs (verified with mod-11 algorithm)
	validCUITs := []int64{
		30710685831, // empresa
		27249999542, // persona fisica
	}
	for _, cuit := range validCUITs {
		if err := ValidarCUIT(cuit); err != nil {
			t.Errorf("ValidarCUIT(%d) unexpected error: %v", cuit, err)
		}
	}

	// Invalid: too short
	if err := ValidarCUIT(123); err == nil {
		t.Error("expected error for short CUIT")
	}

	// Invalid: too long
	if err := ValidarCUIT(201234567890); err == nil {
		t.Error("expected error for long CUIT")
	}
}
