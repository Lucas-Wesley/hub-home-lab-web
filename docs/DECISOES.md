# Decisões do Hub

Estado atual, decisões e próximos passos do Hub. As decisões que valem para todos os apps ficam em
[`ARQUITETURA.md`](ARQUITETURA.md) e [`ROADMAP.md`](ROADMAP.md).

## Estado (11/11/2026)

O Hub começou pelo **painel do servidor**, no ar em `https://hub.internal` (e no endereço antigo, `https://192.168.1.20:8449`). Nesta primeira
versão ele só lê, com uma exceção: a limpeza do Docker.

- **Armazenamento:**
  - os armazenamentos do Proxmox (`local`, `local-lvm`) e a saúde do NVMe;
  - o disco da VM `apps`, em partes: dados dos apps, cache de build, imagens do Docker, sistema e
    livre;
  - quanto cada app guarda, com o que há dentro (livros, histórico do Claude, banco…);
  - o uso do Docker.
- **Servidor:**
  - o processador e a memória do Proxmox, e as VMs e contêineres LXC dele;
  - a memória e a carga da VM `apps`;
  - a memória e o processador de cada app (somando os contêineres dele) e o que sobra para o
    sistema;
  - os contêineres Docker.
- **Limpeza:** apaga o cache de build e as imagens soltas do Docker, agora ou por agenda (a cada
  1, 2, 3, 7, 14 ou 30 dias, numa hora escolhida), e guarda o histórico.
- **Aplicações:** a tela de abertura, com o link de cada app pelo nome `.internal` e como instalá-lo no celular (H12).
- **Ajustes:** Sugestões de melhoria, com a API comum (`/api/sugestoes`).
- **Vigia:** avisa no celular, pelo ntfy, disco cheio, contêiner parado e backup atrasado (H9).
- **Rede:** o AdGuard Home (DNS da casa): pausar e religar o bloqueio, liberar domínios (H10).

## A infraestrutura que o painel mostra

```
Lenovo M920q (NVMe de 256 GB)
└── Proxmox VE 9.2, node pve (192.168.1.10)
    ├── local      (diretório: ISOs, backups, templates)
    ├── local-lvm  (LVM-thin: discos das VMs)
    └── VM 100 "apps", Debian 13 (192.168.1.20), disco de 100 GiB em local-lvm
        └── Docker Compose
            ├── caddy (proxy, HTTPS)
            ├── biblioteca, acervo, flashcards, streaming, prowlarr
            └── hub + docker-proxy
```

## Decisões

### H1. O Hub começa pelo painel do servidor (09/11/2026)

O ROADMAP diz que o Hub é o último, porque ele junta o conteúdo dos apps pelas APIs deles. O
painel do servidor não depende dessas APIs: lê o Proxmox, o disco e o Docker. Por isso pôde vir
antes, sem quebrar a regra. A parte de conteúdo continua para o fim.

### H2. As pastas dos apps são medidas de fora, só para leitura

O Hub monta `/srv/data` só para leitura e mede o tamanho de cada pasta, como o `du`. Isso não lê o
banco de nenhum app: só o tamanho dos arquivos. Assim nenhum app precisou mudar.

