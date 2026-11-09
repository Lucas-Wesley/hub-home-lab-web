package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// vm: memória, carga e tempo ligada da VM apps, lidos do /proc dela. O contêiner vê o /proc da
// VM inteira (a memória não é a do contêiner), que é o que o painel quer.
type vm struct {
	Memoria  uso        `json:"memoria"`
	Carga    [3]float64 `json:"carga"` // média de processos na fila em 1, 5 e 15 minutos
	CPUs     int        `json:"cpus"`
	LigadaHa int64      `json:"ligada_ha"` // segundos
}

func lerVM(proc string) (vm, error) {
	v := vm{CPUs: runtime.NumCPU()}
	mem, err := lerMeminfo(filepath.Join(proc, "meminfo"))
	if err != nil {
		return v, err
	}
	// "Em uso" como no free: o que não está disponível, sem contar o cache que o sistema devolve.
	v.Memoria = uso{Total: mem["MemTotal"], Livre: mem["MemAvailable"]}
	v.Memoria.Usado = v.Memoria.Total - v.Memoria.Livre

	carga, err := os.ReadFile(filepath.Join(proc, "loadavg"))
	if err != nil {
		return v, err
	}
	campos := strings.Fields(string(carga))
	for i := 0; i < 3 && i < len(campos); i++ {
		v.Carga[i], _ = strconv.ParseFloat(campos[i], 64)
	}

	ligada, err := os.ReadFile(filepath.Join(proc, "uptime"))
	if err != nil {
		return v, err
	}
	if campos := strings.Fields(string(ligada)); len(campos) > 0 {
		s, _ := strconv.ParseFloat(campos[0], 64)
		v.LigadaHa = int64(s)
	}
	return v, nil
}

// lerMeminfo devolve os campos do /proc/meminfo em bytes.
func lerMeminfo(caminho string) (map[string]uint64, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	campos := map[string]uint64{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		nome, resto, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}
		partes := strings.Fields(resto)
		if len(partes) == 0 {
			continue
		}
		n, err := strconv.ParseUint(partes[0], 10, 64)
		if err != nil {
			continue
		}
		if len(partes) > 1 && partes[1] == "kB" {
			n *= 1024
		}
		campos[nome] = n
	}
	if campos["MemTotal"] == 0 {
		return nil, fmt.Errorf("%s sem MemTotal", caminho)
	}
	return campos, s.Err()
}

type respostaServidor struct {
	VM              *vm             `json:"vm"`
	VMErro          string          `json:"vm_erro,omitempty"`
	Containers      []container     `json:"containers"`
	DockerErro      string          `json:"docker_erro,omitempty"`
	Proxmox         *visaoDoProxmox `json:"proxmox"`
	ProxmoxErro     string          `json:"proxmox_erro,omitempty"`
	ProxmoxLigado   bool            `json:"proxmox_ligado"` // há um token configurado
	ProxmoxEndereco string          `json:"proxmox_endereco"`
}

// handleServidor: o Proxmox (máquina, discos, VMs) e a VM apps por dentro (memória, contêineres).
func (a *app) handleServidor(w http.ResponseWriter, r *http.Request) {
	resp := respostaServidor{Containers: []container{}, ProxmoxLigado: a.proxmox != nil, ProxmoxEndereco: a.cfg.proxmox.url}
	if v, err := lerVM(a.cfg.proc); err != nil {
		resp.VMErro = "Não deu para ler a memória da VM: " + err.Error()
	} else {
		resp.VM = &v
	}

	if a.docker == nil {
		resp.DockerErro = "O Docker não está ligado ao painel (falta DOCKER_URL)."
	} else if d, err := a.docker.uso(r.Context()); err != nil {
		resp.DockerErro = "O Docker não respondeu: " + err.Error()
	} else {
		resp.Containers = d.Containers
	}

	if a.proxmox != nil {
		p, err := a.proxmox.visao(r.Context())
		if err != nil {
			resp.ProxmoxErro = "O Proxmox não respondeu: " + err.Error()
		} else {
			resp.Proxmox = &p
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
