package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	tokenPessoal = "sk-ant-oat01-pessoal-0000000000000000000000000000000000000000"
	tokenEmpresa = "sk-ant-oat01-empresa-0000000000000000000000000000000000000000"
)

// claudeFalso responde pela CLI: o uso de cada token, ou o erro dele.
type claudeFalso struct {
	usos  map[string]usoDaConta
	erros map[string]error
	vezes int
}

func (f *claudeFalso) medir(_ context.Context, token string) (usoDaConta, error) {
	f.vezes++
	if err := f.erros[token]; err != nil {
		return usoDaConta{}, err
	}
	return f.usos[token], nil
}

func comClaude(t *testing.T) (*ambiente, *claudeFalso, string) {
	t.Helper()
	e := novoAmbiente(t)
	f := &claudeFalso{
		usos: map[string]usoDaConta{
			tokenPessoal: {CincoHoras: janela{Uso: 0.19, ZeraEm: "2026-11-09T20:00:00Z"}, Semana: janela{Uso: 0.32, ZeraEm: "2026-11-14T12:00:00Z"}},
			tokenEmpresa: {CincoHoras: janela{Uso: 0.05}, Semana: janela{Uso: 0.10}},
		},
		erros: map[string]error{},
	}
	pasta := t.TempDir()
	e.app.claude = &contasDoClaude{pasta: pasta, medir: f.medir, relogio: e.app.relogio}
	return e, f, pasta
}

type respostaClaude struct {
	Contas []contaNaTela `json:"contas"`
	Erro   string        `json:"erro"`
}

