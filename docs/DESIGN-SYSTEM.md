# Design system das apps do homelab

Vale para a Biblioteca Digital, o Acervo, o Flashcards, o Streaming, o Hub e as próximas apps.
Há uma cópia igual em `docs/DESIGN-SYSTEM.md` de cada app; os tokens ficam no começo do
`frontend/src/style.css` de cada uma, com os mesmos nomes e valores.

**Terceira versão (18/11/2026):** o design que nasceu na **Trilha de Estudo** vira o padrão de
todas as apps. Ele segue o estilo da **Claude Platform** (`platform.claude.com`: Docs, Console e
Cookbook), a partir de prints escolhidos pelo Lucas (referências em `docs/design/referencias/` da
Trilha). A **implementação de referência** é a da Trilha: este texto é o mesmo do
`docs/DESIGN-SYSTEM.md` de lá, e a **vitrine viva** fica em `#/design` naquele app. Substitui a
versão anterior (neutros frios, fonte do sistema, uma cor por app, vidro e blur).

### O que mudou em relação à versão anterior

- Fundo quase preto no escuro (era cinza-azulado), superfícies e bordas medidos nos prints.
- Serifa para títulos de página e seção (antes era tudo sem serifa).
- Fim da cor por app (`--marca`): o botão principal é invertido (claro sobre escuro) e a cor
  aparece só em links, etiquetas e estados. Cada app se reconhece pelo seu desenho de ícone.
- Fim do vidro/blur: barras e painéis são chapados.
- Fontes embutidas (offline): **Source Serif 4** (títulos), **Inter** (texto e interface),
  **JetBrains Mono** (código).

---

## 1. Princípios (o que se vê nos prints)

1. **Escuro, neutro e quieto.** Fundo quase preto (`#151515`), superfícies um ou dois tons acima,
   bordas finas. Sem gradiente, sem sombra pesada. A cor aparece só em links, etiquetas e estados.
2. **Pequeno e denso, mas respirando.** Botões de 32 px, texto de interface de 14 px, ícones de
   16 px. O espaço vem das margens e das entrelinhas, não do tamanho dos controles.
3. **Serifa para títulos, sem serifa para o resto.** Títulos de página e de seção em serifa
   (o tom "editorial" das Docs); texto corrido e interface em sem serifa; código em monoespaçada.
4. **Leitura em primeiro lugar.** Coluna de texto de ~720 px, 16 px com entrelinha 1,7, contraste
   alto mas não branco puro.
5. **Um botão principal por tela**, invertido (branco sobre o escuro). O resto é discreto.
6. **Três colunas em telas largas:** navegação à esquerda, conteúdo no centro, "nesta página" à
   direita. No celular, uma coluna, com a navegação numa gaveta.

### O que não dá para copiar (e o substituto)

| Na referência | Problema | Aqui |
|---|---|---|
| Fontes próprias da Anthropic (serifa e sem serifa) | proprietárias | **Source Serif 4** (títulos), **Inter** (texto e interface), **JetBrains Mono** (código), todas embutidas no app (offline) |
| Logo e asterisco da marca | marca de terceiros | desenho de ícone próprio de cada app |
| Tema claro | os prints são só do escuro | tema claro derivado dos mesmos princípios (§2); segue o aparelho, com escolha em Ajustes |

---

## 2. Tokens

Medidos nos prints (fundo `#151515`, superfícies `#1a1a19`/`#212121`, bordas `#2c2c2c`/`#383838`,
etiquetas com fundo escuro saturado e texto claro).

### Cor

