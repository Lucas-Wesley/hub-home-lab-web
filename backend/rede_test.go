package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

// adGuardFalso responde como a API do AdGuard Home e guarda a proteção e as regras personalizadas.
type adGuardFalso struct {
	mu       sync.Mutex
	protecao bool
	pausa    int // ms
	regras   []string
}

func (g *adGuardFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u, s, _ := r.BasicAuth(); u != "hub" || s != "segredo" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	responder := func(v any) { json.NewEncoder(w).Encode(v) }
	switch r.Method + " " + r.URL.Path {
	case "GET /control/status":
		responder(map[string]any{"protection_enabled": g.protecao, "protection_disabled_duration": g.pausa})
	case "GET /control/stats":
		responder(map[string]any{"num_dns_queries": 1200, "num_blocked_filtering": 150,
			"top_blocked_domains": []map[string]int{{"doubleclick.net": 90}, {"anuncios.exemplo.com": 60}}})
	case "GET /control/querylog":
		if r.URL.Query().Get("response_status") != "blocked" {
			http.Error(w, "só os bloqueados", http.StatusBadRequest)
			return
		}
		responder(map[string]any{"data": []map[string]any{
			{"client": "192.168.1.50", "client_info": map[string]string{"name": "S24"}, "question": map[string]string{"name": "doubleclick.net"}, "rule": "||doubleclick.net^", "time": "2026-11-11T01:25:55.9Z"},
			{"client": "192.168.1.51", "client_info": map[string]string{"name": ""}, "question": map[string]string{"name": "doubleclick.net"}, "rule": "||doubleclick.net^", "time": "2026-11-11T01:20:00Z"},
			{"client": "192.168.1.51", "client_info": map[string]string{"name": ""}, "question": map[string]string{"name": "anuncios.exemplo.com"}, "rule": "||exemplo.com^", "time": "2026-11-11T01:10:00Z"},
		}})
	case "GET /control/filtering/status":
		responder(map[string]any{"user_rules": g.regras})
	case "POST /control/filtering/set_rules":
		var c struct{ Rules []string }
		json.NewDecoder(r.Body).Decode(&c)
		g.regras = c.Rules
	case "POST /control/protection":
		var c struct {
			Enabled  bool
			Duration int
		}
		json.NewDecoder(r.Body).Decode(&c)
		g.protecao, g.pausa = c.Enabled, c.Duration
	default:
		http.NotFound(w, r)
	}
}

func comAdGuard(t *testing.T) (*ambiente, *adGuardFalso) {
	t.Helper()
	e := novoAmbiente(t)
	g := &adGuardFalso{protecao: true, regras: []string{"! Streamings", "@@||globo.com^", "||rastreador.exemplo^"}}
	srv := httptest.NewServer(g)
	t.Cleanup(srv.Close)
	e.app.adguard = novoAdGuard(configAdGuard{url: srv.URL, usuario: "hub", senha: "segredo", painel: "https://adguard"})
	return e, g
}

func TestRedeMostraOQueOAdGuardBloqueia(t *testing.T) {
	e, _ := comAdGuard(t)
	var r respostaRede
	if s := e.pedir("GET", "/api/rede", nil, &r); s != 200 || r.Rede == nil {
		t.Fatalf("GET /api/rede: %d %+v", s, r)
	}
	v := r.Rede
	if !r.Ligado || r.Painel != "https://adguard" || !v.Protecao || v.PausadoAte != "" || v.Consultas != 1200 || v.Bloqueadas != 150 {
		t.Errorf("visão: %+v", r)
	}
	if len(v.MaisBloqueados) != 2 || v.MaisBloqueados[0] != (contagem{"doubleclick.net", 90}) {
		t.Errorf("mais bloqueados: %+v", v.MaisBloqueados)
	}
	// Um por domínio, o mais novo, com o nome do aparelho quando o AdGuard sabe.
	if len(v.Recentes) != 2 || v.Recentes[0].Aparelho != "S24" || v.Recentes[1].Aparelho != "192.168.1.51" {
		t.Errorf("recentes: %+v", v.Recentes)
	}
	// Só as liberações na forma @@||dominio^; comentários e bloqueios ficam de fora.
	if !slices.Equal(v.Liberados, []string{"globo.com"}) {
		t.Errorf("liberados: %v", v.Liberados)
	}
}

