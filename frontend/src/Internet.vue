<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from './api.js'
import { dataHora } from './formato.js'

// Internet (H15): onde a conexão parou, medido a cada 10 s pelo servidor (no cabo) e pelo Mac (no
// Wi-Fi). Responde a pergunta de quando cai: é da Claro ou é aqui de dentro, e o que reiniciar.
const dados = ref(null)
const erro = ref('')
const aberta = ref('') // início da queda com o registro aberto
let relogio

const CAUSAS = {
  claro: {
    nome: 'Claro ou modem',
    cor: 'vermelho',
    fazer: 'Seu roteador está funcionando. Tire só o modem da Claro da tomada por 30 s. Se não voltar em 5 min, o problema é da Claro: ligue para eles.',
  },
  roteador: {
    nome: 'Roteador',
    cor: 'vermelho',
    fazer: 'O servidor não alcança o roteador: ou o roteador travou, ou o cabo do servidor perdeu a ligação. Tire e recoloque o cabo de rede do servidor; se não voltar, reinicie só o roteador (o modem pode ficar).',
  },
  dns: { nome: 'DNS (AdGuard)', cor: 'ambar', fazer: 'A internet funciona, mas os nomes não resolvem. Pause o bloqueio aqui embaixo; se não resolver, a VM do servidor pode estar travada.' },
  wifi: { nome: 'Wi-Fi', cor: 'ambar', fazer: 'O Mac não alcançou o roteador pelo Wi-Fi. A internet da casa pode estar normal pelo cabo.' },
  'so-wifi': {
    nome: 'Só no Wi-Fi',
    cor: 'ambar',
    fazer: 'A casa tinha internet pelo cabo, mas quem estava no Wi-Fi não. O problema é no Wi-Fi do roteador, não na Claro: reinicie só o roteador.',
  },
  internet: { nome: 'Sem internet no Wi-Fi', cor: 'ambar', fazer: 'O Mac alcança o roteador, mas não a internet.' },
  servidor: {
    nome: 'Servidor fora da rede',
    cor: 'vermelho',
    fazer:
      'A Claro e o roteador estavam funcionando, mas o servidor perdeu a ligação com o roteador. Sem o AdGuard, que roda nele, a casa fica sem DNS e parece sem internet. Tire e recoloque o cabo de rede do servidor; se não voltar, reinicie o roteador.',
  },
  ipv6: { nome: 'IPv6 da Claro', cor: 'ambar', fazer: 'O IPv4 funciona; alguns apps podem ficar lentos ou falhar. Costuma voltar sozinho.' },
}
const causa = (c) => CAUSAS[c] || { nome: c, cor: 'ambar', fazer: '' }

async function carregar() {
  try {
    dados.value = await api('/api/internet')
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão com o Hub.' : e.message
  }
}

const idade = (iso) => (iso ? (Date.now() - new Date(iso).getTime()) / 1000 : Infinity)

// O estado de agora: o servidor manda; o Mac só acrescenta o que é do Wi-Fi.
const agora = computed(() => {
  const d = dados.value
  if (!d?.ligada) return null
  const s = d.servidor && idade(d.servidor.em) < 45 ? d.servidor : null
  const mac = d.mac && idade(d.mac.em) < 90 ? d.mac : null
  if (!s) return { estado: 'sem-dados' }
  if (s.causa === 'roteador' && mac && mac.roteador >= 0) return { estado: 'problema', causa: 'servidor' }
  if (s.causa) return { estado: 'problema', causa: s.causa }
  if (mac?.causa) return { estado: 'problema', causa: mac.causa === 'internet' ? 'so-wifi' : mac.causa }
  return { estado: 'ok' }
})

// Os trechos do caminho, na ordem, com o tempo de resposta.
function trechos(m, quem) {
  if (!m) return []
  const t = [{ nome: quem === 'mac' ? 'Wi-Fi → roteador' : 'Roteador', ms: m.roteador }]
  if (quem === 'servidor') t.push({ nome: 'Claro', ms: m.gateway })
  t.push({ nome: 'Internet', ms: m.internet })
  if (quem === 'mac') t.push({ nome: 'IPv6', ms: m.internet6 })
  t.push({ nome: 'DNS', ms: m.dns })
  return t.filter((x) => x.ms !== -2)
}

const tempoLigada = (s) => {
  if (!s) return ''
  const h = Math.floor(s / 3600)
  if (h >= 24) return `${Math.floor(h / 24)} d ${h % 24} h`
  return h ? `${h} h ${Math.floor((s % 3600) / 60)} min` : `${Math.floor(s / 60)} min`
}

