package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestInicioComecaVazioEGuardaODocumento(t *testing.T) {
	e := novoAmbiente(t)
	var p paginaInicio
	if s := e.pedir("GET", "/api/inicio", nil, &p); s != http.StatusOK || string(p.Conteudo) != "null" {
		t.Fatalf("vazio: %d %s", s, p.Conteudo)
	}

	doc := json.RawMessage(`{"type":"doc","content":[{"type":"taskList","content":[{"type":"taskItem","attrs":{"checked":true},
		"content":[{"type":"paragraph","content":[{"type":"text","text":"Pão"}]}]}]}]}`)
	if s := e.pedir("PUT", "/api/inicio", map[string]any{"conteudo": doc}, &p); s != http.StatusOK || p.AtualizadaEm == "" {
		t.Fatalf("salvar: %d %+v", s, p)
	}
	p = paginaInicio{}
	e.pedir("GET", "/api/inicio", nil, &p)
	var lido struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(p.Conteudo, &lido); err != nil || len(lido.Content) != 1 || lido.Content[0].Type != "taskList" {
		t.Fatalf("lido: %s", p.Conteudo)
	}

	for _, ruim := range []any{"texto", map[string]string{"type": "outro"}, nil} {
		if s := e.pedir("PUT", "/api/inicio", map[string]any{"conteudo": ruim}, nil); s != http.StatusBadRequest {
			t.Fatalf("%v: esperava 400, veio %d", ruim, s)
		}
	}
}
