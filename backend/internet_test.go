package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// H15: três falhas seguidas abrem a queda, duas boas fecham; uma falha ou uma boa solta não contam.
func TestQuedasDe(t *testing.T) {
	inicio := time.Date(2026, 11, 18, 23, 0, 0, 0, time.UTC)
	var ms []medidaNet
	add := func(causas ...string) {
		for _, c := range causas {
			ms = append(ms, medidaNet{Sensor: sensorServidor, Em: hora(inicio.Add(time.Duration(len(ms)) * intervaloDaInternet)), Causa: c})
		}
	}
	add("", "claro", "claro", "") // piscada: não é queda
	add("claro", "claro", "roteador", "claro", "", "claro", "", "", "")
	qs := quedasDe(ms, inicio.Add(time.Hour))
	if len(qs) != 1 {
		t.Fatalf("quedas: %+v", qs)
	}
	q := qs[0]
	if q.Causa != causaClaro || q.Inicio != hora(inicio.Add(40*time.Second)) || q.Fim != hora(inicio.Add(100*time.Second)) || q.Segundos != 60 {
		t.Fatalf("queda: %+v", q)
	}

	add("dns", "dns", "dns", "dns")
	qs = quedasDe(ms, lerHora(ms[len(ms)-1].Em).Add(5*time.Second))
	if len(qs) != 2 || qs[1].Fim != "" || qs[1].Causa != causaDNS || qs[1].Segundos != 35 {
		t.Fatalf("aberta: %+v", qs)
	}

	// O sensor sumiu (Hub reiniciando): a queda fecha na última falha conhecida.
	ms = append(ms, medidaNet{Sensor: sensorServidor, Em: hora(lerHora(ms[len(ms)-1].Em).Add(5 * time.Minute))})
	qs = quedasDe(ms, inicio.Add(time.Hour))
	if len(qs) != 2 || qs[1].Fim == "" || qs[1].Segundos != 40 {
		t.Fatalf("buraco: %+v", qs)
	}
}

func TestCausaDaMedida(t *testing.T) {
	bom := medidaNet{Sensor: sensorServidor, Roteador: 1, Internet: 20, DNS: 3}
	casos := []struct {
		m    func(medidaNet) medidaNet
		wan  *estadoWAN
		quer string
	}{
		{func(m medidaNet) medidaNet { return m }, &estadoWAN{Ligada: true, IPv4: "187.65.178.135"}, ""},
		{func(m medidaNet) medidaNet { m.Roteador = -1; m.Internet = -1; return m }, nil, causaRoteador},
		{func(m medidaNet) medidaNet { return m }, &estadoWAN{Ligada: false}, causaClaro},
		{func(m medidaNet) medidaNet { m.Internet = -1; return m }, &estadoWAN{Erro: "sem acesso"}, causaClaro},
		{func(m medidaNet) medidaNet { m.DNS = -1; return m }, nil, causaDNS},
	}
	for i, c := range casos {
		if got := causaDaMedida(c.m(bom), c.wan); got != c.quer {
			t.Errorf("caso %d: %q, queria %q", i, got, c.quer)
		}
	}
	mac := medidaNet{Sensor: sensorMac, Roteador: 3, Internet: 20, Internet6: 22, DNS: 4}
	if c := causaDaMedida(mac, nil); c != "" {
		t.Errorf("mac bom: %q", c)
	}
	mac.Internet6 = -1
	if c := causaDaMedida(mac, nil); c != causaIPv6 {
		t.Errorf("mac sem IPv6: %q", c)
	}
	mac.Roteador = -1
	if c := causaDaMedida(mac, nil); c != causaWiFi {
		t.Errorf("mac sem roteador: %q", c)
	}
}

// roteadorFalso responde como o rpcd do OpenWrt: login, e o status da wan e da wan6 com a sessão.
type roteadorFalso struct {
	mu      sync.Mutex
	semIP   bool
	logins  int
	vencida bool // a próxima chamada responde "sem permissão", como uma sessão vencida
}

