<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import Medidor from './Medidor.vue'
import { api } from './api.js'
import { QUASE_CHEIO, bytes, dataHora, porcento } from './formato.js'

// O armazenamento de tudo, de fora para dentro: o disco do servidor (Proxmox), o disco da VM
// apps, o que cada app guarda e o que o Docker ocupa.
const dados = ref(null)
const servidor = ref(null)
const erro = ref('')
const medindo = ref(false)

async function carregar(atualizar = false) {
  medindo.value = true
  erro.value = ''
  try {
    const [a, s] = await Promise.all([api(`/api/armazenamento${atualizar ? '?atualizar=1' : ''}`, { tempo: 60000 }), api('/api/servidor', { tempo: 30000 })])
    dados.value = a
    servidor.value = s
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o painel só abre na rede de casa.' : e.message
  } finally {
    medindo.value = false
  }
}

const proxmox = computed(() => servidor.value?.proxmox)
const docker = computed(() => dados.value?.docker)
const disco = computed(() => dados.value?.disco)
const totalDosApps = computed(() => (dados.value?.apps || []).reduce((s, a) => s + a.bytes, 0))

// As partes do disco da VM, na ordem fixa das cores. "Sistema e outros" é o que sobra do usado.
const partes = computed(() => {
  if (!disco.value) return []
  const p = [{ nome: 'Dados dos apps', bytes: totalDosApps.value, cor: 1 }]
  if (docker.value) {
    p.push({ nome: 'Cache de build do Docker', bytes: docker.value.cache_de_build, cor: 2 })
    p.push({ nome: 'Imagens do Docker', bytes: docker.value.imagens, cor: 3 })
  }
  const conhecido = p.reduce((s, x) => s + x.bytes, 0)
  p.push({ nome: 'Sistema e outros', bytes: Math.max(0, disco.value.usado - conhecido), cor: 4 })
  return p
})

const cacheParaLimpar = computed(() => (docker.value ? docker.value.cache_liberavel + docker.value.imagens_soltas : 0))
const vmQuaseCheia = computed(() => disco.value && disco.value.usado / disco.value.total >= QUASE_CHEIO)

const aberto = ref({})

onMounted(() => carregar())
</script>

<template>
  <div class="pagina">
    <div class="cabeca">
      <h1>Armazenamento</h1>
      <button class="botao icone" :disabled="medindo" aria-label="Medir de novo" title="Medir de novo" @click="carregar(true)">
        <Icone nome="atualizar" />
      </button>
    </div>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <p v-else-if="!dados" class="suave">Medindo…</p>

    <template v-if="dados">
      <div v-if="vmQuaseCheia" class="aviso erro">O disco da VM apps passou de {{ Math.round(QUASE_CHEIO * 100) }}%.</div>
      <div v-if="cacheParaLimpar > 1024 ** 3" class="aviso">
        O Docker guarda {{ bytes(cacheParaLimpar) }} que podem ser liberados, quase tudo cache dos builds de deploy.
        <a href="#/limpeza">Ir para Limpeza</a>
      </div>

      <!-- Disco físico e armazenamentos do Proxmox -->
      <p class="rotulo">Servidor (Proxmox)</p>
      <template v-if="proxmox">
        <div class="lista">
          <Medidor
            v-for="s in proxmox.armazenamentos"
            :key="s.nome"
            :nome="s.nome"
            :usado="s.usado"
            :total="s.total"
            :detalhe="s.tipo === 'lvmthin' ? 'Discos das VMs. Mostra o que as VMs já gravaram, não o tamanho reservado.' : `Tipo ${s.tipo}: ${s.conteudo.replaceAll(',', ', ')}`"
          />
        </div>
        <ul v-if="proxmox.discos.length" class="lista discos">
          <li v-for="d in proxmox.discos" :key="d.caminho" class="linha">
            <span>
              <span class="titulo">{{ d.modelo || d.caminho }}</span>
              <span class="suave pequeno">{{ bytes(d.tamanho) }}, {{ d.tipo }}</span>
            </span>
            <span class="chips">
              <span class="chip" :class="d.saude === 'PASSED' || d.saude === 'OK' ? 'ok' : 'erro'">Saúde {{ d.saude }}</span>
              <span v-if="d.vida_util >= 0" class="chip" :class="{ erro: d.vida_util < 20 }">{{ d.vida_util }}% de vida útil</span>
            </span>
          </li>
        </ul>
      </template>
      <div v-else-if="servidor?.proxmox_erro" class="aviso erro">{{ servidor.proxmox_erro }}</div>
      <div v-else class="aviso">O Proxmox ainda não está ligado ao painel. <a href="#/servidor">Veja como ligar</a></div>

      <!-- Disco da VM, em partes -->
      <p class="rotulo">VM apps</p>
      <div v-if="disco" class="vm">
        <p class="numero">{{ bytes(disco.usado) }} <span class="de">de {{ bytes(disco.total) }}</span></p>
        <p class="suave pequeno">{{ bytes(disco.livre) }} livres, {{ porcento(disco.usado, disco.total) }}% em uso</p>
        <div class="pilha" role="img" :aria-label="`Disco da VM: ${partes.map((p) => `${p.nome} ${bytes(p.bytes)}`).join(', ')}, livre ${bytes(disco.livre)}`">
          <div v-for="p in partes.filter((x) => x.bytes > 0)" :key="p.nome" class="parte" :class="`cor-${p.cor}`" :style="{ flexGrow: p.bytes }" :title="`${p.nome}: ${bytes(p.bytes)}`" />
          <div class="parte livre" :style="{ flexGrow: disco.livre }" :title="`Livre: ${bytes(disco.livre)}`" />
        </div>
        <ul class="legenda">
          <li v-for="p in partes" :key="p.nome">
            <span class="amostra" :class="`cor-${p.cor}`" />
            <span class="nome">{{ p.nome }}</span>
            <span class="valor">{{ bytes(p.bytes) }}</span>
          </li>
          <li>
            <span class="amostra livre" />
            <span class="nome">Livre</span>
            <span class="valor">{{ bytes(disco.livre) }}</span>
          </li>
        </ul>
      </div>
      <div v-else class="aviso erro">{{ dados.disco_erro }}</div>

      <!-- Por app -->
      <p class="rotulo">Por app</p>
      <div v-if="dados.apps_erro" class="aviso erro">{{ dados.apps_erro }}</div>
      <ul v-else class="lista">
        <li v-for="a in dados.apps" :key="a.nome" class="por-app">
          <button class="linha linha-botao" :aria-expanded="!!aberto[a.nome]" @click="aberto[a.nome] = !aberto[a.nome]">
            <span class="titulo">{{ a.titulo }}</span>
            <span class="valor">{{ bytes(a.bytes) }}</span>
          </button>
          <ul v-if="aberto[a.nome]" class="itens">
            <li v-for="i in a.itens" :key="i.nome" class="linha">
              <span class="suave">{{ i.titulo }}</span>
              <span class="valor">{{ bytes(i.bytes) }}</span>
            </li>
            <li v-if="!a.itens?.length" class="suave pequeno">Pasta vazia.</li>
            <li v-if="a.sem_acesso" class="suave pequeno">{{ a.sem_acesso }} pasta(s) sem permissão de leitura ficaram de fora.</li>
          </ul>
        </li>
      </ul>

      <!-- Docker -->
      <p class="rotulo">Docker</p>
      <ul v-if="docker" class="lista">
        <li class="linha">
          <span>Imagens</span>
          <span class="valor">{{ bytes(docker.imagens) }}</span>
        </li>
        <li class="linha">
          <span>
            Cache de build
            <span v-if="docker.cache_liberavel" class="chip">{{ bytes(docker.cache_liberavel) }} para liberar</span>
          </span>
          <span class="valor">{{ bytes(docker.cache_de_build) }}</span>
        </li>
        <li class="linha">
          <span>Imagens soltas (sobras de builds)</span>
          <span class="valor">{{ bytes(docker.imagens_soltas) }}</span>
        </li>
      </ul>
      <div v-else class="aviso erro">{{ dados.docker_erro }}</div>

      <p class="medido suave">Medido em {{ dataHora(dados.medido_em) }}. As pastas são medidas de novo a cada minuto.</p>
    </template>
  </div>
