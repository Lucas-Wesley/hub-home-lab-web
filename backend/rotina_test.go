package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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

type respostaEventos struct {
	Eventos    []eventoAdicionado `json:"eventos"`
	Observacao string             `json:"observacao"`
}

func TestEventoComDataPlanejaASemanaEAvisaOChoque(t *testing.T) {
	e := novoAmbiente(t)
	// Quinta, 26/11 (semana de 23/11, ainda sem plano), das 07h às 10h: choca com inglês e trabalho.
	ev := map[string]any{"eventos": []map[string]string{
		{"data": "2026-11-26", "inicio": "07:00", "fim": "10:00", "titulo": "Encontro com amigos"},
		{"data": "2026-11-27", "inicio": "", "fim": "", "titulo": "Pagar o boleto"},
	}}
	var r respostaEventos
	if s := e.pedir("POST", "/api/rotina/eventos", ev, &r); s != http.StatusCreated || len(r.Eventos) != 2 {
		t.Fatalf("eventos: %d %+v", s, r)
	}
	a := r.Eventos[0]
	if a.Semana != "2026-11-23" || a.Dia != 3 || !a.SemanaNova || strings.Join(a.ChocaCom, ",") != "Inglês,Trabalho" {
		t.Fatalf("evento: %+v", a)
	}
	if len(r.Eventos[1].ChocaCom) != 0 {
		t.Fatalf("tarefa sem horário não choca: %+v", r.Eventos[1])
	}

	var s semanaRotina
	e.pedir("GET", "/api/rotina/semanas/2026-11-23", nil, &s)
	if !s.Planejada || len(s.Blocos) != len(rotinaPadrao().Modelo)+2 {
		t.Fatalf("semana: %v %d", s.Planejada, len(s.Blocos))
	}
	// Refazer pelo modelo mantém os eventos.
	s = semanaRotina{}
	e.pedir("POST", "/api/rotina/semanas/2026-11-23?refazer=1", nil, &s)
	if len(s.Blocos) != len(rotinaPadrao().Modelo)+2 {
		t.Fatalf("refeita: %d", len(s.Blocos))
	}

	// O mês: as semanas planejadas no intervalo.
	var lista []semanaRotina
	if st := e.pedir("GET", "/api/rotina/semanas?de=2026-11-02&ate=2026-12-06", nil, &lista); st != http.StatusOK || len(lista) != 1 || lista[0].Inicio != "2026-11-23" {
		t.Fatalf("mês: %d %+v", st, lista)
	}

	ruins := []map[string]string{
		{"data": "26/11", "inicio": "07:00", "fim": "08:00", "titulo": "x"},
		{"data": "2026-11-26", "inicio": "09:00", "fim": "08:00", "titulo": "x"},
		{"data": "2026-11-26", "inicio": "09:00", "fim": "10:00"},
	}
	for _, ruim := range ruins {
		if st := e.pedir("POST", "/api/rotina/eventos", map[string]any{"eventos": []map[string]string{ruim}}, nil); st != http.StatusBadRequest {
			t.Fatalf("%v: esperava 400, veio %d", ruim, st)
		}
	}
}

func TestFalarComOClaudeAdicionaOQueEleEntendeu(t *testing.T) {
	e := novoAmbiente(t)
	var pedido string
	e.app.ia = func(ctx context.Context, sistema, p, esquema string) (json.RawMessage, error) {
		pedido = p
		return json.RawMessage(`{"eventos":[{"data":"2026-11-12","inicio":"19:00","fim":"22:00","titulo":"Encontro com amigos","atividade":"","nota":""},
			{"data":"2026-11-14","inicio":"10:00","fim":"11:00","titulo":"Academia","atividade":"academia","nota":""}],"observacao":"Presumi a noite."}`), nil
	}
	var r respostaEventos
	if s := e.pedir("POST", "/api/rotina/falar", map[string]string{"texto": "quinta das 7 às 10 encontro com amigos e academia sábado às 10"}, &r); s != http.StatusCreated {
		t.Fatalf("falar: %d", s)
	}
	if !strings.Contains(pedido, "2026-11-12 quinta") || !strings.Contains(pedido, "encontro com amigos") {
		t.Fatalf("o pedido ao Claude não tem o calendário e a fala: %s", pedido)
	}
	if len(r.Eventos) != 2 || r.Observacao != "Presumi a noite." || r.Eventos[0].Dia != 3 || r.Eventos[1].Atividade != "academia" {
		t.Fatalf("falar: %+v", r)
	}
	if strings.Join(r.Eventos[1].ChocaCom, ",") != "Trabalho" {
		t.Fatalf("academia às 10 no sábado choca com o trabalho das 9 às 12: %+v", r.Eventos[1])
	}

	e.app.ia = nil
	if s := e.pedir("POST", "/api/rotina/falar", map[string]string{"texto": "x"}, nil); s != http.StatusServiceUnavailable {
		t.Fatalf("sem IA: %d", s)
	}
}

func TestSugerirUmaSemanaFicaSoComOsBlocosValidos(t *testing.T) {
	e := novoAmbiente(t)
	var pedido string
	e.app.ia = func(ctx context.Context, sistema, p, esquema string) (json.RawMessage, error) {
		pedido = p
		return json.RawMessage(`{"explicacao":"Mais programação.","blocos":[
			{"dia":0,"inicio":"07:00","fim":"08:00","atividade":"ingles","nota":""},
			{"dia":0,"inicio":"09:00","fim":"08:00","atividade":"ingles","nota":""},
			{"dia":1,"inicio":"07:00","fim":"08:00","atividade":"inventada","nota":""}]}`), nil
	}
	ativs := rotinaPadrao().Atividades
	corpo := map[string]any{"atividades": ativs, "comentario": "prioriza programação", "anterior": rotinaPadrao().Modelo[:2]}
	var r struct {
		Blocos     []bloco `json:"blocos"`
		Explicacao string  `json:"explicacao"`
	}
	if s := e.pedir("POST", "/api/rotina/sugerir", corpo, &r); s != http.StatusOK || len(r.Blocos) != 1 || r.Explicacao == "" {
		t.Fatalf("sugerir: %d %+v", s, r)
	}
	if !strings.Contains(pedido, "prioriza programação") || !strings.Contains(pedido, "Sugestão anterior") || !strings.Contains(pedido, "futebol, Futebol, 1–1, prioridade 2, fixa") {
		t.Fatalf("pedido: %s", pedido)
	}
}
