package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

func newServer(a *app, web fs.FS, version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.handleHealth(version))

	mux.HandleFunc("GET /api/armazenamento", a.handleArmazenamento)
	mux.HandleFunc("GET /api/servidor", a.handleServidor)

	mux.HandleFunc("GET /api/limpeza", a.handleLimpeza)
	mux.HandleFunc("PUT /api/limpeza/agenda", a.handleSalvarAgenda)
	mux.HandleFunc("POST /api/limpeza", a.handleLimparAgora)

	mux.HandleFunc("GET /api/rede", a.handleRede)
	mux.HandleFunc("POST /api/rede/protecao", a.handleProtecao)
	mux.HandleFunc("POST /api/rede/liberados", a.handleLiberar)
	mux.HandleFunc("DELETE /api/rede/liberados/{dominio}", a.handleBloquearDeNovo)

	mux.HandleFunc("GET /api/sugestoes", a.handleSugestoes)
	mux.HandleFunc("PUT /api/sugestoes/{id}", a.handleGuardarSugestao)
	mux.HandleFunc("PATCH /api/sugestoes/{id}", a.handleMarcarSugestao)
	mux.HandleFunc("DELETE /api/sugestoes/{id}", a.handleApagarSugestao)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		erroJSON(w, http.StatusNotFound, "não encontrado")
	})
	mux.Handle("/", spaHandler(web))
	return mux
}

func (a *app) handleHealth(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := a.db.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "erro", "version": version})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}

// ---------- Utilidades ----------

// spaHandler entrega os arquivos do frontend; caminhos desconhecidos recebem o index.html.
func spaHandler(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if _, err := fs.Stat(web, name); err != nil {
				http.ServeFileFS(w, r, web, "index.html")
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}

func lerCorpo(w http.ResponseWriter, r *http.Request, destino any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := json.NewDecoder(r.Body).Decode(destino); err != nil {
		erroJSON(w, http.StatusBadRequest, "Pedido inválido.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func erroJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"erro": msg})
}

func falha(w http.ResponseWriter, err error) {
	log.Print(err)
	erroJSON(w, http.StatusInternalServerError, err.Error())
}
