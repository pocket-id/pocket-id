package profilepicture

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"

	"github.com/zitadel/exifremove/pkg/exifremove"
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
		stripped, err := exifremove.Remove(data)
		if err == nil {
			return bytes.NewReader(stripped), nil
		}
		return bytes.NewReader(data), nil
	case "png":
		stripped, err := exifremove.Remove(data)
		if err == nil {
			return bytes.NewReader(stripped), nil
		}
		return bytes.NewReader(data), nil
	case "webp":
		return bytes.NewReader(stripWEBPMetadata(data)), nil
	default:
		return bytes.NewReader(data), nil
	}
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