func TestRedePausaEReligaOBloqueio(t *testing.T) {
	e, g := comAdGuard(t)
	var r respostaRede
	if s := e.pedir("POST", "/api/rede/protecao", map[string]any{"ligada": false, "minutos": 10}, &r); s != 200 {
		t.Fatalf("pausar: %d", s)
	}
	if g.protecao || g.pausa != 600000 || r.Rede.Protecao || r.Rede.PausadoAte == "" {
		t.Errorf("pausa de 10 min: falso %v %d, resposta %+v", g.protecao, g.pausa, r.Rede)
	}

	r = respostaRede{}
	e.pedir("POST", "/api/rede/protecao", map[string]any{"ligada": false, "minutos": 0}, &r)
	if g.protecao || g.pausa != 0 || r.Rede.PausadoAte != "" {
		t.Errorf("pausa até religar: %v %d %+v", g.protecao, g.pausa, r.Rede)
	}
	if ps := e.app.problemas(t.Context()); !slices.ContainsFunc(ps, func(p problema) bool { return p.chave == "adguard-pausado" }) {
		t.Errorf("a vigia devia lembrar da pausa sem prazo: %+v", ps)
	}

	r = respostaRede{}
	e.pedir("POST", "/api/rede/protecao", map[string]any{"ligada": true}, &r)
	if !g.protecao || !r.Rede.Protecao {
		t.Error("não religou")
	}
	if s := e.pedir("POST", "/api/rede/protecao", map[string]any{"ligada": false, "minutos": 5000}, nil); s != 400 {
		t.Errorf("pausa longa demais: %d", s)
	}
}

func TestRedeLiberaEBloqueiaDeNovo(t *testing.T) {
	e, g := comAdGuard(t)
	var r respostaRede
	if s := e.pedir("POST", "/api/rede/liberados", map[string]string{"dominio": " https://Anuncios.Exemplo.com/promo?x=1 "}, &r); s != 200 {
		t.Fatalf("liberar: %d", s)
	}
	e.pedir("POST", "/api/rede/liberados", map[string]string{"dominio": "anuncios.exemplo.com"}, &r) // de novo: não repete
	if !slices.Equal(g.regras, []string{"! Streamings", "@@||globo.com^", "||rastreador.exemplo^", "@@||anuncios.exemplo.com^"}) {
		t.Errorf("regras depois de liberar: %v", g.regras)
	}
	if !slices.Equal(r.Rede.Liberados, []string{"globo.com", "anuncios.exemplo.com"}) {
		t.Errorf("liberados: %v", r.Rede.Liberados)
	}

	if s := e.pedir("DELETE", "/api/rede/liberados/anuncios.exemplo.com", nil, &r); s != 200 {
		t.Fatalf("bloquear de novo: %d", s)
	}
	if !slices.Equal(g.regras, []string{"! Streamings", "@@||globo.com^", "||rastreador.exemplo^"}) {
		t.Errorf("regras depois de bloquear de novo: %v", g.regras)
	}

	for _, ruim := range []string{"", "nao tem ponto", "x^y.com", "||a.com^"} {
		if s := e.pedir("POST", "/api/rede/liberados", map[string]string{"dominio": ruim}, nil); s != 400 {
			t.Errorf("%q devia ser recusado: %d", ruim, s)
		}
	}
}

func TestRedeSemAdGuard(t *testing.T) {
	e := novoAmbiente(t)
	var r respostaRede
	if s := e.pedir("GET", "/api/rede", nil, &r); s != 200 || r.Ligado || r.Rede != nil {
		t.Errorf("sem credencial: %d %+v", s, r)
	}
	if s := e.pedir("POST", "/api/rede/protecao", map[string]any{"ligada": false}, nil); s != 404 {
		t.Errorf("pausar sem AdGuard: %d", s)
	}
}
