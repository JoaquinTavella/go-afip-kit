package soap

import "encoding/xml"

// Envelope is a SOAP 1.1 envelope for marshaling requests.
type Envelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	SoapNS  string   `xml:"xmlns:soap,attr"`
	Body    Body     `xml:"soap:Body"`
}

// Body holds the raw XML content inside the SOAP envelope.
type Body struct {
	Content []byte `xml:",innerxml"`
}

// ResponseEnvelope is used for unmarshaling SOAP responses.
// AFIP may use different namespace prefixes (soap, soapenv, etc.),
// so we use a flexible local name match.
type ResponseEnvelope struct {
	XMLName xml.Name     `xml:"Envelope"`
	Body    ResponseBody `xml:"Body"`
}

// ResponseBody holds the inner XML of the response body.
type ResponseBody struct {
	Content []byte `xml:",innerxml"`
}

// NewEnvelope wraps the given XML body bytes in a SOAP 1.1 envelope.
func NewEnvelope(bodyXML []byte) Envelope {
	return Envelope{
		SoapNS: "http://schemas.xmlsoap.org/soap/envelope/",
		Body:   Body{Content: bodyXML},
	}
}
