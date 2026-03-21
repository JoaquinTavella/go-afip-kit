// Package fiscal implements Argentine fiscal rules for electronic invoicing:
// comprobante type determination (A/B/C), amount validation, and IVA calculations.
package fiscal

import "fmt"

// CondicionIVA represents the IVA registration status of a taxpayer.
type CondicionIVA int

const (
	// CondicionDesconocida is the zero value — represents an unknown or unset condition.
	CondicionDesconocida CondicionIVA = 0

	ResponsableInscripto CondicionIVA = iota
	Monotributista
	Exento
	ConsumidorFinal
	NoResponsable
)

// String returns the human-readable name.
func (c CondicionIVA) String() string {
	names := map[CondicionIVA]string{
		ResponsableInscripto: "Responsable Inscripto",
		Monotributista:       "Monotributista",
		Exento:               "Exento",
		ConsumidorFinal:      "Consumidor Final",
		NoResponsable:        "No Responsable",
	}
	if n, ok := names[c]; ok {
		return n
	}
	return "Desconocido"
}

// IsValid returns true if the condition is a known, valid value.
func (c CondicionIVA) IsValid() bool {
	switch c {
	case ResponsableInscripto, Monotributista, Exento, ConsumidorFinal, NoResponsable:
		return true
	default:
		return false
	}
}

// Slug returns the DB-friendly value.
func (c CondicionIVA) Slug() string {
	slugs := map[CondicionIVA]string{
		ResponsableInscripto: "responsable_inscripto",
		Monotributista:       "monotributista",
		Exento:               "exento",
		ConsumidorFinal:      "consumidor_final",
		NoResponsable:        "no_responsable",
	}
	return slugs[c]
}

// ParseCondicionIVA converts a string slug to CondicionIVA.
// Returns CondicionDesconocida and an error for unknown values.
func ParseCondicionIVA(s string) (CondicionIVA, error) {
	m := map[string]CondicionIVA{
		"responsable_inscripto": ResponsableInscripto,
		"monotributista":        Monotributista,
		"exento":                Exento,
		"consumidor_final":      ConsumidorFinal,
		"no_responsable":        NoResponsable,
	}
	if c, ok := m[s]; ok {
		return c, nil
	}
	return CondicionDesconocida, fmt.Errorf("fiscal: condicion IVA desconocida: %q", s)
}
