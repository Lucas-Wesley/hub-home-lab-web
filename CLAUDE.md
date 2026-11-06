# Hub — Ecossistema de apps pessoais do Lucas

Este repositório é o **ponto único de referência** do ecossistema. **Leia primeiro:**
- [`docs/ARQUITETURA.md`](docs/ARQUITETURA.md) — como os apps se encaixam e se comunicam.
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — ordem de desenvolvimento + princípios.
- [`docs/CONTRATO.md`](docs/CONTRATO.md) — contrato comum (ainda **NÃO definido** de propósito; só está registrado que vai existir e será obrigatório).

## O ecossistema em uma frase
Aplicações **independentes** (cada uma com banco + API próprios) que se falam **por API**; o **Hub é só um cliente** que junta tudo e é **construído por último**.

## Apps e repositórios
| App | Repo | Papel |
|-----|------|-------|
| Acervo | github.com/Lucas-Wesley/home-lab-acervo | Gerenciador de conhecimento (Temas + referências) — **2º, no ar** |
| Biblioteca Digital | github.com/Lucas-Wesley/biblioteca-digital | Leitor/biblioteca PDF+EPUB (PWA) com anotações — **1º, no ar** |
| Flashcards | github.com/Lucas-Wesley/flashcards | App estilo Anki — **3º, próximo** |
| Hub | github.com/Lucas-Wesley/hub-home-lab-web | Este repo — interface central (último) |

## Ambiente (homelab)
- **Proxmox VE 9.2** no Lenovo **M920q**, IP **192.168.1.10** (web: `https://192.168.1.10:8006`, SSH `root@192.168.1.10`). Tudo **LAN-only**, não exposto à internet.
- Cada app roda como **container** (Docker Compose) na VM `apps` (`192.168.1.20`), atrás do Caddy. Detalhes em `docs/ARQUITETURA.md` §4; estado atual em `docs/ROADMAP.md` ("Onde paramos").
- **Armazenamento:** somente o **disco interno do M920q**; não haverá SSD externo.

## Regras e convenções
- **Conteúdo do Acervo NÃO sobe pro GitHub** (versionado localmente no homelab). Código das apps pode.
- Construir **cada app sozinho primeiro**; só criar endpoint de integração quando **os dois lados existirem**; **não definir o contrato comum ainda**.
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
