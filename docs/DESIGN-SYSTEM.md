# Design system das apps do homelab

**A fonte é o Hub.** Este arquivo, os tokens e classes do `frontend/src/style.css` e a vitrine viva
em `hub.internal/#/design` são a referência de todas as apps: Biblioteca Digital, Acervo,
Flashcards, Streaming, Trilha de Estudo e as próximas. Mudança de design acontece **aqui primeiro**;
depois, quando o Lucas pedir, é copiada para as outras apps.

| App | Situação (18/11/2026) |
|---|---|
| **Hub** | quarta versão (este documento) |
| Trilha de Estudo | terceira versão: mesmos tokens de cor, fonte e raio; espaço antigo. O `docs/DESIGN-SYSTEM.md` de lá aponta para cá |
| Biblioteca, Acervo, Flashcards, Streaming, plugin | segunda versão (10/11); a cópia do documento nelas está desatualizada |

## Histórico

- **Quarta versão (18/11/2026), espaço.** Pedido do Lucas: a tela "ocupa muito espaço à toa", com
  tudo empilhado na vertical. A referência de **organização de espaço** (não de cor nem de fonte)
  são prints de uma plataforma de curso (§3). Mudou: sem barra no topo; cabeçalho da página numa
  linha só; conteúdo largo (até 1200 px) com painel lateral; quadros com rótulo de seção; títulos e
  listas menores e mais densos.
- **Terceira versão (18/11/2026), estilo.** O design que nasceu na Trilha de Estudo (T14) vira o
  padrão: estilo da **Claude Platform** (Docs, Console e Cookbook), referências em
  [`design/referencias/`](design/referencias/). Fundo quase preto, serifa nos títulos, botão
  principal invertido, fim da cor por app e do vidro/blur, fontes embutidas.
- **Segunda versão (10/11/2026):** neutros frios, fonte do sistema, uma cor por app, vidro e blur.

---

## 1. Princípios

1. **Escuro, neutro e quieto.** Fundo quase preto (`#151515`), superfícies um ou dois tons acima,
   bordas finas. Sem gradiente, sem sombra pesada. A cor aparece só em links, etiquetas e estados.
2. **O espaço vai para o conteúdo.** Nada de barra no topo, título gigante ou bloco sozinho numa
   linha. O que se usa junto fica na mesma linha (título + busca; abas + navegação da semana); o que
   acompanha o conteúdo vai para um painel ao lado, não para baixo.
3. **Pequeno e denso, mas respirando.** Botões de 32 px, interface de 14 px, ícones de 16 px,
   linhas de lista de ~40 px. O respiro vem do agrupamento (quadros, fios) e não do tamanho.
4. **O que mais importa aparece primeiro e maior.** Números importantes em destaque no topo
   (quadros de número), o resto em listas compactas.
5. **Serifa para títulos, sem serifa para o resto.** Título de página e de seção em serifa; texto,
   interface e rótulos sem serifa; código em monoespaçada.
6. **Um botão principal por tela**, invertido (claro sobre o escuro). O resto é discreto.
7. **Leitura:** texto corrido nunca passa de 720 px, mesmo com a página larga.

### O que não dá para copiar das referências (e o substituto)

| Na referência | Problema | Aqui |
|---|---|---|
| Fontes próprias da Anthropic | proprietárias | **Source Serif 4** (títulos), **Inter** (texto e interface), **JetBrains Mono** (código), embutidas (offline) |
| Logo e marca | de terceiros | desenho de ícone próprio de cada app |
| Tema claro | os prints são só do escuro | tema claro derivado dos mesmos princípios; segue o aparelho, com escolha em Ajustes |
| Cor de marca da plataforma de curso (azul-petróleo) | a referência dela é só de espaço | botão principal invertido; cor só em estados |

---

## 2. Tokens

### Cor

