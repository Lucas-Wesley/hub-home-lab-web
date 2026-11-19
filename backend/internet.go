package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

// Internet (H15): a internet de casa caía e o Lucas tirava o modem e o roteador da tomada sem saber
// de quem era a culpa. Dois sensores medem a cada 10 s cada trecho do caminho e a tela Rede diz onde
// parou:
//
//   - o servidor (dentro do Hub, no cabo): roteador → gateway da Claro → internet → DNS, mais o
//     estado da porta wan lido no próprio roteador (usuário só de leitura no rpcd do OpenWrt);
//   - o Mac mini (no Wi-Fi, homelab/internet-mac): roteador → internet IPv4 e IPv6 → DNS. Ele manda
//     as medidas para o Hub e, sem rede, guarda e manda quando volta.
//
// Uma queda são três medidas seguidas com falha (30 s) e termina com duas boas. A causa é a falha
// mais vista nela. O ntfy avisa quedas de 1 minuto ou mais: "caiu" na hora (sensor do servidor) e
// "voltou" com a duração e a causa. O ntfy é da rede de casa, então o aviso chega pelo Wi-Fi mesmo
// com a internet fora.

const (
	intervaloDaInternet = 10 * time.Second
	esperaDoTeste       = 2 * time.Second
	falhasParaQueda     = 3
	boasParaVoltar      = 2
	quedaParaAvisar     = time.Minute
	buracoNasMedidas    = time.Minute // sem medida por mais que isso, a sequência recomeça
	guardarMedidas      = 3 * 24 * time.Hour
	naoMedido           = -2 // nos campos em ms; -1 = falhou
)

const (
	sensorServidor = "servidor"
	sensorMac      = "mac"
)

// Causas. "internet" só sai do Mac e vira outra na hora de mostrar: a do servidor, se ele também
// caiu, ou "so-wifi" (a casa tinha internet no cabo, mas o Wi-Fi não).
const (
	causaRoteador = "roteador"
	causaClaro    = "claro"
	causaDNS      = "dns"
	causaWiFi     = "wifi"
	causaInternet = "internet"
	causaSoWiFi   = "so-wifi"
	causaIPv6     = "ipv6"
)

type configInternet struct {
	ligada   bool   // INTERNET_VIGIA=1: mede (fora do servidor só recebe o que o Mac manda)
	roteador string // http://192.168.1.1
	usuario  string
	senha    string // vazia = sem ler a porta wan
	alvos    []string
	dns      string // o AdGuard, 192.168.1.20:53
	registro string // os registros do roteador gravados na VM (homelab/roteador)
}

type medidaNet struct {
	Sensor    string `json:"sensor"`
	Em        string `json:"em"`
	Roteador  int    `json:"roteador"` // ms; -1 falhou; -2 não medido
	Gateway   int    `json:"gateway"`
	Internet  int    `json:"internet"`
	Internet6 int    `json:"internet6"`
	DNS       int    `json:"dns"`
	Causa     string `json:"causa"` // vazia = tudo certo
}

type estadoWAN struct {
	Ligada   bool   `json:"ligada"` // o cabo do modem com sinal e a interface de pé
	IPv4     string `json:"ipv4"`
	Gateway  string `json:"gateway"`
	Desde    int    `json:"desde"` // segundos com a conexão de pé
	Prefixo6 string `json:"prefixo6"`
	Erro     string `json:"erro,omitempty"`
}

func causaDaMedida(m medidaNet, wan *estadoWAN) string {
	if m.Sensor == sensorMac {
		switch {
		case m.Roteador == -1:
			return causaWiFi
		case m.Internet == -1:
			return causaInternet
		case m.DNS == -1:
			return causaDNS
		case m.Internet6 == -1:
			return causaIPv6
		}
		return ""
	}
	switch {
	case m.Roteador == -1:
		return causaRoteador
	case wan != nil && wan.Erro == "" && (!wan.Ligada || wan.IPv4 == ""):
		return causaClaro
	case m.Internet == -1:
		return causaClaro
	case m.DNS == -1:
		return causaDNS
	}
	return ""
}

// ---------- Quedas ----------

type queda struct {
	Sensor   string   `json:"sensor"`
	Inicio   string   `json:"inicio"`
	Fim      string   `json:"fim,omitempty"` // vazio = ainda fora
	Causa    string   `json:"causa"`
	Segundos int      `json:"segundos"`
	Registro []string `json:"registro,omitempty"` // o que o roteador anotou no período
	avisada  int      // 0 nada, 1 avisou que caiu, 2 avisou que voltou
}

