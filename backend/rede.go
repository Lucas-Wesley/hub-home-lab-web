package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// Rede (H10): o AdGuard Home, o DNS da casa. O Hub mostra o que ele bloqueia e tem os controles de
// quando algo quebra: pausar o bloqueio (o DNS continua respondendo, só sem bloquear, e tudo volta a
// abrir na hora para a casa toda) e liberar um domínio. É a segunda exceção ao "só lê" (a primeira
// é a limpeza, H5). O Hub entra com um usuário próprio do AdGuard, cuja senha fica no hub.env.

type configAdGuard struct {
	url     string // a API, por dentro (http://host.docker.internal:3001)
	usuario string
	senha   string // vazia = Hub sem a tela de Rede
	painel  string // o painel do AdGuard pelo Caddy, para o link
}

type clienteAdGuard struct {
	cfg  configAdGuard
	http *http.Client

	regras sync.Mutex // ler e gravar as regras personalizadas sem perder uma mudança no meio
}

func novoAdGuard(cfg configAdGuard) *clienteAdGuard {
	cfg.url = strings.TrimRight(cfg.url, "/")
	return &clienteAdGuard{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

func (g *clienteAdGuard) chamar(ctx context.Context, metodo, caminho string, corpo, destino any) error {
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, g.cfg.url+"/control"+caminho, leitor)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.cfg.usuario, g.cfg.senha)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: %s %s", caminho, resp.Status, strings.TrimSpace(string(msg)))
	}
	if destino == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}

// ---------- O que a tela mostra ----------

type contagem struct {
	Dominio string `json:"dominio"`
	Vezes   int    `json:"vezes"`
}

type bloqueio struct {
	Dominio  string `json:"dominio"`
	Aparelho string `json:"aparelho"`
	Quando   string `json:"quando"`
	Regra    string `json:"regra"`
}

type visaoDaRede struct {
	Protecao       bool       `json:"protecao"`
	PausadoAte     string     `json:"pausado_ate,omitempty"` // vazio com a proteção desligada = até religar
	Consultas      int        `json:"consultas"`             // nas últimas 24 h
	Bloqueadas     int        `json:"bloqueadas"`
	MaisBloqueados []contagem `json:"mais_bloqueados"`
	Recentes       []bloqueio `json:"recentes"` // um por domínio, o mais novo primeiro
	Liberados      []string   `json:"liberados"`
}

func (g *clienteAdGuard) estado(ctx context.Context) (protecao bool, pausadoAte time.Time, err error) {
	var s struct {
		ProtectionEnabled  bool  `json:"protection_enabled"`
		ProtectionDisabled int64 `json:"protection_disabled_duration"` // ms que faltam
	}
	if err := g.chamar(ctx, http.MethodGet, "/status", nil, &s); err != nil {
		return false, time.Time{}, err
	}
	if !s.ProtectionEnabled && s.ProtectionDisabled > 0 {
		pausadoAte = time.Now().Add(time.Duration(s.ProtectionDisabled) * time.Millisecond)
	}
	return s.ProtectionEnabled, pausadoAte, nil
}

func (g *clienteAdGuard) visao(ctx context.Context) (visaoDaRede, error) {
	v := visaoDaRede{MaisBloqueados: []contagem{}, Recentes: []bloqueio{}, Liberados: []string{}}
	protecao, ate, err := g.estado(ctx)
	if err != nil {
		return v, err
	}
	v.Protecao, v.PausadoAte = protecao, hora(ate)

	var st struct {
		Consultas  int              `json:"num_dns_queries"`
		Bloqueadas int              `json:"num_blocked_filtering"`
		Top        []map[string]int `json:"top_blocked_domains"`
	}
	if err := g.chamar(ctx, http.MethodGet, "/stats", nil, &st); err != nil {
		return v, err
	}
	v.Consultas, v.Bloqueadas = st.Consultas, st.Bloqueadas
	for _, m := range st.Top {
		for d, n := range m {
			v.MaisBloqueados = append(v.MaisBloqueados, contagem{d, n})
		}
		if len(v.MaisBloqueados) == 10 {
			break
		}
	}

	var log struct {
		Data []struct {
			Client     string `json:"client"`
			ClientInfo struct {
				Name string `json:"name"`
			} `json:"client_info"`
			Question struct {
				Name string `json:"name"`
			} `json:"question"`
			Rule string `json:"rule"`
			Time string `json:"time"`
		} `json:"data"`
	}
	if err := g.chamar(ctx, http.MethodGet, "/querylog?limit=200&response_status=blocked", nil, &log); err != nil {
		return v, err
	}
	vistos := map[string]bool{}
	for _, c := range log.Data {
		d := strings.TrimSuffix(c.Question.Name, ".")
		if vistos[d] || len(v.Recentes) == 30 {
			continue
		}
		vistos[d] = true
		quando, _ := time.Parse(time.RFC3339Nano, c.Time)
		v.Recentes = append(v.Recentes, bloqueio{Dominio: d, Aparelho: primeiro(c.ClientInfo.Name, c.Client), Quando: hora(quando), Regra: c.Rule})
	}

	regras, err := g.regrasPersonalizadas(ctx)
	if err != nil {
		return v, err
	}
	for _, r := range regras {
		if d, ok := dominioLiberado(r); ok {
			v.Liberados = append(v.Liberados, d)
		}
	}
	return v, nil
}

// ---------- Controles ----------

