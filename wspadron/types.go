// Package wspadron implements the AFIP WS-SR-PADRON (Web Service de Padrón de Contribuyentes)
// client for querying taxpayer registration data (Alcance 5 - Constancia de Inscripción).
package wspadron

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const (
	padronNS = "http://impl.service.ws_padron.afip.gov/"
)

// PersonaType represents the type of taxpayer.
type PersonaType string

const (
	PersonaFisica   PersonaType = "FISICA"
	PersonaJuridica PersonaType = "JURIDICA"
)

// EstadoClave represents the taxpayer registration status.
type EstadoClave string

const (
	EstadoActivo   EstadoClave = "ACTIVO"
	EstadoAnulado  EstadoClave = "ANULADO"
	EstadoPendiente EstadoClave = "PENDIENTE"
)

// DomicilioFiscal represents the fiscal address of a taxpayer.
type DomicilioFiscal struct {
	Direccion    string `json:"direccion"`
	Localidad    string `json:"localidad"`
	IDProvincia  int    `json:"id_provincia"`
	Provincia    string `json:"provincia"`
	CodPostal    string `json:"cod_postal"`
}

// Impuesto represents a tax registration for the taxpayer.
type Impuesto struct {
	ID     int    `json:"id"`
	Estado string `json:"estado"`
}

// Actividad represents a business activity code.
type Actividad struct {
	ID int `json:"id"`
}

// CategoriaMonotributo represents the monotributo category.
type CategoriaMonotributo struct {
	ID          int    `json:"id"`
	Descripcion string `json:"descripcion"`
}

// DatosMonotributo holds monotributo-specific data.
type DatosMonotributo struct {
	Impuestos           []Impuesto           `json:"impuestos"`
	Actividades         []Actividad          `json:"actividades"`
	CategoriaMonotributo *CategoriaMonotributo `json:"categoria_monotributo"`
}

// DatosRegimenGeneral holds general regime data.
type DatosRegimenGeneral struct {
	Impuestos   []Impuesto  `json:"impuestos"`
	Actividades []Actividad `json:"actividades"`
}

// Persona represents the complete taxpayer record returned by WS-SR-PADRON A5.
type Persona struct {
	// General data
	TipoPersona   PersonaType  `json:"tipo_persona"`
	TipoDoc       string       `json:"tipo_doc"`
	IDPersona     int64        `json:"id_persona"`
	EstadoClave   EstadoClave  `json:"estado_clave"`
	EsSucesion    string       `json:"es_sucesion"`
	RazonSocial   string       `json:"razon_social"`
	Apellido      string       `json:"apellido"`
	Nombre        string       `json:"nombre"`
	DomicilioFiscal DomicilioFiscal `json:"domicilio_fiscal"`

	// Tax data
	DatosMonotributo    DatosMonotributo    `json:"datos_monotributo"`
	DatosRegimenGeneral DatosRegimenGeneral `json:"datos_regimen_general"`

	// Errors from AFIP
	Errores []string `json:"errores"`
}

// Denominacion returns the full name of the taxpayer.
// For juridical persons, returns RazonSocial.
// For physical persons, returns "Apellido, Nombre".
func (p *Persona) Denominacion() string {
	if p.RazonSocial != "" {
		return p.RazonSocial
	}
	parts := []string{}
	if p.Apellido != "" {
		parts = append(parts, p.Apellido)
	}
	if p.Nombre != "" {
		parts = append(parts, p.Nombre)
	}
	return strings.Join(parts, ", ")
}

// ImpuestosActivos returns all active tax IDs.
func (p *Persona) ImpuestosActivos() []int {
	var ids []int
	for _, imp := range p.DatosMonotributo.Impuestos {
		if imp.Estado == "ACTIVO" {
			ids = append(ids, imp.ID)
		}
	}
	for _, imp := range p.DatosRegimenGeneral.Impuestos {
		if imp.Estado == "ACTIVO" {
			ids = append(ids, imp.ID)
		}
	}
	return ids
}

