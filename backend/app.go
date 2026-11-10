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
	medidas *cacheDeMedidas
	amostra time.Duration // intervalo mínimo entre as duas leituras do processador

	limpando sync.Mutex // uma limpeza por vez, da agenda ou do botão
}

func novoApp(db *sql.DB, fuso *time.Location, relogio func() time.Time, cfg config) (*app, error) {
	a := &app{db: db, fuso: fuso, relogio: relogio, cfg: cfg, medidas: &cacheDeMedidas{}, amostra: time.Second}
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
