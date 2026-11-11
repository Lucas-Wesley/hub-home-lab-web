package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Claude (H11): as duas contas Pro que dão a IA do Acervo, do Flashcards e do Streaming. O Hub guarda
// o token de cada uma, mostra o uso das janelas de 5 horas e da semana e troca a conta ativa quando o
// Lucas pede; nunca sozinho. A conta ativa vai para um arquivo que os três apps leem a cada chamada:
// a troca vale na próxima tarefa, sem reiniciar nada. É a terceira exceção ao "só lê".

const (
	contaPessoal       = "pessoal"
	contaEmpresa       = "empresa"
	validadeDoToken    = 365 * 24 * time.Hour // o do "claude setup-token" dura um ano
	medidaVelha        = 10 * time.Minute
	limiteQuaseNoFim   = 0.9
	avisarVencimentoEm = 30 * 24 * time.Hour
)

// As contas são fixas: a pessoal é a principal do homelab; a da empresa, para quando ela acabar.
var nomesDasContas = []struct{ id, nome string }{
	{contaPessoal, "Pessoal"},
	{contaEmpresa, "Empresa"},
}

func nomeDaConta(id string) string {
	for _, c := range nomesDasContas {
		if c.id == id {
			return c.nome
		}
	}
	return id
}

// O que fica no disco (contas.json, só o Hub lê). O token nunca sai pela API.
type contaGuardada struct {
	Token     string `json:"token"`
	GravadoEm string `json:"gravado_em"`
}

type contasGuardadas struct {
	Ativa  string                   `json:"ativa"`
	Contas map[string]contaGuardada `json:"contas"`
}

type janela struct {
	Uso    float64 `json:"uso"`     // de 0 a 1
	ZeraEm string  `json:"zera_em"` // quando a janela recomeça
}

type usoDaConta struct {
	CincoHoras janela `json:"cinco_horas"`
	Semana     janela `json:"semana"`
	Esgotada   bool   `json:"esgotada"` // o Claude recusa até a janela zerar
}

type medida struct {
	uso    *usoDaConta
	erro   string
	quando time.Time
}

// medidor faz a chamada mínima com o token e lê o uso que o Claude devolve; trocado nos testes.
type medidor func(ctx context.Context, token string) (usoDaConta, error)

type contasDoClaude struct {
	pasta   string // contas.json e ativa/token (a pasta que os apps montam, só para leitura)
	medir   medidor
	relogio func() time.Time

	mu      sync.Mutex // contas.json e o arquivo da conta ativa
	medidas sync.Map   // id → medida
	medindo sync.Map   // id → *sync.Mutex: uma medida por conta de cada vez
}

func novasContasDoClaude(pasta, programa, casa string, relogio func() time.Time) *contasDoClaude {
	return &contasDoClaude{pasta: pasta, relogio: relogio, medir: medidorDaCLI(programa, casa)}
}

func (c *contasDoClaude) ler() (contasGuardadas, error) {
	g := contasGuardadas{Ativa: contaPessoal, Contas: map[string]contaGuardada{}}
	b, err := os.ReadFile(filepath.Join(c.pasta, "contas.json"))
	if errors.Is(err, os.ErrNotExist) {
		return g, nil
	}
	if err != nil {
		return g, err
	}
	if err := json.Unmarshal(b, &g); err != nil {
		return g, err
	}
	if g.Contas == nil {
		g.Contas = map[string]contaGuardada{}
	}
	return g, nil
}

// gravar salva as contas e o token da ativa. Os dois arquivos trocam de uma vez (rename): um app que
// lê no meio vê o token velho ou o novo, nunca metade.
func (c *contasDoClaude) gravar(g contasGuardadas) error {
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	if err := gravarDeUmaVez(filepath.Join(c.pasta, "contas.json"), b); err != nil {
		return err
	}
	return gravarDeUmaVez(filepath.Join(c.pasta, "ativa", "token"), []byte(g.Contas[g.Ativa].Token+"\n"))
}

func gravarDeUmaVez(caminho string, conteudo []byte) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return err
	}
	tmp := caminho + ".novo"
	if err := os.WriteFile(tmp, conteudo, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, caminho)
}

