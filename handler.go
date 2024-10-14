package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/peterstace/simplefeatures/geom"
)

//go:embed main.html
var mainHTML string

//go:embed main.js
var mainJS string

//go:embed main.css
var mainCSS string

type Handler struct {
	vertSource     *TileFetcher
	assembler      *Assembler
	coverageSource *CoverageFetcher
	mux            *http.ServeMux
}

func NewHandler(vertSource *TileFetcher, assembler *Assembler, coverage *CoverageFetcher) *Handler {
	mux := http.NewServeMux()
	h := &Handler{
		vertSource:     vertSource,
		assembler:      assembler,
		coverageSource: coverage,
		mux:            mux,
	}
	h.mux.HandleFunc("GET /{$}", static(mainHTML, "text/html"))
	h.mux.HandleFunc("GET /main.js", static(mainJS, "application/javascript"))
	h.mux.HandleFunc("GET /main.css", static(mainCSS, "text/css"))
	h.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, _ *http.Request) { w.Write(notFoundTile) })
	h.mux.HandleFunc("GET /tiles/{z}/{x}/{y}", h.tiles)
	h.mux.HandleFunc("GET /download", h.download)
	h.mux.HandleFunc("GET /coverage", h.coverage)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loggingMiddleware(h.mux).ServeHTTP(w, r)
}

func static(str, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", contentType)
		io.WriteString(w, str)
	}
}

func parseQuery[T any](name string, w http.ResponseWriter, r *http.Request, f func(string) (T, error)) (T, bool) {
	var zero T
	q := r.URL.Query()
	if !q.Has(name) {
		http.Error(w, "missing "+name+" query parameter", http.StatusBadRequest)
		return zero, false
	}
	s := q.Get(name)
	v, err := f(s)
	if err != nil {
		http.Error(w, "invalid "+name+" query parameter", http.StatusBadRequest)
		return zero, false
	}
	return v, true
}

func parseFloat64Query(name string, w http.ResponseWriter, r *http.Request) (float64, bool) {
	return parseQuery(name, w, r, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	})
}

func parseIntQuery(name string, w http.ResponseWriter, r *http.Request) (int, bool) {
	return parseQuery(name, w, r, strconv.Atoi)
}

func parseDateQuery(name string, w http.ResponseWriter, r *http.Request) (Date, bool) {
	return parseQuery(name, w, r, func(s string) (Date, error) {
		_, err := time.Parse(time.DateOnly, s)
		return Date(s), err
	})
}

func (h *Handler) tiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var z, x, y int
	for name, dest := range map[string]*int{"z": &z, "x": &x, "y": &y} {
		s := r.PathValue(name)
		v, err := strconv.Atoi(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		*dest = v
	}

	date, ok := parseDateQuery("date", w, r)
	if !ok {
		return
	}

	raw, err := h.vertSource.GetRaw(ctx, date, z, x, y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(raw)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	var x, y, xRes, yRes, zoom, downsample int
	for name, dest := range map[string]*int{"x": &x, "y": &y, "xres": &xRes, "yres": &yRes, "zoom": &zoom, "downsample": &downsample} {
		var ok bool
		*dest, ok = parseIntQuery(name, w, r)
		if !ok {
			return
		}
	}

	date, ok := parseDateQuery("date", w, r)
	if !ok {
		return
	}

	xy := image.Pt(x, y)
	resolution := image.Pt(xRes, yRes)
	img, err := h.assembler.Assemble(r.Context(), Date(date), zoom, xy, resolution, downsample)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var jpegBuf bytes.Buffer
	opts := &jpeg.Options{Quality: 95} // TODO: Make configurable.
	if err := jpeg.Encode(&jpegBuf, img, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(jpegBuf.Bytes())
}

func (h *Handler) coverage(w http.ResponseWriter, r *http.Request) {
	minLon, ok := parseFloat64Query("minlon", w, r)
	if !ok {
		return
	}

	minLat, ok := parseFloat64Query("minlat", w, r)
	if !ok {
		return
	}

	maxLon, ok := parseFloat64Query("maxlon", w, r)
	if !ok {
		return
	}

	maxLat, ok := parseFloat64Query("maxlat", w, r)
	if !ok {
		return
	}

	if minLat < -90 || minLat > 90 {
		http.Error(w, "invalid minlat query parameter", http.StatusBadRequest)
		return
	}

	if maxLat < -90 || maxLat > 90 {
		http.Error(w, "invalid maxlat query parameter", http.StatusBadRequest)
		return
	}

	env := geom.NewEnvelope(
		geom.XY{X: minLon, Y: minLat},
		geom.XY{X: maxLon, Y: maxLat},
	)

	dates, err := h.coverageSource.GetCoverage(r.Context(), env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(dates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