function duracao(s) {
  if (s < 60) return `${s} s`
  if (s < 3600) return `${Math.round(s / 60)} min`
  return `${Math.floor(s / 3600)} h ${String(Math.floor((s % 3600) / 60)).padStart(2, '0')} min`
}

const hora = (iso) => new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', timeZone: 'America/Sao_Paulo' })

// Resumo de 7 dias e a faixa das últimas 24 h.
const resumo = computed(() => {
  const qs = dados.value?.quedas || []
  const semana = qs.filter((q) => idade(q.inicio) < 7 * 86400)
  const porCausa = {}
  for (const q of semana) porCausa[q.causa] = (porCausa[q.causa] || 0) + 1
  return {
    quedas: semana.length,
    minutos: Math.round(semana.reduce((s, q) => s + q.segundos, 0) / 60),
    porCausa: Object.entries(porCausa).sort((a, b) => b[1] - a[1]),
  }
})

const faixa = computed(() => {
  const fim = Date.now()
  const inicio = fim - 86400000
  return (dados.value?.quedas || [])
    .map((q) => {
      const a = new Date(q.inicio).getTime()
      const b = q.fim ? new Date(q.fim).getTime() : fim
      return { q, de: Math.max(a, inicio), ate: b }
    })
    .filter((x) => x.ate > inicio)
    .map(({ q, de, ate }) => ({
      q,
      left: ((de - inicio) / 86400000) * 100,
      width: Math.max(0.4, ((ate - de) / 86400000) * 100),
    }))
})

const marcas = computed(() => {
  const fim = new Date()
  return [24, 18, 12, 6, 0].map((h) => {
    const d = new Date(fim.getTime() - h * 3600000)
    return { left: 100 - (h / 24) * 100, texto: h === 0 ? 'agora' : hora(d.toISOString()) }
  })
})

onMounted(() => {
  carregar()
  relogio = setInterval(carregar, 10000)
})
onBeforeUnmount(() => clearInterval(relogio))
</script>

<template>
  <section class="internet">
    <h2 class="secao primeira">Internet</h2>
    <div v-if="erro" class="aviso">{{ erro }}</div>
    <div v-else-if="dados && !dados.ligada" class="aviso neutro">A vigia da internet está desligada neste Hub (INTERNET_VIGIA).</div>

    <template v-if="agora">
      <div class="estado-net" :class="agora.estado === 'ok' ? 'bom' : agora.estado === 'problema' ? causa(agora.causa).cor : 'neutro'">
        <p class="situacao">
          <span class="ponto" aria-hidden="true"></span>
          <template v-if="agora.estado === 'ok'">Internet funcionando</template>
          <template v-else-if="agora.estado === 'problema'">Sem internet: {{ causa(agora.causa).nome }}</template>
          <template v-else>Sem medidas recentes do servidor</template>
        </p>
        <p v-if="agora.estado === 'problema'" class="fazer">{{ causa(agora.causa).fazer }}</p>
        <p v-if="dados.wan && !dados.wan.erro" class="suave pequeno">
          Claro: {{ dados.wan.ipv4 || 'sem IP' }}<template v-if="dados.wan.desde"> · conectada há {{ tempoLigada(dados.wan.desde) }}</template>
          <template v-if="dados.wan.prefixo6"> · IPv6 {{ dados.wan.prefixo6 }}</template>
        </p>
        <p v-else-if="dados.wan?.erro" class="suave pequeno">{{ dados.wan.erro }}</p>
      </div>

      <div class="caminhos">
        <div v-for="s in [{ quem: 'servidor', nome: 'Servidor (cabo)', m: dados.servidor }, { quem: 'mac', nome: 'Mac mini (Wi-Fi)', m: dados.mac }]" :key="s.quem" class="caminho">
          <span class="quem">{{ s.nome }}</span>
          <template v-if="s.m && idade(s.m.em) < 90">
            <span v-for="t in trechos(s.m, s.quem)" :key="t.nome" class="trecho" :class="{ falhou: t.ms < 0 }" :title="t.ms < 0 ? 'Sem resposta' : `${t.ms} ms`">
              {{ t.nome }} <b>{{ t.ms < 0 ? '✕' : `${t.ms} ms` }}</b>
            </span>
          </template>
          <span v-else-if="s.m" class="suave pequeno">sem notícias desde {{ dataHora(s.m.em) }}</span>
          <span v-else class="suave pequeno">{{ s.quem === 'mac' ? 'o sensor do Mac ainda não mandou nada' : 'medindo…' }}</span>
        </div>
      </div>

      <h3 class="sub">Últimas 24 h</h3>
      <div class="faixa" role="img" :aria-label="`${faixa.length} quedas nas últimas 24 horas`">
        <span v-for="f in faixa" :key="f.q.sensor + f.q.inicio" class="trecho-fora" :class="causa(f.q.causa).cor" :style="{ left: `${f.left}%`, width: `${f.width}%` }" :title="`${hora(f.q.inicio)} · ${duracao(f.q.segundos)} · ${causa(f.q.causa).nome}`" />
      </div>
      <div class="marcas pequeno">
        <span v-for="m in marcas" :key="m.left" :style="{ left: `${m.left}%` }">{{ m.texto }}</span>
      </div>

      <p class="suave pequeno resumo">
        <template v-if="resumo.quedas">
          7 dias: {{ resumo.quedas }} {{ resumo.quedas === 1 ? 'queda' : 'quedas' }}, {{ resumo.minutos }} min fora
          <template v-for="[c, n] in resumo.porCausa" :key="c"> · {{ causa(c).nome }}: {{ n }}</template>
        </template>
        <template v-else>Nenhuma queda nos últimos 7 dias.</template>
      </p>

      <ul v-if="dados.quedas.length" class="lista quedas">
        <li v-for="q in dados.quedas" :key="q.sensor + q.inicio">
          <button class="queda" :aria-expanded="aberta === q.sensor + q.inicio" @click="aberta = aberta === q.sensor + q.inicio ? '' : q.sensor + q.inicio">
            <span class="esquerda">
              <span>{{ dataHora(q.inicio) }} <span class="suave">· {{ q.fim ? duracao(q.segundos) : `fora há ${duracao(q.segundos)}` }}</span></span>
              <span class="suave pequeno">{{ q.sensor === 'mac' ? 'visto pelo Mac, no Wi-Fi' : 'visto pelo servidor, no cabo' }}</span>
            </span>
            <span class="etiqueta" :class="causa(q.causa).cor">{{ causa(q.causa).nome }}</span>
          </button>
          <div v-if="aberta === q.sensor + q.inicio" class="detalhe">
            <p class="pequeno">{{ causa(q.causa).fazer }}</p>
            <template v-if="q.registro?.length">
              <p class="pequeno suave">O que o roteador anotou:</p>
              <pre class="registro">{{ q.registro.map((l) => l.replace(/^\S+/, (t) => hora(t)).replace(/^(\d\d:\d\d) /, '$1  ')).join('\n') }}</pre>
            </template>
            <p v-else class="pequeno suave">O roteador não anotou nada sobre a conexão nesse período.</p>
          </div>
        </li>
      </ul>
    </template>
  </section>