// medirSeVelha mede a conta se a última medida passou do prazo e devolve a mais nova.
func (c *contasDoClaude) medirSeVelha(ctx context.Context, id, token string, prazo time.Duration) medida {
	trava, _ := c.medindo.LoadOrStore(id, &sync.Mutex{})
	trava.(*sync.Mutex).Lock()
	defer trava.(*sync.Mutex).Unlock()
	if m, ok := c.medidas.Load(id); ok && c.relogio().Sub(m.(medida).quando) < prazo {
		return m.(medida)
	}
	m := medida{quando: c.relogio()}
	if u, err := c.medir(ctx, token); err != nil {
		m.erro = err.Error()
	} else {
		m.uso = &u
	}
	c.medidas.Store(id, m)
	return m
}

// ---------- A medida pelo Claude Code ----------

// medidorDaCLI pede "ok" ao Haiku, sem ferramentas nem instruções do Claude Code (gasta quase nada
// do limite), e lê o rate_limit_event que a saída stream-json traz: o uso das duas janelas da conta.
func medidorDaCLI(programa, casa string) medidor {
	return func(ctx context.Context, token string) (usoDaConta, error) {
		ctx, cancelar := context.WithTimeout(ctx, time.Minute)
		defer cancelar()
		cmd := exec.CommandContext(ctx, programa, "-p", "ok", "--model", "haiku",
			"--output-format", "stream-json", "--verbose", "--no-session-persistence", "--max-turns", "1",
			"--tools", "", "--system-prompt", "Responda ok.", "--strict-mcp-config", "--setting-sources", "")
		cmd.Dir = os.TempDir()
		// Só o necessário: o token da conta medida, nada do ambiente do Hub.
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + casa, "DISABLE_AUTOUPDATER=1", "CLAUDE_CODE_OAUTH_TOKEN=" + token}
		cmd.WaitDelay = 5 * time.Second
		var saida, erros bytes.Buffer
		cmd.Stdout, cmd.Stderr = &saida, &erros
		errExec := cmd.Run()
		if ctx.Err() != nil {
			return usoDaConta{}, errors.New("o Claude não respondeu em 1 minuto")
		}
		u, err := lerUsoDoClaude(saida.Bytes())
		if err != nil && errExec != nil {
			return u, fmt.Errorf("%w (%s)", err, primeiro(strings.TrimSpace(erros.String()), errExec.Error()))
		}
		return u, err
	}
}

// lerUsoDoClaude acha o uso das janelas nas linhas da saída; sem ele, explica o erro do Claude.
func lerUsoDoClaude(saida []byte) (usoDaConta, error) {
	type janelaDoClaude struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    int64   `json:"resetsAt"`
		Status      string  `json:"status"`
	}
	var u usoDaConta
	achou := false
	resultado := ""
	linhas := bufio.NewScanner(bytes.NewReader(saida))
	linhas.Buffer(make([]byte, 64<<10), 4<<20)
	for linhas.Scan() {
		var l struct {
			Type          string `json:"type"`
			IsError       bool   `json:"is_error"`
			Result        string `json:"result"`
			RateLimitInfo struct {
				Status  string                    `json:"status"`
				Janelas map[string]janelaDoClaude `json:"unifiedWindows"`
			} `json:"rate_limit_info"`
		}
		if json.Unmarshal(linhas.Bytes(), &l) != nil {
			continue
		}
		switch l.Type {
		case "rate_limit_event":
			achou = true
			cinco, semana := l.RateLimitInfo.Janelas["five_hour"], l.RateLimitInfo.Janelas["seven_day"]
			u.CincoHoras = janela{Uso: cinco.Utilization, ZeraEm: horaUnix(cinco.ResetsAt)}
			u.Semana = janela{Uso: semana.Utilization, ZeraEm: horaUnix(semana.ResetsAt)}
			u.Esgotada = l.RateLimitInfo.Status == "rejected" || cinco.Status == "rejected" || semana.Status == "rejected"
		case "result":
			if l.IsError {
				resultado = l.Result
			}
		}
	}
	if achou {
		return u, nil
	}
	if strings.Contains(resultado, "Not logged in") || strings.Contains(resultado, "/login") ||
		strings.Contains(resultado, "401") || strings.Contains(strings.ToLower(resultado), "invalid") {
		return u, errors.New("login recusado: o token venceu ou foi revogado")
	}
	if resultado != "" {
		return u, errors.New(primeiraLinhaDe(resultado))
	}
	return u, errors.New("o Claude não informou o uso")
}

