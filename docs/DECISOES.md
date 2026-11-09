# Decisões do Hub

Estado atual, decisões e próximos passos do Hub. As decisões que valem para todos os apps ficam em
[`ARQUITETURA.md`](ARQUITETURA.md) e [`ROADMAP.md`](ROADMAP.md).

## Estado (09/11/2026)

O Hub começou pelo **painel do servidor**, no ar em `https://192.168.1.20:8449`. Nesta primeira
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
  - os contêineres Docker.
- **Limpeza:** apaga o cache de build e as imagens soltas do Docker, agora ou por agenda (a cada
  1, 2, 3, 7, 14 ou 30 dias, numa hora escolhida), e guarda o histórico.
- **Ajustes:** Sugestões de melhoria, com a API comum (`/api/sugestoes`).

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

## Próximos passos

- [ ] Criar o token do Proxmox (H3) e conferir a tela Servidor com os números reais.
- [ ] Conferir se o disco da VM tem `discard` ligado e se o `fstrim.timer` roda: sem isso, o que a
  VM apaga não volta para o `local-lvm`.
- [ ] Manutenção por app: cada app com uma API para ver e limpar o que é temporário (histórico do
  Claude, pastas de trabalho, mídia sem uso). Ver o pedido de 06/11 no ROADMAP.
- [ ] Estado da chave do Claude por app e as pesquisas em andamento.
- [ ] Alertas fora do painel (por exemplo, disco acima de 85%), se fizer falta.
