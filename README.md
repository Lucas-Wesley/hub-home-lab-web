# Hub

Aplicação central que vai **reunir e controlar** as outras aplicações do ecossistema (Acervo,
Biblioteca Digital, Flashcards) numa só interface — primeiro web, depois mobile e desktop.

Repositório: `github.com/Lucas-Wesley/hub-home-lab-web`

> **Status:** **sem implementação por enquanto — de propósito.** O Hub é o **último** a ser
> construído, porque ele só *consome* as APIs dos outros apps (é um cliente, não dono dos dados).
> Construí-lo cedo demais trava tudo.

## Por que este repositório existe agora

Além de ser o futuro app central, o Hub é o **ponto único de referência** do ecossistema. Os
documentos que valem para **todos** os apps vivem aqui:

- [`docs/ARQUITETURA.md`](docs/ARQUITETURA.md) — como as aplicações se encaixam e se comunicam.
- [`docs/CONTRATO.md`](docs/CONTRATO.md) — o **contrato comum** (formato de dados + convenções de
  API) que todos os apps seguem para conversar entre si.
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — a **ordem de desenvolvimento** das aplicações e os
  princípios que valem pra todas.

## O que o Hub será (quando chegar a hora)

Uma interface que lista e cruza o conteúdo dos apps: o Acervo (Temas, textos principais,
referências), a Biblioteca Digital (livros, anotações) e os baralhos do Flashcards — consumindo
as APIs de cada um. Não terá banco próprio de conteúdo; no máximo, estado de UI.