func (f *roteadorFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req struct {
		Params []json.RawMessage `json:"params"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var sessao, objeto string
	json.Unmarshal(req.Params[0], &sessao)
	json.Unmarshal(req.Params[1], &objeto)
	responder := func(res ...any) { json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": res}) }
	if objeto == "session" {
		var c map[string]string
		json.Unmarshal(req.Params[3], &c)
		if c["username"] != "hub" || c["password"] != "segredo" {
			responder(6)
			return
		}
		f.logins++
		responder(0, map[string]any{"ubus_rpc_session": "sessao-1", "expires": 300})
		return
	}
	if sessao != "sessao-1" || f.vencida {
		f.vencida = false
		responder(6)
		return
	}
	switch objeto {
	case "network.interface.wan":
		st := map[string]any{"up": true, "uptime": 3600, "route": []map[string]any{{"target": "0.0.0.0", "nexthop": "187.65.176.1"}}}
		if !f.semIP {
			st["ipv4-address"] = []map[string]any{{"address": "187.65.178.135", "mask": 22}}
		}
		responder(0, st)
	case "network.interface.wan6":
		responder(0, map[string]any{"ipv6-prefix": []map[string]any{{"address": "2804:14d:4e83:440e::", "mask": 64}}})
	}
}

// ntfyFalso guarda os títulos dos avisos.
type ntfyFalso struct {
	mu      sync.Mutex
	titulos []string
	chegou  chan struct{}
}

func (n *ntfyFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var c struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&c)
	n.mu.Lock()
	n.titulos = append(n.titulos, c.Title)
	n.mu.Unlock()
	select { // sem travar quando chegam mais avisos do que o teste espera
	case n.chegou <- struct{}{}:
	default:
	}
}

func (n *ntfyFalso) esperar(t *testing.T, quantos int) []string {
	t.Helper()
	for {
		n.mu.Lock()
		got := append([]string{}, n.titulos...)
		n.mu.Unlock()
		if len(got) >= quantos {
			return got
		}
		select {
		case <-n.chegou:
		case <-time.After(3 * time.Second):
			t.Fatalf("esperava %d avisos, chegaram %v", quantos, got)
		}
	}
}

type ambienteDaInternet struct {
	*ambiente
	roteador *roteadorFalso
	ntfy     *ntfyFalso
	falhar   map[string]bool // ip ou "dns" → falha
}

func comInternet(t *testing.T) *ambienteDaInternet {
	t.Helper()
	e := &ambienteDaInternet{ambiente: novoAmbiente(t), roteador: &roteadorFalso{}, ntfy: &ntfyFalso{chegou: make(chan struct{}, 10)}, falhar: map[string]bool{}}
	rot := httptest.NewServer(e.roteador)
	t.Cleanup(rot.Close)
	ntfy := httptest.NewServer(e.ntfy)
	t.Cleanup(ntfy.Close)

	registro := filepath.Join(t.TempDir(), "roteador.log")
	os.WriteFile(registro, []byte(strings.Join([]string{
		"2026-11-09T14:59:00-03:00 roteador hostapd: phy0-ap0: STA conectou",
		"2026-11-09T15:00:05-03:00 roteador netifd: Network device 'wan' link is down",
		"2026-11-09T15:00:06-03:00 roteador kernel: [  137.7] mt7530-mdio mdio-bus:1f wan: Link is Down",
		"2026-11-09T15:00:50-03:00 roteador netifd: wan (5506): udhcpc: lease of 187.65.178.135 obtained",
		"2026-11-09T16:00:00-03:00 roteador netifd: depois, fora do período",
	}, "\n")+"\n"), 0o644)

	e.app.cfg.internet = configInternet{ligada: true, roteador: rot.URL, usuario: "hub", senha: "segredo", alvos: []string{"1.1.1.1", "8.8.8.8"}, registro: registro}
	e.app.internet = novoSensor(e.app.cfg.internet)
	e.app.internet.pingar = func(_ context.Context, ip string) int {
		if e.falhar[ip] || (strings.HasPrefix(rot.URL, "http://"+ip)) && e.falhar["roteador"] {
			return -1
		}
		return 12
	}
	e.app.internet.resolver = func(context.Context, string) int {
		if e.falhar["dns"] {
			return -1
		}
		return 3
	}
	e.app.avisos = novoAvisador(ntfy.URL + "/rede")
	return e
}

func (e *ambienteDaInternet) rodadas(n int) {
	for range n {
		e.app.rodadaDaInternet(context.Background())
		e.agora = e.agora.Add(intervaloDaInternet)
	}
}

func TestInternetAchaAQuedaDaClaroEAvisa(t *testing.T) {
	e := comInternet(t)
	e.rodadas(3)
	var r respostaInternet
	e.pedir("GET", "/api/internet", nil, &r)
	if !r.Ligada || r.Servidor == nil || r.Servidor.Causa != "" || r.Servidor.Gateway != 12 || r.WAN == nil || r.WAN.IPv4 != "187.65.178.135" || r.WAN.Prefixo6 != "2804:14d:4e83:440e::/64" {
		t.Fatalf("tudo certo: %+v %+v", r.Servidor, r.WAN)
	}
	if e.roteador.logins != 1 {
		t.Fatalf("entrou %d vezes no roteador", e.roteador.logins)
	}
	e.roteador.vencida = true // a sessão venceu: entra de novo sozinho
	e.rodadas(1)
	if e.roteador.logins != 2 {
		t.Fatalf("não renovou a sessão: %d logins", e.roteador.logins)
	}

	// A Claro some: o roteador responde, mas fica sem IP e a internet não responde.
	e.roteador.semIP = true
	e.falhar["1.1.1.1"], e.falhar["8.8.8.8"] = true, true
	e.rodadas(7) // 70 s
	if got := e.ntfy.esperar(t, 1); got[0] != "Internet caiu: Claro ou modem" {
		t.Fatalf("aviso: %v", got)
	}
	e.rodadas(3)

	e.roteador.semIP = false
	e.falhar = map[string]bool{}
	e.rodadas(3)
	got := e.ntfy.esperar(t, 2)
	if len(got) != 2 || got[1] != "Internet voltou" {
		t.Fatalf("avisos: %v", got)
	}

	e.pedir("GET", "/api/internet", nil, &r)
	if len(r.Quedas) != 1 {
		t.Fatalf("quedas: %+v", r.Quedas)
	}
	q := r.Quedas[0]
	if q.Causa != causaClaro || q.Sensor != sensorServidor || q.Fim == "" || q.Segundos != 100 {
		t.Fatalf("queda: %+v", q)
	}
	if len(q.Registro) != 3 || !strings.Contains(q.Registro[0], "link is down") || !strings.Contains(q.Registro[2], "lease") {
		t.Fatalf("registro do roteador: %q", q.Registro)
	}

	// Mais rodadas não avisam de novo nem duplicam a queda.
	e.rodadas(5)
	e.pedir("GET", "/api/internet", nil, &r)
	if len(r.Quedas) != 1 || len(e.ntfy.titulos) != 2 {
		t.Fatalf("repetiu: %d quedas, avisos %v", len(r.Quedas), e.ntfy.titulos)
	}
}

func TestInternetPeloWiFiDoMac(t *testing.T) {
	e := comInternet(t)
	e.rodadas(20) // o servidor, no cabo, com internet o tempo todo

	inicio := e.agora.Add(-200 * time.Second)
	lote := func(de time.Time, n int, m medidaNet) []medidaNet {
		var ms []medidaNet
		for i := range n {
			m.Em = hora(de.Add(time.Duration(i) * intervaloDaInternet))
			ms = append(ms, m)
		}
		return ms
	}
	bom := medidaNet{Roteador: 4, Internet: 20, Internet6: 21, DNS: 3}
	semInternet := medidaNet{Roteador: 4, Internet: -1, Internet6: -1, DNS: 3}
	var ms []medidaNet
	ms = append(ms, lote(inicio, 2, bom)...)
	ms = append(ms, lote(inicio.Add(20*time.Second), 9, semInternet)...) // 90 s sem internet só no Wi-Fi
	ms = append(ms, lote(inicio.Add(110*time.Second), 3, bom)...)
	if s := e.pedir("POST", "/api/internet/medidas", ms, nil); s != http.StatusNoContent {
		t.Fatalf("status %d", s)
	}
	if got := e.ntfy.esperar(t, 1); got[0] != "Internet voltou" {
		t.Fatalf("avisos: %v", got)
	}

	var r respostaInternet
	e.pedir("GET", "/api/internet", nil, &r)
	if r.Mac == nil || r.Mac.Causa != "" {
		t.Fatalf("última do Mac: %+v", r.Mac)
	}
	if len(r.Quedas) != 1 || r.Quedas[0].Sensor != sensorMac || r.Quedas[0].Causa != causaSoWiFi || r.Quedas[0].Segundos != 90 {
		t.Fatalf("quedas: %+v", r.Quedas)
	}

	// O mesmo lote de novo (o Mac reenviou) não duplica nada.
	e.pedir("POST", "/api/internet/medidas", ms, nil)
	e.pedir("GET", "/api/internet", nil, &r)
	if len(r.Quedas) != 1 {
		t.Fatalf("duplicou: %+v", r.Quedas)
	}
}

func TestInternetDesligada(t *testing.T) {
	e := novoAmbiente(t)
	var r respostaInternet
	if s := e.pedir("GET", "/api/internet", nil, &r); s != http.StatusOK || r.Ligada || len(r.Quedas) != 0 {
		t.Fatalf("status %d: %+v", s, r)
	}
}

// O caso de 19/11: o servidor perde o roteador, o Mac continua alcançando o roteador e a internet,
// mas fica sem DNS (o AdGuard está no servidor). A causa é o servidor, nos dois.
func TestInternetServidorForaDaRede(t *testing.T) {
	e := comInternet(t)
	e.rodadas(2)
	inicio := e.agora
	e.falhar["roteador"], e.falhar["1.1.1.1"], e.falhar["8.8.8.8"] = true, true, true
	e.app.internet.pingar = func(_ context.Context, ip string) int { return -1 }
	e.rodadas(9)
	e.app.internet.pingar = func(_ context.Context, ip string) int { return 12 }
	e.rodadas(3)

	var ms []medidaNet
	for i := range 12 {
		m := medidaNet{Roteador: 3, Internet: 28, Internet6: 30, DNS: 90, Em: hora(inicio.Add(time.Duration(i) * intervaloDaInternet))}
		if i >= 1 && i <= 9 {
			m.DNS = -1
		}
		ms = append(ms, m)
	}
	e.pedir("POST", "/api/internet/medidas", ms, nil)

	var r respostaInternet
	e.pedir("GET", "/api/internet", nil, &r)
	if len(r.Quedas) != 2 {
		t.Fatalf("quedas: %+v", r.Quedas)
	}
	for _, q := range r.Quedas {
		if q.Causa != causaServidor {
			t.Errorf("%s: causa %q, queria servidor", q.Sensor, q.Causa)
		}
	}
}

// 20/11: a janela de 24 h apagava o começo de uma queda antiga medida por medida, e cada pedaço
// que sobrava virava uma queda "nova" com aviso de "voltou" — 30 avisos de uma queda só.
func TestQuedaAntigaSaindoDaJanelaNaoAvisaDeNovo(t *testing.T) {
	e := comInternet(t)
	t0 := e.agora
	var ms []medidaNet
	for i := range 90 {
		c := ""
		if i >= 30 && i < 60 {
			c = causaClaro
		}
		ms = append(ms, medidaNet{Sensor: sensorServidor, Em: hora(t0.Add(time.Duration(i) * intervaloDaInternet)), Causa: c})
	}
	if err := e.app.gravarMedidas(ms); err != nil {
		t.Fatal(err)
	}
	e.agora = t0.Add(20 * time.Minute)
	if err := e.app.atualizarQuedas(sensorServidor); err != nil {
		t.Fatal(err)
	}
	e.ntfy.esperar(t, 1)

	// Um dia depois, a janela passa por cima do começo da queda, 10 s de cada vez.
	for e.agora = t0.Add(24*time.Hour + 4*time.Minute); e.agora.Before(t0.Add(24*time.Hour + 11*time.Minute)); e.agora = e.agora.Add(intervaloDaInternet) {
		if err := e.app.atualizarQuedas(sensorServidor); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	e.app.db.QueryRow(`SELECT COUNT(*) FROM internet_quedas`).Scan(&n)
	time.Sleep(100 * time.Millisecond)
	e.ntfy.mu.Lock()
	avisos := len(e.ntfy.titulos)
	e.ntfy.mu.Unlock()
	if n != 1 || avisos != 1 {
		t.Fatalf("%d quedas gravadas e %d avisos; esperava 1 e 1", n, avisos)
	}
}
