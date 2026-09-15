package profilepicture

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateProfilePictureRejectsOversizedPixelCountBeforeDecode(t *testing.T) {
	_, err := CreateProfilePicture(bytes.NewReader(pngHeaderWithDimensions(12_000, 12_000)))

	require.ErrorIs(t, err, ErrInvalidImage)
	require.ErrorIs(t, err, errImageDimensionsTooLarge)
}

func TestCreateProfilePictureAcceptsImageWithinLimits(t *testing.T) {
	var input bytes.Buffer
	require.NoError(t, png.Encode(&input, image.NewNRGBA(image.Rect(0, 0, 400, 300))))

	output, err := CreateProfilePicture(bytes.NewReader(input.Bytes()))
	require.NoError(t, err)

	config, format, err := image.DecodeConfig(output)
	require.NoError(t, err)
	require.Equal(t, "png", format)
	require.Equal(t, 300, config.Width)
	require.Equal(t, 300, config.Height)
}

func TestStripMetadataRejectsOversizedPixelCount(t *testing.T) {
	_, err := StripMetadata(bytes.NewReader(pngHeaderWithDimensions(12_000, 12_000)), "png")

	require.ErrorIs(t, err, ErrInvalidImage)
	require.ErrorIs(t, err, errImageDimensionsTooLarge)
}

func pngHeaderWithDimensions(width, height uint32) []byte {
	var out bytes.Buffer
	out.Write([]byte("\x89PNG\r\n\x1a\n"))

	const headerLength = 13
	data := make([]byte, headerLength)
	binary.BigEndian.PutUint32(data[0:4], width)
	binary.BigEndian.PutUint32(data[4:8], height)
	data[8] = 8

	_ = binary.Write(&out, binary.BigEndian, uint32(headerLength))
	out.WriteString("IHDR")
	out.Write(data)
	_ = binary.Write(&out, binary.BigEndian, crc32.ChecksumIEEE(append([]byte("IHDR"), data...)))

	return out.Bytes()
}
