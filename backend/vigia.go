package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Vigia do servidor (H9): de tempos em tempos confere os discos, os contêineres e o backup e avisa no
// celular, pelo ntfy, o que pede ação. Um problema só avisa depois de visto em duas conferências
// seguidas (um deploy recria os contêineres e não deve avisar) e só uma vez: volta a avisar se
// sumir e aparecer de novo.

const (
	intervaloDaVigia = 10 * time.Minute
	discoCheio       = 0.85
	backupAtrasado   = 26 * time.Hour // roda todo dia às 3h; uma folga para a VM que estava desligada
)

type problema struct {
	chave  string
	titulo string
	texto  string
}

// vigia lembra em quantas conferências seguidas cada problema apareceu.
type vigia struct {
	vistos map[string]int
}

// novos devolve os problemas que acabaram de completar a segunda conferência seguida.
func (v *vigia) novos(atuais []problema) []problema {
	if v.vistos == nil {
		v.vistos = map[string]int{}
	}
	agora := map[string]bool{}
	var avisar []problema
	for _, p := range atuais {
		agora[p.chave] = true
		v.vistos[p.chave]++
		if v.vistos[p.chave] == 2 {
			avisar = append(avisar, p)
		}
	}
	for chave := range v.vistos {
		if !agora[chave] {
			delete(v.vistos, chave)
		}
	}
	return avisar
}

func (a *app) vigiar(ctx context.Context) {
	if a.avisos == nil {
		return
	}
	var v vigia
	t := time.NewTicker(intervaloDaVigia)
	defer t.Stop()
	for {
		for _, p := range v.novos(a.problemas(ctx)) {
			a.avisos.avisar(aviso{Titulo: p.titulo, Texto: p.texto, Link: a.cfg.link, Urgente: true})
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// problemas: o que está errado agora. O que não dá para ler (Proxmox sem token, Docker fora) fica
// de fora: o painel já mostra esses erros.
func (a *app) problemas(ctx context.Context) []problema {
	var ps []problema
	if d, err := lerDisco(a.cfg.apps); err == nil {
		if p := discoQuaseCheio("disco-vm", "Disco da VM apps", d); p != nil {
			ps = append(ps, *p)
		}
	}
	if a.proxmox != nil {
		if v, err := a.proxmox.visao(ctx); err == nil {
			for _, s := range v.Armazenamentos {
				if p := discoQuaseCheio("proxmox-"+s.Nome, "Armazenamento "+s.Nome+" do Proxmox", s.uso); s.Ativo && p != nil {
					ps = append(ps, *p)
				}
			}
			for _, d := range v.Discos {
				if p := discoFisicoComProblema(d); p != nil {
					ps = append(ps, *p)
				}
			}
		}
	}
	if a.docker != nil {
		if u, err := a.docker.uso(ctx); err == nil {
			for _, c := range u.Containers {
				if p := conteinerComProblema(c); p != nil {
					ps = append(ps, *p)
				}
			}
		}
	}
	if a.adguard != nil {
		if protecao, ate, err := a.adguard.estado(ctx); err == nil && !protecao && ate.IsZero() {
			ps = append(ps, problema{chave: "adguard-pausado", titulo: "Bloqueio de anúncios desligado",
				texto: "O bloqueio do AdGuard foi pausado sem prazo. Religue na tela Rede do Hub quando terminar."})
		}
	}
	if a.claude != nil {
		ps = append(ps, a.claude.problemas(ctx)...)
	}
	if a.cfg.backup != "" {
		if p := backupComProblema(a.cfg.backup, a.relogio()); p != nil {
			ps = append(ps, *p)
		}
	}
	return ps
}

func discoQuaseCheio(chave, nome string, u uso) *problema {
	if u.Total == 0 || float64(u.Usado)/float64(u.Total) < discoCheio {
		return nil
	}
	return &problema{chave: chave,
		titulo: fmt.Sprintf("%s em %.0f%%", nome, 100*float64(u.Usado)/float64(u.Total)),
		texto:  "Livre: " + gigas(u.Livre) + ". O Hub mostra o que ocupa espaço e limpa o Docker."}
}

func discoFisicoComProblema(d discoFisico) *problema {
	saude := strings.ToUpper(d.Saude)
	switch {
	case saude != "" && saude != "PASSED" && saude != "OK" && saude != "UNKNOWN":
		return &problema{chave: "saude-" + d.Caminho, titulo: "Disco do servidor com problema",
			texto: fmt.Sprintf("O SMART de %s (%s) diz %s. Hora de garantir o backup e trocar o disco.", d.Caminho, d.Modelo, d.Saude)}
	case d.VidaUtil >= 0 && d.VidaUtil <= 10:
		return &problema{chave: "vida-" + d.Caminho, titulo: "Disco do servidor no fim da vida útil",
			texto: fmt.Sprintf("Restam %d%% da vida útil de %s (%s).", d.VidaUtil, d.Caminho, d.Modelo)}
	}
	return nil
}

func conteinerComProblema(c container) *problema {
	switch {
	case c.Estado != "running":
		return &problema{chave: "conteiner-" + c.Nome, titulo: "Contêiner parado: " + c.Nome, texto: c.Status}
	case strings.Contains(c.Status, "(unhealthy)"):
		return &problema{chave: "conteiner-" + c.Nome, titulo: "Contêiner com problema: " + c.Nome,
			texto: c.Status + ". A conferência de saúde dele está falhando."}
	}
	return nil
}

// backupComProblema lê o arquivo que o homelab-backup.sh deixa a cada rodada: "<data> ok ..." ou
// "<data> FALHOU ...". A falha o próprio script avisa; aqui o caso é o backup que parou de rodar.
func backupComProblema(arquivo string, agora time.Time) *problema {
	b, err := os.ReadFile(arquivo)
	if err != nil {
		return &problema{chave: "backup", titulo: "Backup sem registro",
			texto: "Não achei o resultado do último backup. Confira o homelab-backup.timer na VM apps."}
	}
	campos := strings.Fields(string(b))
	var quando time.Time
	if len(campos) > 0 {
		quando, err = time.Parse(time.RFC3339, campos[0])
	}
	if len(campos) == 0 || err != nil {
		log.Printf("vigia: resultado do backup ilegível: %q", b)
		return nil
	}
	if atraso := agora.Sub(quando); atraso > backupAtrasado {
		return &problema{chave: "backup", titulo: "Backup atrasado",
			texto: fmt.Sprintf("O último rodou há %.0f horas. Confira com journalctl -u homelab-backup na VM apps.", atraso.Hours())}
	}
	return nil
}

func gigas(bytes uint64) string {
	return strings.Replace(fmt.Sprintf("%.1f GB", float64(bytes)/1e9), ".", ",", 1)
}
