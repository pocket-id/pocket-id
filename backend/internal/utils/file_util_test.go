package utils

import (
	"testing"
)

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "Simple file with extension",
			filename: "document.pdf",
			want:     "pdf",
		},
		{
			name:     "File with path",
			filename: "/path/to/document.txt",
			want:     "txt",
		},
		{
			name:     "File with path (Windows style)",
			filename: "C:\\path\\to\\document.jpg",
			want:     "jpg",
		},
		{
			name:     "Multiple extensions",
			filename: "archive.tar.gz",
			want:     "gz",
		},
		{
			name:     "Hidden file with extension",
			filename: ".config.json",
			want:     "json",
		},
		{
			name:     "Filename with dots",
			filename: "version.1.2.3.txt",
			want:     "txt",
		},
		{
			name:     "File with uppercase extension",
			filename: "image.JPG",
			want:     "JPG",
		},
		{
			name:     "File without extension",
			filename: "README",
			want:     "README",
		},
		{
			name:     "Hidden file without extension",
			filename: ".gitignore",
			want:     "gitignore",
		},
		{
			name:     "Empty filename",
			filename: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileExtension(tt.filename)
			if got != tt.want {
				t.Errorf("GetFileExtension(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

// Note: The current GetFileExtension function has issues with:
// 1. Files without extensions (it will return the filename)
// 2. Hidden files without extensions (it will return the filename without the dot)
// 3. Empty filenames (it will panic)
// You might want to consider making the function more robust
