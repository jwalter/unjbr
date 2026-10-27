// Package server wires up HTTP routes and handlers.
package server

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"time"

	"github.com/jwalter/unjbr/web"
)

// Server is the application's HTTP handler.
type Server struct {
	logger *slog.Logger
	pages  map[string]*template.Template
	mux    *http.ServeMux
}

// New builds a Server with all routes registered.
func New(logger *slog.Logger) *Server {
	s := &Server{
		logger: logger,
		pages:  mustParsePages(),
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	static, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
}

// ServeHTTP logs each request and dispatches it to the router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(rec, r)
	s.logger.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rec.status,
		"duration", time.Since(start),
	)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", map[string]any{"Title": "unjbr"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok"))
}

// mustParsePages parses every page template together with the shared layout,
// so each page can define its own "content" block.
func mustParsePages() map[string]*template.Template {
	layout := template.Must(template.ParseFS(web.FS, "templates/layout.html"))
	files, err := fs.Glob(web.FS, "templates/*.html")
	if err != nil {
		panic(err)
	}
	pages := make(map[string]*template.Template)
	for _, f := range files {
		name := path.Base(f)
		if name == "layout.html" {
			continue
		}
		pages[name] = template.Must(template.Must(layout.Clone()).ParseFS(web.FS, f))
	}
	return pages
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	t, ok := s.pages[page]
	if !ok {
		s.logger.Error("unknown page", "page", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		s.logger.Error("render template", "page", page, "err", err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
