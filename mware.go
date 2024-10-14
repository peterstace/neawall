package main

import (
	"bytes"
	"encoding/hex"
	"log"
	"net/http"
)

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		irw := &interceptingResponseWriter{inner: w}
		h.ServeHTTP(irw, r)
		if r.Context().Err() != nil {
			return // Request was cancelled by the user.
		}
		if irw.statusCode != http.StatusOK {
			log.Printf("> %d %s\n%s", irw.statusCode, r.URL, hex.Dump(irw.body.Bytes()))
		}
	})
}

type interceptingResponseWriter struct {
	inner      http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *interceptingResponseWriter) Header() http.Header {
	return w.inner.Header()
}

func (w *interceptingResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.body.Write(p)
	return w.inner.Write(p)
}

func (w *interceptingResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
}