// Actividades returns all activity IDs (from both regimes).
func (p *Persona) Actividades() []int {
	var ids []int
	for _, act := range p.DatosMonotributo.Actividades {
		ids = append(ids, act.ID)
	}
	for _, act := range p.DatosRegimenGeneral.Actividades {
		ids = append(ids, act.ID)
	}
	return ids
}

// HasImpuesto checks if the taxpayer has a specific active tax.
func (p *Persona) HasImpuesto(id int) bool {
	for _, impID := range p.ImpuestosActivos() {
		if impID == id {
			return true
		}
	}
	return false
}

// --- SOAP request/response types ---

type getPersonaRequest struct {
	XMLName         xml.Name `xml:"getPersona"`
	XMLNS           string   `xml:"xmlns,attr"`
	Token           string   `xml:"token"`
	Sign            string   `xml:"sign"`
	CuitRepresentada int64   `xml:"cuitRepresentada"`
	IDPersona       int64    `xml:"idPersona"`
}

type getPersonaResponse struct {
	XMLName xml.Name `xml:"getPersonaResponse"`
	Return  *personaReturn `xml:"personaReturn"`
}

type personaReturn struct {
	DatosGenerales   *datosGenerales   `xml:"datosGenerales"`
	DatosMonotributo *datosMonotributoXML `xml:"datosMonotributo"`
	DatosRegimenGeneral *datosRegimenGeneralXML `xml:"datosRegimenGeneral"`
	ErrorConstancia  []errorItem `xml:"errorConstancia"`
	ErrorMonotributo []errorItem `xml:"errorMonotributo"`
	ErrorRegimenGeneral []errorItem `xml:"errorRegimenGeneral"`
}

type datosGenerales struct {
	TipoPersona   string `xml:"tipoPersona"`
	TipoClave     string `xml:"tipoClave"`
	IDPersona     int64  `xml:"idPersona"`
	EstadoClave   string `xml:"estadoClave"`
	EsSucesion    string `xml:"esSucesion"`
	RazonSocial   string `xml:"razonSocial"`
	Apellido      string `xml:"apellido"`
	Nombre        string `xml:"nombre"`
	DomicilioFiscal *domicilioFiscalXML `xml:"domicilioFiscal"`
}

type domicilioFiscalXML struct {
	Direccion   string `xml:"direccion"`
	Localidad   string `xml:"localidad"`
	IDProvincia int    `xml:"idProvincia"`
	CodPostal   string `xml:"codPostal"`
}

type datosMonotributoXML struct {
	Impuestos           []impuestoXML           `xml:"impuesto"`
	Actividades         []actividadXML          `xml:"actividadMonotributista"`
	CategoriaMonotributo *categoriaMonotributoXML `xml:"categoriaMonotributo"`
}

type datosRegimenGeneralXML struct {
	Impuestos   []impuestoXML  `xml:"impuesto"`
	Actividades []actividadXML `xml:"actividad"`
}

type impuestoXML struct {
	ID     int    `xml:"idImpuesto"`
	Estado string `xml:"estado"`
}

type actividadXML struct {
	ID int `xml:"idActividad"`
}

type categoriaMonotributoXML struct {
	ID          int    `xml:"idCategoria"`
	Descripcion string `xml:"descripcionCategoria"`
}

type errorItem struct {
	Error string `xml:"error"`
}

// Known AFIP tax IDs (impuestos)
const (
	ImpuestoIVA           = 30  // IVA
	ImpuestoGanancias     = 10  // Ganancias
	ImpuestoExento        = 32  // Exento
	ImpuestoNoInscripto   = 33  // No Inscripto
	ImpuestoNoAlcanzado   = 34  // No Alcanzado
	ImpuestoEmpleador     = 301 // Empleador
	ImpuestoMonotributo   = 20  // Monotributo
	ImpuestoMonotributoSocial = 21 // Monotributo Social
)

