package jetbrains

import "encoding/xml"

// JetBrainsList represents a list element in JetBrains configuration XML
type JetBrainsList struct {
	XMLName xml.Name              `xml:"list"`
	Options []JetBrainsListOption `xml:"option"`
}
