package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// clienteDocker fala com o Docker da VM por um proxy (docker-socket-proxy) que só libera ler o
// uso do disco e limpar imagens e cache de build: o painel não cria nem entra em contêineres.
// A versão da API fica fixa, para o formato das respostas não mudar quando o Docker atualizar.
type clienteDocker struct {
	base string
	http *http.Client
}

const versaoDaAPIDoDocker = "/v1.47"

func novoDocker(base string) *clienteDocker {
	return &clienteDocker{base: strings.TrimRight(base, "/") + versaoDaAPIDoDocker, http: &http.Client{}}
}

type usoDoDocker struct {
	Imagens        int64       `json:"imagens"`        // bytes no disco, contando as camadas uma vez
	ImagensSoltas  int64       `json:"imagens_soltas"` // imagens sem nome e sem contêiner: o que a limpeza apaga
	CacheDeBuild   int64       `json:"cache_de_build"`
	CacheLiberavel int64       `json:"cache_liberavel"` // o cache que nenhum build está usando
	Containers     []container `json:"containers"`
}

type container struct {
	Nome   string `json:"nome"`
	Imagem string `json:"imagem"`
	Estado string `json:"estado"` // running, exited…
	Status string `json:"status"` // "Up 5 hours", como o Docker escreve
	Bytes  int64  `json:"bytes"`  // o que o contêiner gravou fora da imagem
}

// Os campos de GET /system/df que o painel usa.
type dfDoDocker struct {
	LayersSize int64
	Images     []struct {
		RepoTags   []string
		Size       int64
		SharedSize int64
		Containers int
	}
	Containers []struct {
		Names  []string
		Image  string
		State  string
		Status string
		SizeRw int64
	}
	BuildCache []struct {
		Size   int64
		InUse  bool
		Shared bool
	}
}

func (d *clienteDocker) uso(ctx context.Context) (usoDoDocker, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var df dfDoDocker
	if err := d.chamar(ctx, http.MethodGet, "/system/df", &df); err != nil {
		return usoDoDocker{}, err
	}
	// As contas são as do "docker system df".
	u := usoDoDocker{Imagens: df.LayersSize, Containers: []container{}}
	for _, img := range df.Images {
		if img.Containers == 0 && solta(img.RepoTags) {
			if img.SharedSize > 0 {
				u.ImagensSoltas += img.Size - img.SharedSize
			} else {
				u.ImagensSoltas += img.Size
			}
		}
	}
	for _, c := range df.BuildCache {
		if c.Shared {
			continue
		}
		u.CacheDeBuild += c.Size
		if !c.InUse {
			u.CacheLiberavel += c.Size
		}
	}
	for _, c := range df.Containers {
		nome := ""
		if len(c.Names) > 0 {
			nome = strings.TrimPrefix(c.Names[0], "/")
		}
		u.Containers = append(u.Containers, container{Nome: nome, Imagem: c.Image, Estado: c.State, Status: c.Status, Bytes: c.SizeRw})
	}
	return u, nil
}

// solta: imagem sem nome, que sobrou de um build antigo.
func solta(tags []string) bool {
	for _, t := range tags {
		if t != "<none>:<none>" {
			return false
		}
	}
	return true
}

// limparCache apaga todo o cache de build que não está em uso. O próximo deploy de cada app
// demora um pouco mais, porque baixa as dependências de novo.
func (d *clienteDocker) limparCache(ctx context.Context) (int64, error) {
	var r struct{ SpaceReclaimed int64 }
	err := d.chamar(ctx, http.MethodPost, "/build/prune?all=true", &r)
	return r.SpaceReclaimed, err
}

// limparImagens apaga só as imagens soltas (sem nome). As imagens com nome sem contêiner são a
// versão anterior de cada app, que o deploy guarda para voltar se a nova falhar.
func (d *clienteDocker) limparImagens(ctx context.Context) (int64, error) {
	var r struct{ SpaceReclaimed int64 }
	err := d.chamar(ctx, http.MethodPost, "/images/prune", &r)
	return r.SpaceReclaimed, err
}

func (d *clienteDocker) chamar(ctx context.Context, metodo, caminho string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, metodo, d.base+caminho, nil)
	if err != nil {
		return err
	}
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return fmt.Errorf("%s %s: %s %s", metodo, caminho, resp.Status, strings.TrimSpace(string(corpo)))
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}
