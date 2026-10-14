package pkcs12

import (
	"bytes"
	"errors"
)

// normalizeBER only changes ASN.1 framing and constructed octet strings
// Primitive octet payloads remain byte-for-byte intact, including the authenticated safe verified by verifyMac
func normalizeBER(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input) > 1<<20 {
		return nil, errors.New("pkcs12: ASN.1 input is empty or exceeds 1 MiB")
	}
	budget := 65536
	value, consumed, err := readBERValue(input, 0, &budget)
	if err != nil {
		return nil, err
	}
	if consumed != len(input) {
		return nil, errors.New("pkcs12: trailing data found")
	}
	return value.encode(), nil
}

type berValue struct {
	tag     []byte
	content []byte
}

func (v berValue) isTag(tag byte) bool { return len(v.tag) == 1 && v.tag[0] == tag }

func (v berValue) encode() []byte {
	result := append([]byte(nil), v.tag...)
	n := len(v.content)
	if n < 128 {
		result = append(result, byte(n))
	} else {
		var length [8]byte
		i := len(length)
		for n > 0 {
			i--
			length[i] = byte(n)
			n >>= 8
		}
		result = append(result, 0x80|byte(len(length)-i))
		result = append(result, length[i:]...)
	}
	return append(result, v.content...)
}

func readBERValue(input []byte, depth int, budget *int) (berValue, int, error) {
	fail := func() (berValue, int, error) {
		return berValue{}, 0, errors.New("pkcs12: malformed or excessively nested BER data")
	}
	if depth > 32 || *budget <= 0 || len(input) < 2 || input[0] == 0 {
		return fail()
	}
	*budget -= 1
	pos := 1
	if input[0]&0x1f == 0x1f {
		for {
			if pos >= len(input) || pos > 6 || (pos == 1 && input[pos]&0x7f == 0) {
				return fail()
			}
			b := input[pos]
			pos++
			if b&0x80 == 0 {
				break
			}
		}
	}
	value := berValue{tag: append([]byte(nil), input[:pos]...)}
	if pos >= len(input) {
		return fail()
	}
	lengthByte := input[pos]
	pos++
	indefinite := lengthByte == 0x80
	n := int(lengthByte)
	if lengthByte > 0x80 {
		count := int(lengthByte & 0x7f)
		if count > 4 || count > len(input)-pos {
			return fail()
		}
		n = 0
		for _, b := range input[pos : pos+count] {
			n = n<<8 | int(b)
			if n > len(input) {
				return fail()
			}
		}
		pos += count
	}
	constructed := input[0]&0x20 != 0
	if indefinite && !constructed {
		return fail()
	}
	end := len(input)
	if !indefinite {
		if n > len(input)-pos {
			return fail()
		}
		end = pos + n
	}
	if !constructed {
		value.content = input[pos:end]
		return value, end, nil
	}
	var children []berValue
	terminated := false
	for pos < end {
		if indefinite && end-pos >= 2 && input[pos] == 0 && input[pos+1] == 0 {
			pos += 2
			terminated = true
			break
		}
		child, consumed, err := readBERValue(input[pos:end], depth+1, budget)
		if err != nil {
			return berValue{}, 0, err
		}
		children = append(children, child)
		pos += consumed
	}
	if indefinite && !terminated {
		return fail()
	}
	// EncryptedContentInfo uses an implicit [0] OCTET STRING while ContentInfo uses an explicit [0] wrapper
	if value.isTag(0x30) && len(children) == 3 && children[0].isTag(0x06) && children[1].isTag(0x30) && children[2].isTag(0xa0) {
		var encrypted bytes.Buffer
		chunks := children[2].content
		for len(chunks) > 0 {
			child, consumed, err := readBERValue(chunks, depth+1, budget)
			if err != nil || !child.isTag(0x04) {
				return fail()
			}
			encrypted.Write(child.content)
			chunks = chunks[consumed:]
		}
		children[2] = berValue{tag: []byte{0x80}, content: encrypted.Bytes()}
	}
	var content bytes.Buffer
	for _, child := range children {
		if value.isTag(0x24) {
			if !child.isTag(0x04) {
				return fail()
			}
			content.Write(child.content)
		} else {
			content.Write(child.encode())
		}
	}
	if value.isTag(0x24) {
		value.tag = []byte{0x04}
	}
	value.content = content.Bytes()
	return value, pos, nil
}
