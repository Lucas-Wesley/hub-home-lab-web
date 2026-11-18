package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IA da Rotina (H14): entender um pedido falado ("quinta das 19 às 22, encontro com amigos") e sugerir
// uma semana a partir das metas. Usa o Claude Code da imagem com a conta ativa da tela Claude (H11).

// perguntador: manda instrução e pedido ao Claude e devolve a resposta no formato do esquema. Trocado
// nos testes.
type perguntador func(ctx context.Context, sistema, pedido, esquema string) (json.RawMessage, error)

const modeloDaRotina = "sonnet"

func perguntadorDaCLI(programa, casa, arquivoDoToken string) perguntador {
	return func(ctx context.Context, sistema, pedido, esquema string) (json.RawMessage, error) {
		b, err := os.ReadFile(arquivoDoToken)
		token := strings.TrimSpace(string(b))
		if err != nil || token == "" {
			return nil, errors.New("Nenhuma conta do Claude ativa: escolha uma na tela Claude.")
		}
		cmd := exec.CommandContext(ctx, programa, "-p", pedido, "--model", modeloDaRotina,
			"--output-format", "json", "--json-schema", esquema, "--no-session-persistence",
			"--tools", "", "--system-prompt", sistema, "--strict-mcp-config", "--setting-sources", "")
		cmd.Dir = os.TempDir()
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + casa, "DISABLE_AUTOUPDATER=1", "CLAUDE_CODE_OAUTH_TOKEN=" + token}
		cmd.WaitDelay = 5 * time.Second
		var saida, erros bytes.Buffer
		cmd.Stdout, cmd.Stderr = &saida, &erros
		errExec := cmd.Run()
		if ctx.Err() != nil {
			return nil, errors.New("O Claude demorou demais para responder.")
		}
		var r struct {
			Result      string          `json:"result"`
			IsError     bool            `json:"is_error"`
			Estruturado json.RawMessage `json:"structured_output"`
		}
		if err := json.Unmarshal(saida.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("O Claude falhou: %s", primeiro(strings.TrimSpace(erros.String()), fmt.Sprint(errExec)))
		}
		if r.IsError || errExec != nil {
			return nil, fmt.Errorf("O Claude falhou: %s", primeiro(strings.TrimSpace(r.Result), strings.TrimSpace(erros.String())))
		}
		if len(r.Estruturado) == 0 || string(r.Estruturado) == "null" {
			return nil, errors.New("O Claude respondeu fora do formato.")
		}
		return r.Estruturado, nil
	}
}

func arquivoDoTokenAtivo(contas string) string {
	return filepath.Join(contas, "ativa", "token")
}

var diasPorExtenso = []string{"segunda", "terça", "quarta", "quinta", "sexta", "sábado", "domingo"}

func linhaDoBloco(b bloco, nomes map[string]string) string {
	nome := b.Titulo
	if nome == "" {
		nome = nomes[b.Atividade] + " [" + b.Atividade + "]"
	}
	return fmt.Sprintf("%s %s–%s %s", diasPorExtenso[b.Dia], b.Inicio, b.Fim, nome)
}

// ---------- Falar: um pedido vira eventos ----------

const sistemaFalar = `Você transforma um pedido em português, dito por voz, em eventos da agenda semanal do Lucas.
Ele só ADICIONA coisas: nunca remove nem move. Responda só no formato pedido.

Regras:
- data: AAAA-MM-DD, pelo calendário dado. "quinta" sem mais nada é a próxima quinta a partir de hoje (hoje mesmo se hoje for quinta e o horário ainda não passou). "semana que vem", "daqui a 2 semanas", "dia 5" seguem o calendário.
- inicio e fim: HH:MM, 24 horas. Sem fim dito, 1 hora de duração. Sem horário nenhum (uma tarefa: "pagar o boleto na sexta"), inicio e fim vazios.
- Horário ambíguo ("das 7 às 10"): decida pelo contexto. Encontro, jantar, festa, bar, cinema → noite (19:00–22:00). Médico, banco, reunião → horário comercial. Diga o que presumiu em observacao.
- titulo: curto, como ele disse, com a primeira letra maiúscula ("Encontro com amigos").
- atividade: o id de uma atividade da lista só se o pedido for claramente uma delas ("academia sábado às 10" → academia); senão vazio.
- nota: detalhes extras ditos (lugar, pessoa); senão vazio.
- Vários eventos num pedido viram vários itens. Se não houver nada para adicionar, eventos vazio e explique em observacao.`

