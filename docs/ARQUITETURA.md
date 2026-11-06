# Arquitetura do Ecossistema

> Visão de nível alto de como as aplicações pessoais se encaixam. Vale para **todos** os projetos.
> Atualizado em **06/11/2026**.

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
| **Acervo** | Gerenciador de conhecimento (Temas + referências) | github.com/Lucas-Wesley/home-lab-acervo | No ar (`:8444`) |
| **Biblioteca Digital** | Leitor/biblioteca de PDF e EPUB com anotações | github.com/Lucas-Wesley/biblioteca-digital | No ar (`:8443`) |
| **Flashcards** | App estilo Anki (Vue + Go + SQLite) | github.com/Lucas-Wesley/flashcards | No ar (`:8445`) |
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

**A primeira já existe (06/11/2026): Acervo → Biblioteca, só leitura.** O Pesquisador do Acervo
(D27) chama a busca no texto dos livros da Biblioteca (`GET /api/search?q=`, R32), pela rede
interna do Docker (`http://biblioteca:8080`), e cita livro e página com o link de volta para o
ponto do livro. É uma consulta, não uma troca de conteúdo: o formato é o da própria API da
Biblioteca, e o Fragmento continua para quando um app **mandar** conteúdo para o outro.

---

## 4. Convenções do ecossistema (manter simples)

Contexto que simplifica tudo: **usuário único + tudo na LAN**. Então **nada** de Kubernetes,
message bus ou service mesh. O padrão mínimo:

- Toda app roda como **container no Proxmox**.
- Toda app expõe **HTTP + JSON**.
- Todas ficam atrás do **mesmo reverse proxy** na LAN, com nomes estáveis.
- Serviços **LAN-only, não expostos à internet**.
- Armazenamento de arquivos/conteúdo no **disco interno** do homelab.

Como isso ficou na prática, com a Biblioteca e o Acervo:

- **Stack:** API em **Go**, banco **SQLite** do próprio app e front-end **Vue** como **PWA**, com
  offline.
- **Onde roda:** VM `apps` (`192.168.1.20`) no Proxmox, com **Docker Compose**, um container por
  app e dados em `/srv/data/<app>`.
- **Proxy:** **Caddy** com `tls internal`, uma porta por app (`:8443` Biblioteca, `:8444` Acervo, `:8445` Flashcards).
  - O certificado raiz para instalar nos aparelhos fica em `http://192.168.1.20/ca.crt`.
  - A configuração está na pasta `proxy/` do homelab, copiada para `/opt/proxy`.
- **Publicar:** `git push` na main e `./deploy.sh` de cada app.
  - O script roda os testes, constrói na VM, confere a saúde e volta ao commit anterior se falhar.
  - Segredos ficam em `/opt/apps/<app>.env`, fora do git.
- **Visual:** design system comum (neutros, azul e terracota só como contraste, vidro com blur,
  listas finas, Literata na leitura). A especificação fica em `docs/DESIGN-SYSTEM.md` de cada app,
  igual em todas.
- **Ajustes e sugestões de melhoria:** toda app tem **Ajustes** no fim do menu, com a seção
  **Sugestões de melhoria**. O Lucas anota ali o que pode melhorar; depois pede ao Claude Code para
  ver as sugestões, e ele lê, faz e responde. A API é a mesma em todas (campos em português):
  - `GET /api/sugestoes`: lista, com as novas primeiro.
  - `PUT /api/sugestoes/{id}` com `{texto, tela}`: cria. O id é gerado no aparelho, então reenviar
    não duplica.
  - `PATCH /api/sugestoes/{id}` com `{estado, resposta}`: o que o Claude Code usa ao terminar.
    Estados: `nova`, `feita` ou `descartada`.
  - `DELETE /api/sugestoes/{id}`.
  - Cada sugestão tem `id`, `texto`, `tela` (de onde o Lucas veio), `estado`, `resposta`,
    `criada_em` e `atualizada_em`.

---

## 5. Ordem de construção

```
1. Biblioteca Digital (no ar)  →  2. Acervo (no ar)  →  3. Flashcards (no ar) + APIs de integração
                                                          (próximo)  →  4. Hub
```

**Regra de ouro:** o Hub é o último. Cada app precisa existir e ser útil sozinho antes de amarrar.

---

## 6. Em aberto

- [ ] **Nome do ecossistema/suite** (e do Hub).
- [x] Reverse proxy: Caddy, uma porta por app. Nomes na LAN (em vez de IP) ficam para depois.
- [ ] Autenticação compartilhada (ver CONTRATO — por ora, token simples).
