package jetbrains

import "encoding/xml"

// JetBrainsMap represents a map element in JetBrains configuration XML
type JetBrainsMap struct {
	XMLName xml.Name         `xml:"map"`
	Entries []JetBrainsEntry `xml:"entry"`
}
