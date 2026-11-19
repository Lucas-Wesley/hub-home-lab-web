<script setup>
import { computed, onMounted, ref } from 'vue'
import Medidor from './Medidor.vue'
import { api } from './api.js'
import { bytes, duracao, porcento } from './formato.js'

// A máquina inteira: o Proxmox (processador, memória, VMs) e a VM apps por dentro (contêineres).
const dados = ref(null)
const erro = ref('')

onMounted(async () => {
  try {
    dados.value = await api('/api/servidor', { tempo: 30000 })
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o painel só abre na rede de casa.' : e.message
  }
})

const no = computed(() => dados.value?.proxmox?.no)

// Memória por app, do maior para o menor, e o que sobra do "em uso" da VM: o sistema, o Docker
// e o que roda fora dos contêineres. As barras são relativas ao maior, para comparar os apps.
const memoria = computed(() => {
  const apps = dados.value?.memoria_por_app || []
  if (!apps.length) return null
  const soma = apps.reduce((t, m) => t + m.memoria, 0)
  const resto = dados.value.vm ? Math.max(0, dados.value.vm.memoria.usado - soma) : 0
  const maior = Math.max(resto, ...apps.map((m) => m.memoria))
  return { apps, soma, resto, maior }
})

const processador = (fracao) => `${(fracao * 100).toLocaleString('pt-BR', { maximumFractionDigits: fracao < 0.1 ? 1 : 0 })}%`
const NOMES_DO_TIPO = { qemu: 'VM', lxc: 'Contêiner LXC' }
const NOMES_DO_ESTADO = { running: 'Ligada', stopped: 'Desligada', paused: 'Pausada' }
</script>

<template>
  <div class="pagina">
    <div class="topo-pagina"><h1>Servidor</h1></div>
    <div v-if="erro" class="aviso">{{ erro }}</div>
    <p v-else-if="!dados" class="suave">Lendo…</p>

    <template v-if="dados">
      <template v-if="dados.proxmox">
        <p class="explicacao pequeno">
          Proxmox: node {{ no.nome }}, {{ no.versao.split('/').slice(0, 2).join(' ') }}, ligado há {{ duracao(no.ligado_ha) }}. {{ no.modelo }}.
        </p>
        <div class="numeros-quadro">
          <div class="numero-quadro">
            <p class="rotulo-secao">Processador</p>
            <span class="valor-grande">{{ Math.round(no.cpu * 100) }}%</span>
            <span class="pequeno">de {{ no.cpus }} threads</span>
          </div>
          <div class="numero-quadro">
            <p class="rotulo-secao">Memória</p>
            <span class="valor-grande">{{ porcento(no.memoria.usado, no.memoria.total) }}%</span>
            <span class="pequeno">{{ bytes(no.memoria.usado) }} de {{ bytes(no.memoria.total) }}</span>
          </div>
          <div class="numero-quadro">
            <p class="rotulo-secao">Disco do sistema</p>
            <span class="valor-grande">{{ porcento(no.raiz.usado, no.raiz.total) }}%</span>
            <span class="pequeno">{{ bytes(no.raiz.usado) }} de {{ bytes(no.raiz.total) }}</span>
          </div>
          <div class="numero-quadro">
            <p class="rotulo-secao">Máquinas ligadas</p>
            <span class="valor-grande">{{ dados.proxmox.maquinas.filter((m) => m.estado === 'running').length }}</span>
            <span class="pequeno">de {{ dados.proxmox.maquinas.length }}</span>
          </div>
        </div>
      </template>

      <div class="quadros">
        <section v-if="dados.proxmox" class="quadro">
          <div class="quadro-topo"><h2 class="rotulo-secao">Proxmox</h2></div>
          <div class="lista">
            <Medidor nome="Memória" :usado="no.memoria.usado" :total="no.memoria.total" />
            <Medidor v-if="no.swap.total" nome="Swap" :usado="no.swap.usado" :total="no.swap.total" />
            <Medidor nome="Disco do sistema (raiz)" :usado="no.raiz.usado" :total="no.raiz.total" />
          </div>
          <div v-for="a in dados.proxmox.avisos" :key="a" class="aviso neutro">Não veio: {{ a }}</div>
        </section>
        <div v-else-if="dados.proxmox_erro" class="aviso">{{ dados.proxmox_erro }}</div>
        <section v-else class="quadro instrucoes">
          <div class="quadro-topo"><h2 class="rotulo-secao">Proxmox</h2></div>
          <p>O Proxmox ainda não está ligado ao painel. Para ligar, crie um usuário só de leitura no shell do Proxmox ({{ dados.proxmox_endereco }}, em pve → Shell):</p>
          <pre><code>pveum user add hub@pve --comment "Painel do Hub, só leitura"
