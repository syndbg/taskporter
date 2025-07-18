package jetbrains

import "encoding/xml"

// JetBrainsMethod represents a method element in JetBrains configuration XML
type JetBrainsMethod struct {
	XMLName xml.Name          `xml:"method"`
	Version string            `xml:"v,attr"`
	Options []JetBrainsOption `xml:"option"`
}
