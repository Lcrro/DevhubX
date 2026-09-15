package logfmt

import (
	"encoding/binary"
	"encoding/xml"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

var ansiRE = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\a]*(?:\a|\x1b\\))`)

// Bytes turns common PowerShell output encodings into readable UTF-8 before
// it is persisted. Invalid UTF-8 is kept as replacement characters instead
// of allowing control bytes to corrupt the log viewer.
func Bytes(p []byte) string {
	if len(p) >= 2 && ((p[0] == 0xff && p[1] == 0xfe) || (p[0] == 0xfe && p[1] == 0xff)) {
		little := p[0] == 0xff
		p = p[2:]
		if len(p)%2 == 1 {
			p = p[:len(p)-1]
		}
		units := make([]uint16, len(p)/2)
		for i := range units {
			if little {
				units[i] = binary.LittleEndian.Uint16(p[i*2:])
			} else {
				units[i] = binary.BigEndian.Uint16(p[i*2:])
			}
		}
		return Clean(string(utf16.Decode(units)))
	}
	return Clean(string(p))
}

// Clean strips terminal control sequences and the XML envelope PowerShell
// uses for CLIXML errors. It intentionally leaves ordinary XML-like output
// untouched unless the CLIXML marker is present.
func Clean(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = ansiRE.ReplaceAllString(text, "")
	if strings.Contains(text, "#< CLIXML") || strings.Contains(text, "<Objs") {
		text = cleanCLIXML(text)
	}
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "�")
	}
	return text
}

func cleanCLIXML(text string) string {
	text = strings.TrimSpace(strings.TrimPrefix(text, "#< CLIXML"))
	var values []string
	decoder := xml.NewDecoder(strings.NewReader(text))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "S" {
			continue
		}
		var value string
		if err := decoder.DecodeElement(&value, &start); err == nil && strings.TrimSpace(value) != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		text = strings.ReplaceAll(text, "#< CLIXML", "")
		return strings.TrimSpace(text)
	}
	return strings.Join(values, "\n") + "\n"
}
