package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestArmazenamentoMedeCadaAppEJuntaOBanco(t *testing.T) {
	e := novoAmbiente(t)
	var r respostaArmazenamento
	if s := e.pedir("GET", "/api/armazenamento", nil, &r); s != http.StatusOK {
		t.Fatalf("status %d", s)
	}
	if r.Disco == nil || r.Disco.Total == 0 || r.Disco.Usado > r.Disco.Total {
		t.Fatalf("disco: %+v %s", r.Disco, r.DiscoErro)
	}
	if len(r.Apps) != 2 || r.Apps[0].Nome != "biblioteca" || r.Apps[0].Titulo != "Biblioteca Digital" {
		t.Fatalf("apps fora de ordem: %+v", r.Apps)
	}
	bib := r.Apps[0]
	// Os livros primeiro, depois o banco (.db e -wal num item só).
	if len(bib.Itens) != 2 || bib.Itens[0].Titulo != "Livros" || bib.Itens[1].Titulo != itemBanco {
		t.Fatalf("itens da biblioteca: %+v", bib.Itens)
	}
	if bib.Itens[0].Bytes < 300000 || bib.Itens[1].Bytes < 48000 || bib.Bytes != bib.Itens[0].Bytes+bib.Itens[1].Bytes {
		t.Fatalf("tamanhos: %+v", bib)
	}
	acervo := r.Apps[1]
	if acervo.Itens[0].Titulo != "Claude: histórico e configuração" || acervo.Itens[1].Titulo != itemOutros {
		t.Fatalf("itens do acervo: %+v", acervo.Itens)
	}

	d := r.Docker
	if d == nil {
		t.Fatalf("sem docker: %s", r.DockerErro)
	}
	// Só o cache que ninguém usa e não é compartilhado conta como liberável; só a imagem sem nome é solta.
	if d.CacheDeBuild != 14_642_571_481+1000 || d.CacheLiberavel != 14_642_571_481 || d.ImagensSoltas != 99770000 || d.Imagens != 1946260943 {
		t.Fatalf("docker: %+v", d)
	}
}

func TestArmazenamentoGuardaAMedidaPorUmMinuto(t *testing.T) {
	e := novoAmbiente(t)
	var antes respostaArmazenamento
	e.pedir("GET", "/api/armazenamento", nil, &antes)
	escrever(t, e.apps+"/flashcards/flashcards.db", 1000)

	var r respostaArmazenamento
	e.pedir("GET", "/api/armazenamento", nil, &r)
	if len(r.Apps) != 2 {
		t.Fatalf("mediu de novo antes de um minuto: %d apps", len(r.Apps))
	}
	e.pedir("GET", "/api/armazenamento?atualizar=1", nil, &r)
	if len(r.Apps) != 3 {
		t.Fatalf("?atualizar=1 não mediu de novo: %d apps", len(r.Apps))
	}
}

func TestArmazenamentoSemDocker(t *testing.T) {
	e := novoAmbiente(t)
	e.app.docker = nil
	var r respostaArmazenamento
	e.pedir("GET", "/api/armazenamento", nil, &r)
	if r.Docker != nil || !strings.Contains(r.DockerErro, "DOCKER_URL") || len(r.Apps) != 2 {
		t.Fatalf("sem docker, o resto deveria vir igual: %+v", r)
	}
}

func TestServidorLeVMEProxmox(t *testing.T) {
	e := novoAmbiente(t)
	var r respostaServidor
	e.pedir("GET", "/api/servidor", nil, &r)
	if r.VM == nil || r.VM.Memoria.Total != 8000000*1024 || r.VM.Memoria.Usado != 1500000*1024 || r.VM.Carga[1] != 0.20 || r.VM.LigadaHa != 18000 {
		t.Fatalf("vm: %+v %s", r.VM, r.VMErro)
	}
	if len(r.Containers) != 2 || r.Containers[0].Nome != "caddy" {
		t.Fatalf("containers: %+v", r.Containers)
	}
	p := r.Proxmox
	if p == nil {
		t.Fatalf("proxmox: %s", r.ProxmoxErro)
	}
	if p.No.CPUs != 12 || p.No.Memoria.Total != 16642998272 || !strings.HasPrefix(p.No.Versao, "pve-manager/9.2") {
		t.Fatalf("nó: %+v", p.No)
	}
	if len(p.Armazenamentos) != 2 || p.Armazenamentos[0].Nome != "local" || p.Armazenamentos[1].Usado != 20000000000 {
		t.Fatalf("armazenamentos: %+v", p.Armazenamentos)
	}
	if len(p.Maquinas) != 1 || p.Maquinas[0].Nome != "apps" || p.Maquinas[0].Tipo != "qemu" {
		t.Fatalf("máquinas: %+v", p.Maquinas)
	}
	// Sem permissão para os discos, vem um aviso e o resto.
	if len(p.Discos) != 0 || len(p.Avisos) != 1 || !strings.Contains(p.Avisos[0], "403") {
		t.Fatalf("avisos: %+v", p.Avisos)
	}
}

