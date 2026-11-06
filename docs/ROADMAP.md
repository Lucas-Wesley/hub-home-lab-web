# Roadmap do Ecossistema

> Ordem sugerida para desenvolver as aplicações. Atualizado em **06/11/2026**.
> Vale para todos os projetos — este é o repo **Hub**, ponto único de referência.

---

## Onde paramos (06/11/2026)

| App | Estado | Endereço |
|-----|--------|----------|
| Biblioteca Digital | **No ar**, em uso | `https://192.168.1.20:8443` |
| Acervo | **No ar**, primeira versão completa | `https://192.168.1.20:8444` |
| Flashcards | **Próximo passo** | — |
| Hub | Último | — |

**Biblioteca Digital:**
- PWA que lê EPUB (Readium) e PDF (PDF.js), com modo texto para PDF.
- Marcações e notas, com seleção própria no toque.
- Progresso no servidor e leitura offline de livros baixados.
- Busca de livros em fontes abertas.
- Ajustes de leitura guardados por livro, com o estilo pronto "Acervo" como padrão.
- Detalhes em `biblioteca-digital/docs/DECISOES.md`.

**Acervo:**
- Um vídeo do YouTube (colado, ou mandado pelo "Compartilhar" do celular) entra numa fila de uma
  tarefa por vez: `yt-dlp` extrai a legenda e o Claude Code headless organiza e traduz o texto.
- A IA sugere o Tema e cria ou integra o texto, numa worktree git.
- O Lucas aprova pelo diff, em Revisões.
- Cada Tema é uma página com capítulos, que podem ser separados em páginas próprias, com edição no
  app e leitura offline.
- Ainda não existem a verificação de veracidade nem a integração com a Biblioteca, que está só
  planejada em `home-lab-acervo/docs/PESQUISADOR.md`.
- Detalhes em `home-lab-acervo/docs/DECISOES.md` e `CONSTRUCAO.md`.

**Comum às duas:**
- Mesma infraestrutura (ver [`ARQUITETURA.md`](ARQUITETURA.md) §4).
- Design system único, em `docs/DESIGN-SYSTEM.md` de cada app.

**Integrações:** nenhuma ainda. O contrato comum continua não definido, e vai nascer quando o
primeiro par de apps precisar conversar. O candidato natural é Biblioteca/Acervo → Flashcards.

## Ordem de desenvolvimento

### 1º — Biblioteca Digital
Mais "resolvida" tecnicamente (readers são tecnologia madura). Começar por ela permite montar o
**esqueleto reutilizável** (estrutura do projeto, API, token de auth, container no Proxmox, armazenamento
no disco interno, PWA) num problema mais fácil, antes da parte difícil. É **útil sozinha na hora** (vira
um e-reader funcional) e **alimenta o Acervo** (livro → referência), então o Acervo já nasce com
fonte de conteúdo real.
Repo: `github.com/Lucas-Wesley/biblioteca-digital`.

### 2º — Acervo
O coração e o mais ambicioso/incerto (pipeline com Claude Code headless, verificação, merge).
Construí-lo depois significa chegar nele com o esqueleto já provado e com conteúdo real entrando.
**Começar pela fatia mais simples:** extrair transcrição → resumir → salvar o Markdown. Deixar
fact-checking e merge automático para fases seguintes.
Repo: `github.com/Lucas-Wesley/home-lab-acervo`.

### 3º — Flashcards (+ API) — próximo
É o **consumidor** da ponta (recebe cards) e já tem repo + plano. Finalizá-lo e dar uma API faz mais
sentido depois que já existe quem mande cards pra ele — e agora existem: a seleção de texto da
Biblioteca e os Temas do Acervo. Deve seguir a mesma infraestrutura e o mesmo design system.
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