- **Banco de dados:** o `.db`, o `-wal` e o `-shm` do SQLite aparecem juntos, como um item só.
- **Nomes:** as pastas conhecidas têm nome em português (`home` vira "Claude: histórico e
  configuração"). Uma pasta nova aparece com o nome do disco até entrar na lista, em
  `backend/armazenamento.go`.
- **Frequência:** a medida vale por um minuto, porque percorrer as pastas custa disco. O botão
  "Medir de novo" força uma medida nova.

Apagar coisas de dentro de um app (histórico do Claude, mídia sem uso) fica para quando cada app
tiver uma API de manutenção, porque só o app sabe o que está em uso.

### H3. Proxmox: token só de leitura e certificado fixado

O Hub lê a API do Proxmox com o token de um usuário próprio (`hub@pve`), com o papel
`PVEAuditor`, que só lê. Para criar o usuário e o token, no shell do Proxmox:

```
pveum user add hub@pve --comment "Painel do Hub, só leitura"
pveum acl modify / --users hub@pve --roles PVEAuditor
pveum user token add hub@pve painel --privsep 0
```

O token vai em `/opt/apps/hub.env`, na VM, fora do Git:

```
PROXMOX_TOKEN=hub@pve!painel=<value que o último comando mostrou>
```

O certificado do Proxmox é autoassinado. Em vez de desligar a conferência, o Hub só aceita o
certificado com a impressão digital SHA-256 do README do homelab (`PROXMOX_FINGERPRINT`, em
`compose.prod.yaml`). Quando o certificado for renovado (vence em 28/10/2028), o painel avisa
"a impressão digital mudou" até a variável ser atualizada.

O Proxmox não sabe quanto uma VM usa por dentro do disco dela. Por isso o disco da VM `apps` é
medido de dentro, pelo próprio Hub. O `local-lvm` mostra o que as VMs já gravaram, e não o
tamanho reservado a elas.

### H4. Docker por um proxy que só deixa ler e limpar

O socket do Docker dá poder de root na VM, então o Hub não o recebe. Quem recebe é o
`docker-socket-proxy` (`tecnativa/docker-socket-proxy:v0.5.0`), numa rede interna só dele e do
Hub. Ele libera só três grupos de endpoints:

- `/system`: o uso do disco;
- `/images`: limpar as imagens soltas;
- `/build`: limpar o cache de build.

Criar, listar, parar ou entrar em contêineres dá 403, o que foi testado com o Docker de verdade.

A API do Docker vai com a versão fixa (`/v1.47`), para o formato das respostas não mudar quando o
Docker atualizar.

### H5. O que a limpeza apaga

O que mais ocupa o disco da VM é o cache de build. Cada deploy constrói o app na VM, e o Docker
guarda as camadas. Em 09/11/2026 eram 14,6 GiB de 19 GiB usados, contra uns 50 MiB de dados dos
apps.

- **Cache de build:** todo o que não está em uso (`POST /build/prune?all=true`). Depois disso, o
  próximo deploy de cada app demora um pouco mais, porque baixa as dependências de novo.
- **Imagens soltas:** só as sem nome (`POST /images/prune`). As imagens com nome sem contêiner
  ficam: são a versão anterior de cada app, que o deploy guarda para voltar se a nova falhar.
- **Nunca:** os dados dos apps, os volumes e os contêineres.

### H6. Agenda da limpeza

- **Padrão:** a agenda nasce ligada, toda semana, às 4h de Brasília.
- **Ao salvar:** a próxima limpeza fica para a próxima vez em que o relógio marcar a hora
  escolhida. Depois de cada limpeza, a seguinte fica N dias à frente, na mesma hora.
- **Servidor desligado:** se perdeu várias limpezas, o Hub limpa uma vez ao voltar e marca a
  próxima no futuro, sem repetir as perdidas.
- **Uma por vez:** a agenda e o botão nunca limpam ao mesmo tempo. Uma segunda limpeza recebe
  "Já há uma limpeza em andamento".
- **Histórico:** guarda as últimas 100 limpezas, com o que cada uma liberou e o erro, se houve.

### H7. Cor do Hub: ciano petróleo

`#0e7490` no claro e `#67cbe0` no escuro. Ela não se confunde com o azul da Biblioteca, o verde do
Acervo, o violeta do Flashcards, o âmbar do Streaming nem com o vermelho de erro. As barras do
disco usam as quatro primeiras cores da paleta categórica do skill de visualização, validadas nos
dois temas. O texto fica sempre nas cores de texto, e a legenda tem o valor de cada parte.

### H8. Memória de cada app pelo cgroup, não pelo proxy do Docker (10/11/2026)

O "docker stats" sai de `/containers/{id}/stats`, mas liberar `/containers` no proxy também
liberaria `/containers/{id}/json`, que mostra as variáveis de ambiente de cada app (as chaves).
Por isso o Hub lê o cgroup v2 da VM, montado só para leitura em `/host/cgroup` (`CGROUP_DIR`).

- **Memória:** `memory.current` menos o `inactive_file` do `memory.stat` de cada
  `system.slice/docker-<id>.scope`, a mesma conta do "docker stats".
- **Processador:** o `usage_usec` do `cpu.stat`, lido duas vezes com 1 s de intervalo (as
  leituras do Docker e do Proxmox ficam no meio, para a tela não esperar à toa). É a fração do
  processador da VM inteira.
- **Por app:** os contêineres se juntam pelo projeto do Compose (o rótulo
  `com.docker.compose.project`, que vem no `/system/df`): o Streaming soma o app e o procurador.
- **O resto:** a tela mostra a diferença entre o "em uso" da VM e a soma dos apps como "Sistema e
  o resto" (o Linux, o Docker, o que roda fora dos contêineres).

### H9. Vigia do servidor com aviso no celular (10/11/2026)

O Hub confere o servidor a cada 10 minutos e avisa pelo **ntfy** do homelab (`NTFY_URL`, tópico
`servidor`; sem a variável, não confere):

- disco da VM e armazenamentos do Proxmox acima de **85%**;
- NVMe com o SMART diferente de `PASSED` ou com 10% ou menos de vida útil;
- contêiner parado ou com a conferência de saúde falhando (`unhealthy`), pelo `/system/df` que o
  proxy já libera (H4, H8);
- **backup atrasado**: o arquivo `/var/lib/homelab-backup/ultimo`, montado só para leitura
  (`BACKUP_STATUS`), sem rodada há mais de 26 h. A falha do backup quem avisa é o próprio script.

Um problema só avisa depois de aparecer em **duas conferências seguidas** (um deploy recria os
contêineres e não deve avisar) e **uma vez só**; avisa de novo se sumir e voltar. A lembrança fica na
memória: se o Hub reiniciar com um problema em aberto, ele avisa de novo. Tocar abre o Hub
(`HUB_LINK`).

### H10. Rede: controlar o AdGuard Home pelo Hub (10/11/2026)

Pedido do Lucas antes de ligar o AdGuard para a casa toda: ter no Hub o botão de voltar se algo
quebrar. A tela **Rede** mostra o estado do bloqueio, as consultas e os bloqueios das últimas 24 h,
os domínios bloqueados agora (um por domínio, com o aparelho) e os liberados, e tem três controles:

- **Pausar o bloqueio** por 10 min, 1 h ou até religar, e **religar**. Pausado, o AdGuard continua
  respondendo o DNS, só sem bloquear: tudo volta a abrir na hora, na casa toda. É o "voltar" de
  verdade, sem mexer no roteador (que não tem API).
- **Liberar** um domínio (digitado, colado como link ou tocado na lista de bloqueados): grava a regra
  `@@||dominio^` nas regras personalizadas.
- **Bloquear de novo** um liberado: tira só essa regra. As outras regras ficam para o painel do AdGuard.

É a segunda exceção ao "só lê" (a primeira é a limpeza, H5). O Hub entra com um **usuário próprio**
no AdGuard (`hub`), não com o do Lucas: a senha fica no `hub.env` e pode ser trocada sem mexer na
dele. A API escuta só na ponte do Docker (`host.docker.internal:3001`). A vigia (H9) avisa no ntfy se
o bloqueio ficar pausado **sem prazo**. Nos testes, um AdGuard falso (`rede_test.go`).

### H11. Claude: as duas contas num lugar só, com o uso e a troca à mão (10/11/2026)

O Lucas tem duas contas Pro: a **Pessoal**, principal do homelab, e a da **Empresa**, para uso
eventual. Antes, cada app tinha o próprio `CLAUDE_CODE_OAUTH_TOKEN` no `.env` (e o Streaming estava
numa conta diferente dos outros sem ninguém lembrar). Agora a configuração é uma só:

- O Hub guarda os dois tokens em `/srv/data/claude/contas.json` (só o Hub lê, `0600`) e escreve o da
  conta ativa em `/srv/data/claude/ativa/token`. Os três apps montam só a pasta `ativa`, só para
  leitura, e leem o arquivo **a cada chamada** (`CLAUDE_TOKEN_ARQUIVO`): a troca vale na próxima
  tarefa, sem reiniciar. Fora do backup de propósito: perdido, é gerar outro token.
- A tela **Claude** mostra, por conta, o uso das janelas de **5 horas** e da **semana** e quando cada
  uma zera, e tem **Usar esta conta** e **Trocar token** (colar o do `claude setup-token`; o Hub
  confere com o Claude antes de guardar). O token nunca sai pela API.
- O uso vem do próprio Claude Code: uma chamada mínima ao Haiku, sem ferramentas nem instruções,
  devolve o `rate_limit_event` com as duas janelas. O token do `setup-token` não tem permissão para
  a API de uso da conta (`user:profile`). Por isso a imagem do Hub passou a ter o Claude Code.
- **A troca é sempre do Lucas, nunca automática** (pedido dele; revezar contas sozinho para fugir do
  limite é zona cinzenta nos termos). A vigia (H9) só avisa: a conta ativa em 90% de uma janela,
  esgotada ou com o login recusado, e um token a 30 dias de vencer (um ano depois de colado).
- Sem custo na tela: a assinatura não cobra por uso.

É a terceira exceção ao "só lê". Nos testes, um Claude falso no lugar da CLI (`claude_test.go`).

### H12. Aplicações: os apps pelo nome, numa tela só (11/11/2026)

Com o item 3 do `SUGESTOES.md` (nomes em vez de IP), cada app ganhou um nome: `biblioteca.internal`,
`acervo.internal`, `flashcards.internal`, `streaming.internal`, `hub.internal`, `adguard.internal`,
`ntfy.internal` e `prowlarr.internal`.

- **DNS:** uma regra só no AdGuard (*DNS rewrite*), `*.internal → 192.168.1.20`, criada com o usuário
  `hub`. `.internal` é o domínio reservado pela ICANN para redes privadas; `.local` é do mDNS e `.casa`
  existe na internet.
- **Caddy:** cada site responde pelo nome, na porta 443, e pelo IP:porta de antes. Os PWAs já
  instalados, os links guardados nos apps e o app do ntfy no celular seguem no endereço antigo, que
  não tem data para sair. A porta das telas do Streaming (`http://192.168.1.20:8447`) fica sem nome,
  porque nem todo aparelho DLNA resolve nomes.
- **Os links novos que os apps geram** (avisos, cartões, resumos) usam os nomes: `*_LINK` e
  `ADGUARD_PAINEL` mudaram de padrão.
- **A tela Aplicações** passou a ser a tela de abertura do Hub: um cartão por app, com o ícone dele, e
  os painéis do servidor. Tem o passo a passo de instalar cada app no celular no endereço novo e o
  aviso de que um app instalado pelo IP é outro app para o celular (os livros offline da Biblioteca
  precisam ser baixados de novo). No celular, "Armazenamento" vira "Disco" na barra de abas, para caber
  a sétima.
- **Fora de casa:** os nomes só funcionam pelo Tailscale com o DNS dividido (`internal` → `192.168.1.20`),
  configurado no painel do Tailscale.

## Próximos passos

- [ ] Criar o token do Proxmox (H3) e conferir a tela Servidor com os números reais.
- [ ] Conferir se o disco da VM tem `discard` ligado e se o `fstrim.timer` roda: sem isso, o que a
  VM apaga não volta para o `local-lvm`.
- [ ] Manutenção por app: cada app com uma API para ver e limpar o que é temporário (histórico do
  Claude, pastas de trabalho, mídia sem uso). Ver o pedido de 06/11 no ROADMAP.
- [ ] As pesquisas em andamento de cada app (a chave do Claude já está na tela Claude, H11).