func TestProxmoxRecusaOutroCertificado(t *testing.T) {
	e := novoAmbiente(t)
	cfg := e.app.cfg.proxmox
	cfg.impressao = strings.Repeat("AB:", 31) + "AB"
	p, err := novoProxmox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	e.app.proxmox = p
	var r respostaServidor
	e.pedir("GET", "/api/servidor", nil, &r)
	if r.Proxmox != nil || !strings.Contains(r.ProxmoxErro, "impressão digital") {
		t.Fatalf("aceitou um certificado com outra impressão: %+v", r)
	}

	if _, err := novoProxmox(configProxmox{url: e.proxmox.URL, token: tokenDeTeste, impressao: "não é hex"}); err == nil {
		t.Fatal("aceitou uma impressão digital inválida")
	}
}

func TestLimparAgoraGravaOHistorico(t *testing.T) {
	e := novoAmbiente(t)
	var l limpeza
	if s := e.pedir("POST", "/api/limpeza", oQueLimpar{CacheDeBuild: true, ImagensSoltas: true}, &l); s != http.StatusOK {
		t.Fatalf("status %d", s)
	}
	if l.CacheLiberado != 14_642_571_481 || l.ImagensLiberado != 99770000 || l.Erro != "" || l.Origem != "manual" {
		t.Fatalf("limpeza: %+v", l)
	}
	if got := e.docker.limpezas(); len(got) != 2 || got[0] != "POST /v1.47/build/prune?all=true" || got[1] != "POST /v1.47/images/prune" {
		t.Fatalf("pedidos ao docker: %v", got)
	}

	var r respostaLimpeza
	e.pedir("GET", "/api/limpeza", nil, &r)
	if len(r.Historico) != 1 || r.Historico[0].CacheLiberado != l.CacheLiberado || r.Historico[0].TerminadaEm == "" || !r.DockerLigado {
		t.Fatalf("histórico: %+v", r)
	}

	var erro map[string]string
	if s := e.pedir("POST", "/api/limpeza", oQueLimpar{}, &erro); s != http.StatusBadRequest {
		t.Fatalf("limpar nada deveria falhar: %d", s)
	}
}

func TestLimpezaQueFalhaFicaNoHistorico(t *testing.T) {
	e := novoAmbiente(t)
	e.docker.falhar = true
	var l limpeza
	e.pedir("POST", "/api/limpeza", oQueLimpar{CacheDeBuild: true}, &l)
	if !strings.Contains(l.Erro, "Cache de build") || !strings.Contains(l.Erro, "500") {
		t.Fatalf("erro: %+v", l)
	}
	if got := e.docker.limpezas(); len(got) != 1 {
		t.Fatalf("limpou o que não foi pedido: %v", got)
	}
}

func TestUmaLimpezaPorVez(t *testing.T) {
	e := novoAmbiente(t)
	e.docker.esperar = make(chan struct{})
	pronto := make(chan int)
	go func() {
		pronto <- e.pedir("POST", "/api/limpeza", oQueLimpar{CacheDeBuild: true}, nil)
	}()
	// Espera a primeira chegar ao Docker.
	for i := 0; len(e.docker.limpezas()) == 0; i++ {
		if i > 200 {
			t.Fatal("a primeira limpeza não começou")
		}
		time.Sleep(5 * time.Millisecond)
	}
	var r respostaLimpeza
	e.pedir("GET", "/api/limpeza", nil, &r)
	if !r.EmAndamento {
		t.Fatal("a limpeza em andamento não aparece")
	}
	if s := e.pedir("POST", "/api/limpeza", oQueLimpar{CacheDeBuild: true}, nil); s != http.StatusConflict {
		t.Fatalf("segunda limpeza ao mesmo tempo: %d", s)
	}
	close(e.docker.esperar)
	if s := <-pronto; s != http.StatusOK {
		t.Fatalf("primeira limpeza: %d", s)
	}
}

