package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

// Limpeza do Docker da VM: o cache de build que cada deploy deixa (o que mais ocupa o disco) e
// as imagens soltas. Roda pela agenda (a cada N dias, numa hora) ou pelo botão "Limpar agora".

type agenda struct {
	Ativa         bool   `json:"ativa"`
	ACadaDias     int    `json:"a_cada_dias"`
	Hora          int    `json:"hora"` // hora local, de 0 a 23
	CacheDeBuild  bool   `json:"cache_de_build"`
	ImagensSoltas bool   `json:"imagens_soltas"`
	ProximaEm     string `json:"proxima_em"` // vazio com a agenda desligada
}

type limpeza struct {
	ID              int64  `json:"id"`
	Origem          string `json:"origem"` // agenda ou manual
	IniciadaEm      string `json:"iniciada_em"`
	TerminadaEm     string `json:"terminada_em"`
	CacheLiberado   int64  `json:"cache_liberado"`
	ImagensLiberado int64  `json:"imagens_liberado"`
	Erro            string `json:"erro"`
}

type oQueLimpar struct {
	CacheDeBuild  bool `json:"cache_de_build"`
	ImagensSoltas bool `json:"imagens_soltas"`
}

var errLimpezaEmAndamento = errors.New("Já há uma limpeza em andamento.")

// criarAgenda grava a agenda padrão no primeiro uso: toda semana, às 4h, cache e imagens.
func (a *app) criarAgenda() error {
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM agenda_limpeza`).Scan(&n); err != nil || n > 0 {
		return err
	}
	_, err := a.db.Exec(`INSERT INTO agenda_limpeza (id, ativa, a_cada_dias, hora, proxima_em) VALUES (1, 1, 7, 4, ?)`,
		hora(proximaNaHora(a.relogio(), 4, a.fuso)))
	return err
}

func (a *app) lerAgenda(ctx context.Context) (agenda, error) {
	var ag agenda
	err := a.db.QueryRowContext(ctx, `SELECT ativa, a_cada_dias, hora, cache_de_build, imagens_soltas, proxima_em
		FROM agenda_limpeza WHERE id = 1`).Scan(&ag.Ativa, &ag.ACadaDias, &ag.Hora, &ag.CacheDeBuild, &ag.ImagensSoltas, &ag.ProximaEm)
	if !ag.Ativa {
		ag.ProximaEm = ""
	}
	return ag, err
}

// proximaNaHora: a primeira vez, depois de agora, em que o relógio local marca a hora pedida.
func proximaNaHora(agora time.Time, h int, fuso *time.Location) time.Time {
	l := agora.In(fuso)
	t := time.Date(l.Year(), l.Month(), l.Day(), h, 0, 0, 0, fuso)
	if !t.After(agora) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// seguinte: a limpeza depois da que estava marcada, N dias à frente. Se o servidor ficou
// desligado e perdeu várias, pula para a primeira que ainda está no futuro.
func seguinte(marcada, agora time.Time, dias, h int, fuso *time.Location) time.Time {
	l := marcada.In(fuso)
	t := time.Date(l.Year(), l.Month(), l.Day()+dias, h, 0, 0, 0, fuso)
	for !t.After(agora) {
		t = t.AddDate(0, 0, dias)
	}
	return t
}

// agendaDeLimpeza confere a agenda a cada minuto.
func (a *app) agendaDeLimpeza(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		a.conferirAgenda(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// conferirAgenda roda a limpeza se chegou a hora e marca a próxima.
func (a *app) conferirAgenda(ctx context.Context) {
	ag, err := a.lerAgenda(ctx)
	if err != nil {
		log.Print("agenda de limpeza: ", err)
		return
	}
	marcada := lerHora(ag.ProximaEm)
	agora := a.relogio()
	if !ag.Ativa || marcada.IsZero() || agora.Before(marcada) || a.docker == nil {
		return
	}
	// Marca a próxima antes de limpar: se a limpeza travar ou o app reiniciar, não repete em laço.
	proxima := seguinte(marcada, agora, ag.ACadaDias, ag.Hora, a.fuso)
	if _, err := a.db.ExecContext(ctx, `UPDATE agenda_limpeza SET proxima_em = ? WHERE id = 1`, hora(proxima)); err != nil {
		log.Print("agenda de limpeza: ", err)
		return
	}
	if _, err := a.limpar(ctx, "agenda", oQueLimpar{CacheDeBuild: ag.CacheDeBuild, ImagensSoltas: ag.ImagensSoltas}); err != nil {
		log.Print("limpeza da agenda: ", err)
	}
}

// limpar roda a limpeza e grava o resultado no histórico, com o erro, se houver.
func (a *app) limpar(ctx context.Context, origem string, o oQueLimpar) (limpeza, error) {
	if !a.limpando.TryLock() {
		return limpeza{}, errLimpezaEmAndamento
	}
	defer a.limpando.Unlock()

	l := limpeza{Origem: origem, IniciadaEm: a.agora()}
	res, err := a.db.ExecContext(ctx, `INSERT INTO limpezas (origem, iniciada_em) VALUES (?, ?)`, l.Origem, l.IniciadaEm)
	if err != nil {
		return l, err
	}
	l.ID, _ = res.LastInsertId()

	// O prune do cache pode levar minutos quando há muito acumulado.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	var erros []string
	if o.CacheDeBuild {
		n, err := a.docker.limparCache(ctx)
		l.CacheLiberado = n
		if err != nil {
			erros = append(erros, "Cache de build: "+err.Error())
		}
	}
	if o.ImagensSoltas {
		n, err := a.docker.limparImagens(ctx)
		l.ImagensLiberado = n
		if err != nil {
			erros = append(erros, "Imagens: "+err.Error())
		}
	}
	l.Erro = strings.Join(erros, " ")
	l.TerminadaEm = a.agora()
	// Grava mesmo se o pedido foi cancelado: o histórico precisa mostrar como terminou.
	_, err = a.db.ExecContext(context.WithoutCancel(ctx), `UPDATE limpezas SET terminada_em = ?, cache_liberado = ?, imagens_liberado = ?, erro = ?
		WHERE id = ?`, l.TerminadaEm, l.CacheLiberado, l.ImagensLiberado, l.Erro, l.ID)
	a.db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM limpezas WHERE id <= ?`, l.ID-100)
	return l, err
}