// Known provinces (ID -> name)
var Provincias = map[int]string{
	1:  "Buenos Aires",
	2:  "CABA",
	3:  "Catamarca",
	4:  "Córdoba",
	5:  "Corrientes",
	6:  "Entre Ríos",
	7:  "Jujuy",
	8:  "Mendoza",
	9:  "La Rioja",
	10: "Salta",
	11: "San Juan",
	12: "San Luis",
	13: "Santa Fe",
	14: "Santiago del Estero",
	15: "Tucumán",
	16: "Chaco",
	17: "Chubut",
	18: "Formosa",
	19: "Misiones",
	20: "Neuquén",
	21: "La Pampa",
	22: "Río Negro",
	23: "Santa Cruz",
	24: "Tierra del Fuego",
}

// provinciaNombre returns the province name for a given ID.
func provinciaNombre(id int) string {
	if name, ok := Provincias[id]; ok {
		return name
	}
	return fmt.Sprintf("Desconocida (%d)", id)
}

// toPersona converts the raw XML response to a Persona struct.
func (pr *personaReturn) toPersona() *Persona {
	if pr == nil || pr.DatosGenerales == nil {
		return nil
	}

	dg := pr.DatosGenerales
	p := &Persona{
		TipoPersona: PersonaType(dg.TipoPersona),
		TipoDoc:     dg.TipoClave,
		IDPersona:   dg.IDPersona,
		EstadoClave: EstadoClave(dg.EstadoClave),
		EsSucesion:  dg.EsSucesion,
		RazonSocial: dg.RazonSocial,
		Apellido:    dg.Apellido,
		Nombre:      dg.Nombre,
	}

	if dg.DomicilioFiscal != nil {
		p.DomicilioFiscal = DomicilioFiscal{
			Direccion:   dg.DomicilioFiscal.Direccion,
			Localidad:   dg.DomicilioFiscal.Localidad,
			IDProvincia: dg.DomicilioFiscal.IDProvincia,
			Provincia:   provinciaNombre(dg.DomicilioFiscal.IDProvincia),
			CodPostal:   dg.DomicilioFiscal.CodPostal,
		}
	}

	// Monotributo data
	if pr.DatosMonotributo != nil {
		p.DatosMonotributo = DatosMonotributo{}
		for _, imp := range pr.DatosMonotributo.Impuestos {
			p.DatosMonotributo.Impuestos = append(p.DatosMonotributo.Impuestos, Impuesto{
				ID:     imp.ID,
				Estado: imp.Estado,
			})
		}
		for _, act := range pr.DatosMonotributo.Actividades {
			p.DatosMonotributo.Actividades = append(p.DatosMonotributo.Actividades, Actividad{ID: act.ID})
		}
		if pr.DatosMonotributo.CategoriaMonotributo != nil {
			p.DatosMonotributo.CategoriaMonotributo = &CategoriaMonotributo{
				ID:          pr.DatosMonotributo.CategoriaMonotributo.ID,
				Descripcion: pr.DatosMonotributo.CategoriaMonotributo.Descripcion,
			}
		}
	}

	// Regimen General data
	if pr.DatosRegimenGeneral != nil {
		p.DatosRegimenGeneral = DatosRegimenGeneral{}
		for _, imp := range pr.DatosRegimenGeneral.Impuestos {
			p.DatosRegimenGeneral.Impuestos = append(p.DatosRegimenGeneral.Impuestos, Impuesto{
				ID:     imp.ID,
				Estado: imp.Estado,
			})
		}
		for _, act := range pr.DatosRegimenGeneral.Actividades {
			p.DatosRegimenGeneral.Actividades = append(p.DatosRegimenGeneral.Actividades, Actividad{ID: act.ID})
		}
	}

	// Collect errors
	for _, e := range pr.ErrorConstancia {
		if e.Error != "" {
			p.Errores = append(p.Errores, e.Error)
		}
	}
	for _, e := range pr.ErrorMonotributo {
		if e.Error != "" {
			p.Errores = append(p.Errores, e.Error)
		}
	}
	for _, e := range pr.ErrorRegimenGeneral {
		if e.Error != "" {
			p.Errores = append(p.Errores, e.Error)
		}
	}

	return p
}