func horaUnix(s int64) string {
	if s == 0 {
		return ""
	}
	return hora(time.Unix(s, 0))
}

func primeiraLinhaDe(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}

// ---------- O que a tela mostra ----------

type contaNaTela struct {
	ID          string      `json:"id"`
	Nome        string      `json:"nome"`
	Ativa       bool        `json:"ativa"`
	Configurada bool        `json:"configurada"`
	GravadoEm   string      `json:"gravado_em,omitempty"`
	VenceEm     string      `json:"vence_em,omitempty"`
	Uso         *usoDaConta `json:"uso"`
	MedidoEm    string      `json:"medido_em,omitempty"`
	Erro        string      `json:"erro,omitempty"`
}

// visao monta as duas contas; com medir, mede antes as que estão com a medida velha (em paralelo).
func (c *contasDoClaude) visao(ctx context.Context, medir bool) ([]contaNaTela, error) {
	g, err := c.ler()
	if err != nil {
		return nil, err
	}
	contas := make([]contaNaTela, len(nomesDasContas))
	var espera sync.WaitGroup
	for i, n := range nomesDasContas {
		guardada, ok := g.Contas[n.id]
		ct := contaNaTela{ID: n.id, Nome: n.nome, Ativa: g.Ativa == n.id, Configurada: ok && guardada.Token != ""}
		if ct.Configurada {
			ct.GravadoEm = guardada.GravadoEm
			if t := lerHora(guardada.GravadoEm); !t.IsZero() {
				ct.VenceEm = hora(t.Add(validadeDoToken))
			}
		}
		contas[i] = ct
		if ct.Configurada && medir {
			espera.Add(1)
			go func() {
				defer espera.Done()
				c.medirSeVelha(ctx, n.id, guardada.Token, time.Minute)
			}()
		}
	}
	espera.Wait()
	for i := range contas {
		if m, ok := c.medidas.Load(contas[i].ID); ok && contas[i].Configurada {
			contas[i].Uso, contas[i].Erro, contas[i].MedidoEm = m.(medida).uso, m.(medida).erro, hora(m.(medida).quando)
		}
	}
	return contas, nil
}

// problemasDoClaude: para a vigia (H9). Só pede ação: a conta ativa perto do limite, sem login, ou um
// token perto de vencer. A troca de conta continua sendo do Lucas.
func (c *contasDoClaude) problemas(ctx context.Context) []problema {
	g, err := c.ler()
	if err != nil {
		log.Printf("claude: %v", err)
		return nil
	}
	var ps []problema
	for _, n := range nomesDasContas {
		guardada, ok := g.Contas[n.id]
		if !ok || guardada.Token == "" {
			continue
		}
		if t := lerHora(guardada.GravadoEm); !t.IsZero() {
			if falta := t.Add(validadeDoToken).Sub(c.relogio()); falta < avisarVencimentoEm {
				ps = append(ps, problema{chave: "claude-vence-" + n.id, titulo: "Token do Claude perto de vencer",
					texto: fmt.Sprintf("O token da conta %s vence em %d dias. Gere outro com \"claude setup-token\" e cole no Hub, na tela Claude.", n.nome, max(0, int(falta.Hours()/24)))})
			}
		}
	}
	ativa, ok := g.Contas[g.Ativa]
	if !ok || ativa.Token == "" {
		return ps
	}
	nome := nomeDaConta(g.Ativa)
	m := c.medirSeVelha(ctx, g.Ativa, ativa.Token, medidaVelha)
	switch {
	case strings.HasPrefix(m.erro, "login recusado"):
		ps = append(ps, problema{chave: "claude-login", titulo: "Claude sem login",
			texto: fmt.Sprintf("A conta %s, a ativa, recusou o token. As tarefas de IA dos apps vão falhar. Cole um token novo ou troque de conta no Hub.", nome)})
	case m.uso == nil:
	case m.uso.Esgotada:
		ps = append(ps, problema{chave: "claude-limite", titulo: "Limite do Claude esgotado",
			texto: fmt.Sprintf("A conta %s, a ativa, chegou ao limite. Se não der para esperar, troque de conta no Hub.", nome)})
	case m.uso.CincoHoras.Uso >= limiteQuaseNoFim:
		ps = append(ps, problema{chave: "claude-5h", titulo: fmt.Sprintf("Claude em %.0f%% do limite de 5 horas", 100*m.uso.CincoHoras.Uso),
			texto: fmt.Sprintf("Conta %s, zera %s. Troque de conta no Hub se precisar.", nome, quandoZera(m.uso.CincoHoras.ZeraEm))})
	case m.uso.Semana.Uso >= limiteQuaseNoFim:
		ps = append(ps, problema{chave: "claude-semana", titulo: fmt.Sprintf("Claude em %.0f%% do limite da semana", 100*m.uso.Semana.Uso),
			texto: fmt.Sprintf("Conta %s, zera %s. Troque de conta no Hub se precisar.", nome, quandoZera(m.uso.Semana.ZeraEm))})
	}
	return ps
}