// ---------- API ----------

type respostaLimpeza struct {
	Agenda       agenda    `json:"agenda"`
	Historico    []limpeza `json:"historico"`
	DockerLigado bool      `json:"docker_ligado"`
	EmAndamento  bool      `json:"em_andamento"`
}

func (a *app) handleLimpeza(w http.ResponseWriter, r *http.Request) {
	ag, err := a.lerAgenda(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT id, origem, iniciada_em, terminada_em, cache_liberado, imagens_liberado, erro
		FROM limpezas ORDER BY id DESC LIMIT 20`)
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	historico := []limpeza{}
	for rows.Next() {
		var l limpeza
		if err := rows.Scan(&l.ID, &l.Origem, &l.IniciadaEm, &l.TerminadaEm, &l.CacheLiberado, &l.ImagensLiberado, &l.Erro); err != nil {
			falha(w, err)
			return
		}
		historico = append(historico, l)
	}
	if err := rows.Err(); err != nil {
		falha(w, err)
		return
	}
	emAndamento := !a.limpando.TryLock()
	if !emAndamento {
		a.limpando.Unlock()
	}
	writeJSON(w, http.StatusOK, respostaLimpeza{Agenda: ag, Historico: historico, DockerLigado: a.docker != nil, EmAndamento: emAndamento})
}

// Frequências que o painel oferece: todo dia, a cada 2 ou 3 dias, toda semana, a cada 2 semanas e todo mês.
var frequencias = map[int]bool{1: true, 2: true, 3: true, 7: true, 14: true, 30: true}

// handleSalvarAgenda grava a agenda e marca a próxima limpeza na próxima vez que der a hora escolhida.
func (a *app) handleSalvarAgenda(w http.ResponseWriter, r *http.Request) {
	var ag agenda
	if !lerCorpo(w, r, &ag) {
		return
	}
	if !frequencias[ag.ACadaDias] {
		erroJSON(w, http.StatusBadRequest, "Frequência inválida: use 1, 2, 3, 7, 14 ou 30 dias.")
		return
	}
	if ag.Hora < 0 || ag.Hora > 23 {
		erroJSON(w, http.StatusBadRequest, "A hora vai de 0 a 23.")
		return
	}
	if ag.Ativa && !ag.CacheDeBuild && !ag.ImagensSoltas {
		erroJSON(w, http.StatusBadRequest, "Escolha o que limpar.")
		return
	}
	proxima := ""
	if ag.Ativa {
		proxima = hora(proximaNaHora(a.relogio(), ag.Hora, a.fuso))
	}
	_, err := a.db.ExecContext(r.Context(), `UPDATE agenda_limpeza SET ativa = ?, a_cada_dias = ?, hora = ?, cache_de_build = ?,
		imagens_soltas = ?, proxima_em = ? WHERE id = 1`, ag.Ativa, ag.ACadaDias, ag.Hora, ag.CacheDeBuild, ag.ImagensSoltas, proxima)
	if err != nil {
		falha(w, err)
		return
	}
	salva, err := a.lerAgenda(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, salva)
}

// handleLimparAgora roda a limpeza e só responde no fim, com o que foi liberado.
func (a *app) handleLimparAgora(w http.ResponseWriter, r *http.Request) {
	if a.docker == nil {
		erroJSON(w, http.StatusServiceUnavailable, "O Docker não está ligado ao painel.")
		return
	}
	var o oQueLimpar
	if !lerCorpo(w, r, &o) {
		return
	}
	if !o.CacheDeBuild && !o.ImagensSoltas {
		erroJSON(w, http.StatusBadRequest, "Escolha o que limpar.")
		return
	}
	// Se a página fechar no meio, a limpeza termina mesmo assim.
	l, err := a.limpar(context.WithoutCancel(r.Context()), "manual", o)
	if errors.Is(err, errLimpezaEmAndamento) {
		erroJSON(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}
