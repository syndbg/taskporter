package jetbrains

import "encoding/xml"

// JetBrainsConfiguration represents the root element of a JetBrains run configuration XML file
type JetBrainsConfiguration struct {
	XMLName       xml.Name                  `xml:"component"`
	Configuration JetBrainsRunConfiguration `xml:"configuration"`
}