| Token | Escuro | Claro | Uso |
|---|---|---|---|
| `--fundo` | `#151515` | `#fbfaf8` | página, quadros |
| `--superficie` | `#1b1b1a` | `#f4f3ef` | lateral de navegação, cartões cheios, blocos de código |
| `--superficie-2` | `#232322` | `#ecebe6` | campos, botão secundário, hover, trilho de barras |
| `--ativo` | `#2e2e2d` | `#e4e3dd` | item ativo da navegação, aba ativa, dia escolhido |
| `--borda` | `#2c2c2b` | `#e3e2dc` | bordas e divisórias |
| `--borda-forte` | `#3d3d3b` | `#cfcec7` | hover de campo, borda do item escolhido |
| `--pontos` | `#262625` | `#e6e5e0` | grade de pontos atrás do painel central (`.fundo-pontos`) |
| `--texto` | `#ececea` | `#1d1d1b` | texto principal |
| `--texto-suave` | `#a3a29d` | `#5f5e59` | descrições, navegação inativa, preenchimento do medidor |
| `--texto-fraco` | `#73726d` | `#8b8a84` | rótulos de seção, datas, placeholders |
| `--primario` | `#f5f5f3` | `#1d1d1b` | botão principal (fundo) |
| `--primario-texto` | `#151515` | `#fbfaf8` | botão principal (texto) |
| `--link` | `#7cb7f2` | `#1f63c4` | links, ação do quadro |
| `--foco` | `#7cb7f2` | `#1f63c4` | contorno de foco |

**Etiquetas e estados** (fundo escuro saturado + texto claro), como `--azul-fundo` e `--azul`:

| Cor | Escuro (fundo / texto) | Claro (fundo / texto) | Uso |
|---|---|---|---|
| `azul` | `#0c2440` / `#86bdf7` | `#e3eefc` / `#1d58ad` | informação, hoje, em andamento |
| `verde` | `#10281a` / `#7fd49a` | `#e3f4e8` / `#1d7a3e` | concluído, ligado, meta atingida |
| `vermelho` | `#3a1410` / `#f28b74` | `#fbe7e3` / `#b3361f` | erro, falta, choque, quase cheio |
| `ambar` | `#33240c` / `#efb663` | `#fbf0dc` / `#8f5a0b` | aguardando, atenção |
| `cinza` | `#262625` / `#b4b3ad` | `#ecebe6` / `#55544f` | neutro, bloqueado |

**Tokens próprios de uma app:** quando uma peça precisa de cores que não são estado (as cores das
atividades da Rotina, `--r-azul` … `--r-cinza`), a app cria tokens com um **prefixo da peça**,
nos dois temas, no bloco de tokens do `style.css`, e documenta aqui. Uso só em barras, pontos e
fundos leves (`color-mix` com 16–18% sobre `--fundo`), nunca em texto corrido nem em botão.

| Peça | Tokens | Onde |
|---|---|---|
| Rotina (Hub) | `--r-azul`, `--r-violeta`, `--r-verde`, `--r-laranja`, `--r-amarelo`, `--r-rosa`, `--r-turquesa`, `--r-cinza`, via as classes `.cor-*` (que definem `--cor`) | fio do bloco, ponto da meta, pedaço da grade |

### Tipografia

| Token | Valor |
|---|---|
| `--fonte-titulo` | `"Source Serif 4 Variable", Georgia, serif` |
| `--fonte` | `"Inter Variable", system-ui, sans-serif` |
| `--fonte-mono` | `"JetBrains Mono Variable", ui-monospace, monospace` |

| Estilo | Tamanho / entrelinha / peso | Fonte |
|---|---|---|
| Título de página (h1) | 26 / 1,2 / 500 | serifa |
| Seção (h2) | 20 / 1,3 / 500 | serifa |
| Subseção (h3) | 16 / 1,4 / 600 | sem serifa |
| Rótulo de seção (`.rotulo-secao`) | 12 / 1,3 / 600, caixa alta, espaçado 0,07em, `--texto-fraco` | sem serifa |
| Número em destaque | 24 / 1,2 / 600, algarismos tabulares | sem serifa |
| Destaque (`.lead`) | 15 / 1,6 / 400, `--texto-suave` | sem serifa |
| Texto de leitura (`.leitura`) | 16 / 1,7 / 400 | sem serifa |
| Interface | 14 / 1,45 / 400–500 | sem serifa |
| Item de lista | 14,5 / 600 (nome) + 13 `--texto-suave` (nota) | sem serifa |
| Pequeno (datas, valores) | 12,5 / 1,4 / 500 | sem serifa |
| Código | 13,5 / 1,6 | mono |

