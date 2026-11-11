# Roadmap do Ecossistema

> Ordem sugerida para desenvolver as aplicações. Atualizado em **09/11/2026**.
> Vale para todos os projetos — este é o repo **Hub**, ponto único de referência.

---

## Onde paramos (06/11/2026)

| App | Estado | Endereço |
|-----|--------|----------|
| Biblioteca Digital | **No ar**, em uso | `https://biblioteca.internal` |
| Acervo | **No ar**, primeira versão completa | `https://acervo.internal` |
| Flashcards | **No ar**, primeira versão (06/11/2026) | `https://flashcards.internal` |
| Extensão do Chrome | **Feita** (06/11/2026), instalada sem compactação | — (cliente dos apps) |
| Hub | **Painel do servidor no ar** (09/11/2026); a parte de conteúdo é a última | `https://hub.internal` |

**Biblioteca Digital:**
- PWA que lê EPUB (Readium) e PDF (PDF.js), com modo texto para PDF.
- Marcações e notas, com seleção própria no toque.
- Progresso no servidor e leitura offline de livros baixados.
- Busca de livros em fontes abertas.
- Ajustes de leitura guardados por livro, com o estilo pronto "Acervo" como padrão.
- **Estatísticas de leitura** (06/11/2026): tempo e páginas por dia e por livro, dias seguidos,
  calendário, médias e previsão do tempo que falta. Coletando dados; revisão com o Lucas depois
  de alguns dias de uso.
- **Busca no texto dos livros** (R32): tela "Buscar nos meus livros", que abre o livro no trecho, e
  API usada pelo Pesquisador do Acervo.
- **Capítulos e texto de um livro pela API** (R33), para o Acervo resumir capítulos.
- Detalhes em `biblioteca-digital/docs/DECISOES.md`.

**Acervo:**
- Um vídeo do YouTube (colado, ou mandado pelo "Compartilhar" do celular) entra numa fila de uma
  tarefa por vez: `yt-dlp` extrai a legenda e o Claude Code headless organiza e traduz o texto.
- A IA sugere o Tema e cria ou integra o texto, numa worktree git.
- O Lucas aprova pelo diff, em Revisões.
- Cada Tema é uma página com capítulos, que podem ser separados em páginas próprias, com edição no
  app e leitura offline.
- **Mais informações sobre um trecho:** seleciono um texto num Tema e a IA pesquisa na internet
  (`WebSearch`/`WebFetch`, só nessa tarefa) e devolve uma explicação com fontes (D26).
- **Pesquisador** (D27): um tema vira uma Referência com fonte em cada parte. O Lucas escolhe as
  fontes (seus livros, Wikipédia, YouTube, sites ou só a IA) e a profundidade (rápida ou detalhada,
  com conferência), e pode revisar o esboço antes da busca.
- **Aprofundar uma pesquisa** (D29): outra rodada com mais fontes e o que o Lucas quer a mais,
  partindo do texto atual e dos Temas que já tratam do assunto, com a seção "O que há de novo".
- **Resumir capítulos de um livro** (D30): um ou mais capítulos de um livro da Biblioteca viram
  um resumo fiel, com a página de cada parte, e a IA sugere o Tema existente ou novo.
- Ainda não existem a verificação de veracidade das Referências de vídeo nem o envio de trechos da
  Biblioteca para o Acervo (planejado em `home-lab-acervo/docs/PESQUISADOR.md`).
- Detalhes em `home-lab-acervo/docs/DECISOES.md` e `CONSTRUCAO.md`.

**Flashcards:**
- Baralhos, notas de frente e verso ou nos dois sentidos (vocabulário), estudo com quatro notas,
  desfazer, busca, suspensão e importação de texto/CSV.
- Agendamento FSRS-6 (`go-fsrs`), com o dia de estudo virando às 4h em Brasília.
- **Gerar baralho com IA** (Claude Code headless, como no Acervo): pedido → proposta → revisão e aprovação; a
  instrução da IA é editável em Ajustes.
- Imagens nos cartões, da Wikipédia (pela IA ou à mão) ou do aparelho.
- **Cartões de um link** de qualquer site, com o **áudio** da pronúncia: o app lê as frases de
  exemplo da página ou, se ela não as mostra do jeito comum, a IA as tira do texto; vão só as 3 mais
  didáticas (F29, F31). Na revisão, o baralho pode ser trocado (padrão: Inglês).
- Ainda sem estudo offline e sem integração.
- Detalhes e próximos passos em `flashcards/docs/DECISOES.md`.

**Extensão do Chrome** (`home-lab-plugin-web`):
- Num vídeo do YouTube, um botão discreto no canto manda o link ao Acervo; numa página com frases
  de exemplo, de qualquer site, manda ao Flashcards (pelo ícone, qualquer página). Guarda só o
  histórico dos envios e os ajustes.
