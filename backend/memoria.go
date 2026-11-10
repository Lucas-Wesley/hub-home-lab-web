package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A memória e o processador de cada contêiner vêm do cgroup v2 da VM, montado só para leitura
// (CGROUP_DIR). O Docker põe cada contêiner em system.slice/docker-<id>.scope. Assim o proxy do
// Docker continua sem liberar /containers, que mostraria as variáveis de ambiente (as chaves).

// consumo: uma leitura do cgroup de um contêiner.
type consumo struct {
	memoria int64 // bytes, como o "docker stats": memory.current sem o cache de arquivos inativo
	cpuUsec int64 // tempo de processador acumulado
}

// lerCgroups lê todos os contêineres de uma vez, pelo id completo.
func lerCgroups(dir string) (map[string]consumo, error) {
	if _, err := os.Stat(filepath.Join(dir, "system.slice")); err != nil {
		return nil, err
	}
	pastas, err := filepath.Glob(filepath.Join(dir, "system.slice", "docker-*.scope"))
	if err != nil {
		return nil, err
	}
	leituras := map[string]consumo{}
	for _, p := range pastas {
		id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "docker-"), ".scope")
		c, err := lerCgroup(p)
		if err != nil {
			continue // o contêiner parou entre o Glob e a leitura
		}
		leituras[id] = c
	}
	return leituras, nil
}

func lerCgroup(pasta string) (consumo, error) {
	atual, err := os.ReadFile(filepath.Join(pasta, "memory.current"))
	if err != nil {
		return consumo{}, err
	}
	memoria, err := strconv.ParseInt(strings.TrimSpace(string(atual)), 10, 64)
	if err != nil {
		return consumo{}, err
	}
	stat, err := lerChaves(filepath.Join(pasta, "memory.stat"))
	if err != nil {
		return consumo{}, err
	}
	if inativo := stat["inactive_file"]; inativo < memoria {
		memoria -= inativo
	}
	cpu, err := lerChaves(filepath.Join(pasta, "cpu.stat"))
	if err != nil {
		return consumo{}, err
	}
	return consumo{memoria: memoria, cpuUsec: cpu["usage_usec"]}, nil
}

// lerChaves lê arquivos do cgroup no formato "nome valor" por linha.
func lerChaves(caminho string) (map[string]int64, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	chaves := map[string]int64{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		nome, valor, ok := strings.Cut(s.Text(), " ")
		if !ok {
			continue
		}
		if n, err := strconv.ParseInt(valor, 10, 64); err == nil {
			chaves[nome] = n
		}
	}
	return chaves, s.Err()
}

// amostrar lê os cgroups duas vezes, com pelo menos `intervalo` entre as leituras, para tirar o
// uso do processador. `enquanto` roda no meio (as leituras do Docker e do Proxmox), para a
// espera não somar ao tempo da resposta.
func amostrar(ctx context.Context, dir string, intervalo time.Duration, enquanto func()) (antes, depois map[string]consumo, segundos float64, err error) {
	inicio := time.Now()
	antes, err = lerCgroups(dir)
	enquanto()
	if err != nil {
		return nil, nil, 0, err
	}
	select {
	case <-time.After(intervalo - time.Since(inicio)):
	case <-ctx.Done():
	}
	depois, err = lerCgroups(dir)
	return antes, depois, time.Since(inicio).Seconds(), err
}

// memoriaDoApp junta os contêineres de um app (o projeto do Compose): o Streaming tem dois.
type memoriaDoApp struct {
	Nome       string   `json:"nome"`
	Titulo     string   `json:"titulo"`
	Memoria    int64    `json:"memoria"`
	CPU        float64  `json:"cpu"` // fração do processador da VM inteira, de 0 a 1
	Containers []string `json:"containers"`
}

func porApp(containers []container, antes, depois map[string]consumo, segundos float64, cpus int) []memoriaDoApp {
	grupos := map[string]*memoriaDoApp{}
	for _, c := range containers {
		agora, ok := depois[c.id]
		if !ok {
			continue // parado: não usa memória
		}
		nome := c.Projeto
		if nome == "" {
			nome = c.Nome
		}
		g := grupos[nome]
		if g == nil {
			g = &memoriaDoApp{Nome: nome, Titulo: tituloOu(titulosDosApps, nome), Containers: []string{}}
			grupos[nome] = g
		}
		g.Memoria += agora.memoria
		if a, ok := antes[c.id]; ok && segundos > 0 && cpus > 0 && agora.cpuUsec >= a.cpuUsec {
			g.CPU += float64(agora.cpuUsec-a.cpuUsec) / (segundos * 1e6 * float64(cpus))
		}
		g.Containers = append(g.Containers, c.Nome)
	}
	apps := make([]memoriaDoApp, 0, len(grupos))
	for _, g := range grupos {
		sort.Strings(g.Containers)
		apps = append(apps, *g)
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Memoria != apps[j].Memoria {
			return apps[i].Memoria > apps[j].Memoria
		}
		return apps[i].Nome < apps[j].Nome
	})
	return apps
}
