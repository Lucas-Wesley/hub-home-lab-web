# Contrato Comum do Ecossistema

> **Status (03/11/2026): ainda NÃO definido — de propósito.**
> Por ora só registramos **uma decisão**: vai existir um contrato comum e **toda aplicação do
> ecossistema terá que segui-lo** para trocar conteúdo com as outras. O formato exato fica para
> quando os apps estiverem mais maduros, e provavelmente vai mudar no caminho.
>
> Tudo **abaixo desta linha** são **ideias iniciais, não decididas** — só um ponto de partida pra
> quando formos definir de verdade. Nada aqui está valendo ainda.

---

## 1. A unidade de troca: o **Fragmento de Conteúdo**

Quase toda integração é "um pedaço de texto + de onde ele veio". Esse é o objeto que os apps trocam:

```json
{
  "texto": "o conteúdo textual em si",
  "formato": "markdown",              // "markdown" | "texto"
  "fonte": {
    "tipo": "livro",                  // "youtube" | "livro" | "artigo" | "web" | "manual"
    "titulo": "Título da fonte",
    "autor": "Autor, se houver",      // ou null
    "referencia": "URL, ISBN ou id interno da fonte",
    "localizador": "p.45"             // "12:34" (timestamp) | "p.45" | "cap.3" | null
  },
  "tags": ["tema:segunda-guerra-mundial"],
  "origem_app": "biblioteca",         // quem gerou: "biblioteca" | "acervo" | "flashcards" | "hub"
  "criado_em": "2026-11-03T17:00:00Z" // ISO 8601, UTC
}
```

Campos obrigatórios: `texto`, `fonte.tipo`, `origem_app`, `criado_em`. O resto é opcional.

---

## 2. Convenções de API

- **Transporte:** HTTP + JSON, UTF-8.
- **Versão na URL:** `/api/v1/...` (permite evoluir sem quebrar quem já usa).
- **IDs:** cada app gera os seus (UUID). **Nenhum app depende do id interno do outro** — a
  referência cruzada, quando precisar, vai no campo `fonte.referencia`.
- **Datas:** sempre ISO 8601 em UTC.
- **Erros:** formato consistente —
  ```json
  { "erro": { "codigo": "nao_encontrado", "mensagem": "descrição legível" } }
  ```
- **Autenticação:** usuário único + LAN → por ora um **token simples compartilhado**
  (`Authorization: Bearer <token>`). Centralizar/trocar por algo mais robusto fica para depois
  (item em aberto na arquitetura).

---

## 3. Endpoints de integração (ilustrativos)

Cada app define a própria API; estes são os pontos de contato entre eles:

| App | Endpoint | Recebe/Devolve |
|-----|----------|----------------|
| Acervo | `POST /api/v1/referencias` | recebe um **Fragmento** → cria uma referência (estado `cru`) |
| Flashcards | `POST /api/v1/cards` | recebe um **Fragmento** (ou `{frente, verso}`) → cria um card |
| Biblioteca Digital | `GET /api/v1/livros` | lista a biblioteca |
| Biblioteca Digital | `GET /api/v1/livros/{id}` | detalhes + anotações de um livro |

### Primeira integração feita (06/11/2026)

**Acervo → Flashcards**, com um endpoint próprio, **sem** o contrato comum (que segue sem definir):
`POST /api/textos` no Flashcards recebe `{titulo, baralho_id | baralho, contexto, texto, link, secoes}` e cria um
pedido à IA (cada cartão guarda o link do capítulo de onde saiu); para escolher o baralho, o Acervo lê `GET /api/baralhos`. O Acervo
acompanha e decide a proposta por `GET /api/geracoes/{id}` e `POST /api/geracoes/{id}/{aprovar|descartar|refazer}` (o `aprovar` aceita `baralho_id` opcional, para trocar o baralho na revisão; F31).
A chamada vai pela rede interna do Docker (`http://flashcards:8080`), sem passar pelo Caddy. Detalhes em
`acervo/docs/DECISOES.md` (D28) e `flashcards/docs/DECISOES.md` (F28). Quando o contrato existir, esse envio vira um
Fragmento.

---

## 4. Fluxos de exemplo

**Biblioteca Digital → Acervo (parágrafo vira referência):**
1. Na Biblioteca Digital, seleciono um parágrafo do livro.
2. A Biblioteca Digital monta um Fragmento: `texto`, `fonte.tipo="livro"`, `fonte.titulo`, `fonte.localizador="p.45"`, `origem_app="biblioteca"`.
3. `POST /api/v1/referencias` na API do **Acervo**.
4. O Acervo recebe e inicia o pipeline dele (cru → organizado → ...).

**Biblioteca Digital/Acervo → Flashcards (seleção vira card):**
1. Seleciono um trecho (num livro ou numa referência).
2. Monto um Fragmento.
3. `POST /api/v1/cards` na API do **Flashcards**.

---

## 5. Regras de ouro

- Um app **nunca** lê o banco de dados do outro — só a API.
- Quem recebe um Fragmento é **dono** do que faz com ele (o Acervo decide como organizar; o
  Flashcards decide como virar card).
- Mudou o contrato de forma incompatível? Sobe a versão (`/api/v2`) — não quebra quem usa a v1.
