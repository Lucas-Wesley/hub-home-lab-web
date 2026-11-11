# Hub — Ecossistema de apps pessoais do Lucas

Este repositório é o **ponto único de referência** do ecossistema. **Leia primeiro:**
- [`docs/ARQUITETURA.md`](docs/ARQUITETURA.md) — como os apps se encaixam e se comunicam.
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — ordem de desenvolvimento + princípios.
- [`docs/CONTRATO.md`](docs/CONTRATO.md) — contrato comum (ainda **NÃO definido** de propósito; só está registrado que vai existir e será obrigatório).
- [`docs/DECISOES.md`](docs/DECISOES.md) — o app Hub em si: o painel do servidor (H1–H8), estado e próximos passos.

## O ecossistema em uma frase
Aplicações **independentes** (cada uma com banco + API próprios) que se falam **por API**; o **Hub é só um cliente** que junta tudo e é **construído por último**.

## Apps e repositórios
| App | Repo | Papel |
|-----|------|-------|
| Acervo | github.com/Lucas-Wesley/home-lab-acervo | Gerenciador de conhecimento (Temas + referências) — **2º, no ar** |
| Biblioteca Digital | github.com/Lucas-Wesley/biblioteca-digital | Leitor/biblioteca PDF+EPUB (PWA) com anotações — **1º, no ar** |
| Flashcards | github.com/Lucas-Wesley/flashcards | App estilo Anki (FSRS-6) — **3º, no ar** |
| Streaming | github.com/Lucas-Wesley/home-lab-streaming | Canais, filmes e séries, com o celular de controle remoto (`:8446`); pronto no Mac, ainda não publicado |
| Hub | github.com/Lucas-Wesley/hub-home-lab-web | Este repo — painel do servidor **no ar** (`:8449`); a parte de conteúdo é a última |

## Ambiente (homelab)
- **Proxmox VE 9.2** no Lenovo **M920q**, IP **192.168.1.10** (web: `https://192.168.1.10:8006`, SSH `root@192.168.1.10`). Tudo **LAN-only**, não exposto à internet.
- Cada app roda como **container** (Docker Compose) na VM `apps` (`192.168.1.20`), atrás do Caddy. Detalhes em `docs/ARQUITETURA.md` §4; estado atual em `docs/ROADMAP.md` ("Onde paramos").
- **Armazenamento:** somente o **disco interno do M920q**; não haverá SSD externo.

## Sugestões de melhoria dos apps
Cada app tem **Ajustes → Sugestões de melhoria** (contrato em `docs/ARQUITETURA.md` §4). Quando o Lucas pedir para "ver as sugestões":
1. Ler as novas: `curl -s --cacert certs/homelab-ca.crt https://192.168.1.20:<porta>/api/sugestoes` (8443 Biblioteca, 8444 Acervo, 8445 Flashcards, 8446 Streaming, 8449 Hub; o `certs/` fica na pasta `homelab`).
2. Conferir com ele o que não estiver claro, fazer, testar e publicar.
3. Marcar cada uma: `PATCH /api/sugestoes/{id}` com `{"estado":"feita","resposta":"<o que mudou>"}` (ou `descartada`, com o motivo).

## Regras e convenções
- **Conteúdo do Acervo NÃO sobe pro GitHub** (versionado localmente no homelab). Código das apps pode.
- Construir **cada app sozinho primeiro**; só criar endpoint de integração quando **os dois lados existirem**; **não definir o contrato comum ainda**.
- **Painel do servidor:** só lê, menos a limpeza do Docker e a tela Rede (pausar o AdGuard e liberar domínios, H10, com o usuário `hub` do AdGuard, senha no `hub.env`). O Hub nunca recebe o socket do Docker (só o `docker-proxy`, H4) lê a memória de cada app pelo cgroup, só para leitura (H8), e lê o Proxmox com um token `PVEAuditor` (H3); o token fica em `/opt/apps/hub.env`, nunca no Git.
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