### Espaço, raio, tamanho

- Escala de espaço: 4, 8, 12, 16, 24, 32, 48, 64 (`--e1` … `--e8`).
- Raio: `--raio-e` 4 px (etiqueta, `kbd`), `--raio-p` 6 px (botão, campo, item de navegação),
  `--raio` 10 px (quadro, cartão, código), `--raio-g` 16 px (painel central, diálogo).
- Altura de controle: **32 px** (`--controle`), **36 px** no toque (`pointer: coarse`).
- Larguras: navegação `--nav` 240 px · página `--pagina` 1200 px · painel lateral `--lateral`
  320 px · texto `--leitura` 720 px.
- Respiro da página: 24 px no topo, 32 px nas laterais (16 px no celular).
- Ícone: 16 px, traço 1,5 px. Movimento: `--transicao` 120 ms ease, só em cor e fundo.

---

## 3. Espaço e layout

### A referência de espaço

Prints de uma plataforma de curso de idiomas, mandados pelo Lucas em 18/11/2026 (não guardados no
repositório). Valem **só pela organização do espaço**, não pela cor nem pela fonte. O que se tira
deles:

- **Sem barra no topo.** A marca abre a lateral (rótulo pequeno em caixa alta + nome); a lateral
  tem a navegação e, no pé, o que é secundário.
- **Cabeçalho numa linha:** à esquerda o caminho ou título; à direita as ações (busca, avisos,
  conta). Nada fica sozinho numa linha.
- **Conteúdo + painel lateral:** na aula, o vídeo e o texto ao centro e, à direita, o progresso e a
  lista de módulos. O painel acompanha o conteúdo em vez de ir para o pé da página.