const esquemaFalar = `{"type":"object","properties":{"eventos":{"type":"array","items":{"type":"object","properties":{"data":{"type":"string"},"inicio":{"type":"string"},"fim":{"type":"string"},"titulo":{"type":"string"},"atividade":{"type":"string"},"nota":{"type":"string"}},"required":["data","inicio","fim","titulo","atividade","nota"]}},"observacao":{"type":"string"}},"required":["eventos","observacao"]}`

func (a *app) handleFalar(w http.ResponseWriter, r *http.Request) {
	if a.ia == nil {
		erroJSON(w, http.StatusServiceUnavailable, "A IA não está ligada neste Hub (falta a pasta das contas do Claude).")
		return
	}
	var corpo struct {
		Texto string `json:"texto"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	texto := strings.TrimSpace(corpo.Texto)
	if texto == "" || len(texto) > 1000 {
		erroJSON(w, http.StatusBadRequest, "Diga ou escreva o que adicionar.")
		return
	}
	c, err := a.lerConfigRotina(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	hoje := a.relogio().In(a.fuso)
	var p strings.Builder
	fmt.Fprintf(&p, "Agora: %s, %s.\n\nCalendário (próximos 60 dias):\n", diasPorExtenso[(int(hoje.Weekday())+6)%7], hoje.Format("2006-01-02 15:04"))
	for i := 0; i < 60; i++ {
		d := hoje.AddDate(0, 0, i)
		fmt.Fprintf(&p, "%s %s\n", d.Format("2006-01-02"), diasPorExtenso[(int(d.Weekday())+6)%7])
	}
	p.WriteString("\nAtividades (id: nome):\n")
	for _, at := range c.Atividades {
		fmt.Fprintf(&p, "%s: %s\n", at.ID, at.Nome)
	}
	fmt.Fprintf(&p, "\nPedido: %s", texto)

	ctx, cancelar := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancelar()
	bruto, err := a.ia(ctx, sistemaFalar, p.String(), esquemaFalar)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	var entendido struct {
		Eventos    []eventoNovo `json:"eventos"`
		Observacao string       `json:"observacao"`
	}
	if err := json.Unmarshal(bruto, &entendido); err != nil {
		erroJSON(w, http.StatusBadGateway, "O Claude respondeu fora do formato.")
		return
	}
	feitos := []eventoAdicionado{}
	if len(entendido.Eventos) > 0 {
		if feitos, err = a.adicionarEventos(r.Context(), entendido.Eventos); err != nil {
			erroJSON(w, http.StatusBadGateway, "Não deu para adicionar o que o Claude entendeu: "+err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"eventos": feitos, "observacao": entendido.Observacao})
}

// ---------- Sugerir: uma semana a partir das metas ----------

const sistemaSugerir = `Você monta a semana padrão do Lucas a partir das atividades e metas dele. É um rascunho para ele ter uma noção geral; ele pode pedir para refazer com comentários.

Regras:
- Use só os ids de atividade da lista. Cada bloco: dia (0 segunda … 6 domingo), inicio e fim em HH:MM (24 h, fim depois do início, no mesmo dia), atividade, nota curta ou vazia.
- Atividades sem meta (meta 0) e as fixas ficam nos mesmos dias e horários do modelo atual (trabalho, almoço, namorada, futebol…): são compromissos.
- As com meta: o número de blocos por semana fica entre o mínimo e o máximo. Prioridade 1 é a mais importante: se não couber tudo, corte das de prioridade 3.
- Sono de pelo menos 6 h entre o último bloco de um dia e o primeiro do seguinte. Sem dois blocos no mesmo horário.
- Domingo o mais livre possível. Estudo pesado não depois das 23h.
- Se houver um pedido do Lucas ou uma sugestão anterior, parta da anterior e mude só o que ele pediu.
- explicacao: 2 a 4 frases sobre as escolhas e o que ficou de fora.`

const esquemaSugerir = `{"type":"object","properties":{"explicacao":{"type":"string"},"blocos":{"type":"array","items":{"type":"object","properties":{"dia":{"type":"integer"},"inicio":{"type":"string"},"fim":{"type":"string"},"atividade":{"type":"string"},"nota":{"type":"string"}},"required":["dia","inicio","fim","atividade","nota"]}}},"required":["explicacao","blocos"]}`

func (a *app) handleSugerir(w http.ResponseWriter, r *http.Request) {
	if a.ia == nil {
		erroJSON(w, http.StatusServiceUnavailable, "A IA não está ligada neste Hub (falta a pasta das contas do Claude).")
		return
	}
	var corpo struct {
		Atividades []atividade `json:"atividades"`
		Comentario string      `json:"comentario"`
		Anterior   []bloco     `json:"anterior"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	existe, err := validarAtividades(corpo.Atividades)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(corpo.Comentario) > 2000 {
		erroJSON(w, http.StatusBadRequest, "Comentário longo demais.")
		return
	}
	c, err := a.lerConfigRotina(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	nomes := map[string]string{}
	for _, at := range append(append([]atividade{}, c.Atividades...), corpo.Atividades...) {
		nomes[at.ID] = at.Nome
	}

	var p strings.Builder
	p.WriteString("Atividades (id, nome, meta mín–máx por semana, prioridade, fixa):\n")
	for _, at := range corpo.Atividades {
		fmt.Fprintf(&p, "- %s, %s, %d–%d, prioridade %d%s\n", at.ID, at.Nome, at.MetaMin, at.MetaMax, at.Prioridade, map[bool]string{true: ", fixa", false: ""}[at.Fixa])
	}
	p.WriteString("\nModelo atual:\n")
	for _, b := range c.Modelo {
		if existe[b.Atividade] {
			p.WriteString(linhaDoBloco(b, nomes) + "\n")
		}
	}
	if len(corpo.Anterior) > 0 {
		p.WriteString("\nSugestão anterior:\n")
		for _, b := range corpo.Anterior {
			if b.Dia >= 0 && b.Dia <= 6 {
				p.WriteString(linhaDoBloco(b, nomes) + "\n")
			}
		}
	}
	if s := strings.TrimSpace(corpo.Comentario); s != "" {
		fmt.Fprintf(&p, "\nPedido do Lucas: %s\n", s)
	}

	ctx, cancelar := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancelar()
	bruto, err := a.ia(ctx, sistemaSugerir, p.String(), esquemaSugerir)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	var sugestao struct {
		Explicacao string  `json:"explicacao"`
		Blocos     []bloco `json:"blocos"`
	}
	if err := json.Unmarshal(bruto, &sugestao); err != nil {
		erroJSON(w, http.StatusBadGateway, "O Claude respondeu fora do formato.")
		return
	}
	// Fica só o que é válido: um bloco torto não derruba a sugestão inteira.
	validos := []bloco{}
	for _, b := range sugestao.Blocos {
		b.ID = novoID("s")
		b.Titulo, b.Feito, b.Extra = "", false, false
		if validarBlocos([]bloco{b}, existe) == nil {
			validos = append(validos, b)
		}
	}
	if len(validos) == 0 {
		erroJSON(w, http.StatusBadGateway, "O Claude não devolveu uma semana válida. Tente de novo.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocos": validos, "explicacao": sugestao.Explicacao})
}
