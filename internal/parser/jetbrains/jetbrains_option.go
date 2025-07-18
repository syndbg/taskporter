package jetbrains

import "encoding/xml"

// JetBrainsOption represents an option element in JetBrains configuration XML
type JetBrainsOption struct {
	XMLName xml.Name       `xml:"option"`
	Name    string         `xml:"name,attr"`
	Value   string         `xml:"value,attr"`
	Map     *JetBrainsMap  `xml:"map"`
	List    *JetBrainsList `xml:"list"`
}
