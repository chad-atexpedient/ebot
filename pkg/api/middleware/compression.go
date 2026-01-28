// Package middleware provides HTTP middleware for ebot
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// CompressionMiddleware adds gzip compression to responses
func CompressionMiddleware(minSize int) func(http.Handler) http.Handler {
	if minSize == 0 {
		minSize = 1024 // Default 1KB minimum
	}

	// Pool of gzip writers for reuse
	gzipPool := sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
			return w
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Get a gzip writer from the pool
			gz := gzipPool.Get().(*gzip.Writer)
			defer gzipPool.Put(gz)

			gz.Reset(w)
			defer gz.Close()

			// Wrap the response writer
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length") // Length will change after compression
			
			gzw := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gz,
				minSize:        minSize,
			}

			next.ServeHTTP(gzw, r)
		})
	}
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer  *gzip.Writer
	minSize int
	written int
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	// Check if we should compress based on size
	if w.written+len(b) < w.minSize {
		// Too small to compress, write directly
		w.written += len(b)
		return w.ResponseWriter.Write(b)
	}

	w.written += len(b)
	return w.Writer.Write(b)
}

// BrotliMiddleware adds Brotli compression (higher compression ratio than gzip)
// Note: Requires import of "github.com/andybalholm/brotli"
func BrotliMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts br (Brotli)
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
				next.ServeHTTP(w, r)
				return
			}

			// For now, fall back to next handler
			// In production, implement Brotli compression similar to gzip
			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeCompressionMiddleware only compresses specific content types
func ContentTypeCompressionMiddleware(compressibleTypes []string) func(http.Handler) http.Handler {
	if len(compressibleTypes) == 0 {
		// Default compressible types
		compressibleTypes = []string{
			"application/json",
			"application/javascript",
			"text/html",
			"text/css",
			"text/plain",
			"text/xml",
			"application/xml",
		}
	}

	compressMap := make(map[string]bool)
	for _, ct := range compressibleTypes {
		compressMap[ct] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer to intercept Content-Type
			ctw := &contentTypeWrapper{
				ResponseWriter: w,
				compressMap:    compressMap,
				request:        r,
				next:           next,
			}

			next.ServeHTTP(ctw, r)
		})
	}
}

type contentTypeWrapper struct {
	http.ResponseWriter
	compressMap map[string]bool
	request     *http.Request
	next        http.Handler
	headerWritten bool
}

func (w *contentTypeWrapper) WriteHeader(statusCode int) {
	if !w.headerWritten {
		w.headerWritten = true
		
		// Check if content type is compressible
		ct := w.Header().Get("Content-Type")
		shouldCompress := false
		
		for compressType := range w.compressMap {
			if strings.Contains(ct, compressType) {
				shouldCompress = true
				break
			}
		}

		// Only set compression headers if type is compressible
		if !shouldCompress {
			w.Header().Del("Content-Encoding")
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *contentTypeWrapper) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