</template>

<style scoped>
.cabeca {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.aviso {
  margin-bottom: 10px;
}

.rotulo:first-of-type {
  margin-top: 8px;
}

.pequeno {
  font-size: 0.82rem;
}

.linha {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0;
}

.linha > span:first-child {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
}

.linha-botao {
  width: 100%;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.linha-botao:hover .titulo {
  color: var(--marca);
}

.titulo {
  font-weight: 500;
  transition: color 0.15s ease;
}

.valor {
  color: var(--texto-suave);
  font-size: 0.9rem;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.discos {
  margin-top: 8px;
}

.itens {
  margin: 0 0 10px;
  padding: 0 0 0 14px;
  list-style: none;
  border-left: 2px solid var(--linha);
}

.itens .linha {
  padding: 6px 0;
  font-size: 0.9rem;
}

/* Cores das partes do disco: as quatro primeiras da paleta categórica, validadas nos dois temas. */
.vm {
  --serie-1: #2a78d6;
  --serie-2: #eb6834;
  --serie-3: #1baf7a;
  --serie-4: #eda100;
}

@media (prefers-color-scheme: dark) {
  .vm {
    --serie-1: #3987e5;
    --serie-2: #d95926;
    --serie-3: #199e70;
    --serie-4: #c98500;
  }
}

.numero {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.de {
  color: var(--texto-suave);
  font-size: 1rem;
  font-weight: 500;
  letter-spacing: 0;
}

.vm > .suave {
  margin: 2px 0 14px;
}

.pilha {
  display: flex;
  gap: 2px;
  height: 14px;
}

.parte {
  flex-basis: 0;
  min-width: 4px;
  border-radius: 4px;
}

.cor-1 {
  background: var(--serie-1);
}

.cor-2 {
  background: var(--serie-2);
}

.cor-3 {
  background: var(--serie-3);
}

.cor-4 {
  background: var(--serie-4);
}

.livre {
  background: var(--superficie-forte);
}

.legenda {
  display: grid;
  gap: 6px;
  margin: 14px 0 0;
  padding: 0;
  list-style: none;
  font-size: 0.9rem;
}

.legenda li {
  display: grid;
  grid-template-columns: 10px 1fr auto;
  align-items: center;
  gap: 10px;
}

.amostra {
  width: 10px;
  height: 10px;
  border-radius: 3px;
}

.medido {
  margin-top: 28px;
  font-size: 0.8rem;
}
</style>
