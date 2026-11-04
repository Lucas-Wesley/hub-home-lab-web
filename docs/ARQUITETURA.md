# Arquitetura do Ecossistema

> Visão de nível alto de como as aplicações pessoais se encaixam. Vale para **todos** os projetos.
> Atualizado em **03/11/2026**.

---

## 1. Princípio

Um conjunto de **aplicações independentes e fracamente acopladas** (loosely coupled). Cada app:

- é **dono do próprio banco de dados** (ninguém lê o banco do outro);
- expõe uma **API** (HTTP + JSON);
- **conversa com os outros pela API**, nunca por banco compartilhado;
- é **útil sozinho**, mesmo sem os demais.

O **Hub** é apenas um **cliente** que consome essas APIs e junta tudo numa interface. Não é um
"monolito" que contém os outros.

```
                    ┌──────────────────────────────┐
                    │   Hub  (web → mobile/desktop) │   ← só CONSOME as APIs
                    │        cliente, sem BD próprio │
                    └───────────────┬──────────────┘
            ┌───────────────────────┼────────────────────────┐
            │                       │                         │
     ┌──────▼──────┐         ┌──────▼───────┐         ┌───────▼──────┐
     │   Acervo    │◄───────►│  Biblioteca  │         │  Flashcards  │
     │  API + BD   │         │ Digital API+BD│◄───────►│   API + BD   │
     └─────────────┘         └──────────────┘         └──────────────┘
```

---

## 2. As aplicações

| App | O que é | Repositório | Estado |
|-----|---------|-------------|--------|
| **Acervo** | Gerenciador de conhecimento (Temas + referências) | github.com/Lucas-Wesley/home-lab-acervo | Em planejamento |
| **Biblioteca Digital** | Leitor/biblioteca de PDF e EPUB com anotações | github.com/Lucas-Wesley/biblioteca-digital | Em planejamento |
| **Flashcards** | App estilo Anki (Vue + Go + SQLite) | github.com/Lucas-Wesley/flashcards | Repo já existe (plano) |
| **Hub** | Interface central que reúne todos | github.com/Lucas-Wesley/hub-home-lab-web | Último a construir |

---

## 3. Integrações previstas

| De → Para | O que acontece |
|-----------|----------------|
| Biblioteca Digital → Acervo | Capítulo/parágrafo vira **texto de referência** ou alimenta um **Tema** |
| Biblioteca Digital → Flashcards | **Seleção** de texto vira **flashcard** |
| Acervo → Flashcards | Trecho de referência/Tema vira **flashcard** |
| Hub → todos | Lê as APIs e mostra tudo junto |

Toda integração é uma **chamada de API** que troca um **Fragmento de Conteúdo** — ver
[`CONTRATO.md`](CONTRATO.md).

---

## 4. Convenções do ecossistema (manter simples)

Contexto que simplifica tudo: **usuário único + tudo na LAN**. Então **nada** de Kubernetes,
message bus ou service mesh. O padrão mínimo:

- Toda app roda como **container no Proxmox**.
- Toda app expõe **HTTP + JSON**.
- Todas ficam atrás do **mesmo reverse proxy** na LAN, com nomes estáveis.
- Serviços **LAN-only, não expostos à internet**.
- Armazenamento de arquivos/conteúdo no **SSD externo** do homelab.

---

## 5. Ordem de construção

```
1. Acervo (em andamento)  →  2. Biblioteca Digital  →  3. cada app expõe sua API
                                                        (Flashcards ganha API)  →  4. Hub
```

**Regra de ouro:** o Hub é o último. Cada app precisa existir e ser útil sozinho antes de amarrar.

---

## 6. Em aberto

- [ ] **Nome do ecossistema/suite** (e do Hub).
- [ ] Reverse proxy e esquema de nomes na LAN.
- [ ] Autenticação compartilhada (ver CONTRATO — por ora, token simples).
