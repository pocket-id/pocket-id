package email

import (
	"fmt"
	"strings"
	"unicode"
)

const maxLineLength = 78
const continuePrefix = "    "
const addressSeparator = ", "

type Composer struct {
	isClosed bool
	content  strings.Builder
}

func NewComposer() *Composer {
	return &Composer{}
}

type Address struct {
	Name  string
	Email string
}

func (c *Composer) AddAddressHeader(name string, addresses []Address) {
	c.content.WriteString(genAddressHeader(name, addresses, maxLineLength))
	c.content.WriteString("\n")
}

func genAddressHeader(name string, addresses []Address, maxLength int) string {
	hl := &headerLine{
		maxLineLength:  maxLength,
		continuePrefix: continuePrefix,
	}

	hl.Write(name)
	hl.Write(": ")

	for i, addr := range addresses {
		var email string
		if i < len(addresses)-1 {
			email = fmt.Sprintf("<%s>%s", addr.Email, addressSeparator)
		} else {
			email = fmt.Sprintf("<%s>", addr.Email)
		}
		if isPrintableASCII(addr.Name) {
			writeHeaderAtom(hl, addr.Name)
		} else {
			writeHeaderQ(hl, addr.Name)
		}
		writeHeaderAtom(hl, " ")
		writeHeaderAtom(hl, email)
	}
	hl.EndLine()
	return hl.String()
}

func (c *Composer) AddHeader(name, value string) {
	if isPrintableASCII(value) && len(value)+len(name)+len(": ") < maxLineLength {
		c.AddHeaderRaw(name, value)
		return
	}

	c.content.WriteString(genHeader(name, value, maxLineLength))
	c.content.WriteString("\n")
}

func genHeader(name, value string, maxLength int) string {
	// add content as raw header when it is printable ASCII and shorter than maxLineLength
	hl := &headerLine{
		maxLineLength:  maxLength,
		continuePrefix: continuePrefix,
	}

	hl.Write(name)
	hl.Write(": ")
	writeHeaderQ(hl, value)
	hl.EndLine()
	return hl.String()
}

const qEncStart = "=?utf-8?q?"
const qEncEnd = "?="

type headerLine struct {
	buffer         strings.Builder
	line           strings.Builder
	maxLineLength  int
	continuePrefix string
}

func (h *headerLine) FitsLine(length int) bool {
	return h.line.Len()+len(h.continuePrefix)+length+2 < h.maxLineLength
}

func (h *headerLine) Write(str string) {
	h.line.WriteString(str)
}

func (h *headerLine) EndLineWith(str string) {
	h.line.WriteString(str)
	h.EndLine()
}

func (h *headerLine) EndLine() {
	if h.line.Len() == 0 {
		return
	}

	if h.buffer.Len() != 0 {
		h.buffer.WriteString("\n")
		h.buffer.WriteString(h.continuePrefix)
	}
	h.buffer.WriteString(h.line.String())
	h.line.Reset()
}

func (h *headerLine) String() string {
	return h.buffer.String()
}

func writeHeaderQ(header *headerLine, value string) {

	// current line does not fit event the first character - do \n
	if !header.FitsLine(len(qEncStart) + len(convertRunes(value[0:1])[0]) + len(qEncEnd)) {
		header.EndLineWith("")
	}

	header.Write(qEncStart)

	for _, token := range convertRunes(value) {
		if header.FitsLine(len(token) + len(qEncEnd)) {
			header.Write(token)
		} else {
			header.EndLineWith(qEncEnd)
			header.Write(qEncStart)
			header.Write(token)
		}
	}

	header.Write(qEncEnd)
}

func writeHeaderAtom(header *headerLine, value string) {
	if !header.FitsLine(len(value)) {
		header.EndLine()
	}
	header.Write(value)
}

func (c *Composer) AddHeaderRaw(name, value string) {
	if c.isClosed {
		panic("composer had already written body!")
	}
	header := fmt.Sprintf("%s: %s\n", name, value)
	c.content.WriteString(header)
}

func (c *Composer) Body(body string) {
	c.content.WriteString("\n")
	c.content.WriteString(body)
	c.isClosed = true
}

func (c *Composer) String() string {
	return c.content.String()
}

func convertRunes(str string) []string {
	var enc = make([]string, 0, len(str))
	for _, r := range str {
		if r == ' ' {
			enc = append(enc, "_")
		} else if isPrintableASCIIRune(r) &&
			r != '=' &&
			r != '?' &&
			r != '_' {
			enc = append(enc, string(r))
		} else {
			enc = append(enc, string(toHex([]byte(string(r)))))
		}
	}
	return enc
}

func toHex(in []byte) []byte {
	enc := make([]byte, 0, len(in)*2)
	for _, b := range in {
		enc = append(enc, '=')
		enc = append(enc, hex(b/16))
		enc = append(enc, hex(b%16))
	}
	return enc
}

func hex(n byte) byte {
	if n > 9 {
		return n + (65 - 10)
	} else {
		return n + 48
	}
}

func isPrintableASCII(str string) bool {
	for _, r := range str {
		if !unicode.IsPrint(r) || r >= unicode.MaxASCII {
			return false
		}
	}
	return true
}

func isPrintableASCIIRune(r rune) bool {
	return r > 31 && r < 127
}
