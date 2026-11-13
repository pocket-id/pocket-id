package profilepicture

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripMetadata(t *testing.T) {
	t.Run("removes WEBP EXIF and XMP chunks", func(t *testing.T) {
		input := webpFile(
			webpChunk("VP8X", []byte{0x0c, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
			webpChunk("VP8 ", []byte{1, 2, 3, 4}),
			webpChunk("EXIF", []byte("gps")),
			webpChunk("XMP ", []byte("xmp")),
			webpChunk("ICCP", []byte("profile")),
		)

		output := stripReader(t, input, "webp")

		assert.Contains(t, string(output), "VP8 ")
		assert.NotContains(t, string(output), "gps")
		assert.NotContains(t, string(output), "xmp")
		assert.Contains(t, string(output), "profile")
		assert.Equal(t, byte(0), output[20]&0x0c)
	})

	t.Run("removes JPEG EXIF and XMP segments", func(t *testing.T) {
		var encoded bytes.Buffer
		require.NoError(t, jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil))
		app1 := func(payload string) []byte {
			return append([]byte{0xFF, 0xE1, 0, byte(len(payload) + 2)}, payload...) //nolint:gosec
		}
		input := append([]byte{0xFF, 0xD8}, app1("Exif\x00\x00gps")...)
		input = append(input, app1("http://ns.adobe.com/xap/1.0/\x00xmp")...)
		input = append(input, encoded.Bytes()[2:]...)

		output := stripReader(t, input, "jpg")

		assert.Equal(t, encoded.Bytes(), output)
	})

	t.Run("removes PNG EXIF and text chunks", func(t *testing.T) {
		var encoded bytes.Buffer
		require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 4, 4))))
		// The IHDR chunk always ends at byte 33, right after the signature
		input := append([]byte{}, encoded.Bytes()[:33]...)
		input = append(input, pngChunk("eXIf", []byte("gps"))...)
		input = append(input, pngChunk("iTXt", []byte("XML:com.adobe.xmp\x00xmp"))...)
		input = append(input, encoded.Bytes()[33:]...)

		output := stripReader(t, input, "png")

		assert.Equal(t, encoded.Bytes(), output)
	})

	t.Run("leaves non photo metadata formats unchanged", func(t *testing.T) {
		input := []byte(`<svg><metadata>secret</metadata><path d="M0 0"/></svg>`)

		output := stripReader(t, input, "svg")

		assert.Equal(t, input, output)
	})

	t.Run("leaves malformed photo data unchanged", func(t *testing.T) {
		input := []byte("fake-png-content")

		output := stripReader(t, input, "png")

		assert.Equal(t, input, output)
	})
}

func stripReader(t *testing.T, input []byte, ext string) []byte {
	t.Helper()

	reader, err := StripMetadata(bytes.NewReader(input), ext)
	require.NoError(t, err)

	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	return output
}

func webpFile(chunks ...[]byte) []byte {
	var out bytes.Buffer
	out.WriteString("RIFF")
	out.Write([]byte{0, 0, 0, 0})
	out.WriteString("WEBP")
	for _, chunk := range chunks {
		out.Write(chunk)
	}
	binary.LittleEndian.PutUint32(out.Bytes()[4:8], uint32(out.Len()-8)) //nolint:gosec
	return out.Bytes()
}

func webpChunk(chunkType string, data []byte) []byte {
	var out bytes.Buffer
	out.WriteString(chunkType)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(data))) //nolint:gosec
	out.Write(data)
	if len(data)%2 == 1 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

func pngChunk(chunkType string, data []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, uint32(len(data))) //nolint:gosec
	out.WriteString(chunkType)
	out.Write(data)
	_ = binary.Write(&out, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(chunkType), data...)))
	return out.Bytes()
}
