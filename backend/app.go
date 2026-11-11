package main

import (
	"database/sql"
	"os"
	"sync"
	"time"
)

type config struct {
	apps    string // pasta com os dados de cada app (/srv/data na VM, montada só para leitura)
	proc    string // /proc da VM: memória, carga e tempo ligada
	docker  string // endereço do proxy do Docker; vazio = painel sem Docker
	cgroup  string // cgroup v2 da VM: memória e processador de cada contêiner; vazio = sem
	proxmox configProxmox
	// Vigia (H9): o ntfy do homelab com o tópico (vazio, sem avisos), o endereço do Hub que a
	// notificação abre e o resultado do último backup (vazio, o backup não é conferido).
	ntfy   string
	link   string
	backup string
	// Rede (H10): a API do AdGuard Home, com o usuário do Hub.
	adguard configAdGuard
	// Claude (H11): a pasta com os tokens das contas (vazia = sem a tela Claude), o programa do
	// Claude Code e a pasta onde ele guarda as próprias configurações nas medidas.
	claudeContas   string
	claudePrograma string
	claudeCasa     string
}

type configProxmox struct {
	url       string // https://192.168.1.10:8006
	no        string // nome do node (pve)
	token     string // "usuario@realm!nome=segredo"; vazio = painel sem Proxmox
	impressao string // SHA-256 do certificado do Proxmox, que é autoassinado
}

func configDoAmbiente() config {
	return config{
		apps:   env("APPS_DIR", "./data/apps"),
		proc:   env("PROC_DIR", "/proc"),
		docker: os.Getenv("DOCKER_URL"),
		cgroup: os.Getenv("CGROUP_DIR"),
		proxmox: configProxmox{
			url:       env("PROXMOX_URL", "https://192.168.1.10:8006"),
			no:        env("PROXMOX_NODE", "pve"),
			token:     os.Getenv("PROXMOX_TOKEN"),
			impressao: os.Getenv("PROXMOX_FINGERPRINT"),
		},
		ntfy:   os.Getenv("NTFY_URL"),
		link:   env("HUB_LINK", "https://192.168.1.20:8449"),
		backup: os.Getenv("BACKUP_STATUS"),
		adguard: configAdGuard{
			url:     env("ADGUARD_URL", "http://host.docker.internal:3001"),
			usuario: env("ADGUARD_USUARIO", "hub"),
			senha:   os.Getenv("ADGUARD_SENHA"),
			painel:  env("ADGUARD_PAINEL", "https://192.168.1.20:8450"),
		},
		claudeContas:   os.Getenv("CLAUDE_CONTAS"),
		claudePrograma: env("CLAUDE_BIN", "claude"),
		claudeCasa:     env("CLAUDE_CASA", os.TempDir()),
	}
}

// app junta o que os handlers e a agenda de limpeza usam.
type app struct {
	db      *sql.DB
	fuso    *time.Location
	relogio func() time.Time
	cfg     config
	docker  *clienteDocker  // nil sem DOCKER_URL
	proxmox *clienteProxmox // nil sem PROXMOX_TOKEN
	avisos  *avisador       // nil sem NTFY_URL
	adguard *clienteAdGuard // nil sem ADGUARD_SENHA
	claude  *contasDoClaude // nil sem CLAUDE_CONTAS
	medidas *cacheDeMedidas
	amostra time.Duration // intervalo mínimo entre as duas leituras do processador

	limpando sync.Mutex // uma limpeza por vez, da agenda ou do botão
}

func novoApp(db *sql.DB, fuso *time.Location, relogio func() time.Time, cfg config) (*app, error) {
	a := &app{db: db, fuso: fuso, relogio: relogio, cfg: cfg, medidas: &cacheDeMedidas{}, amostra: time.Second, avisos: novoAvisador(cfg.ntfy)}
	if cfg.docker != "" {
		a.docker = novoDocker(cfg.docker)
	}
	if cfg.proxmox.token != "" {
		p, err := novoProxmox(cfg.proxmox)
		if err != nil {
			return nil, err
		}
		a.proxmox = p
	}
	if cfg.adguard.senha != "" {
		a.adguard = novoAdGuard(cfg.adguard)
	}
	if cfg.claudeContas != "" {
		a.claude = novasContasDoClaude(cfg.claudeContas, cfg.claudePrograma, cfg.claudeCasa, relogio)
	}
	if err := a.criarAgenda(); err != nil {
		return nil, err
	}
	return a, nil
}

func hora(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func lerHora(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func (a *app) agora() string {
	return hora(a.relogio())
}