// quedasDe acha as quedas numa sequência de medidas de um sensor, em ordem.
func quedasDe(ms []medidaNet, agora time.Time) []queda {
	var qs []queda
	var atual *queda
	var falhas, boas int
	contagem := map[string]int{}
	var ultimaFalha, anterior, primeiraBoa time.Time

	fechar := func(fim time.Time) {
		if atual != nil {
			atual.Fim = hora(fim)
			atual.Segundos = int(fim.Sub(lerHora(atual.Inicio)).Seconds())
			atual.Causa = maisVista(contagem)
			qs = append(qs, *atual)
		}
		atual, falhas, boas, contagem = nil, 0, 0, map[string]int{}
	}
	var inicioDaSequencia time.Time
	for _, m := range ms {
		em := lerHora(m.Em)
		if !anterior.IsZero() && em.Sub(anterior) > buracoNasMedidas {
			fechar(ultimaFalha.Add(intervaloDaInternet)) // o sensor sumiu: fecha no que se sabe
		}
		anterior = em
		if m.Causa == "" {
			if boas++; boas == 1 {
				primeiraBoa = em
			}
			if atual != nil && boas >= boasParaVoltar {
				fechar(primeiraBoa) // voltou na primeira medida boa
			} else if atual == nil {
				falhas, contagem = 0, map[string]int{}
			}
			continue
		}
		boas = 0
		if falhas == 0 && atual == nil {
			inicioDaSequencia = em
		}
		falhas++
		contagem[m.Causa]++
		ultimaFalha = em
		if atual == nil && falhas >= falhasParaQueda {
			atual = &queda{Sensor: m.Sensor, Inicio: hora(inicioDaSequencia)}
		}
	}
	if atual != nil {
		atual.Causa = maisVista(contagem)
		atual.Segundos = int(agora.Sub(lerHora(atual.Inicio)).Seconds())
		qs = append(qs, *atual)
	}
	return qs
}

func maisVista(contagem map[string]int) string {
	var melhor string
	for c, n := range contagem {
		if n > contagem[melhor] || (n == contagem[melhor] && c < melhor) {
			melhor = c
		}
	}
	return melhor
}

// ---------- Banco ----------

const esquemaInternet = `
CREATE TABLE IF NOT EXISTS internet_medidas (
	sensor    TEXT NOT NULL,
	em        TEXT NOT NULL,
	roteador  INTEGER NOT NULL,
	gateway   INTEGER NOT NULL,
	internet  INTEGER NOT NULL,
	internet6 INTEGER NOT NULL,
	dns       INTEGER NOT NULL,
	causa     TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (sensor, em)
);
CREATE TABLE IF NOT EXISTS internet_quedas (
	sensor  TEXT NOT NULL,
	inicio  TEXT NOT NULL,
	fim     TEXT NOT NULL DEFAULT '',
	causa   TEXT NOT NULL,
	avisada INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (sensor, inicio)
);
`

