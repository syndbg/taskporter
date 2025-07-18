package jetbrains

import "encoding/xml"

// JetBrainsExternalSystemSettings represents external system settings in JetBrains configuration XML
type JetBrainsExternalSystemSettings struct {
	XMLName xml.Name          `xml:"ExternalSystemSettings"`
	Options []JetBrainsOption `xml:"option"`
}
