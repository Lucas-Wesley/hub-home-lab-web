# Roadmap do Ecossistema

> Ordem sugerida para desenvolver as aplicações. Atualizado em **03/11/2026**.
> Vale para todos os projetos — este é o repo **Hub**, ponto único de referência.

---

## Ordem de desenvolvimento

### 1º — Biblioteca Digital
Mais "resolvida" tecnicamente (readers são tecnologia madura). Começar por ela permite montar o
**esqueleto reutilizável** (estrutura do projeto, API, token de auth, container no Proxmox, montar o
SSD externo, PWA) num problema mais fácil, antes da parte difícil. É **útil sozinha na hora** (vira
um e-reader funcional) e **alimenta o Acervo** (livro → referência), então o Acervo já nasce com
fonte de conteúdo real.
Repo: `github.com/Lucas-Wesley/biblioteca-digital`.

### 2º — Acervo
O coração e o mais ambicioso/incerto (pipeline com Claude Code headless, verificação, merge).
Construí-lo depois significa chegar nele com o esqueleto já provado e com conteúdo real entrando.
**Começar pela fatia mais simples:** extrair transcrição → resumir → salvar o Markdown. Deixar
fact-checking e merge automático para fases seguintes.
Repo: a criar.

### 3º — Flashcards (+ API)
É o **consumidor** da ponta (recebe cards) e já tem repo + plano. Finalizá-lo e dar uma API faz mais
sentido depois que já existe quem mande cards pra ele.
Repo: `github.com/Lucas-Wesley/flashcards`.

### 4º — Hub
**Último sempre.** Só consome as APIs dos outros; construir cedo trava tudo.
Repo: `github.com/Lucas-Wesley/hub-home-lab-web`.

---

## Princípios (valem pra qualquer ordem)

- **Cada app sozinho primeiro.** Só adicionar o endpoint de integração quando os **dois lados
  existirem** (ex.: só criar "Biblioteca → Acervo" quando o Acervo já receber).
- **Não definir o contrato comum ainda.** Ele só vira concreto quando dois apps precisarem conversar
  de verdade — ver [`CONTRATO.md`](CONTRATO.md).
- **Começar por fatias verticais finas** dentro de cada app, não tentar o app inteiro de uma vez.

---

## Alternativas

- **Vitória rápida / embalo:** o Flashcards é o que está mais perto de pronto — dá pra fazê-lo
  primeiro se quiser ganhar ritmo.
- **Prioridade no Acervo:** pode começar por ele, mas então comece pela fatia mínima pra não travar
  no mais difícil.
