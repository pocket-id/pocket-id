package frontend

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"golang.org/x/sync/singleflight"
)

// Content encodings the frontend assets can be served with
const (
	encodingBrotli = "br"
	encodingGzip   = "gzip"
)

// Compression levels used when compressing assets
// They lean towards a better compression ratio rather than speed, because each asset is compressed at most once, on its first request, and the result is then kept in memory for the lifetime of the process
const (
	gzipCompressionLevel   = gzip.BestCompression
	brotliCompressionLevel = 6
)

// Size boundaries for the assets that are compressed
// Assets below the minimum barely shrink, and often grow, once framing overhead is accounted for
// The maximum bounds how much memory a single cached asset can take
const (
	minCompressibleSize = 1024
	maxCompressibleSize = 8 << 20
)

// Extensions of the assets that are worth compressing, mapped to the content type they are served with
// Anything not listed here is either already compressed (fonts, images) or unknown, and is served as-is
// These take the place of mime.TypeByExtension because that reads the platform's MIME database, which is the Windows registry on Windows and /etc/mime.types on Linux, so the same bundle would otherwise be served with different content types depending on the host
var compressibleContentTypes = map[string]string{
	".css":         "text/css; charset=utf-8",
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".json":        "application/json",
	".map":         "application/json",
	".mjs":         "text/javascript; charset=utf-8",
	".svg":         "image/svg+xml",
	".txt":         "text/plain; charset=utf-8",
	".wasm":        "application/wasm",
	".webmanifest": "application/manifest+json",
	".xml":         "text/xml; charset=utf-8",
}

// FileServerWithCaching wraps http.FileServer to add caching headers and to serve compressed assets
type FileServerWithCaching struct {
	distFS                  fs.FS
	fileServer              http.Handler
	compressor              *assetCompressor
	lastModified            time.Time
	lastModifiedHeaderValue string
}

func NewFileServerWithCaching(distFS fs.FS) *FileServerWithCaching {
	now := time.Now()
	return &FileServerWithCaching{
		distFS:                  distFS,
		fileServer:              http.FileServer(http.FS(distFS)),
		compressor:              newAssetCompressor(distFS),
		lastModified:            now,
		lastModifiedHeaderValue: now.UTC().Format(http.TimeFormat),
	}
}

func (f *FileServerWithCaching) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First, set cache headers
	// Check if the request is for an immutable asset
	if isImmutableAsset(r) {
		// Set the cache control header as immutable with a long expiration
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// Check if the client has a cached version
		ifModifiedSince := r.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			ifModifiedSinceTime, err := time.Parse(http.TimeFormat, ifModifiedSince)
			if err == nil && f.lastModified.Before(ifModifiedSinceTime.Add(1*time.Second)) {
				// Client's cached version is up to date
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Cache other assets for up to 24 hours, but set Last-Modified too
		w.Header().Set("Last-Modified", f.lastModifiedHeaderValue)
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}

	// Try serving a compressed copy of the asset, falling back to the plain file server when that isn't possible
	if f.serveCompressed(w, r) {
		return
	}

	f.fileServer.ServeHTTP(w, r)
}

// serveCompressed writes a compressed copy of the requested asset and reports whether it did
// When it returns false the response body hasn't been touched, and the caller must serve the asset uncompressed
func (f *FileServerWithCaching) serveCompressed(w http.ResponseWriter, r *http.Request) bool {
	// Only GET and HEAD can be answered from memory, and the file server is left to reject anything else
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	// Assets we don't recognize are left entirely to the file server, including the content type it picks for them
	name, contentType, size, ok := f.bundledAsset(r.URL.Path)
	if !ok {
		return false
	}

	// Setting the content type here, rather than letting http.ServeContent derive it, keeps it identical whether or not the asset ends up compressed
	w.Header().Set("Content-Type", contentType)

	// Assets outside the size range are always served as-is, so they must not get the "Vary" header below
	if size < minCompressibleSize || size > maxCompressibleSize {
		return false
	}

	// The response for this asset depends on the request's Accept-Encoding, so caches in front of Pocket ID need to key on it too
	w.Header().Add("Vary", "Accept-Encoding")

	// Pick the encoding to use, if the client accepts any of the supported ones
	encoding := negotiateEncoding(r.Header.Get("Accept-Encoding"))
	if encoding == "" {
		return false
	}

	// Compress the asset, or grab the result of a previous compression from the cache
	// A nil result means compressing the asset doesn't pay off, so it's served as-is
	data, err := f.compressor.Get(name, encoding)
	if err != nil {
		// Failing to compress an asset is not a reason to fail the request, so it's served uncompressed instead
		slog.WarnContext(r.Context(), "Failed to compress frontend asset", "asset", name, "encoding", encoding, "error", err)
		return false
	}
	if data == nil {
		return false
	}

	w.Header().Set("Content-Encoding", encoding)

	// http.ServeContent refuses to set a content length on encoded responses, so we set it ourselves to avoid falling back to a chunked response
	// It is overwritten by http.ServeContent, with the correct value, when the client asks for a range
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	// The modification time is left empty because conditional requests and the Last-Modified header are already handled by the caller
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))

	return true
}

