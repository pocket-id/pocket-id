package utils

import (
	"mime"
	"path/filepath"
	"strings"
)

func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 0 && ext[0] == '.' {
		return ext[1:]
	}
	return filename
}

// SplitFileName splits a full file name into name and extension.
func SplitFileName(fullName string) (name, ext string) {
	dot := strings.LastIndex(fullName, ".")
	if dot == -1 || dot == 0 {
		return fullName, "" // no extension or hidden file like .gitignore
	}
	return fullName[:dot], fullName[dot+1:]
}

func GetImageMimeType(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "svg":
		return "image/svg+xml"
	case "ico":
		return "image/x-icon"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "heic":
		return "image/heic"
	default:
		return ""
	}
}

func GetFontMimeType(ext string) string {
	switch ext {
	case "woff2":
		return "font/woff2"
	case "woff":
		return "font/woff"
	case "ttf":
		return "font/ttf"
	case "otf":
		return "font/otf"
	default:
		return ""
	}
}

func GetImageExtensionFromMimeType(mimeType string) string {
	// Normalize and strip parameters like `; charset=utf-8`
	mt := strings.TrimSpace(strings.ToLower(mimeType))
	if v, _, err := mime.ParseMediaType(mt); err == nil {
		mt = v
	}
	switch mt {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/svg+xml":
		return "svg"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return "ico"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/avif":
		return "avif"
	case "image/heic", "image/heif":
		return "heic"
	default:
		return ""
	}
}
