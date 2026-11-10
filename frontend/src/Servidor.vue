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
    <h1>Servidor</h1>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <p v-else-if="!dados" class="suave">Lendo…</p>

    <template v-if="dados">
      <p class="rotulo">Proxmox</p>
      <template v-if="dados.proxmox">
        <p class="suave pequeno">
          Node {{ no.nome }}, {{ no.versao.split('/').slice(0, 2).join(' ') }}, ligado há {{ duracao(no.ligado_ha) }}. {{ no.modelo }}.
        </p>
        <div class="numeros">
          <div>
            <p class="rotulo-numero">Processador</p>
            <p class="numero">{{ Math.round(no.cpu * 100) }}%</p>
            <p class="suave pequeno">de {{ no.cpus }} threads</p>
          </div>
          <div>
            <p class="rotulo-numero">Memória</p>
            <p class="numero">{{ porcento(no.memoria.usado, no.memoria.total) }}%</p>
            <p class="suave pequeno">{{ bytes(no.memoria.usado) }} de {{ bytes(no.memoria.total) }}</p>
          </div>
        </div>
        <div class="lista">
          <Medidor nome="Memória" :usado="no.memoria.usado" :total="no.memoria.total" />
          <Medidor v-if="no.swap.total" nome="Swap" :usado="no.swap.usado" :total="no.swap.total" />
          <Medidor nome="Disco do sistema (raiz)" :usado="no.raiz.usado" :total="no.raiz.total" />
        </div>

        <p class="rotulo">Máquinas no Proxmox</p>
        <ul class="lista">
          <li v-for="m in dados.proxmox.maquinas" :key="m.id" class="linha">
            <span class="esquerda">
              <span class="titulo">{{ m.id }} {{ m.nome }}</span>
              <span class="chip">{{ NOMES_DO_TIPO[m.tipo] || m.tipo }}</span>
              <span class="chip" :class="m.estado === 'running' ? 'ok' : ''">{{ NOMES_DO_ESTADO[m.estado] || m.estado }}</span>
            </span>
            <span class="valor">
              {{ Math.round(m.cpu * 100) }}% de {{ m.cpus }} CPUs, {{ bytes(m.memoria) }} de {{ bytes(m.memoria_max) }}, disco de {{ bytes(m.disco_max) }}
            </span>
          </li>
          <li v-if="!dados.proxmox.maquinas.length" class="suave linha">Nenhuma máquina.</li>
        </ul>
        <div v-for="a in dados.proxmox.avisos" :key="a" class="aviso">Não veio: {{ a }}</div>
      </template>
      <div v-else-if="dados.proxmox_erro" class="aviso erro">{{ dados.proxmox_erro }}</div>
      <div v-else class="instrucoes">
        <p>O Proxmox ainda não está ligado ao painel. Para ligar, crie um usuário só de leitura no shell do Proxmox ({{ dados.proxmox_endereco }}, em pve → Shell):</p>
        <pre><code>pveum user add hub@pve --comment "Painel do Hub, só leitura"
pveum acl modify / --users hub@pve --roles PVEAuditor
pveum user token add hub@pve painel --privsep 0</code></pre>
        <p>Depois ponha o token em <code>/opt/apps/hub.env</code> na VM, na linha <code>PROXMOX_TOKEN=hub@pve!painel=&lt;value&gt;</code>, e publique o Hub de novo.</p>
      </div>

      <p class="rotulo">VM apps por dentro</p>
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
      <div v-else class="aviso erro">{{ dados.vm_erro }}</div>

      <p class="rotulo">Memória por app</p>
      <template v-if="memoria">
        <p v-if="dados.vm" class="suave pequeno">
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
      <div v-else-if="dados.memoria_erro" class="aviso">{{ dados.memoria_erro }}</div>
      <div v-else-if="dados.docker_erro" class="aviso erro">{{ dados.docker_erro }}</div>

      <p class="rotulo">Contêineres</p>
      <ul v-if="dados.containers.length" class="lista">
        <li v-for="c in dados.containers" :key="c.nome" class="linha">
          <span class="esquerda">
            <span class="titulo">{{ c.nome }}</span>
            <span class="suave pequeno">{{ c.imagem.length > 40 ? c.imagem.slice(0, 40) + '…' : c.imagem }}</span>
          </span>
          <span class="chip" :class="c.estado === 'running' ? 'ok' : 'erro'">{{ c.status }}</span>
        </li>
      </ul>
      <div v-else-if="dados.docker_erro" class="aviso erro">{{ dados.docker_erro }}</div>
    </template>
  </div>
</template>

<style scoped>
.rotulo:first-of-type {
  margin-top: 8px;
}

.pequeno {
  font-size: 0.82rem;
}

.numeros {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 48px;
  margin: 18px 0 14px;
}

.numeros p {
  margin: 0;
}

.rotulo-numero {
  color: var(--texto-suave);
  font-size: 0.82rem;
  font-weight: 600;
}

.numero {
  font-size: 1.75rem;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.linha {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px 12px;
  padding: 12px 0;
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
  font-size: 0.88rem;
  font-variant-numeric: tabular-nums;
  white-space: pre-wrap;
}

.aviso {
  margin-top: 10px;
}

.consumo {
  padding: 12px 0;
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
  height: 8px;
  border-radius: 999px;
  background: var(--azul-fundo);
  overflow: hidden;
}

.preenchido {
  height: 100%;
  min-width: 4px;
  border-radius: 999px;
  background: var(--azul);
}

.trilha.resto .preenchido {
  background: var(--texto-suave);
}

.conteineres {
  margin: 6px 0 0;
}

.instrucoes p {
  line-height: 1.5;
}

pre {
  overflow-x: auto;
  padding: 12px 14px;
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.8rem;
  line-height: 1.6;
}

code {
  font-family: ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 0.85em;
}
</style>
