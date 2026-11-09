package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// clienteProxmox lê a API do Proxmox com um token só de leitura (papel PVEAuditor). O certificado
// do Proxmox é autoassinado: em vez de desligar a conferência, o painel aceita só o certificado
// com a impressão digital configurada (a mesma do README do homelab).
type clienteProxmox struct {
	base  string
	no    string
	token string
	http  *http.Client
}

func novoProxmox(cfg configProxmox) (*clienteProxmox, error) {
	impressao, err := hex.DecodeString(strings.ReplaceAll(strings.TrimSpace(cfg.impressao), ":", ""))
	if err != nil || len(impressao) != sha256.Size {
		return nil, errors.New("PROXMOX_FINGERPRINT precisa ser o SHA-256 do certificado do Proxmox (AA:BB:…)")
	}
	conferir := func(certs [][]byte, _ [][]*x509.Certificate) error {
		if len(certs) == 0 {
			return errors.New("o Proxmox não mandou certificado")
		}
		soma := sha256.Sum256(certs[0])
		if !bytes.Equal(soma[:], impressao) {
			return errors.New("o certificado do Proxmox não é o esperado (a impressão digital mudou)")
		}
		return nil
	}
	transporte := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify:    true, // a conferência é a da impressão digital, logo abaixo
		VerifyPeerCertificate: conferir,
	}}
	return &clienteProxmox{
		base:  strings.TrimRight(cfg.url, "/") + "/api2/json",
		no:    cfg.no,
		token: cfg.token,
		http:  &http.Client{Transport: transporte, Timeout: 15 * time.Second},
	}, nil
}

type visaoDoProxmox struct {
	No             noDoProxmox     `json:"no"`
	Armazenamentos []armazenamento `json:"armazenamentos"`
	Maquinas       []maquina       `json:"maquinas"`
	Discos         []discoFisico   `json:"discos"`
	Avisos         []string        `json:"avisos,omitempty"` // partes que não vieram (falta de permissão…)
}

type noDoProxmox struct {
	Nome     string  `json:"nome"`
	CPU      float64 `json:"cpu"` // fração de 0 a 1
	CPUs     int     `json:"cpus"`
	Modelo   string  `json:"modelo"`
	Memoria  uso     `json:"memoria"`
	Swap     uso     `json:"swap"`
	Raiz     uso     `json:"raiz"` // o disco do próprio Proxmox
	LigadoHa int64   `json:"ligado_ha"`
	Versao   string  `json:"versao"`
}

type armazenamento struct {
	Nome     string `json:"nome"`
	Tipo     string `json:"tipo"`
	Conteudo string `json:"conteudo"`
	Ativo    bool   `json:"ativo"`
	uso
}

type maquina struct {
	ID         int     `json:"id"`
	Nome       string  `json:"nome"`
	Tipo       string  `json:"tipo"` // qemu (VM) ou lxc (contêiner do Proxmox)
	Estado     string  `json:"estado"`
	CPU        float64 `json:"cpu"`
	CPUs       float64 `json:"cpus"`
	Memoria    int64   `json:"memoria"`
	MemoriaMax int64   `json:"memoria_max"`
	DiscoMax   int64   `json:"disco_max"` // tamanho do disco; o uso de dentro de uma VM o Proxmox não sabe
	LigadaHa   int64   `json:"ligada_ha"`
}

type discoFisico struct {
	Caminho  string `json:"caminho"`
	Modelo   string `json:"modelo"`
	Tipo     string `json:"tipo"`
	Tamanho  int64  `json:"tamanho"`
	Saude    string `json:"saude"`
	VidaUtil int    `json:"vida_util"` // % da vida útil que resta (SMART); -1 quando o disco não diz
}