// pausar desliga o bloqueio por alguns minutos (0 = até religar). O DNS continua respondendo.
func (g *clienteAdGuard) pausar(ctx context.Context, minutos int) error {
	corpo := map[string]any{"enabled": false}
	if minutos > 0 {
		corpo["duration"] = minutos * 60 * 1000
	}
	return g.chamar(ctx, http.MethodPost, "/protection", corpo, nil)
}

func (g *clienteAdGuard) religar(ctx context.Context) error {
	return g.chamar(ctx, http.MethodPost, "/protection", map[string]any{"enabled": true}, nil)
}

func (g *clienteAdGuard) regrasPersonalizadas(ctx context.Context) ([]string, error) {
	var f struct {
		UserRules []string `json:"user_rules"`
	}
	err := g.chamar(ctx, http.MethodGet, "/filtering/status", nil, &f)
	return f.UserRules, err
}

// mudarRegras lê as regras personalizadas, aplica a mudança e grava. Regras em branco somem.
func (g *clienteAdGuard) mudarRegras(ctx context.Context, mudar func([]string) []string) error {
	g.regras.Lock()
	defer g.regras.Unlock()
	regras, err := g.regrasPersonalizadas(ctx)
	if err != nil {
		return err
	}
	novas := slices.DeleteFunc(mudar(regras), func(r string) bool { return strings.TrimSpace(r) == "" })
	return g.chamar(ctx, http.MethodPost, "/filtering/set_rules", map[string]any{"rules": novas}, nil)
}

// primeiro devolve o primeiro valor não vazio.
func primeiro(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

func regraDeLiberar(dominio string) string { return "@@||" + dominio + "^" }

// dominioLiberado reconhece só a forma que o Hub grava (@@||dominio^); as outras regras ficam para o
// painel do AdGuard.
func dominioLiberado(regra string) (string, bool) {
	if !strings.HasPrefix(regra, "@@||") || !strings.HasSuffix(regra, "^") {
		return "", false
	}
	d := strings.TrimSuffix(strings.TrimPrefix(regra, "@@||"), "^")
	return d, reDominio.MatchString(d)
}

var reDominio = regexp.MustCompile(`^([a-z0-9_]([a-z0-9_-]*[a-z0-9_])?\.)+[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// normalizarDominio aceita o domínio ou um endereço colado inteiro (https://x.com/caminho).
func normalizarDominio(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil {
			s = u.Hostname()
		}
	}
	s = strings.TrimSuffix(strings.SplitN(s, "/", 2)[0], ".")
	if !reDominio.MatchString(s) {
		return "", errors.New("Isso não parece um domínio (exemplo: anuncios.exemplo.com).")
	}
	return s, nil
}

// ---------- API ----------

func (a *app) adGuardOu404(w http.ResponseWriter) bool {
	if a.adguard == nil {
		erroJSON(w, http.StatusNotFound, "O AdGuard não está ligado ao Hub (falta ADGUARD_SENHA).")
		return false
	}
	return true
}

type respostaRede struct {
	Ligado bool         `json:"ligado"` // há credencial do AdGuard
	Painel string       `json:"painel"`
	Rede   *visaoDaRede `json:"rede"`
	Erro   string       `json:"erro,omitempty"`
}

func (a *app) handleRede(w http.ResponseWriter, r *http.Request) {
	resp := respostaRede{Ligado: a.adguard != nil}
	if a.adguard == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.Painel = a.adguard.cfg.painel
	if v, err := a.adguard.visao(r.Context()); err != nil {
		resp.Erro = "O AdGuard não respondeu: " + err.Error()
	} else {
		resp.Rede = &v
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleProtecao: {"ligada": true} religa; {"ligada": false, "minutos": 10} pausa (0 = até religar).
func (a *app) handleProtecao(w http.ResponseWriter, r *http.Request) {
	if !a.adGuardOu404(w) {
		return
	}
	var corpo struct {
		Ligada  bool `json:"ligada"`
		Minutos int  `json:"minutos"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	if corpo.Minutos < 0 || corpo.Minutos > 24*60 {
		erroJSON(w, http.StatusBadRequest, "A pausa vai de 1 minuto a 24 horas (ou 0, até religar).")
		return
	}
	var err error
	if corpo.Ligada {
		err = a.adguard.religar(r.Context())
	} else {
		err = a.adguard.pausar(r.Context(), corpo.Minutos)
	}
	if err != nil {
		falha(w, err)
		return
	}
	a.handleRede(w, r)
}

func (a *app) handleLiberar(w http.ResponseWriter, r *http.Request) {
	if !a.adGuardOu404(w) {
		return
	}
	var corpo struct {
		Dominio string `json:"dominio"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	d, err := normalizarDominio(corpo.Dominio)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	err = a.adguard.mudarRegras(r.Context(), func(regras []string) []string {
		if slices.Contains(regras, regraDeLiberar(d)) {
			return regras
		}
		return append(regras, regraDeLiberar(d))
	})
	if err != nil {
		falha(w, err)
		return
	}
	a.handleRede(w, r)
}

func (a *app) handleBloquearDeNovo(w http.ResponseWriter, r *http.Request) {
	if !a.adGuardOu404(w) {
		return
	}
	d, err := normalizarDominio(r.PathValue("dominio"))
	if err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	err = a.adguard.mudarRegras(r.Context(), func(regras []string) []string {
		return slices.DeleteFunc(regras, func(x string) bool { return x == regraDeLiberar(d) })
	})
	if err != nil {
		falha(w, err)
		return
	}
	a.handleRede(w, r)
}