func tokenAtivo(t *testing.T, pasta string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(pasta, "ativa", "token"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

func TestClaudeGuardaOsTokensETrocaAContaAtiva(t *testing.T) {
	e, _, pasta := comClaude(t)
	var r respostaClaude
	if s := e.pedir("GET", "/api/claude", nil, &r); s != 200 || len(r.Contas) != 2 || r.Contas[0].Configurada || r.Contas[1].Configurada {
		t.Fatalf("sem contas: %d %+v", s, r)
	}

	r = respostaClaude{}
	if s := e.pedir("PUT", "/api/claude/contas/pessoal", map[string]string{"token": " " + tokenPessoal + "\n"}, &r); s != 200 {
		t.Fatalf("token da pessoal: %d %+v", s, r)
	}
	p := r.Contas[0]
	if !p.Configurada || !p.Ativa || p.Uso == nil || p.Uso.CincoHoras.Uso != 0.19 || p.GravadoEm == "" || p.VenceEm != "2027-11-09T18:00:00Z" {
		t.Errorf("pessoal: %+v", p)
	}
	if tokenAtivo(t, pasta) != tokenPessoal {
		t.Error("o arquivo dos apps devia ter o token da pessoal")
	}

	// A empresa entra sem virar a ativa; trocar é pedido à parte.
	e.pedir("PUT", "/api/claude/contas/empresa", map[string]string{"token": tokenEmpresa}, &r)
	if tokenAtivo(t, pasta) != tokenPessoal || r.Contas[1].Ativa {
		t.Error("guardar o token da empresa não devia trocar a conta ativa")
	}
	r = respostaClaude{}
	if s := e.pedir("POST", "/api/claude/ativa", map[string]string{"conta": "empresa"}, &r); s != 200 || !r.Contas[1].Ativa || r.Contas[0].Ativa {
		t.Fatalf("trocar para a empresa: %d %+v", s, r)
	}
	if tokenAtivo(t, pasta) != tokenEmpresa {
		t.Error("o arquivo dos apps devia ter o token da empresa")
	}

	// O token nunca sai pela API.
	var bruto map[string]any
	e.pedir("GET", "/api/claude", nil, &bruto)
	if s := strings.Join(slices.Collect(func(yield func(string) bool) {
		for _, c := range bruto["contas"].([]any) {
			for _, v := range c.(map[string]any) {
				if s, ok := v.(string); ok && !yield(s) {
					return
				}
			}
		}
	}), " "); strings.Contains(s, "sk-ant") {
		t.Errorf("a API mostrou um token: %s", s)
	}
	if info, _ := os.Stat(filepath.Join(pasta, "contas.json")); info.Mode().Perm() != 0o600 {
		t.Errorf("contas.json com permissão %v", info.Mode().Perm())
	}
}

func TestClaudeRecusaTokenRuimEContaSemToken(t *testing.T) {
	e, f, pasta := comClaude(t)
	f.erros["sk-ant-oat01-vencido-000000000000000000000000000000000000000000"] = errors.New("login recusado: o token venceu ou foi revogado")
	casos := []struct {
		caminho, campo, valor string
		status                int
	}{
		{"/api/claude/contas/pessoal", "token", "curto", 400},
		{"/api/claude/contas/pessoal", "token", "sk-ant-oat01-vencido-000000000000000000000000000000000000000000", 400},
		{"/api/claude/contas/outra", "token", tokenPessoal, 404},
	}
	for _, c := range casos {
		if s := e.pedir("PUT", c.caminho, map[string]string{c.campo: c.valor}, nil); s != c.status {
			t.Errorf("PUT %s %q: %d, queria %d", c.caminho, c.valor, s, c.status)
		}
	}
	if s := e.pedir("POST", "/api/claude/ativa", map[string]string{"conta": "empresa"}, nil); s != 400 {
		t.Errorf("ativar conta sem token: %d", s)
	}
	if _, err := os.Stat(filepath.Join(pasta, "contas.json")); err == nil {
		t.Error("nada devia ter sido gravado")
	}
}

func TestClaudeMedeSoQuandoAMedidaEstaVelha(t *testing.T) {
	e, f, _ := comClaude(t)
	e.pedir("PUT", "/api/claude/contas/pessoal", map[string]string{"token": tokenPessoal}, nil)
	f.vezes = 0
	e.pedir("GET", "/api/claude?medir=1", nil, nil)
	if f.vezes != 0 {
		t.Errorf("acabou de medir ao guardar o token: mediu de novo %d vez(es)", f.vezes)
	}
	e.agora = e.agora.Add(2 * time.Minute)
	e.pedir("GET", "/api/claude?medir=1", nil, nil)
	e.pedir("GET", "/api/claude", nil, nil) // sem medir=1, só o que já sabe
	if f.vezes != 1 {
		t.Errorf("mediu %d vezes, queria 1", f.vezes)
	}
}

func TestClaudeAvisaOQuePedeAcao(t *testing.T) {
	e, f, _ := comClaude(t)
	e.pedir("PUT", "/api/claude/contas/pessoal", map[string]string{"token": tokenPessoal}, nil)
	chaves := func() []string {
		e.agora = e.agora.Add(medidaVelha) // a vigia mede de novo
		var cs []string
		for _, p := range e.app.claude.problemas(t.Context()) {
			cs = append(cs, p.chave)
		}
		return cs
	}
	if cs := chaves(); len(cs) != 0 {
		t.Errorf("tudo bem, mas avisou: %v", cs)
	}
	f.usos[tokenPessoal] = usoDaConta{CincoHoras: janela{Uso: 0.93, ZeraEm: "2026-11-09T20:00:00Z"}, Semana: janela{Uso: 0.5}}
	if cs := chaves(); !slices.Equal(cs, []string{"claude-5h"}) {
		t.Errorf("5 horas no fim: %v", cs)
	}
	f.usos[tokenPessoal] = usoDaConta{CincoHoras: janela{Uso: 1}, Esgotada: true}
	if cs := chaves(); !slices.Equal(cs, []string{"claude-limite"}) {
		t.Errorf("esgotada: %v", cs)
	}
	f.erros[tokenPessoal] = errors.New("login recusado: o token venceu ou foi revogado")
	if cs := chaves(); !slices.Equal(cs, []string{"claude-login"}) {
		t.Errorf("sem login: %v", cs)
	}
	delete(f.erros, tokenPessoal)
	f.usos[tokenPessoal] = usoDaConta{}
	e.agora = e.agora.Add(340 * 24 * time.Hour)
	if cs := chaves(); !slices.Equal(cs, []string{"claude-vence-pessoal"}) {
		t.Errorf("perto de vencer: %v", cs)
	}
}

func TestLerUsoDoClaude(t *testing.T) {
	saida := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}
{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1794378600,"rateLimitType":"five_hour","unifiedWindows":{"five_hour":{"utilization":0.19,"resetsAt":1794378600,"status":"allowed"},"seven_day":{"utilization":0.32,"resetsAt":1794859200,"status":"allowed"}}}}
{"type":"result","is_error":false,"result":"ok","total_cost_usd":0.00006}`
	u, err := lerUsoDoClaude([]byte(saida))
	if err != nil || u.CincoHoras.Uso != 0.19 || u.Semana.Uso != 0.32 || u.Esgotada || u.CincoHoras.ZeraEm != hora(time.Unix(1794378600, 0)) {
		t.Errorf("uso: %+v %v", u, err)
	}

	_, err = lerUsoDoClaude([]byte(`{"type":"result","is_error":true,"result":"Invalid API key · Please run /login"}`))
	if err == nil || !strings.HasPrefix(err.Error(), "login recusado") {
		t.Errorf("login: %v", err)
	}
	u, _ = lerUsoDoClaude([]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","unifiedWindows":{"five_hour":{"utilization":1,"status":"rejected"},"seven_day":{"utilization":0.7}}}}`))
	if !u.Esgotada {
		t.Errorf("esgotada: %+v", u)
	}
}