func TestAgendaPadraoESalvar(t *testing.T) {
	e := novoAmbiente(t) // segunda, 9/11/2026, 15h em Brasília
	var r respostaLimpeza
	e.pedir("GET", "/api/limpeza", nil, &r)
	ag := r.Agenda
	if !ag.Ativa || ag.ACadaDias != 7 || ag.Hora != 4 || !ag.CacheDeBuild || !ag.ImagensSoltas {
		t.Fatalf("agenda padrão: %+v", ag)
	}
	if want := time.Date(2026, 11, 10, 4, 0, 0, 0, e.fuso); !lerHora(ag.ProximaEm).Equal(want) {
		t.Fatalf("próxima: %s, queria %s", ag.ProximaEm, want)
	}

	var salva agenda
	if s := e.pedir("PUT", "/api/limpeza/agenda", agenda{Ativa: true, ACadaDias: 2, Hora: 23, CacheDeBuild: true}, &salva); s != http.StatusOK {
		t.Fatalf("status %d", s)
	}
	if want := time.Date(2026, 11, 9, 23, 0, 0, 0, e.fuso); !lerHora(salva.ProximaEm).Equal(want) || salva.ImagensSoltas {
		t.Fatalf("agenda salva: %+v", salva)
	}

	e.pedir("PUT", "/api/limpeza/agenda", agenda{Ativa: false, ACadaDias: 2, Hora: 23}, &salva)
	if salva.Ativa || salva.ProximaEm != "" {
		t.Fatalf("agenda desligada: %+v", salva)
	}

	for _, ruim := range []agenda{{Ativa: true, ACadaDias: 5, Hora: 4, CacheDeBuild: true}, {Ativa: true, ACadaDias: 7, Hora: 24, CacheDeBuild: true}, {Ativa: true, ACadaDias: 7, Hora: 4}} {
		if s := e.pedir("PUT", "/api/limpeza/agenda", ruim, nil); s != http.StatusBadRequest {
			t.Fatalf("aceitou %+v: %d", ruim, s)
		}
	}
}

func TestAgendaRodaNaHoraEMarcaAProxima(t *testing.T) {
	e := novoAmbiente(t)
	e.pedir("PUT", "/api/limpeza/agenda", agenda{Ativa: true, ACadaDias: 2, Hora: 4, CacheDeBuild: true}, nil)
	ctx := t.Context()

	e.agora = time.Date(2026, 11, 10, 3, 59, 0, 0, e.fuso)
	e.app.conferirAgenda(ctx)
	if n := len(e.docker.limpezas()); n != 0 {
		t.Fatalf("limpou antes da hora: %d", n)
	}

	e.agora = time.Date(2026, 11, 10, 4, 0, 30, 0, e.fuso)
	e.app.conferirAgenda(ctx)
	e.app.conferirAgenda(ctx) // o minuto seguinte não repete
	if got := e.docker.limpezas(); len(got) != 1 || !strings.Contains(got[0], "build/prune") {
		t.Fatalf("limpezas: %v", got)
	}
	var r respostaLimpeza
	e.pedir("GET", "/api/limpeza", nil, &r)
	if want := time.Date(2026, 11, 12, 4, 0, 0, 0, e.fuso); !lerHora(r.Agenda.ProximaEm).Equal(want) {
		t.Fatalf("próxima: %s, queria %s", r.Agenda.ProximaEm, want)
	}
	if len(r.Historico) != 1 || r.Historico[0].Origem != "agenda" {
		t.Fatalf("histórico: %+v", r.Historico)
	}

	// Servidor desligado por uma semana: limpa uma vez e marca a próxima no futuro.
	e.agora = time.Date(2026, 11, 19, 10, 0, 0, 0, e.fuso)
	e.app.conferirAgenda(ctx)
	e.pedir("GET", "/api/limpeza", nil, &r)
	if want := time.Date(2026, 11, 20, 4, 0, 0, 0, e.fuso); !lerHora(r.Agenda.ProximaEm).Equal(want) || len(r.Historico) != 2 {
		t.Fatalf("depois de uma semana desligado: próxima %s, %d limpezas", r.Agenda.ProximaEm, len(r.Historico))
	}
}

func TestAgendaSemDockerNaoGravaNada(t *testing.T) {
	e := novoAmbiente(t)
	e.app.docker = nil
	e.agora = time.Date(2026, 11, 10, 5, 0, 0, 0, e.fuso)
	e.app.conferirAgenda(t.Context())
	var r respostaLimpeza
	e.pedir("GET", "/api/limpeza", nil, &r)
	if len(r.Historico) != 0 || r.DockerLigado {
		t.Fatalf("sem docker: %+v", r)
	}
	if s := e.pedir("POST", "/api/limpeza", oQueLimpar{CacheDeBuild: true}, nil); s != http.StatusServiceUnavailable {
		t.Fatalf("limpar sem docker: %d", s)
	}
}

func TestSugestoesEHealth(t *testing.T) {
	e := novoAmbiente(t)
	var h map[string]string
	if s := e.pedir("GET", "/api/health", nil, &h); s != http.StatusOK || h["version"] != "teste" {
		t.Fatalf("health: %d %v", s, h)
	}
	var s melhoria
	e.pedir("PUT", "/api/sugestoes/abcdefgh-1", map[string]string{"texto": "Mostrar a temperatura", "tela": "#/servidor"}, &s)
	e.pedir("PATCH", "/api/sugestoes/abcdefgh-1", map[string]string{"estado": "feita", "resposta": "Feito."}, &s)
	if s.Estado != "feita" || s.Tela != "#/servidor" {
		t.Fatalf("sugestão: %+v", s)
	}
}