| Token | Escuro | Claro | Uso |
|---|---|---|---|
| `--fundo` | `#151515` | `#fbfaf8` | página |
| `--superficie` | `#1b1b1a` | `#f4f3ef` | cartões, blocos de código, caixas |
| `--superficie-2` | `#232322` | `#ecebe6` | campos, botão secundário, hover |
| `--ativo` | `#2e2e2d` | `#e4e3dd` | item ativo da navegação, aba ativa |
| `--borda` | `#2c2c2b` | `#e3e2dc` | bordas e divisórias |
| `--borda-forte` | `#3d3d3b` | `#cfcec7` | campo em foco sem cor, borda de botão |
| `--video` | `#000` | `#000` | fundo de player de vídeo (barras do vídeo) |
| `--pontos` | `#262625` | `#e6e5e0` | grade de pontos atrás do painel central (`.fundo-pontos`) |
| `--texto` | `#ececea` | `#1d1d1b` | texto principal |
| `--texto-suave` | `#a3a29d` | `#5f5e59` | descrições, navegação inativa |
| `--texto-fraco` | `#73726d` | `#8b8a84` | rótulos, datas, placeholders |
| `--primario` | `#f5f5f3` | `#1d1d1b` | botão principal (fundo) |
| `--primario-texto` | `#151515` | `#fbfaf8` | botão principal (texto) |
| `--link` | `#7cb7f2` | `#1f63c4` | links |
| `--foco` | `#7cb7f2` | `#1f63c4` | contorno de foco |

**Etiquetas e estados** (fundo escuro saturado + texto claro, como as categorias do Cookbook):

| Token | Escuro (fundo / texto) | Claro (fundo / texto) | Uso |
|---|---|---|---|
| `azul` | `#0c2440` / `#86bdf7` | `#e3eefc` / `#1d58ad` | informação, em andamento |
| `verde` | `#10281a` / `#7fd49a` | `#e3f4e8` / `#1d7a3e` | concluído, aprovado |
| `vermelho` | `#3a1410` / `#f28b74` | `#fbe7e3` / `#b3361f` | erro, atrasado |
| `ambar` | `#33240c` / `#efb663` | `#fbf0dc` / `#8f5a0b` | aguardando, beta, atenção |
| `cinza` | `#262625` / `#b4b3ad` | `#ecebe6` / `#55544f` | neutro, bloqueado |

### Tipografia

| Token | Valor |
|---|---|
| `--fonte-titulo` | `"Source Serif 4 Variable", Georgia, serif` |
| `--fonte` | `"Inter Variable", system-ui, sans-serif` |
| `--fonte-mono` | `"JetBrains Mono Variable", ui-monospace, monospace` |

| Estilo | Tamanho / entrelinha / peso | Fonte |
|---|---|---|
| Título de página (h1) | 30 / 1,2 / 500 | serifa |
| Seção (h2) | 23 / 1,3 / 500 | serifa |
| Subseção (h3) | 18 / 1,4 / 600 | sem serifa |
| Destaque (lead) | 17 / 1,6 / 400, `--texto-suave` | sem serifa |
| Texto de leitura | 16 / 1,7 / 400 | sem serifa |
| Interface | 14 / 1,45 / 400–500 | sem serifa |
| Pequeno (rótulos, datas) | 12,5 / 1,4 / 500 | sem serifa |
| Código | 13,5 / 1,6 | mono |

### Espaço, raio, tamanho

- Escala de espaço: 4, 8, 12, 16, 24, 32, 48, 64 (`--e1` … `--e8`).
- Raio: `--raio-e` 4 px (etiqueta, `kbd`, código em linha), `--raio-p` 6 px (botão, campo, item de
  navegação), `--raio` 10 px (cartão, bloco de código, abas da página), `--raio-g` 16 px (painel
  central tipo login/diálogo).
- Altura de controle: **32 px** (`--controle`), **36 px** no toque (`pointer: coarse`).
- Ícone: 16 px, traço 1,5 px.
- Larguras e alturas nomeadas: barra do topo `--barra` 56 px · navegação `--nav` 260 px · texto
  `--leitura` 720 px · "nesta página" `--sumario` 240 px.
- Movimento: `--transicao` 120 ms ease, só em cor e fundo.

---