// bundledAsset maps a request path to a compressible file in the bundle, returning the name to look it up with, the content type to serve it with, and its uncompressed size
func (f *FileServerWithCaching) bundledAsset(urlPath string) (name string, contentType string, size int64, ok bool) {
	// Assets whose extension isn't in the list are served as-is
	ext := strings.ToLower(path.Ext(urlPath))
	contentType, ok = compressibleContentTypes[ext]
	if !ok {
		return "", "", 0, false
	}

	// Resolve the request path the same way http.FileServer does, so the compressed copy is cached under the name of the file that is actually served
	name = strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(urlPath, "/")), "/")
	if !fs.ValidPath(name) {
		return "", "", 0, false
	}

	// Directories and paths that aren't in the bundle are left to the file server, which knows how to handle them
	info, err := fs.Stat(f.distFS, name)
	if err != nil || info.IsDir() {
		return "", "", 0, false
	}

	return name, contentType, info.Size(), true
}

func isImmutableAsset(r *http.Request) bool {
	switch {
	// Fonts
	case strings.HasPrefix(r.URL.Path, "/fonts/"):
		return true

	// Compiled SvelteKit assets
	case strings.HasPrefix(r.URL.Path, "/_app/immutable/"):
		return true

	default:
		return false
	}
}

// assetCompressor compresses assets from the frontend bundle on demand, keeping the result in memory so each asset is only ever compressed once per encoding
// The bundle is embedded in the binary and its contents are fixed at build time, so the cache is bounded by the size of the bundle and never needs to be evicted
type assetCompressor struct {
	distFS fs.FS
	cache  sync.Map
	sf     singleflight.Group
}

func newAssetCompressor(distFS fs.FS) *assetCompressor {
	return &assetCompressor{distFS: distFS}
}

// Get returns the asset "name" compressed with the given encoding, compressing it first if this is the first request for it
// It returns a nil slice, and no error, when compressing the asset doesn't make it smaller and it should be served uncompressed
func (c *assetCompressor) Get(name string, encoding string) ([]byte, error) {
	key := encoding + ":" + name

	// Serve straight from the cache if this asset has been compressed before
	cached, ok := c.cache.Load(key)
	if ok {
		return cached.([]byte), nil
	}

	// Compress the asset, using singleflight so a burst of requests for a cold asset only compresses it once
	res, err, _ := c.sf.Do(key, func() (any, error) {
		cCached, cOk := c.cache.Load(key)
		if cOk {
			return cCached, nil
		}

		compressed, cErr := c.compress(name, encoding)
		if cErr != nil {
			return nil, cErr
		}

		c.cache.Store(key, compressed)

		return compressed, nil
	})
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

// compress reads an asset from the bundle and compresses it in memory
// It returns a nil slice when the compressed asset isn't any smaller than the original
func (c *assetCompressor) compress(name string, encoding string) ([]byte, error) {
	// The whole asset is read in memory, which is safe because callers only pass files from the embedded bundle whose size has already been checked
	src, err := fs.ReadFile(c.distFS, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset '%s': %w", name, err)
	}

	// Create the writer for the requested encoding
	buf := &bytes.Buffer{}
	var cw io.WriteCloser
	switch encoding {
	case encodingBrotli:
		cw = brotli.NewWriterLevel(buf, brotliCompressionLevel)
	case encodingGzip:
		cw, err = gzip.NewWriterLevel(buf, gzipCompressionLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip writer: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported encoding '%s'", encoding)
	}

	// Compress the asset, closing the writer to flush the trailing bytes
	_, err = cw.Write(src)
	if err != nil {
		_ = cw.Close()
		return nil, fmt.Errorf("failed to compress asset '%s': %w", name, err)
	}
	err = cw.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to compress asset '%s': %w", name, err)
	}

	// Sending the compressed copy would only waste bandwidth when it's not actually smaller
	if buf.Len() >= len(src) {
		return nil, nil
	}

	return buf.Bytes(), nil
}

// negotiateEncoding returns the best of the supported content encodings for the given Accept-Encoding header
// It returns an empty string when the client doesn't accept any of them, in which case the asset must be served uncompressed
func negotiateEncoding(acceptEncoding string) string {
	if acceptEncoding == "" {
		return ""
	}

	// Weights start negative to tell "not mentioned in the header" apart from "explicitly rejected with q=0"
	brWeight, gzipWeight, wildcardWeight := -1.0, -1.0, -1.0

	// Accept-Encoding is a comma-separated list of codings, each with an optional "q" weight
	for part := range strings.SplitSeq(acceptEncoding, ",") {
		coding, weight := parseAcceptEncodingPart(part)
		switch coding {
		case encodingBrotli:
			brWeight = weight
		case encodingGzip:
			gzipWeight = weight
		case "*":
			wildcardWeight = weight
		}
	}

	// Codings the client didn't mention are covered by the "*" entry, when there is one
	if brWeight < 0 {
		brWeight = wildcardWeight
	}
	if gzipWeight < 0 {
		gzipWeight = wildcardWeight
	}

	// Brotli wins ties because it compresses better, but an explicit preference for gzip is honored
	switch {
	case brWeight > 0 && brWeight >= gzipWeight:
		return encodingBrotli
	case gzipWeight > 0:
		return encodingGzip
	default:
		return ""
	}
}

// parseAcceptEncodingPart parses a single entry of an Accept-Encoding header, returning the lowercased coding name and its weight
func parseAcceptEncodingPart(part string) (coding string, weight float64) {
	rawCoding, params, _ := strings.Cut(part, ";")
	coding = strings.ToLower(strings.TrimSpace(rawCoding))

	// Entries without a "q" parameter have an implicit weight of 1
	weight = 1
	for param := range strings.SplitSeq(params, ";") {
		key, value, found := strings.Cut(param, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}

		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			continue
		}

		weight = parsed
	}

	return coding, weight
}
