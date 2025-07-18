package jetbrains

import "encoding/xml"

// JetBrainsListOption represents an option element within a list in JetBrains configuration XML
type JetBrainsListOption struct {
	XMLName xml.Name `xml:"option"`
	Value   string   `xml:"value,attr"`
}
