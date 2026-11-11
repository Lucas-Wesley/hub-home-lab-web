# Hub

Aplicação central que vai **reunir e controlar** as outras aplicações do ecossistema (Acervo,
Biblioteca Digital, Flashcards) numa só interface — primeiro web, depois mobile e desktop.

Repositório: `github.com/Lucas-Wesley/hub-home-lab-web`

> **Status (10/11/2026):** no ar o **painel do servidor** (`https://hub.internal`):
> armazenamento do Proxmox, da VM e de cada app, memória e processador de cada app, contêineres e a limpeza do Docker com
> agenda. A parte de conteúdo (juntar Acervo, Biblioteca e Flashcards) continua para o fim, porque
> depende das APIs dos outros apps. Estado e decisões em [`docs/DECISOES.md`](docs/DECISOES.md).

## Rodar e publicar

- **Local:** `make dev-api` (API em `:8080`, lendo `data/apps` como se fosse `/srv/data`) e
  `make dev-web` (Vite).
- **Testes:** `make test` (Go, Vitest e Playwright com um Docker falso).
- **Publicar:** `git push origin main` e `./deploy.sh`.
- **Proxmox:** sem `/opt/apps/hub.env` com o `PROXMOX_TOKEN`, o painel abre sem os números do
  Proxmox e mostra como criar o token (ver `docs/DECISOES.md`, H3).

## Por que este repositório existe

Além de ser o app central, o Hub é o **ponto único de referência** do ecossistema. Os
documentos que valem para **todos** os apps vivem aqui:

- [`docs/ARQUITETURA.md`](docs/ARQUITETURA.md) — como as aplicações se encaixam e se comunicam.
- [`docs/CONTRATO.md`](docs/CONTRATO.md) — o **contrato comum** (formato de dados + convenções de
  API) que todos os apps seguem para conversar entre si.
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — a **ordem de desenvolvimento** das aplicações e os
  princípios que valem pra todas.

## O que o Hub será

Além do painel do servidor, uma interface que lista e cruza o conteúdo dos apps: o Acervo (Temas,
textos principais, referências), a Biblioteca Digital (livros, anotações) e os baralhos do
Flashcards, consumindo as APIs de cada um. Não terá banco próprio de conteúdo: o banco dele guarda
só a agenda e o histórico da limpeza e as sugestões.
