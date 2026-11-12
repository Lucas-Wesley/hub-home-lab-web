package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

// Início: uma página de anotações livre, como no Notion (texto, lista, checklist e toggle). O
// documento é o JSON do editor (Tiptap) e o Hub só o guarda: confere que é um documento e mais nada.
type paginaInicio struct {
	Conteudo     json.RawMessage `json:"conteudo"` // null enquanto nada foi escrito
	AtualizadaEm string          `json:"atualizada_em"`
}

func (a *app) handleInicio(w http.ResponseWriter, r *http.Request) {
	p := paginaInicio{Conteudo: json.RawMessage("null")}
	var conteudo string
	err := a.db.QueryRowContext(r.Context(), `SELECT conteudo, atualizada_em FROM inicio WHERE id = 1`).Scan(&conteudo, &p.AtualizadaEm)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		falha(w, err)
		return
	}
	if err == nil {
		p.Conteudo = json.RawMessage(conteudo)
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *app) handleSalvarInicio(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Conteudo json.RawMessage `json:"conteudo"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	var doc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(corpo.Conteudo, &doc); err != nil || doc.Type != "doc" {
		erroJSON(w, http.StatusBadRequest, "Documento inválido.")
		return
	}
	var compacto bytes.Buffer
	if err := json.Compact(&compacto, corpo.Conteudo); err != nil {
		erroJSON(w, http.StatusBadRequest, "Documento inválido.")
		return
	}
	agora := a.agora()
	_, err := a.db.ExecContext(r.Context(), `INSERT INTO inicio (id, conteudo, atualizada_em) VALUES (1, ?, ?)
		ON CONFLICT (id) DO UPDATE SET conteudo = excluded.conteudo, atualizada_em = excluded.atualizada_em`,
		compacto.String(), agora)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, paginaInicio{Conteudo: compacto.Bytes(), AtualizadaEm: agora})
}