## 3. Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ ◢ App          aba   aba                                ⚙   ◐   [ Ação ]       │ ← barra 56 px
├───────────────┬──────────────────────────────────────────────┬───────────────┤
│ grupo         │ Seção › Página                               │ ┌───────────┐ │
│  item         │                                              │ │ resumo    │ │
│ ▌ item ativo  │ Título serifado               [Copiar ▾]     │ │ etiquetas │ │
│   item        │ destaque em cinza                            │ └───────────┘ │
│               │                                              │ ▌ Seção ativa │
│ grupo         │ texto de leitura…                            │   Seção       │
│  item         │                                              │     Sub       │
└───────────────┴──────────────────────────────────────────────┴───────────────┘
```

- **Barra do topo:** marca à esquerda (serifa, 19 px), abas de texto no centro (a ativa em pílula
  `--ativo`), à direita os botões discretos de Ajustes e de tema e **um** botão principal.
- **Navegação lateral:** títulos de grupo em 14 px/600 `--texto`; itens em 14 px `--texto-suave`;
  item ativo com fundo `--ativo` e texto `--texto`, raio 6 px. A busca (`.busca`) fica dentro das
  telas que a têm, não na lateral nem na barra.
- **Conteúdo:** trilha de navegação (`.migalhas`, pequena, com `›`), cabeçalho com título serifado
  e ação pequena à direita (`.cabecalho`), destaque, texto.
- **"Nesta página":** um cartão de resumo opcional e o sumário com fio de 1 px à esquerda; a seção
  atual em `--texto`/600 com o fio mais forte; subseções recuadas.
- **Celular (< 900 px):** uma coluna; a navegação lateral vira gaveta (botão ☰ na barra); o "nesta
  página" some. As abas do topo dão lugar a uma **barra de abas fixa no pé** (`.abas-celular`:
  ícone + nome, aba ativa com fundo `--superficie-2`). **Abaixo de 1200 px** o "nesta página" some.

Nem toda app precisa das três colunas: o `Pagina.vue` da Trilha já omite a lateral e/ou o "nesta
página" quando não há o que pôr ali.

---

## 4. Componentes

| Componente | Como é |
|---|---|
| **Botão principal** (`.botao.primario`) | fundo `--primario`, texto `--primario-texto`, 32 px, raio 6, 14 px/500. Um por tela |
| **Botão secundário** (`.botao`) | fundo `--superficie-2`, borda `--borda`, texto `--texto` |
| **Botão discreto** (`.botao.discreto`) | sem fundo nem borda; hover com `--superficie-2` (ex.: "Copiar página") |
| **Botão de ícone** (`.botao.icone`) | quadrado na altura do controle, só o ícone; na barra usa-se com `discreto` |
| **Botão largo** (`.botao.largo`) | ocupa a largura toda (ex.: dentro do painel central) |
| **Botão dividido** (`.botao-dividido`) | botão + seta ▾ grudados, divisória de 1 px |
| **Botão flutuante** (`.flutuante`) | botão principal fixo no canto inferior direito (como "Ask Docs") |
| **Campo** (`.campo`) | fundo `--superficie-2`, borda `--borda`, 32 px (área de texto livre), placeholder `--texto-fraco`; foco com contorno `--foco` |
| **Busca** (`.busca`) | campo com ícone de lupa; quando a tela tem atalho de teclado, ele aparece em `<kbd>` à direita (ex.: `Ctrl K`) |
| **Seletor em pílula** (`.seletor`) | "Categoria **Todas** ▾": rótulo fraco + valor forte |
| **Abas** (`.abas`) | texto `--texto-suave`; ativa com fundo `--ativo` e texto `--texto` (como as linguagens do bloco de código) |
| **Abas da página** (`.abas-pagina`) | barra em pílula `--superficie` com borda e raio 10, presa no topo ao rolar (abaixo da barra do app); aba 14/500 `--texto-suave`, ativa com fundo `--fundo` e contorno `--borda`; contagem em pílula `--superficie-2` 12 px; ✓ verde quando a parte está pronta. Rola de lado no celular |
| **Cartão** (`.cartao`) | fundo transparente ou `--superficie` (`.cheio`), borda 1 px `--borda`, raio 10, padding 16; título 16/600, descrição `--texto-suave` cortada em 2 linhas; hover com borda `--borda-forte` |
| **Lista em tabela** (`.tabela`) | cabeçalho 13 px `--texto-fraco`; linhas com divisória `--borda`; título 15/600 + descrição `--texto-suave`; etiquetas; data à direita em `--texto-fraco` |
| **Etiqueta** (`.etiqueta.azul` etc.) | 12 px/500, padding 2×6, raio 4, cores da tabela de etiquetas |
| **Código** | em linha: mono, fundo `--superficie-2`, borda `--borda`, raio 4. Em bloco (`.codigo`): cartão `--superficie` com abas no topo e botão de copiar |
| **Painel central** (`.painel`) | raio 16, borda `--borda`, padding 24, largura até 440 px, sobre fundo com **grade de pontos** sutil (`.fundo-pontos`, cor `--pontos`; como o login do Console). Para diálogos e telas vazias |
| **Diálogo** (`.dialogo-fundo` + `.painel`) | véu escuro (preto 55%) centralizado; o painel cresce até 520 px e rola |
| **Aviso** (`.aviso`, `.aviso.neutro`) | faixa de erro: fundo `--vermelho-fundo`, texto `--vermelho`, raio 6; a neutra usa `--superficie-2`/`--texto-suave` |
| **Barra de progresso** (`.progresso`) | trilho de 4 px `--superficie-2`, preenchimento `--verde` |
| **Lista de verificação** (`.verificacao`) | ícone de check verde de 16 px + texto 14 px |
| **Sumário "nesta página"** (`.sumario`) | ver layout |
| **Comentário** (`.comentario`) | para propostas: cartão pequeno com autor, âncora citada em `--texto-fraco` e texto; o trecho comentado no texto ganha sublinhado pontilhado `ambar` |

**Foco:** contorno de 2 px `--foco` com 2 px de afastamento, sempre visível no teclado
(`:focus-visible`).

**Movimento:** transições de 120 ms só em cor e fundo. Nada de animação decorativa.

---

## 5. Convenções entre apps

O que já era combinado no ecossistema e continua valendo:

- **Ajustes em níveis:** a primeira tela só tem o título "Ajustes" e a lista das seções (nome,
  descrição de uma linha e uma seta); cada seção abre numa página própria com "‹ Ajustes" para
  voltar, o nome da seção como título e uma descrição. Nada de listar tudo numa tela só.
- **Sugestões de melhoria:** toda app tem a seção em Ajustes (campo de texto, botão "Enviar
  sugestão" e a lista com estado e resposta), sobre a API comum `/api/sugestoes`
  (ver ARQUITETURA); as que usam IA têm também **Instruções da IA**.
- **Tema:** escuro é a base; o claro deriva dos mesmos tokens. Segue o aparelho por padrão, com
  escolha em Ajustes ou no botão da barra (`tema.js` da Trilha é a referência).
- **Ícone da app:** cada app tem o seu desenho de linha, diferente dos outros (o da Trilha é um
  traço em degraus; o do Hub, dois servidores empilhados). Ícones de interface: 16 px, traço
  1,5 px, desenhados à mão (`Icone.vue`).
- **Vitrine:** cada app portada mantém a sua vitrine em `#/design` (sem aba; só pelo endereço),
  espelhando a da Trilha.

## 6. Regras para quem for mexer

- Toda cor, raio e tamanho vem de um token. Valor solto no componente é erro.
- Um botão principal por tela.
- Título de página e seção em serifa; nunca serifa em botão, rótulo ou navegação.
- Texto de leitura nunca passa de 720 px de largura.
- Mudou o design? Atualize este arquivo **e** a vitrine em `#/design`.