func quandoZera(iso string) string {
	t := lerHora(iso)
	if t.IsZero() {
		return "em breve"
	}
	fuso, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fuso = time.UTC
	}
	return t.In(fuso).Format("02/01 às 15:04")
}

// ---------- API ----------

func (a *app) comClaude(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.claude == nil {
			erroJSON(w, http.StatusNotFound, "As contas do Claude não estão ligadas ao Hub (falta CLAUDE_CONTAS).")
			return
		}
		h(w, r)
	}
}

func (a *app) handleClaude(w http.ResponseWriter, r *http.Request) {
	contas, err := a.claude.visao(r.Context(), r.URL.Query().Get("medir") == "1")
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contas": contas})
}

// handleContaAtiva: {"conta": "empresa"}. Os apps passam a usar essa conta na próxima chamada.
func (a *app) handleContaAtiva(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Conta string `json:"conta"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	a.claude.mu.Lock()
	g, err := a.claude.ler()
	if err == nil {
		if guardada, ok := g.Contas[corpo.Conta]; !ok || guardada.Token == "" {
			a.claude.mu.Unlock()
			erroJSON(w, http.StatusBadRequest, "Essa conta ainda não tem token: cole o token dela primeiro.")
			return
		}
		g.Ativa = corpo.Conta
		err = a.claude.gravar(g)
	}
	a.claude.mu.Unlock()
	if err != nil {
		falha(w, err)
		return
	}
	log.Printf("claude: conta ativa agora é %s", corpo.Conta)
	a.handleClaude(w, r)
}

// handleTokenDaConta: {"token": "..."}. Confere o token com o Claude antes de guardar.
func (a *app) handleTokenDaConta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("conta")
	if nomeDaConta(id) == id {
		erroJSON(w, http.StatusNotFound, "Conta desconhecida.")
		return
	}
	var corpo struct {
		Token string `json:"token"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	token := strings.Join(strings.Fields(corpo.Token), "")
	if len(token) < 40 {
		erroJSON(w, http.StatusBadRequest, "Token curto demais: confira se copiou inteiro.")
		return
	}
	u, err := a.claude.medir(r.Context(), token)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "O Claude não aceitou esse token: "+err.Error())
		return
	}
	a.claude.mu.Lock()
	g, err := a.claude.ler()
	if err == nil {
		g.Contas[id] = contaGuardada{Token: token, GravadoEm: a.agora()}
		if _, ok := g.Contas[g.Ativa]; !ok {
			g.Ativa = id
		}
		err = a.claude.gravar(g)
	}
	a.claude.mu.Unlock()
	if err != nil {
		falha(w, err)
		return
	}
	a.claude.medidas.Store(id, medida{uso: &u, quando: a.relogio()})
	log.Printf("claude: token da conta %s trocado", id)
	a.handleClaude(w, r)
}
