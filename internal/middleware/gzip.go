package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// DecompressRequest распаковывает тело запроса, если указан Content-Encoding: gzip.
// При неподдерживаемой кодировке возвращает 415.
func DecompressRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc, ok := normalizedContentEncoding(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if enc != "gzip" {
			http.Error(w, "unsupported Content-Encoding", http.StatusUnsupportedMediaType)
			return
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "invalid gzip body", http.StatusBadRequest)
			return
		}
		defer func() { _ = zr.Close() }()

		r2 := r.Clone(r.Context())
		r2.Body = &readCloser{
			Reader: zr,
			closeFn: func() error {
				_ = zr.Close()
				return r.Body.Close()
			},
		}
		// Avoid confusing downstream by leaving the encoding header in place.
		r2.Header = r.Header.Clone()
		r2.Header.Del("Content-Encoding")

		next.ServeHTTP(w, r2)
	})
}

func normalizedContentEncoding(r *http.Request) (enc string, ok bool) {
	rawEnc := r.Header.Get("Content-Encoding")
	enc = strings.TrimSpace(strings.ToLower(rawEnc))
	return enc, enc != "" && enc != "identity"
}

// CompressResponse сжимает ответ gzip, если клиент поддерживает Accept-Encoding: gzip
// и Content-Type — application/json или text/html.
func CompressResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !clientAcceptsGzip(r) || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			request:        r,
		}
		defer func() { _ = gw.Close() }()

		next.ServeHTTP(gw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	request *http.Request

	statusCode  int
	wroteHeader bool
	headerSent  bool
	gz          *gzip.Writer
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.wroteHeader = true
	w.statusCode = statusCode
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if w.gz == nil && w.shouldCompressNow() {
		w.enableCompression()
	}

	if w.gz != nil {
		if !w.headerSent {
			if !w.wroteHeader {
				w.statusCode = http.StatusOK
			}
			w.ResponseWriter.WriteHeader(w.statusCode)
			w.headerSent = true
		}
		return w.gz.Write(p)
	}

	if !w.headerSent {
		if !w.wroteHeader {
			w.statusCode = http.StatusOK
		}
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.headerSent = true
	}
	return w.ResponseWriter.Write(p)
}

func (w *gzipResponseWriter) Close() error {
	if !w.headerSent && w.wroteHeader {
		// Handler called WriteHeader but wrote no body.
		// Still must send headers.
		if w.gz == nil && w.shouldCompressNow() {
			w.enableCompression()
		}
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.headerSent = true
	}
	if w.gz != nil {
		return w.gz.Close()
	}
	return nil
}

func (w *gzipResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.gz != nil {
			_ = w.gz.Flush()
		}
		f.Flush()
	}
}

func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *gzipResponseWriter) shouldCompressNow() bool {
	h := w.Header()

	// If handler already decided content encoding, don't double-encode.
	if ce := strings.TrimSpace(strings.ToLower(h.Get("Content-Encoding"))); ce != "" && ce != "identity" {
		return false
	}
	if h.Get("Content-Range") != "" {
		return false
	}

	ct := h.Get("Content-Type")
	return isCompressibleContentType(ct)
}

func (w *gzipResponseWriter) enableCompression() {
	h := w.Header()
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	h.Del("Content-Length")

	w.gz = gzip.NewWriter(w.ResponseWriter)
}

func clientAcceptsGzip(r *http.Request) bool {
	ae := r.Header.Get("Accept-Encoding")
	if ae == "" {
		return false
	}
	return headerContainsGzip(ae)
}

func headerContainsGzip(ae string) bool {
	parts := strings.Split(ae, ",")
	for _, p := range parts {
		token := strings.TrimSpace(strings.ToLower(p))
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "gzip") {
			continue
		}
		if strings.Contains(token, "q=0") {
			return false
		}
		return true
	}
	return false
}

func isCompressibleContentType(ct string) bool {
	ct = strings.TrimSpace(strings.ToLower(ct))
	if ct == "" {
		return false
	}
	// tolerate "application/json; charset=utf-8"
	if strings.HasPrefix(ct, "application/json") {
		return true
	}
	if strings.HasPrefix(ct, "text/html") {
		return true
	}
	return false
}

type readCloser struct {
	io.Reader
	closeFn func() error
}

func (r *readCloser) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	return nil
}
