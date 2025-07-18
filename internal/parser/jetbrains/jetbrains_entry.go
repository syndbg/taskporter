package jetbrains

import "encoding/xml"

// JetBrainsEntry represents an entry element in JetBrains configuration XML
type JetBrainsEntry struct {
	XMLName xml.Name `xml:"entry"`
	Key     string   `xml:"key,attr"`
	Value   string   `xml:"value,attr"`
}