func (a *app) gravarMedidas(ms []medidaNet) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range ms {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO internet_medidas (sensor, em, roteador, gateway, internet, internet6, dns, causa)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, m.Sensor, m.Em, m.Roteador, m.Gateway, m.Internet, m.Internet6, m.DNS, m.Causa); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *app) medidasDesde(sensor string, desde time.Time) ([]medidaNet, error) {
	rows, err := a.db.Query(`SELECT sensor, em, roteador, gateway, internet, internet6, dns, causa FROM internet_medidas
		WHERE sensor = ? AND em >= ? ORDER BY em`, sensor, hora(desde))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ms []medidaNet
	for rows.Next() {
		var m medidaNet
		if err := rows.Scan(&m.Sensor, &m.Em, &m.Roteador, &m.Gateway, &m.Internet, &m.Internet6, &m.DNS, &m.Causa); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, rows.Err()
}

// atualizarQuedas refaz as quedas das últimas 24 h do sensor, grava e avisa o que for novo.
func (a *app) atualizarQuedas(sensor string) error {
	agora := a.relogio()
	ms, err := a.medidasDesde(sensor, agora.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	for _, q := range quedasDe(ms, agora) {
		var avisada int
		var fimGravado string
		err := a.db.QueryRow(`SELECT avisada, fim FROM internet_quedas WHERE sensor = ? AND inicio = ?`, q.Sensor, q.Inicio).Scan(&avisada, &fimGravado)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		q.avisada = avisada
		nova := a.avisarQueda(&q)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = a.db.Exec(`INSERT INTO internet_quedas (sensor, inicio, fim, causa, avisada) VALUES (?, ?, ?, ?, ?)`, q.Sensor, q.Inicio, q.Fim, q.Causa, nova)
		} else {
			_, err = a.db.Exec(`UPDATE internet_quedas SET fim = ?, causa = ?, avisada = ? WHERE sensor = ? AND inicio = ?`, q.Fim, q.Causa, nova, q.Sensor, q.Inicio)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// avisarQueda manda o que falta avisar e devolve o novo estado do aviso.
func (a *app) avisarQueda(q *queda) int {
	if q.avisada == 2 || time.Duration(q.Segundos)*time.Second < quedaParaAvisar {
		return q.avisada
	}
	causa := a.causaParaMostrar(*q)
	if q.Fim == "" {
		if q.Sensor != sensorServidor || q.avisada == 1 {
			return q.avisada
		}
		a.avisos.avisar(aviso{Titulo: "Internet caiu: " + nomeDaCausa(causa), Texto: oQueFazer(causa), Link: a.cfg.link + "/#/rede", Urgente: true})
		return 1
	}
	texto := fmt.Sprintf("Ficou %s fora (%s, %s). %s", duracaoEmTexto(q.Segundos), nomeDaCausa(causa), quemMediu(q.Sensor), oQueFazerDepois(causa))
	a.avisos.avisar(aviso{Titulo: "Internet voltou", Texto: texto, Link: a.cfg.link + "/#/rede"})
	return 2
}

// causaParaMostrar resolve o "internet" do Mac com o que o servidor viu no mesmo período.
func (a *app) causaParaMostrar(q queda) string {
	if q.Sensor != sensorMac || q.Causa != causaInternet {
		return q.Causa
	}
	fim := q.Fim
	if fim == "" {
		fim = a.agora()
	}
	var causa string
	err := a.db.QueryRow(`SELECT causa FROM internet_quedas WHERE sensor = ? AND inicio <= ? AND (fim = '' OR fim >= ?) ORDER BY inicio DESC LIMIT 1`,
		sensorServidor, fim, q.Inicio).Scan(&causa)
	if err == nil {
		return causa
	}
	return causaSoWiFi
}

func nomeDaCausa(c string) string {
	switch c {
	case causaRoteador:
		return "roteador"
	case causaClaro:
		return "Claro ou modem"
	case causaDNS:
		return "DNS (AdGuard)"
	case causaWiFi:
		return "Wi-Fi"
	case causaSoWiFi:
		return "só no Wi-Fi"
	case causaIPv6:
		return "IPv6 da Claro"
	}
	return c
}

func oQueFazer(c string) string {
	switch c {
	case causaRoteador:
		return "O servidor não alcança o roteador. Reinicie só o roteador."
	case causaClaro:
		return "O roteador está de pé, mas a Claro não responde. Tire só o modem da tomada por 30 s; se não voltar em 5 min, é com a Claro."
	case causaDNS:
		return "A internet está de pé, mas o AdGuard não resolve nomes. Veja a tela Rede do Hub."
	}
	return "Veja a tela Rede do Hub."
}

func oQueFazerDepois(c string) string {
	switch c {
	case causaClaro:
		return "O roteador ficou de pé o tempo todo."
	case causaSoWiFi:
		return "A casa tinha internet pelo cabo; o problema foi no Wi-Fi do roteador."
	}
	return ""
}

func quemMediu(sensor string) string {
	if sensor == sensorMac {
		return "visto pelo Mac, no Wi-Fi"
	}
	return "visto pelo servidor, no cabo"
}

func duracaoEmTexto(s int) string {
	switch {
	case s < 60:
		return fmt.Sprintf("%d s", s)
	case s < 3600:
		return fmt.Sprintf("%d min", (s+30)/60)
	}
	return fmt.Sprintf("%d h %02d min", s/3600, (s%3600)/60)
}

func (a *app) apagarMedidasAntigas() {
	if _, err := a.db.Exec(`DELETE FROM internet_medidas WHERE em < ?`, hora(a.relogio().Add(-guardarMedidas))); err != nil {
		log.Printf("internet: apagar medidas antigas: %v", err)
	}
}

// ---------- O sensor do servidor ----------

type sensorDaInternet struct {
	cfg      configInternet
	roteador *clienteRoteador // nil sem senha
	pingar   func(ctx context.Context, ip string) int
	resolver func(ctx context.Context, nome string) int

	mu     sync.Mutex
	ultima *medidaNet
	wan    *estadoWAN
	rodada int
}

func novoSensor(cfg configInternet) *sensorDaInternet {
	s := &sensorDaInternet{cfg: cfg, pingar: pingar}
	s.resolver = func(ctx context.Context, nome string) int { return resolver(ctx, cfg.dns, nome) }
	if cfg.senha != "" {
		s.roteador = &clienteRoteador{url: strings.TrimRight(cfg.roteador, "/"), usuario: cfg.usuario, senha: cfg.senha,
			http: &http.Client{Timeout: esperaDoTeste}}
	}
	return s
}

func (s *sensorDaInternet) medir(ctx context.Context, agora time.Time) (medidaNet, *estadoWAN) {
	ctx, cancel := context.WithTimeout(ctx, 3*esperaDoTeste)
	defer cancel()
	m := medidaNet{Sensor: sensorServidor, Em: hora(agora), Gateway: naoMedido, Internet6: naoMedido}
	ipRoteador := strings.TrimPrefix(strings.TrimPrefix(s.cfg.roteador, "http://"), "https://")

	var wan *estadoWAN
	if s.roteador != nil {
		w := s.roteador.wan(ctx)
		wan = &w
	}
	s.mu.Lock()
	s.rodada++
	// Um nome aleatório a cada minuto obriga o AdGuard a perguntar lá fora; nos outros, um nome
	// comum (que ele responde do cache) só confere que ele está de pé.
	nome := "google.com"
	if s.rodada%6 == 1 {
		nome = fmt.Sprintf("vigia-%d.google.com", rand.IntN(1e9))
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	internet := []int{}
	teste := func(f func()) {
		wg.Add(1)
		go func() { defer wg.Done(); f() }()
	}
	teste(func() { m.Roteador = s.pingar(ctx, ipRoteador) })
	if wan != nil && wan.Gateway != "" {
		teste(func() { m.Gateway = s.pingar(ctx, wan.Gateway) })
	}
	for _, alvo := range s.cfg.alvos {
		teste(func() {
			ms := s.pingar(ctx, alvo)
			mu.Lock()
			internet = append(internet, ms)
			mu.Unlock()
		})
	}
	teste(func() { m.DNS = s.resolver(ctx, nome) })
	wg.Wait()

	m.Internet = -1
	for _, ms := range internet {
		if ms >= 0 && (m.Internet < 0 || ms < m.Internet) {
			m.Internet = ms
		}
	}
	m.Causa = causaDaMedida(m, wan)
	return m, wan
}

func (a *app) vigiarInternet(ctx context.Context) {
	if a.internet == nil || !a.cfg.internet.ligada {
		return
	}
	t := time.NewTicker(intervaloDaInternet)
	defer t.Stop()
	var rodadas int
	for {
		a.rodadaDaInternet(ctx)
		if rodadas++; rodadas%360 == 0 {
			a.apagarMedidasAntigas()
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (a *app) rodadaDaInternet(ctx context.Context) {
	m, wan := a.internet.medir(ctx, a.relogio())
	a.internet.mu.Lock()
	a.internet.ultima, a.internet.wan = &m, wan
	a.internet.mu.Unlock()
	if err := a.gravarMedidas([]medidaNet{m}); err != nil {
		log.Printf("internet: gravar: %v", err)
	} else if err := a.atualizarQuedas(sensorServidor); err != nil {
		log.Printf("internet: quedas: %v", err)
	}
}

// resolver pergunta ao DNS da casa; NXDOMAIN também é resposta (o nome aleatório não existe).
func resolver(ctx context.Context, servidor, nome string) int {
	r := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "udp", servidor)
	}}
	ctx, cancel := context.WithTimeout(ctx, esperaDoTeste)
	defer cancel()
	inicio := time.Now()
	_, err := r.LookupHost(ctx, nome)
	var dnsErr *net.DNSError
	if err != nil && !(errors.As(err, &dnsErr) && dnsErr.IsNotFound) {
		return -1
	}
	return max(1, int(time.Since(inicio).Milliseconds()))
}

// ---------- O roteador (rpcd do OpenWrt, só leitura) ----------

type clienteRoteador struct {
	url, usuario, senha string
	http                *http.Client

	mu     sync.Mutex
	sessao string
}

var errSemPermissao = errors.New("sem permissão")

func (r *clienteRoteador) chamar(ctx context.Context, sessao, objeto, metodo string, args any) (json.RawMessage, error) {
	corpo, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "call", "params": []any{sessao, objeto, metodo, args}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url+"/ubus", bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res struct {
		Result []json.RawMessage `json:"result"`
		Error  *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&res); err != nil {
		return nil, err
	}
	if res.Error != nil {
		if res.Error.Code == -32002 { // sessão vencida ou sem acesso
			return nil, errSemPermissao
		}
		return nil, fmt.Errorf("ubus: erro %d", res.Error.Code)
	}
	if len(res.Result) == 0 {
		return nil, errors.New("ubus: resposta vazia")
	}
	var codigo int
	json.Unmarshal(res.Result[0], &codigo)
	if codigo == 6 {
		return nil, errSemPermissao
	}
	if codigo != 0 {
		return nil, fmt.Errorf("ubus: código %d", codigo)
	}
	if len(res.Result) < 2 {
		return json.RawMessage("{}"), nil
	}
	return res.Result[1], nil
}

func (r *clienteRoteador) entrar(ctx context.Context) error {
	res, err := r.chamar(ctx, strings.Repeat("0", 32), "session", "login", map[string]string{"username": r.usuario, "password": r.senha})
	if err != nil {
		return fmt.Errorf("entrar no roteador: %w", err)
	}
	var s struct {
		Sessao string `json:"ubus_rpc_session"`
	}
	json.Unmarshal(res, &s)
	r.sessao = s.Sessao
	return nil
}

func (r *clienteRoteador) ler(ctx context.Context, objeto string, destino any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for tentativa := 0; tentativa < 2; tentativa++ {
		if r.sessao == "" {
			if err := r.entrar(ctx); err != nil {
				return err
			}
		}
		res, err := r.chamar(ctx, r.sessao, objeto, "status", map[string]any{})
		if errors.Is(err, errSemPermissao) {
			r.sessao = ""
			continue
		}
		if err != nil {
			return err
		}
		return json.Unmarshal(res, destino)
	}
	return errSemPermissao
}

func (r *clienteRoteador) wan(ctx context.Context) estadoWAN {
	var w struct {
		Up     bool `json:"up"`
		Uptime int  `json:"uptime"`
		IPv4   []struct {
			Address string `json:"address"`
		} `json:"ipv4-address"`
		Route []struct {
			Target  string `json:"target"`
			Nexthop string `json:"nexthop"`
		} `json:"route"`
	}
	if err := r.ler(ctx, "network.interface.wan", &w); err != nil {
		return estadoWAN{Erro: "Não consegui ler o roteador: " + err.Error()}
	}
	e := estadoWAN{Ligada: w.Up, Desde: w.Uptime}
	if len(w.IPv4) > 0 {
		e.IPv4 = w.IPv4[0].Address
	}
	for _, rt := range w.Route {
		if rt.Target == "0.0.0.0" {
			e.Gateway = rt.Nexthop
		}
	}
	var w6 struct {
		Prefixos []struct {
			Address string `json:"address"`
			Mask    int    `json:"mask"`
		} `json:"ipv6-prefix"`
	}
	if r.ler(ctx, "network.interface.wan6", &w6) == nil && len(w6.Prefixos) > 0 {
		e.Prefixo6 = fmt.Sprintf("%s/%d", w6.Prefixos[0].Address, w6.Prefixos[0].Mask)
	}
	return e
}

// ---------- Registros do roteador ----------

// linhasDoRoteador devolve o que o roteador anotou sobre a conexão entre dois momentos. O arquivo
// vem do rsyslog da VM (homelab/roteador), uma linha por registro começando pela hora RFC 3339.
func linhasDoRoteador(arquivo string, de, ate time.Time) []string {
	f, err := os.Open(arquivo)
	if err != nil {
		return nil
	}
	defer f.Close()
	var linhas []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		l := sc.Text()
		campos := strings.SplitN(l, " ", 3)
		if len(campos) < 3 {
			continue
		}
		t, err := time.Parse(time.RFC3339, campos[0])
		if err != nil || t.Before(de) || t.After(ate) || !registroImportante(campos[2]) {
			continue
		}
		linhas = append(linhas, t.Format(time.RFC3339)+" "+campos[2])
	}
	if len(linhas) > 25 {
		linhas = append(linhas[:12], append([]string{"…"}, linhas[len(linhas)-12:]...)...)
	}
	return linhas
}

func registroImportante(texto string) bool {
	for _, s := range []string{"netifd", "udhcpc", "odhcp6c", "Link is", "link is", "kernel", "watchdog", "reboot"} {
		if strings.Contains(texto, s) {
			return !strings.Contains(texto, "br-lan: port") && !strings.Contains(texto, "promiscuous")
		}
	}
	return false
}

// ---------- API ----------

type respostaInternet struct {
	Ligada    bool       `json:"ligada"`
	Servidor  *medidaNet `json:"servidor"`
	WAN       *estadoWAN `json:"wan"`
	Mac       *medidaNet `json:"mac"`
	Quedas    []queda    `json:"quedas"` // 30 dias, a mais nova primeiro
	Intervalo int        `json:"intervalo"`
}

func (a *app) handleInternet(w http.ResponseWriter, r *http.Request) {
	resp := respostaInternet{Ligada: a.internet != nil && a.cfg.internet.ligada, Quedas: []queda{}, Intervalo: int(intervaloDaInternet.Seconds())}
	if a.internet != nil {
		a.internet.mu.Lock()
		resp.Servidor, resp.WAN = a.internet.ultima, a.internet.wan
		a.internet.mu.Unlock()
	}
	var mac medidaNet
	err := a.db.QueryRow(`SELECT sensor, em, roteador, gateway, internet, internet6, dns, causa FROM internet_medidas
		WHERE sensor = ? ORDER BY em DESC LIMIT 1`, sensorMac).Scan(&mac.Sensor, &mac.Em, &mac.Roteador, &mac.Gateway, &mac.Internet, &mac.Internet6, &mac.DNS, &mac.Causa)
	if err == nil {
		resp.Mac = &mac
	}

	rows, err := a.db.Query(`SELECT sensor, inicio, fim, causa, avisada FROM internet_quedas WHERE inicio >= ? ORDER BY inicio DESC LIMIT 100`,
		hora(a.relogio().Add(-30*24*time.Hour)))
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var q queda
		if err := rows.Scan(&q.Sensor, &q.Inicio, &q.Fim, &q.Causa, &q.avisada); err != nil {
			falha(w, err)
			return
		}
		fim := lerHora(q.Fim)
		if q.Fim == "" {
			fim = a.relogio()
		}
		q.Segundos = int(fim.Sub(lerHora(q.Inicio)).Seconds())
		resp.Quedas = append(resp.Quedas, q)
	}
	for i := range resp.Quedas {
		q := &resp.Quedas[i]
		q.Causa = a.causaParaMostrar(*q)
		if i < 20 && a.cfg.internet.registro != "" {
			fim := lerHora(q.Fim)
			if q.Fim == "" {
				fim = a.relogio()
			}
			q.Registro = linhasDoRoteador(a.cfg.internet.registro, lerHora(q.Inicio).Add(-2*time.Minute), fim.Add(time.Minute))
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleMedidasDoMac recebe as medidas do sensor do Wi-Fi (homelab/internet-mac), em lote.
func (a *app) handleMedidasDoMac(w http.ResponseWriter, r *http.Request) {
	var ms []medidaNet
	if !lerCorpo(w, r, &ms) {
		return
	}
	if len(ms) > 50000 {
		erroJSON(w, http.StatusBadRequest, "Medidas demais num pedido só.")
		return
	}
	limite := a.relogio().Add(time.Minute)
	ms = slices.DeleteFunc(ms, func(m medidaNet) bool {
		t := lerHora(m.Em)
		return t.IsZero() || t.After(limite) || t.Before(limite.Add(-guardarMedidas))
	})
	for i := range ms {
		ms[i].Sensor = sensorMac
		ms[i].Em = hora(lerHora(ms[i].Em))
		ms[i].Gateway = naoMedido
		ms[i].Causa = causaDaMedida(ms[i], nil)
	}
	if err := a.gravarMedidas(ms); err != nil {
		falha(w, err)
		return
	}
	if err := a.atualizarQuedas(sensorMac); err != nil {
		falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