func (p *clienteProxmox) visao(ctx context.Context) (visaoDoProxmox, error) {
	v := visaoDoProxmox{Armazenamentos: []armazenamento{}, Maquinas: []maquina{}, Discos: []discoFisico{}}

	// Sem o estado da máquina não há o que mostrar; o resto pode faltar com um aviso.
	var st struct {
		CPU     float64 `json:"cpu"`
		CPUInfo struct {
			CPUs  int    `json:"cpus"`
			Model string `json:"model"`
		} `json:"cpuinfo"`
		Memory     struct{ Total, Used, Free uint64 }  `json:"memory"`
		Swap       struct{ Total, Used, Free uint64 }  `json:"swap"`
		RootFS     struct{ Total, Used, Avail uint64 } `json:"rootfs"`
		Uptime     int64                               `json:"uptime"`
		PVEVersion string                              `json:"pveversion"`
	}
	if err := p.ler(ctx, "/nodes/"+p.no+"/status", &st); err != nil {
		return v, err
	}
	v.No = noDoProxmox{
		Nome: p.no, CPU: st.CPU, CPUs: st.CPUInfo.CPUs, Modelo: st.CPUInfo.Model,
		Memoria:  uso{Total: st.Memory.Total, Usado: st.Memory.Used, Livre: st.Memory.Free},
		Swap:     uso{Total: st.Swap.Total, Usado: st.Swap.Used, Livre: st.Swap.Free},
		Raiz:     uso{Total: st.RootFS.Total, Usado: st.RootFS.Used, Livre: st.RootFS.Avail},
		LigadoHa: st.Uptime, Versao: st.PVEVersion,
	}

	var storages []struct {
		Storage, Type, Content string
		Total, Used, Avail     uint64
		Active                 int
	}
	if err := p.ler(ctx, "/nodes/"+p.no+"/storage", &storages); err != nil {
		v.Avisos = append(v.Avisos, "Armazenamentos: "+err.Error())
	}
	for _, s := range storages {
		v.Armazenamentos = append(v.Armazenamentos, armazenamento{
			Nome: s.Storage, Tipo: s.Type, Conteudo: s.Content, Ativo: s.Active == 1,
			uso: uso{Total: s.Total, Usado: s.Used, Livre: s.Avail},
		})
	}
	sort.Slice(v.Armazenamentos, func(i, j int) bool { return v.Armazenamentos[i].Nome < v.Armazenamentos[j].Nome })

	var recursos []struct {
		VMID        int     `json:"vmid"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Status      string  `json:"status"`
		Node        string  `json:"node"`
		CPU         float64 `json:"cpu"`
		MaxCPU      float64 `json:"maxcpu"`
		Mem, MaxMem int64
		MaxDisk     int64 `json:"maxdisk"`
		Uptime      int64 `json:"uptime"`
	}
	if err := p.ler(ctx, "/cluster/resources?type=vm", &recursos); err != nil {
		v.Avisos = append(v.Avisos, "Máquinas: "+err.Error())
	}
	for _, r := range recursos {
		v.Maquinas = append(v.Maquinas, maquina{
			ID: r.VMID, Nome: r.Name, Tipo: r.Type, Estado: r.Status, CPU: r.CPU, CPUs: r.MaxCPU,
			Memoria: r.Mem, MemoriaMax: r.MaxMem, DiscoMax: r.MaxDisk, LigadaHa: r.Uptime,
		})
	}
	sort.Slice(v.Maquinas, func(i, j int) bool { return v.Maquinas[i].ID < v.Maquinas[j].ID })

	var discos []struct {
		DevPath string          `json:"devpath"`
		Model   string          `json:"model"`
		Type    string          `json:"type"`
		Size    int64           `json:"size"`
		Health  string          `json:"health"`
		Wearout json.RawMessage `json:"wearout"` // número, ou "N/A"
	}
	if err := p.ler(ctx, "/nodes/"+p.no+"/disks/list", &discos); err != nil {
		v.Avisos = append(v.Avisos, "Discos: "+err.Error())
	}
	for _, d := range discos {
		vida := -1
		var n float64
		if json.Unmarshal(d.Wearout, &n) == nil {
			vida = int(n)
		}
		v.Discos = append(v.Discos, discoFisico{Caminho: d.DevPath, Modelo: d.Model, Tipo: d.Type, Tamanho: d.Size, Saude: d.Health, VidaUtil: vida})
	}
	return v, nil
}

// ler faz um GET na API e decodifica o campo "data" da resposta.
func (p *clienteProxmox) ler(ctx context.Context, caminho string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+caminho, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "PVEAPIToken="+p.token)
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// A resposta de erro do Proxmox não traz o token; o Status já diz o motivo (401, 403…).
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s: %s", caminho, resp.Status)
	}
	var corpo struct{ Data json.RawMessage }
	if err := json.NewDecoder(resp.Body).Decode(&corpo); err != nil {
		return err
	}
	return json.Unmarshal(corpo.Data, destino)
}
