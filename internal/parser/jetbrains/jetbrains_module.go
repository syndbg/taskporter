package jetbrains

import "encoding/xml"

// JetBrainsModule represents a module element in JetBrains configuration XML
type JetBrainsModule struct {
	XMLName xml.Name `xml:"module"`
	Name    string   `xml:"name,attr"`
}