- **Números em quadros lado a lado** no topo do painel ("Dias estudados 1", "Aulas concluídas 2/34").
- **Quadros com rótulo de seção** em caixa alta pequena e a ação à direita ("Próximas aulas ·
  Ir para as aulas").
- **Listas densas:** linhas de ~40 px, valor à direita (porcentagem, duração), ícone pequeno.
- **O valor fica na linha do rótulo** ("Progresso da aula … 43% · 25:41 de 59:06"), com a barra
  logo abaixo e a ação ao lado.

### O layout

```
┌───────────────┬──────────────────────────────────────────────────────────────────┐
│ ▣ HOMELAB     │ Título   [ ferramenta principal: busca / adicionar ]     ↻  [Ação] │ ← .topo-pagina
│   Hub         │ Abas  Abas  Abas                        ‹ 16–22 nov. ›  Dia  Sem.  │ ← .ferramentas
│               │ ┌──────┐┌──────┐┌──────┐┌──────┐                                   │
│ ⌂ Início      │ │RÓTULO││RÓTULO││RÓTULO││RÓTULO│  ← .numeros-quadro                 │
│ ▌ Rotina      │ │ 61%  ││ 9%   ││ 1    ││ 3    │                                    │
│   Servidor    │ └──────┘└──────┘└──────┘└──────┘                                    │
│   …           │ ┌─────────────────────────────────────┐ ┌──────────────────────┐   │
│               │ │ conteúdo principal                  │ │ RÓTULO        Ação → │   │
│               │ │ (lista do dia, grade, quadros)      │ │ linha            7/7 │   │
│               │ │                                     │ │ linha            3/4 │   │ ← .lateral
│               │ └─────────────────────────────────────┘ └──────────────────────┘   │
│ versão   ◐    │                                    .duas-colunas                   │
└───────────────┴──────────────────────────────────────────────────────────────────┘
```

- **Lateral** (`--nav` 240 px, fundo `--superficie`, presa ao rolar): marca no alto (ícone num
  quadradinho de 32 px + "HOMELAB" 10,5 px em caixa alta + nome em serifa 18 px); itens de 14 px
  `--texto-suave`, o ativo com fundo `--ativo`; no pé, a versão e o botão de tema.
- **Página** (`.pagina`): até 1200 px, centrada, 24 px de respiro no topo.
- **Cabeçalho** (`.topo-pagina`): `h1` + `.meio` (a ferramenta principal, até 640 px) + `.acoes`
  (ou um botão solto, sempre empurrado à direita). Quebra em duas linhas só quando não cabe.
- **Barra de ferramentas** (`.ferramentas`): abas à esquerda; à direita (`.direita`) a navegação
  (‹ período ›) e as alternâncias de visão.
- **Duas colunas** (`.duas-colunas`): conteúdo `1fr` + `.lateral` 320 px a partir de 1100 px; o
  painel fica preso ao rolar. Abaixo de 1100 px, o painel desce para depois do conteúdo.
- **Quadros lado a lado** (`.quadros`): grade automática de colunas de no mínimo 340 px.
- **Texto explicativo** (`.explicacao`): uma ou duas linhas em `--texto-suave`, até 720 px, logo
  abaixo do cabeçalho ou das ferramentas. Explicação longa vai para Ajustes, não para o topo.
- **Celular (< 900 px):** sem lateral; **barra de abas fixa no pé** (`.abas-celular`: ícone + nome,
  nome curto quando não cabe, aba ativa com fundo `--superficie-2`). O cabeçalho quebra em título e
  ferramenta; as abas da página rolam de lado numa linha só (< 700 px). O tema fica em Ajustes.

---

## 4. Componentes

| Componente | Como é |
|---|---|
| **Cabeçalho da página** (`.topo-pagina`, `.meio`, `.acoes`) | ver §3 |
| **Barra de ferramentas** (`.ferramentas`, `.direita`) | ver §3 |
| **Quadro** (`.quadro`, `.quadro-topo`) | borda 1 px `--borda`, raio 10, padding 16, fundo `--fundo`. Topo com `.rotulo-secao` (pode ser um `h2`) e, à direita, a ação como link 13/500 `--link` (`a` ou `.link-acao`). Lista dentro do quadro sem fio no topo nem no fim |
| **Quadro de número** (`.numeros-quadro` > `.numero-quadro`) | rótulo de seção, valor 24/600 (`.valor-grande`) e um `.pequeno` embaixo; colunas de no mínimo 150 px |
| **Rótulo de seção** (`.rotulo-secao`) | 12/600, caixa alta, espaçado, `--texto-fraco`. Nome de quadro, de grupo, de número |
| **Botão principal** (`.botao.primario`) | fundo `--primario`, texto `--primario-texto`, 32 px, raio 6, 14/500. Um por tela |
| **Botão secundário** (`.botao`) | fundo `--superficie-2`, borda `--borda`, texto `--texto` |
| **Botão discreto** (`.botao.discreto`) | sem fundo nem borda; hover com `--superficie-2` |
| **Botão de ícone** (`.botao.icone`) | quadrado na altura do controle, só o ícone (com `aria-label`) |
| **Botão largo / perigo** (`.botao.largo`, `.botao.perigo`) | largura toda / texto `--vermelho` |
| **Botão dividido** (`.botao-dividido`) | botão + seta ▾ grudados, divisória de 1 px |
| **Campo** (`.campo`) | fundo `--superficie-2`, borda `--borda`, 32 px, **raio 6** (nunca pílula), placeholder `--texto-fraco`; foco com contorno `--foco` |
| **Busca** (`.busca`) | campo com lupa à esquerda; atalho em `<kbd>` à direita quando existe |
| **Seletor em pílula** (`.seletor`) | "Categoria **Todas** ▾": rótulo fraco + valor forte |
| **Abas** (`.abas`) | 28 px, texto `--texto-suave`; ativa com fundo `--ativo` e texto `--texto`. Também para alternâncias (Dia / Semana inteira) |
| **Seletor de dias** (Rotina) | os 7 dias numa faixa com borda, raio 10; cada dia "S 16" numa linha; escolhido com fundo `--ativo` e borda `--borda-forte`; hoje em `--azul`; "!" vermelho no canto quando há choque |
| **Lista** (`.lista`) | linhas separadas por fio `--borda`, padding vertical 8–10 px |
| **Lista em tabela** (`.tabela`) | cabeçalho 13 px `--texto-fraco`; título 15/600 + descrição `--texto-suave` |
| **Etiqueta** (`.etiqueta.azul` etc.) | 20 px de altura, 12/500, padding 0×6, raio 4 |
| **Cartão** (`.cartao`, `.cheio`) | borda `--borda`, raio 10, padding 16; título 16/600, descrição em até 2 linhas; hover `--borda-forte` |
| **Medidor de uso** (`Medidor.vue`) | nome à esquerda e "x de y" à direita na mesma linha; trilho de 6 px `--superficie-2`, preenchimento `--texto-suave`; vermelho (trilho e preenchimento) a partir de 85% |
| **Barra de progresso** (`.progresso`) | trilho de 4 px `--superficie-2`, preenchimento `--verde`. Para o que se conclui (meta, curso), não para uso |
| **Aviso** (`.aviso`, `.aviso.neutro`) | erro: `--vermelho-fundo`/`--vermelho`, raio 6; neutro: `--superficie-2`/`--texto-suave` |
| **Recado** (`.recado`) | aviso rápido no pé da tela, some em 4 s |
| **Painel central** (`.painel` sobre `.fundo-pontos`) | raio 16, borda, padding 24, até 440 px. Para diálogos e telas vazias |
| **Diálogo / folha** (`.dialogo-fundo` + `.painel`) | véu preto 55%; no celular a folha sobe do pé |
| **Código** | em linha: mono, `--superficie-2`, raio 4. Em bloco (`.codigo`): cartão `--superficie` |
| **Lista de verificação** (`.verificacao`) | check verde de 16 px + texto 14 px |
| **Comentário** (`.comentario`) | cartão pequeno com autor, âncora em `--texto-fraco` e texto |

**Foco:** contorno de 2 px `--foco` com 2 px de afastamento (`:focus-visible`).

**Cuidado com classes genéricas em estilos `scoped`:** o Vue só marca o último seletor. Um
`.corpo:hover .nome` dentro de uma tela também casa com um `.corpo` do App, que envolve tudo.
Prenda o seletor à peça (`.bloco > .corpo:hover .nome`) e não use nomes genéricos no App.

---

## 5. Convenções entre apps

- **Ajustes em níveis:** a primeira tela tem o título "Ajustes", o tema e a lista das seções
  (nome, descrição de uma linha e uma seta); cada seção abre numa página própria com "‹ Ajustes".
- **Sugestões de melhoria:** toda app tem a seção em Ajustes, sobre a API comum `/api/sugestoes`
  (ver ARQUITETURA); as que usam IA têm também **Instruções da IA**.
- **Tema:** escuro é a base; o claro deriva dos mesmos tokens. Segue o aparelho por padrão, com
  escolha em Ajustes e no botão do pé da lateral (`tema.js`).
- **Ícone da app:** cada app tem o seu desenho de linha (o do Hub: dois servidores empilhados; o
  da Trilha: um traço em degraus). Ícones de interface desenhados à mão em `Icone.vue`.
- **Vitrine:** `#/design`, sem aba, só pelo endereço. A do Hub é a referência.

## 6. Levar o design para outra app

1. Copiar o bloco de tokens e as classes do `frontend/src/style.css` do Hub (até "peças que só o
   Hub tem"), as fontes (`@fontsource-variable/*`) e o `tema.js`.
2. Tirar a barra do topo: marca no alto da lateral, tema no pé dela e em Ajustes.
3. Trocar os cabeçalhos por `.topo-pagina` e agrupar o que vai junto (`.ferramentas`, `.quadro`,
   `.duas-colunas`).
4. Manter só os tokens próprios da app (com prefixo) e registrá-los na tabela do §2.
5. Copiar este arquivo para o `docs/DESIGN-SYSTEM.md` da app como um ponteiro para cá (como na
   Trilha) e atualizar a tabela de situação no topo.

## 7. Regras para quem for mexer

- Toda cor, raio e tamanho vem de um token. Valor solto no componente é erro.
- Um botão principal por tela.
- Título de página e seção em serifa; nunca serifa em botão, rótulo ou navegação.
- Nada sozinho numa linha quando cabe ao lado de outra coisa; o painel lateral antes do pé da
  página.
- Texto corrido nunca passa de 720 px.
- Mudou o design? Atualize este arquivo **e** a vitrine em `#/design`.