- É só um cliente das APIs que os apps já têm. Detalhes em `home-lab-plugin-web/docs/DECISOES.md`.

**Comum às três:**
- Mesma infraestrutura (ver [`ARQUITETURA.md`](ARQUITETURA.md) §4).
- Design system único, em `docs/DESIGN-SYSTEM.md` de cada app.
- **Ajustes → Sugestões de melhoria** em cada app, com a mesma API (`/api/sugestoes`). O Flashcards
  e as próximas apps já nascem com ela.
- No Acervo, Ajustes também mostra as **instruções da IA** de cada etapa, editáveis pelo celular.

**Integrações:** a primeira é **Acervo → Biblioteca** (06/11/2026): o Pesquisador do Acervo busca
no texto dos livros e cita livro e página (ver [`ARQUITETURA.md`](ARQUITETURA.md) §3). É só leitura,
com o formato da API da Biblioteca. O contrato comum (o Fragmento) continua não definido e nasce na
primeira troca de conteúdo; o candidato natural é Biblioteca/Acervo → Flashcards. Depois vieram
**Acervo → Flashcards** (F28) e **Acervo → Biblioteca, capítulos** (R33/D30), também cada uma com
o formato da API de quem responde.

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

### 3º — Flashcards (+ API) — no ar, sem integrações ainda
É o **consumidor** da ponta (recebe cards) e já tem repo + plano. Finalizá-lo e dar uma API faz mais
sentido depois que já existe quem mande cards pra ele — e agora existem: a seleção de texto da
Biblioteca e os Temas do Acervo. Deve seguir a mesma infraestrutura e o mesmo design system.
Repo: `github.com/Lucas-Wesley/flashcards`.

### 4º — Hub
**Último sempre.** Só consome as APIs dos outros; construir cedo trava tudo.
Repo: `github.com/Lucas-Wesley/hub-home-lab-web`.

#### Pedido para o Hub: gestão do servidor e do Claude (anotado em 06/11/2026, começado em 09/11/2026)

**Feito (09/11/2026):** o painel do servidor, só leitura, com o armazenamento do Proxmox, da VM e
de cada app, a memória e os contêineres, e a limpeza do Docker com agenda. Ele não depende das APIs
dos apps, então pôde vir antes da parte de conteúdo. Detalhes e decisões (H1–H8) em
[`DECISOES.md`](DECISOES.md). Falta, do pedido abaixo: limpar o que é temporário dentro de cada
app e a parte do Claude.

O Lucas quer gerenciar pelo Hub, num lugar só:

- **Arquivos temporários e sem uso de cada app:** ver quanto ocupam e limpar. Exemplos:
  - os históricos de sessão do Claude em `/srv/data/<app>/home/.claude/projects` (um `.jsonl` por tarefa);
  - os backups em `.claude/backups`;
  - as pastas de trabalho das tarefas;
  - a mídia sem uso.
- **Claude:**
  - a chave (`CLAUDE_CODE_OAUTH_TOKEN` em `/opt/apps/<app>.env`), só para ver o estado e trocar, sem nunca mostrar o valor;
  - o que está sendo pesquisado agora e o que já foi, em cada app;
  - limpar o histórico;
  - talvez uma memória por app (ver abaixo).
- **Servidor:** memória, processamento e armazenamento, no total e por app/contêiner; o que pesa
  e custa mais.

**Como o Claude roda hoje (para decidir a parte de memória):**
- Cada tarefa é um `claude -p` independente: começa do zero e não lembra das anteriores (sem
  `--resume`/`--continue`). As pesquisas são efêmeras; o contexto vem todo do prompt e das
  instruções em `regras/`.
- Cada app roda o Claude no próprio contêiner, com `HOME=/data/home` próprio. Então os históricos e
  as configurações **já ficam separados por app** (Flashcards, Acervo), em `/srv/data/<app>/home`.
- O que fica gravado é só o **registro** de cada sessão (`.claude/projects/…/*.jsonl`), que nenhuma
  tarefa relê. Desde 06/11/2026 o Flashcards roda com `--no-session-persistence` e não grava mais
  nada. O Acervo ainda grava.
- Uma memória de verdade (por exemplo, "o Lucas prefere cartões curtos") seria um arquivo de texto
  por app que o app inclui no prompt, editável no Ajustes, como já são as instruções. Não seria uma
  sessão longa do Claude.

**Como o Hub lê esses dados (decidido em 09/11/2026):** os números do servidor vêm da API do
Proxmox (token só de leitura), do disco da VM e de um proxy do Docker que só deixa ler e limpar,
sem agente no host. O tamanho de cada app é medido de fora, em `/srv/data`. Limpar o que é de
dentro de um app vai pedir uma API de "manutenção" em cada app; o formato nasce quando o primeiro
app a tiver, seguindo o princípio de não definir o contrato antes de precisar.

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
