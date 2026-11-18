package main

import (
	"net/http"
	"testing"
)

type respostaRotina struct {
	configRotina
	Hoje        string `json:"hoje"`
	DiaDeHoje   int    `json:"dia_de_hoje"`
	SemanaAtual string `json:"semana_atual"`
}

func TestRotinaPadraoEhValida(t *testing.T) {
	if err := rotinaPadrao().validar(); err != nil {
		t.Fatal(err)
	}
}

func TestRotinaNasceComOModeloEGuardaAsMudancas(t *testing.T) {
	e := novoAmbiente(t)
	var r respostaRotina
	if s := e.pedir("GET", "/api/rotina", nil, &r); s != http.StatusOK {
		t.Fatalf("rotina: %d", s)
	}
	// O relógio do teste marca segunda, 09/11/2026.
	if r.Hoje != "2026-11-09" || r.DiaDeHoje != 0 || r.SemanaAtual != "2026-11-09" {
		t.Fatalf("hoje: %+v", r)
	}
	if len(r.Atividades) == 0 || len(r.Modelo) == 0 {
		t.Fatal("sem o modelo padrão")
	}

	c := r.configRotina
	c.Modelo = c.Modelo[:1]
	if s := e.pedir("PUT", "/api/rotina", c, nil); s != http.StatusOK {
		t.Fatalf("salvar: %d", s)
	}
	r = respostaRotina{}
	e.pedir("GET", "/api/rotina", nil, &r)
	if len(r.Modelo) != 1 {
		t.Fatalf("modelo: %d blocos", len(r.Modelo))
	}

	ruins := []func(c *configRotina){
		func(c *configRotina) { c.Modelo[0].Fim = c.Modelo[0].Inicio },
		func(c *configRotina) { c.Modelo[0].Inicio = "7h" },
		func(c *configRotina) { c.Modelo[0].Atividade = "nada" },
		func(c *configRotina) { c.Modelo[0].Dia = 7 },
		func(c *configRotina) { c.Atividades[0].Cor = "#fff" },
		func(c *configRotina) { c.Atividades[0].MetaMin = 5; c.Atividades[0].MetaMax = 2 },
		func(c *configRotina) { c.Atividades = append(c.Atividades, c.Atividades[0]) },
	}
	for i, estraga := range ruins {
		copia := configRotina{Atividades: append([]atividade{}, r.Atividades...), Modelo: append([]bloco{}, r.Modelo...)}
		estraga(&copia)
		if s := e.pedir("PUT", "/api/rotina", copia, nil); s != http.StatusBadRequest {
			t.Fatalf("caso %d: esperava 400, veio %d", i, s)
		}
	}
}

func TestSemanaPlanejadaAPartirDoModeloEDepoisRemanejada(t *testing.T) {
	e := novoAmbiente(t)
	var s semanaRotina
	if st := e.pedir("GET", "/api/rotina/semanas/2026-11-16", nil, &s); st != http.StatusOK || s.Planejada {
		t.Fatalf("antes de planejar: %d %+v", st, s)
	}
	if st := e.pedir("GET", "/api/rotina/semanas/2026-11-18", nil, nil); st != http.StatusBadRequest {
		t.Fatalf("quarta não começa semana: %d", st)
	}

	if st := e.pedir("POST", "/api/rotina/semanas/2026-11-16", nil, &s); st != http.StatusCreated || !s.Planejada || len(s.Blocos) != len(rotinaPadrao().Modelo) {
		t.Fatalf("planejar: %d %d blocos", st, len(s.Blocos))
	}
	if st := e.pedir("POST", "/api/rotina/semanas/2026-11-16", nil, nil); st != http.StatusConflict {
		t.Fatalf("planejar de novo: %d", st)
	}

	// Sexta é feriado: tira o trabalho, marca o dia; quarta à noite livre vira programação.
	var blocos []bloco
	for _, b := range s.Blocos {
		if b.Dia == 4 && b.Atividade == "trabalho" {
			continue
		}
		if b.Dia == 2 && b.Atividade == "namorada" {
			b.Atividade, b.Inicio, b.Fim = "programacao", "20:00", "21:30"
		}
		if b.Dia == 0 && b.Atividade == "ingles" {
			b.Feito = true
		}
		blocos = append(blocos, b)
	}
	corpo := map[string]any{"blocos": blocos, "notas": map[string]string{"4": "Feriado"}}
	if st := e.pedir("PUT", "/api/rotina/semanas/2026-11-16", corpo, &s); st != http.StatusOK {
		t.Fatalf("salvar semana: %d", st)
	}
	s = semanaRotina{}
	e.pedir("GET", "/api/rotina/semanas/2026-11-16", nil, &s)
	if s.Notas["4"] != "Feriado" || len(s.Blocos) != len(blocos) {
		t.Fatalf("lida: %+v", s.Notas)
	}
	feitos := 0
	for _, b := range s.Blocos {
		if b.Feito {
			feitos++
		}
	}
	if feitos != 1 {
		t.Fatalf("feitos: %d", feitos)
	}

	if st := e.pedir("PUT", "/api/rotina/semanas/2026-11-16", map[string]any{"notas": map[string]string{"9": "x"}}, nil); st != http.StatusBadRequest {
		t.Fatalf("nota de dia ruim: %d", st)
	}

	// Refazer pelo modelo.
	s = semanaRotina{}
	if st := e.pedir("POST", "/api/rotina/semanas/2026-11-16?refazer=1", nil, &s); st != http.StatusCreated || len(s.Notas) != 0 {
		t.Fatalf("refazer: %d", st)
	}
	if st := e.pedir("DELETE", "/api/rotina/semanas/2026-11-16", nil, nil); st != http.StatusNoContent {
		t.Fatalf("apagar: %d", st)
	}
	e.pedir("GET", "/api/rotina/semanas/2026-11-16", nil, &s)
	if s.Planejada {
		t.Fatal("continuou planejada")
	}
}