pveum acl modify / --users hub@pve --roles PVEAuditor
pveum user token add hub@pve painel --privsep 0</code></pre>
          <p>Depois ponha o token em <code>/opt/apps/hub.env</code> na VM, na linha <code>PROXMOX_TOKEN=hub@pve!painel=&lt;value&gt;</code>, e publique o Hub de novo.</p>
        </section>

        <section class="quadro">
          <div class="quadro-topo"><h2 class="rotulo-secao">VM apps por dentro</h2></div>
          <div v-if="dados.vm" class="lista">
            <Medidor
              nome="Memória"
              :usado="dados.vm.memoria.usado"
              :total="dados.vm.memoria.total"
              :detalhe="`Sem contar o cache de arquivos, que o sistema devolve quando precisa. Ligada há ${duracao(dados.vm.ligada_ha)}.`"
            />
            <div class="linha">
              <span>Carga média (1, 5 e 15 min)</span>
              <span class="valor">{{ dados.vm.carga.map((c) => c.toLocaleString('pt-BR', { maximumFractionDigits: 2 })).join('   ') }} em {{ dados.vm.cpus }} CPUs</span>
            </div>
          </div>
          <div v-else class="aviso">{{ dados.vm_erro }}</div>
        </section>
      </div>

      <section v-if="dados.proxmox" class="quadro bloco">
        <div class="quadro-topo"><h2 class="rotulo-secao">Máquinas no Proxmox</h2></div>
        <ul class="lista">
          <li v-for="m in dados.proxmox.maquinas" :key="m.id" class="linha">
            <span class="esquerda">
              <span class="titulo">{{ m.id }} {{ m.nome }}</span>
              <span class="etiqueta">{{ NOMES_DO_TIPO[m.tipo] || m.tipo }}</span>
              <span class="etiqueta" :class="{ verde: m.estado === 'running' }">{{ NOMES_DO_ESTADO[m.estado] || m.estado }}</span>
            </span>
            <span class="valor">
              {{ Math.round(m.cpu * 100) }}% de {{ m.cpus }} CPUs, {{ bytes(m.memoria) }} de {{ bytes(m.memoria_max) }}, disco de {{ bytes(m.disco_max) }}
            </span>
          </li>
          <li v-if="!dados.proxmox.maquinas.length" class="suave linha">Nenhuma máquina.</li>
        </ul>
      </section>

      <div class="quadros">
        <section class="quadro">
          <div class="quadro-topo"><h2 class="rotulo-secao">Memória por app</h2></div>
          <template v-if="memoria">
            <p v-if="dados.vm" class="suave pequeno resumo">
              Os apps usam {{ bytes(memoria.soma) }} dos {{ bytes(dados.vm.memoria.usado) }} em uso na VM. O processador é a média do último segundo.
            </p>
            <ul class="lista">
              <li v-for="m in memoria.apps" :key="m.nome" class="consumo">
                <div class="topo">
                  <span class="esquerda">
                    <span class="titulo">{{ m.titulo }}</span>
                    <span v-if="m.containers.length > 1" class="suave pequeno">{{ m.containers.length }} contêineres</span>
                  </span>
                  <span class="valor">{{ bytes(m.memoria) }} · {{ processador(m.cpu) }} do processador</span>
                </div>
                <div class="trilha" :title="dados.vm ? `${porcento(m.memoria, dados.vm.memoria.total)}% da memória da VM` : ''">
                  <div class="preenchido" :style="{ width: `${(m.memoria / memoria.maior) * 100}%` }" />
                </div>
                <p class="suave pequeno conteineres">{{ m.containers.join(', ') }}</p>
              </li>
              <li v-if="memoria.resto" class="consumo">
                <div class="topo">
                  <span class="titulo">Sistema e o resto</span>
                  <span class="valor">{{ bytes(memoria.resto) }}</span>
                </div>
                <div class="trilha resto"><div class="preenchido" :style="{ width: `${(memoria.resto / memoria.maior) * 100}%` }" /></div>
                <p class="suave pequeno conteineres">O Linux, o Docker e o que roda fora dos contêineres.</p>
              </li>
            </ul>
          </template>
          <div v-else-if="dados.memoria_erro" class="aviso neutro">{{ dados.memoria_erro }}</div>
          <div v-else-if="dados.docker_erro" class="aviso">{{ dados.docker_erro }}</div>
        </section>

        <section class="quadro">
          <div class="quadro-topo">
            <h2 class="rotulo-secao">Contêineres</h2>
            <span class="pequeno">{{ dados.containers.length }}</span>
          </div>
          <ul v-if="dados.containers.length" class="lista">
            <li v-for="c in dados.containers" :key="c.nome" class="linha">
              <span class="esquerda">
                <span class="titulo">{{ c.nome }}</span>
                <span class="suave pequeno">{{ c.imagem.length > 40 ? c.imagem.slice(0, 40) + '…' : c.imagem }}</span>
              </span>
              <span class="etiqueta" :class="c.estado === 'running' ? 'verde' : 'vermelho'">{{ c.status }}</span>
            </li>
          </ul>
          <div v-else-if="dados.docker_erro" class="aviso">{{ dados.docker_erro }}</div>
        </section>
      </div>
    </template>
  </div>
</template>

<style scoped>
.bloco {
  margin-bottom: var(--e4);
}

.resumo {
  margin: 0 0 var(--e2);
}

.linha {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px 12px;
  padding: 10px 0;
}

.esquerda {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
}

.titulo {
  font-weight: 500;
}

.valor {
  color: var(--texto-suave);
  font-size: 13.5px;
  font-variant-numeric: tabular-nums;
  white-space: pre-wrap;
}

.aviso {
  margin-top: 10px;
}

.consumo {
  padding: 10px 0;
}

.consumo .topo {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 12px;
  margin-bottom: 8px;
}

.trilha {
  height: 6px;
  border-radius: 999px;
  background: var(--superficie-2);
  overflow: hidden;
}

.preenchido {
  height: 100%;
  min-width: 4px;
  border-radius: 999px;
  background: var(--texto-suave);
}

.trilha.resto .preenchido {
  background: var(--texto-fraco);
}

.conteineres {
  margin: 6px 0 0;
}

.instrucoes p {
  line-height: 1.5;
}

pre {
  overflow-x: auto;
  padding: var(--e3) var(--e4);
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 13px;
  line-height: 1.6;
}
</style>
