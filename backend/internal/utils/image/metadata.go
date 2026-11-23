package profilepicture

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxRIFFSize = ^uint32(0)

// StripMetadata removes EXIF/XMP metadata from JPG, PNG and WEBP images
// Returns a *bytes.Reader so that storage backends receive a seekable reader with a known content length,
// which is required for correct checksum calculation on S3-compatible services
func StripMetadata(file io.Reader, ext string) (*bytes.Reader, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(ext) {
	case "jpg", "jpeg":
		if err := rejectOversizedImage(data); err != nil {
			return nil, err
		}
		return bytes.NewReader(stripJPEGMetadata(data)), nil
	case "png":
		if err := rejectOversizedImage(data); err != nil {
			return nil, err
		}
		return bytes.NewReader(stripPNGMetadata(data)), nil
	case "webp":
		return bytes.NewReader(stripWEBPMetadata(data)), nil
	default:
		return bytes.NewReader(data), nil
	}
}

func rejectOversizedImage(data []byte) error {
	err := validateImageDimensions(bytes.NewReader(data))
	if errors.Is(err, errImageDimensionsTooLarge) {
		return fmt.Errorf("%w: %w", ErrInvalidImage, err)
	}
	return nil
}

func stripJPEGMetadata(data []byte) []byte {
	// Check if the file starts with the SOI marker
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}

	var out bytes.Buffer
	out.Write(data[:2])

	for pos := 2; pos < len(data); {
		// Each segment starts with 0xFF followed by the marker type
		if pos+2 > len(data) || data[pos] != 0xFF {
			return data
		}
		marker := data[pos+1]

		// Markers without a length field carry no payload
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			out.Write(data[pos : pos+2])
			pos += 2
			continue
		}

		// The length field counts itself but not the marker
		if pos+4 > len(data) {
			return data
		}
		segmentEnd := pos + 2 + int(binary.BigEndian.Uint16(data[pos+2:pos+4]))
		if segmentEnd > len(data) {
			return data
		}

		// The entropy-coded image data follows the SOS segment, so everything from here on is copied as is
		if marker == 0xDA {
			out.Write(data[pos:])
			return out.Bytes()
		}

		// APP1 segments hold the EXIF and XMP data
		if marker != 0xE1 {
			out.Write(data[pos:segmentEnd])
		}
		pos = segmentEnd
	}

	return data
}

func stripPNGMetadata(data []byte) []byte {
	// Check if the file starts with the PNG signature
	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(signature) || string(data[:len(signature)]) != signature {
		return data
	}

	var out bytes.Buffer
	out.WriteString(signature)

	for pos := len(signature); pos < len(data); {
		// Each chunk has a 4 byte length, a 4 byte type, the payload and a 4 byte CRC
		if pos+8 > len(data) {
			return data
		}
		chunkEnd := pos + 12 + int(binary.BigEndian.Uint32(data[pos:pos+4]))
		if chunkEnd > len(data) || chunkEnd < pos {
			return data
		}

		// Drop the EXIF chunk and the text chunks, which also carry XMP
		// The chunks are copied whole, so their CRCs stay valid
		switch string(data[pos+4 : pos+8]) {
		case "eXIf", "tEXt", "iTXt", "zTXt":
		default:
			out.Write(data[pos:chunkEnd])
		}
		pos = chunkEnd
	}

	return out.Bytes()
}

func stripWEBPMetadata(data []byte) []byte {
	// Check if the file contains the RIFF...WEBP header
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return data
	}

	var out bytes.Buffer
	// Build the WEBP header
	out.WriteString("RIFF")
	out.Write([]byte{0, 0, 0, 0}) // Size will be filled at the end
	out.WriteString("WEBP")

	for pos := 12; pos < len(data); {
		// Each RIFF chunk needs an 8 byte header
		if pos+8 > len(data) {
			return data
		}

		// Read the chunk type and payload size from the header
		chunkType := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkEnd := pos + 8 + chunkSize
		// Chunks with odd payload sizes include one additional byte at the end
		if chunkSize%2 == 1 {
			chunkEnd++
		}

		// End of chunk can't be more than the actual image data length
		if chunkEnd > len(data) {
			return data
		}

		// Remove chunks with the EXIF or XMP data type
		if chunkType == "EXIF" || chunkType == "XMP " {
			pos = chunkEnd
			continue
		}

		// In the VP8X chunk there is a feature flag if the file contains any EXIF or XMP data
		if chunkType == "VP8X" && chunkSize >= 10 {
			// Copy the chunk because we don't want to modify the original one
			chunk := make([]byte, chunkEnd-pos)
			copy(chunk, data[pos:chunkEnd])

			// Clear the Exif and XMP feature flags
			chunk[8] &^= 0x0c
			out.Write(chunk)
		} else {
			out.Write(data[pos:chunkEnd])
		}

		pos = chunkEnd
	}

	// WEBP image can max be 4GB in size
	riffSize := out.Len() - 8
	if riffSize < 0 || uint64(riffSize) > uint64(maxRIFFSize) {
		return data
	}

	// Set the size in the header (byte 4-7)
	binary.LittleEndian.PutUint32(out.Bytes()[4:8], uint32(riffSize))
	return out.Bytes()
}