</template>

<style scoped>
.internet {
  margin-bottom: 28px;
}

.primeira {
  margin-top: var(--e4);
}

.estado-net {
  padding: 16px;
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  background: var(--superficie);
}

.estado-net.vermelho {
  background: var(--vermelho-fundo);
}

.estado-net.ambar {
  background: var(--ambar-fundo);
}

.situacao {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 600;
}

.ponto {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--texto-fraco);
}

.bom .ponto {
  background: var(--verde);
}

.vermelho .ponto {
  background: var(--vermelho);
}

.ambar .ponto {
  background: var(--ambar);
}

.fazer {
  margin: 6px 0 8px;
  font-size: 14px;
  line-height: 1.5;
}

.estado-net .pequeno {
  margin: 0;
  overflow-wrap: anywhere;
}

.caminhos {
  display: grid;
  gap: 8px;
  margin: 12px 0 0;
}

.caminho {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.quem {
  min-width: 128px;
  font-weight: 500;
}

.trecho {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--verde-fundo);
  color: var(--verde);
  white-space: nowrap;
}

.trecho b {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.trecho.falhou {
  background: var(--vermelho-fundo);
  color: var(--vermelho);
}

.sub {
  margin: 18px 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.faixa {
  position: relative;
  height: 14px;
  border-radius: 4px;
  background: var(--verde-fundo);
  overflow: hidden;
}

.trecho-fora {
  position: absolute;
  top: 0;
  bottom: 0;
  min-width: 2px;
  background: var(--ambar);
}

.trecho-fora.vermelho {
  background: var(--vermelho);
}

.marcas {
  position: relative;
  height: 18px;
  margin-top: 2px;
}

.marcas span {
  position: absolute;
  transform: translateX(-50%);
  white-space: nowrap;
}

.marcas span:first-child {
  transform: none;
}

.marcas span:last-child {
  transform: translateX(-100%);
}

.resumo {
  margin: 8px 0 10px;
}

.queda {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.esquerda {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.detalhe {
  padding: 0 0 12px;
}

.detalhe p {
  margin: 0 0 6px;
}

.registro {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--raio-p);
  background: var(--superficie-2);
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
