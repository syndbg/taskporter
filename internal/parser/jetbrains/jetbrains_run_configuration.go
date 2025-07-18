package jetbrains

import "encoding/xml"

// JetBrainsRunConfiguration represents a single run configuration in a JetBrains XML file
type JetBrainsRunConfiguration struct {
	XMLName                xml.Name                         `xml:"configuration"`
	Name                   string                           `xml:"name,attr"`
	Type                   string                           `xml:"type,attr"`
	FactoryName            string                           `xml:"factoryName,attr"`
	Default                string                           `xml:"default,attr"`
	Options                []JetBrainsOption                `xml:"option"`
	Module                 *JetBrainsModule                 `xml:"module"`
	Method                 *JetBrainsMethod                 `xml:"method"`
	ExternalSystemSettings *JetBrainsExternalSystemSettings `xml:"ExternalSystemSettings"`
}
