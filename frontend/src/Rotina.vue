<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import {
  CORES,
  DIAS,
  LETRAS,
  PRIORIDADES,
  conflitos,
  diaDoMes,
  doDia,
  duracao,
  idDoNome,
  meta,
  minutos,
  novoId,
  placar,
  rotuloSemana,
  somarDias,
  sono,
} from './rotina.js'

// Rotina (H14): o modelo da semana, cada semana planejada a partir dele e as metas de cada atividade.
const recado = inject('recado')

const cfg = ref(null) // { atividades, modelo, hoje, dia_de_hoje, semana_atual }
const erro = ref('')
const aba = ref('semana') // semana | modelo | atividades
const visao = ref('dia') // dia | semana
const inicio = ref('') // segunda-feira da semana aberta
const semana = ref(null) // { inicio, planejada, blocos, notas }
const dia = ref(0)
const editando = ref(null) // { novo, b }
const ocupado = ref(false)

async function carregar() {
  try {
    cfg.value = await api('/api/rotina')
    inicio.value = cfg.value.semana_atual
    dia.value = cfg.value.dia_de_hoje
    await carregarSemana()
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o Hub só abre na rede de casa (ou pelo Tailscale).' : e.message
  }
}

async function carregarSemana() {
  semana.value = null
  try {
    semana.value = await api(`/api/rotina/semanas/${inicio.value}`)
  } catch (e) {
    recado(e.message, true)
  }
}

function mudarSemana(n) {
  inicio.value = somarDias(inicio.value, 7 * n)
  dia.value = inicio.value === cfg.value.semana_atual ? cfg.value.dia_de_hoje : 0
  carregarSemana()
}

function irParaHoje() {
  inicio.value = cfg.value.semana_atual
  dia.value = cfg.value.dia_de_hoje
  carregarSemana()
}

const emModelo = computed(() => aba.value === 'modelo')
const blocos = computed(() => (emModelo.value ? cfg.value?.modelo : semana.value?.blocos) ?? [])
const porId = computed(() => Object.fromEntries((cfg.value?.atividades ?? []).map((a) => [a.id, a])))
const blocosDoDia = computed(() => doDia(blocos.value, dia.value))
const idsEmConflito = computed(() => conflitos(blocos.value))
const sonoDoDia = computed(() => {
  const seguinte = dia.value < 6 ? doDia(blocos.value, dia.value + 1) : doDia(cfg.value?.modelo ?? [], 0)
  return sono(blocosDoDia.value, seguinte)
})
const metas = computed(() => (cfg.value ? placar(cfg.value.atividades, blocos.value) : []))
const faltando = computed(() => metas.value.filter((m) => m.estado === 'falta'))

const relacao = computed(() => {
  if (!cfg.value) return ''
  const atual = cfg.value.semana_atual
  if (inicio.value === atual) return 'Esta semana'
  if (inicio.value === somarDias(atual, 7)) return 'Próxima semana'
  if (inicio.value === somarDias(atual, -7)) return 'Semana passada'
  return ''
})
const ehHoje = (d) => !emModelo.value && inicio.value === cfg.value?.semana_atual && d === cfg.value?.dia_de_hoje
const nomeDe = (b) => porId.value[b.atividade]?.nome ?? 'Atividade apagada'
const corDe = (b) => porId.value[b.atividade]?.cor ?? 'cinza'
const tempo = (b) => duracao(minutos(b.fim) - minutos(b.inicio))

// ---------- Gravar ----------

async function gravar(novos, notas) {
  ocupado.value = true
  try {
    if (emModelo.value) {
      const salvo = await api('/api/rotina', { metodo: 'PUT', corpo: { atividades: cfg.value.atividades, modelo: novos } })
      cfg.value.modelo = salvo.modelo
    } else {
      semana.value = await api(`/api/rotina/semanas/${inicio.value}`, {
        metodo: 'PUT',
        corpo: { blocos: novos, notas: notas ?? semana.value.notas },
      })
    }
    return true
  } catch (e) {
    recado(e.message, true)
    return false
  } finally {
    ocupado.value = false
  }
}

async function planejar(refazer = false) {
  if (refazer && !confirm('Refazer a semana pelo modelo? O que você mudou nela (e o que marcou como feito) se perde.')) return
  ocupado.value = true
  try {
    semana.value = await api(`/api/rotina/semanas/${inicio.value}${refazer ? '?refazer=1' : ''}`, { metodo: 'POST' })
    recado(refazer ? 'Semana refeita pelo modelo.' : 'Semana planejada. Ajuste o que for diferente desta vez.')
  } catch (e) {
    recado(e.message, true)
  } finally {
    ocupado.value = false
  }
}

function alternarFeito(b) {
  gravar(blocos.value.map((x) => (x.id === b.id ? { ...x, feito: !x.feito } : x)))
}

// ---------- Editar um bloco ----------

function somarHora(hhmm, min) {
  const t = Math.min(minutos(hhmm) + min, 24 * 60)
  return `${String(Math.floor(t / 60)).padStart(2, '0')}:${String(t % 60).padStart(2, '0')}`
}

function abrirNovo() {
  const ultimo = blocosDoDia.value.at(-1)
  const comeco = ultimo && ultimo.fim < '23:00' ? ultimo.fim : '08:00'
  editando.value = {
    novo: true,
    b: { id: novoId(), dia: dia.value, inicio: comeco, fim: somarHora(comeco, 60), atividade: cfg.value.atividades[0].id, nota: '', feito: false },
  }
}

function abrirBloco(b) {
  dia.value = b.dia
  editando.value = { novo: false, b: { ...b } }
}

const erroDaEdicao = computed(() => {
  const b = editando.value?.b
  if (!b) return ''
  if (!b.inicio || !b.fim) return 'Preencha o começo e o fim.'
  if (b.fim <= b.inicio) return 'O fim precisa ser depois do começo.'
  return ''
})

async function salvarEdicao() {
  if (erroDaEdicao.value) return
  const { novo, b } = editando.value
  const novos = novo ? [...blocos.value, b] : blocos.value.map((x) => (x.id === b.id ? b : x))
  const mudouDeDia = !novo && blocos.value.find((x) => x.id === b.id)?.dia !== b.dia
  if (await gravar(novos)) {
    editando.value = null
    if (mudouDeDia) {
      recado(`Movido para ${DIAS[b.dia].toLowerCase()}.`)
      dia.value = b.dia
    }
  }
}

async function apagarBloco() {
  const id = editando.value.b.id
  if (await gravar(blocos.value.filter((x) => x.id !== id))) editando.value = null
}

// ---------- O dia inteiro ----------

const notaDoDia = computed({
  get: () => semana.value?.notas?.[dia.value] ?? '',
  set: (v) => {
    const notas = { ...semana.value.notas }
    if (v.trim()) notas[dia.value] = v.trim().slice(0, 120)
    else delete notas[dia.value]
    semana.value.notas = notas
  },
})

function salvarNota() {
  gravar(semana.value.blocos, semana.value.notas)
}

function limparDia() {
  if (!confirm(`Tirar todos os blocos de ${DIAS[dia.value].toLowerCase()}?`)) return
  gravar(blocos.value.filter((b) => b.dia !== dia.value))
}

// Troca o dia aberto pelos blocos do modelo de outro dia (ou do mesmo): "a sexta de feriado como um sábado".
const copiarDe = ref('')
async function copiarDoModelo(origem) {
  const vindos = doDia(cfg.value.modelo, origem).map((b) => ({ ...b, id: novoId(), dia: dia.value, feito: false }))
  if (await gravar([...blocos.value.filter((b) => b.dia !== dia.value), ...vindos])) {
    recado(origem === dia.value ? 'O dia voltou ao modelo.' : `${DIAS[dia.value]} agora segue o modelo de ${DIAS[origem].toLowerCase()}.`)
  }
  copiarDe.value = ''
}

watch(copiarDe, (v) => {
  if (v !== '') copiarDoModelo(Number(v))
})

// ---------- Visão da semana (grade) ----------

const HORA_ALTURA = 30 // px por hora
const gradeInicio = computed(() => {
  const menor = blocos.value.reduce((m, b) => Math.min(m, minutos(b.inicio)), 6 * 60)
  return Math.floor(menor / 60)
})
const horasDaGrade = computed(() => Array.from({ length: 24 - gradeInicio.value }, (_, i) => gradeInicio.value + i))
function posicao(b) {
  const top = ((minutos(b.inicio) - gradeInicio.value * 60) / 60) * HORA_ALTURA
  const altura = ((minutos(b.fim) - minutos(b.inicio)) / 60) * HORA_ALTURA
  return { top: `${top}px`, height: `${Math.max(altura - 2, 4)}px` }
}

// ---------- Atividades ----------

const atividadesEdit = ref([])
watch(aba, (v) => {
  if (v === 'atividades') atividadesEdit.value = cfg.value.atividades.map((a) => ({ ...a }))
})
const atividadesMudaram = computed(() => JSON.stringify(atividadesEdit.value) !== JSON.stringify(cfg.value?.atividades))
const noModelo = computed(() => new Set((cfg.value?.modelo ?? []).map((b) => b.atividade)))

function novaAtividade() {
  atividadesEdit.value.push({ id: '', nome: '', cor: 'azul', meta_min: 0, meta_max: 0, prioridade: 2 })
}

async function salvarAtividades() {
  const usados = new Set(atividadesEdit.value.map((a) => a.id).filter(Boolean))
  const lista = atividadesEdit.value
    .filter((a) => a.id || a.nome.trim())
    .map((a) => {
      const x = { ...a, nome: a.nome.trim(), meta_min: Number(a.meta_min) || 0, meta_max: Number(a.meta_max) || 0, prioridade: Number(a.prioridade) }
      if (x.meta_max < x.meta_min) x.meta_max = x.meta_min
      if (!x.id) {
        x.id = idDoNome(x.nome, usados)
        usados.add(x.id)
      }
      return x
    })
  ocupado.value = true
  try {
    const salvo = await api('/api/rotina', { metodo: 'PUT', corpo: { atividades: lista, modelo: cfg.value.modelo } })
    cfg.value.atividades = salvo.atividades
    atividadesEdit.value = salvo.atividades.map((a) => ({ ...a }))
    recado('Atividades salvas.')
  } catch (e) {
    recado(e.message, true)
  } finally {
    ocupado.value = false
  }
}

onMounted(carregar)
</script>

<template>
  <div class="pagina rotina">
    <h1>Rotina</h1>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <template v-if="cfg">
      <div class="pilulas abas" role="tablist" aria-label="Partes da rotina">
        <button
          v-for="[id, nome] in [
            ['semana', 'Semana'],
            ['modelo', 'Modelo'],
            ['atividades', 'Atividades e metas'],
          ]"
          :key="id"
          role="tab"
          class="pilula"
          :class="{ ativa: aba === id }"
          :aria-selected="aba === id"
          @click="aba = id"
        >
          {{ nome }}
        </button>
      </div>

      <!-- ===== Semana e modelo ===== -->
      <template v-if="aba !== 'atividades'">
        <div v-if="aba === 'semana'" class="navegar">
          <button class="botao icone" aria-label="Semana anterior" @click="mudarSemana(-1)"><Icone nome="voltar" /></button>
          <div class="qual">
            <strong>{{ rotuloSemana(inicio) }}</strong>
            <span v-if="relacao" class="chip">{{ relacao }}</span>
            <button v-else class="botao pequeno-botao" @click="irParaHoje">Ir para hoje</button>
          </div>
          <button class="botao icone" aria-label="Próxima semana" @click="mudarSemana(1)"><Icone nome="avancar" /></button>
        </div>
        <p v-else class="descricao-da-pagina modelo-texto">
          A semana padrão. Cada semana nova nasce como cópia dele; mudar o modelo não mexe nas semanas já planejadas.
        </p>

        <div v-if="aba === 'semana' && semana && !semana.planejada" class="vazio cartao">
          <p>Esta semana ainda não foi planejada.</p>
          <p class="suave pequeno">Ela começa como o modelo; depois é só ajustar o que for diferente (feriado, evento, uma noite livre).</p>
          <button class="botao principal" :disabled="ocupado" @click="planejar()">Planejar a partir do modelo</button>
        </div>

        <template v-else-if="emModelo || semana">
          <div class="pilulas" role="radiogroup" aria-label="Visão">
            <button class="pilula" role="radio" :class="{ ativa: visao === 'dia' }" :aria-checked="visao === 'dia'" @click="visao = 'dia'">Dia</button>
            <button class="pilula" role="radio" :class="{ ativa: visao === 'semana' }" :aria-checked="visao === 'semana'" @click="visao = 'semana'">Semana inteira</button>
          </div>

          <!-- Dias: S T Q Q S S D -->
          <div v-if="visao === 'dia'" class="dias" role="tablist" aria-label="Dia da semana">
            <button
              v-for="(letra, d) in LETRAS"
              :key="d"
              role="tab"
              class="dia"
              :class="{ ativo: dia === d, hoje: ehHoje(d) }"
              :aria-selected="dia === d"
              :aria-label="DIAS[d]"
              @click="dia = d"
            >
              <span class="letra">{{ letra }}</span>
              <span v-if="!emModelo" class="numero">{{ diaDoMes(inicio, d) }}</span>
              <span v-if="!emModelo && semana.notas[d]" class="ponto" aria-hidden="true" />
            </button>
          </div>

          <!-- ===== Um dia ===== -->
          <section v-if="visao === 'dia'" class="um-dia">
            <div class="cabeca-dia">
              <h2>{{ DIAS[dia] }}<span v-if="!emModelo">, {{ diaDoMes(inicio, dia) }}</span></h2>
              <span v-if="ehHoje(dia)" class="chip ok">Hoje</span>
            </div>
            <input
              v-if="!emModelo"
              v-model.lazy="notaDoDia"
              class="campo nota-dia"
              placeholder="Nota do dia: feriado, aniversário, viagem…"
              aria-label="Nota do dia"
              maxlength="120"
              @change="salvarNota"
            />

            <ul v-if="blocosDoDia.length" class="lista blocos">
              <li v-for="b in blocosDoDia" :key="b.id" class="bloco" :class="[`cor-${corDe(b)}`, { feito: b.feito, conflito: idsEmConflito.has(b.id) }]">
                <label v-if="!emModelo" class="marcar" :title="b.feito ? 'Desmarcar' : 'Marcar como feito'">
                  <input type="checkbox" :checked="b.feito" :aria-label="`${nomeDe(b)} feito`" @change="alternarFeito(b)" />
                </label>
                <button class="corpo" @click="abrirBloco(b)">
                  <span class="horario"><span>{{ b.inicio }}</span><span class="ate">–{{ b.fim }}</span></span>
                  <span class="texto">
                    <span class="nome">{{ nomeDe(b) }}</span>
                    <span v-if="b.nota" class="nota">{{ b.nota }}</span>
                  </span>
                  <span class="duracao">{{ tempo(b) }}</span>
                  <span v-if="idsEmConflito.has(b.id)" class="chip erro">Choca</span>
                </button>
              </li>
            </ul>
            <p v-else class="suave livre">Dia livre.</p>

            <p v-if="sonoDoDia" class="sono" :class="{ pouco: sonoDoDia.minutos < 360 }">
              Livre das {{ sonoDoDia.de }} às {{ sonoDoDia.ate }} do dia seguinte: até {{ duracao(sonoDoDia.minutos) }} de sono.
            </p>

            <div class="linha-acoes acoes-dia">
              <button class="botao" @click="abrirNovo">+ Bloco</button>
              <select v-model="copiarDe" class="campo seletor" aria-label="Usar o modelo de um dia">
                <option value="" disabled>{{ emModelo ? 'Copiar outro dia…' : 'Usar o modelo de…' }}</option>
                <option v-for="(nome, d) in DIAS" :key="d" :value="String(d)">{{ !emModelo && d === dia ? `${nome} (voltar ao modelo)` : nome }}</option>
              </select>
              <button v-if="blocosDoDia.length" class="botao perigo" @click="limparDia">Limpar o dia</button>
            </div>
          </section>

          <!-- ===== Semana inteira ===== -->
          <section v-else class="grade-semana" aria-label="Semana inteira">
            <div class="grade" :style="{ '--hora': `${HORA_ALTURA}px`, '--horas': horasDaGrade.length }">
              <div class="coluna-horas">
                <div class="topo-coluna" />
                <div class="corpo-coluna">
                  <span v-for="h in horasDaGrade" :key="h" class="hora">{{ String(h).padStart(2, '0') }}</span>
                </div>
              </div>
              <div v-for="(letra, d) in LETRAS" :key="d" class="coluna" :class="{ hoje: ehHoje(d) }">
                <button class="topo-coluna" :title="DIAS[d]" @click="(dia = d), (visao = 'dia')">
                  {{ letra }}<template v-if="!emModelo"> {{ diaDoMes(inicio, d) }}</template>
                  <span v-if="!emModelo && semana.notas[d]" class="ponto" :title="semana.notas[d]" />
                </button>
                <div class="corpo-coluna">
                  <button
                    v-for="b in doDia(blocos, d)"
                    :key="b.id"
                    class="pedaco"
                    :class="[`cor-${corDe(b)}`, { feito: b.feito, conflito: idsEmConflito.has(b.id) }]"
                    :style="posicao(b)"
                    :title="`${nomeDe(b)} · ${b.inicio}–${b.fim}`"
                    @click="abrirBloco(b)"
                  >
                    {{ nomeDe(b) }}
                  </button>
                </div>
              </div>
            </div>
          </section>

          <!-- ===== Metas ===== -->
          <h2 class="rotulo">Metas {{ emModelo ? 'do modelo' : 'da semana' }}</h2>
          <p v-if="faltando.length" class="aviso erro">
            Faltam na {{ emModelo ? 'semana padrão' : 'semana' }}: {{ faltando.map((m) => `${m.atividade.nome} (${m.falta})`).join(', ') }}.
          </p>
          <ul class="lista metas">
            <li v-for="m in metas" :key="m.atividade.id" class="meta" :class="`cor-${m.atividade.cor}`">
              <span class="ponto-cor" aria-hidden="true" />
              <span class="nome">{{ m.atividade.nome }}</span>
              <span class="suave pequeno">{{ duracao(m.tempo || 0) }}</span>
              <span class="chips">
                <span class="chip" :class="{ erro: m.estado === 'falta', ok: m.estado === 'ok' }">{{ m.planejados }} de {{ meta(m.atividade) }}</span>
                <span v-if="!emModelo && m.feitos" class="chip ok">{{ m.feitos }} {{ m.feitos === 1 ? 'feita' : 'feitas' }}</span>
              </span>
            </li>
          </ul>

          <div v-if="!emModelo" class="linha-acoes rodape-semana">
            <button class="botao" :disabled="ocupado" @click="planejar(true)">Refazer pelo modelo</button>
          </div>
        </template>
      </template>

      <!-- ===== Atividades ===== -->
      <template v-else>
        <p class="descricao-da-pagina modelo-texto">
          A meta é quantas vezes por semana. A prioridade diz o que fica quando a semana aperta: alta primeiro.
        </p>
        <ul class="lista atividades">
          <li v-for="(a, i) in atividadesEdit" :key="a.id || `nova-${i}`" class="atividade" :class="`cor-${a.cor}`">
            <div class="linha-nome">
              <span class="ponto-cor" aria-hidden="true" />
              <input v-model="a.nome" class="campo" placeholder="Nome" aria-label="Nome da atividade" maxlength="60" />
              <button
                class="botao icone"
                :disabled="noModelo.has(a.id)"
                :title="noModelo.has(a.id) ? 'Está no modelo: tire os blocos dela antes' : 'Apagar'"
                aria-label="Apagar atividade"
                @click="atividadesEdit.splice(i, 1)"
              >
                <Icone nome="fechar" />
              </button>
            </div>
            <div class="linha-campos">
              <div class="cores" role="radiogroup" aria-label="Cor">
                <button
                  v-for="c in CORES"
                  :key="c"
                  role="radio"
                  class="amostra"
                  :class="[`cor-${c}`, { ativa: a.cor === c }]"
                  :aria-checked="a.cor === c"
                  :aria-label="c"
                  @click="a.cor = c"
                />
              </div>
              <label class="numero-meta">
                <span class="suave pequeno">Meta</span>
                <input v-model.number="a.meta_min" type="number" min="0" max="21" class="campo" aria-label="Mínimo por semana" />
                <span class="suave">a</span>
                <input v-model.number="a.meta_max" type="number" min="0" max="21" class="campo" aria-label="Máximo por semana" />
                <span class="suave pequeno">× sem.</span>
              </label>
              <select v-model.number="a.prioridade" class="campo prioridade" aria-label="Prioridade">
                <option v-for="(nome, p) in PRIORIDADES" :key="p" :value="Number(p)">{{ nome }}</option>
              </select>
            </div>
          </li>
        </ul>
        <div class="linha-acoes rodape-semana">
          <button class="botao" @click="novaAtividade">+ Atividade</button>
          <button class="botao principal" :disabled="!atividadesMudaram || ocupado" @click="salvarAtividades">Salvar</button>
        </div>
      </template>
    </template>

    <!-- ===== Folha de edição de um bloco ===== -->
    <Transition name="folha">
      <div v-if="editando" class="veu" @click.self="editando = null">
        <form class="folha vidro" :aria-label="editando.novo ? 'Novo bloco' : 'Editar bloco'" @submit.prevent="salvarEdicao">
          <h2>{{ editando.novo ? 'Novo bloco' : 'Editar bloco' }}</h2>
          <label>
            <span class="rotulo-campo">Atividade</span>
            <select v-model="editando.b.atividade" class="campo">
              <option v-for="a in cfg.atividades" :key="a.id" :value="a.id">{{ a.nome }}</option>
              <option v-if="!porId[editando.b.atividade]" :value="editando.b.atividade">Atividade apagada</option>
            </select>
          </label>
          <label>
            <span class="rotulo-campo">Dia</span>
            <select v-model.number="editando.b.dia" class="campo">
              <option v-for="(nome, d) in DIAS" :key="d" :value="d">{{ nome }}<template v-if="!emModelo">, {{ diaDoMes(inicio, d) }}</template></option>
            </select>
          </label>
          <div class="horas">
            <label>
              <span class="rotulo-campo">Começa</span>
              <input v-model="editando.b.inicio" type="time" step="300" class="campo" required />
            </label>
            <label>
              <span class="rotulo-campo">Termina</span>
              <input v-model="editando.b.fim" type="time" step="300" class="campo" required />
            </label>
          </div>
          <label>
            <span class="rotulo-campo">Nota</span>
            <input v-model="editando.b.nota" class="campo" maxlength="300" placeholder="Opcional" />
          </label>
          <p v-if="erroDaEdicao" class="erro-texto">{{ erroDaEdicao }}</p>
          <div class="linha-acoes">
            <button class="botao principal" :disabled="!!erroDaEdicao || ocupado">Salvar</button>
            <button type="button" class="botao" @click="editando = null">Cancelar</button>
            <button v-if="!editando.novo" type="button" class="botao perigo apagar" :disabled="ocupado" @click="apagarBloco">Apagar</button>
          </div>
        </form>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.pequeno {
  font-size: 0.82rem;
}

.abas {
  margin-bottom: 18px;
}

.pilulas {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.modelo-texto {
  margin: 0 0 16px;
}

/* Navegar entre semanas */
.navegar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 16px;
}

.qual {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 4px 10px;
  font-size: 1.0625rem;
}

.pequeno-botao {
  min-height: 26px;
  padding: 0 8px;
  font-size: 0.78rem;
}

.vazio p {
  margin: 0 0 10px;
}

.vazio .botao {
  margin-top: 6px;
}

/* Cores das atividades: uma barra, um ponto ou um fundo leve; o texto fica sempre nas cores de texto. */
.cor-azul { --cor: var(--r-azul); }
.cor-violeta { --cor: var(--r-violeta); }
.cor-verde { --cor: var(--r-verde); }
.cor-laranja { --cor: var(--r-laranja); }
.cor-amarelo { --cor: var(--r-amarelo); }
.cor-rosa { --cor: var(--r-rosa); }
.cor-turquesa { --cor: var(--r-turquesa); }
.cor-cinza { --cor: var(--r-cinza); }

.ponto-cor {
  flex-shrink: 0;
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: var(--cor);
}

/* Dias */
.dias {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
  margin: 14px 0 6px;
}

.dia {
  position: relative;
  display: grid;
  justify-items: center;
  gap: 1px;
  padding: 7px 0 6px;
  border: 1px solid transparent;
  border-radius: var(--raio-p);
  background: transparent;
  color: var(--texto-suave);
  cursor: pointer;
}

.dia:hover {
  background: var(--superficie);
}

.dia .letra {
  font-size: 0.8rem;
  font-weight: 600;
}

.dia .numero {
  font-size: 1.05rem;
  font-variant-numeric: tabular-nums;
  color: var(--texto);
}

.dia.hoje {
  border-color: var(--marca);
}

.dia.ativo {
  background: var(--marca);
  color: var(--marca-texto);
}

.dia.ativo .numero {
  color: var(--marca-texto);
}

.ponto {
  position: absolute;
  top: 5px;
  right: 6px;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--vermelho);
}

/* Um dia */
.cabeca-dia {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 18px 0 8px;
}

.cabeca-dia h2 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 650;
  letter-spacing: -0.012em;
}

.nota-dia {
  margin-bottom: 12px;
}

.bloco {
  display: flex;
  align-items: stretch;
  gap: 4px;
  border-left: 3px solid var(--cor);
}

.bloco .marcar {
  display: flex;
  align-items: center;
  padding: 0 4px 0 10px;
  cursor: pointer;
}

.bloco .marcar input {
  width: 18px;
  height: 18px;
  accent-color: var(--marca);
}

.corpo {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px 6px 12px 10px;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.corpo:hover .nome {
  color: var(--marca);
}

.horario {
  flex-shrink: 0;
  width: 92px;
  color: var(--texto-suave);
  font-size: 0.88rem;
  font-variant-numeric: tabular-nums;
}

.texto {
  display: grid;
  flex: 1;
  min-width: 0;
}

.texto .nome {
  font-size: 1.0625rem;
  font-weight: 650;
  transition: color 0.15s ease;
}

.texto .nota {
  overflow: hidden;
  color: var(--texto-suave);
  font-size: 0.85rem;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.duracao {
  flex-shrink: 0;
  color: var(--texto-fraco);
  font-size: 0.78rem;
}

.bloco.feito .nome {
  color: var(--texto-suave);
  text-decoration: line-through;
}

.bloco.conflito {
  background: var(--vermelho-fundo);
}

.livre {
  padding: 18px 0;
  border-top: 1px solid var(--linha);
  border-bottom: 1px solid var(--linha);
}

.sono {
  margin: 10px 0 0;
  color: var(--texto-fraco);
  font-size: 0.82rem;
}

.sono.pouco {
  color: var(--vermelho);
}

.acoes-dia {
  margin-top: 14px;
}

.seletor {
  width: auto;
  min-height: 32px;
  padding: 4px 10px;
  font-size: 0.87rem;
}

/* Semana inteira */
.grade-semana {
  margin-top: 14px;
  overflow-x: auto;
}

.grade {
  display: grid;
  grid-template-columns: 30px repeat(7, minmax(40px, 1fr));
  gap: 0 3px;
}

.topo-coluna {
  position: relative;
  height: 28px;
  border: 0;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
}

.coluna.hoje .topo-coluna {
  color: var(--marca);
}

.topo-coluna .ponto {
  top: 4px;
  right: 2px;
}

.corpo-coluna {
  position: relative;
  height: calc(var(--hora) * var(--horas));
  background: repeating-linear-gradient(to bottom, var(--linha) 0 1px, transparent 1px var(--hora));
}

.coluna-horas .corpo-coluna {
  display: grid;
  grid-auto-rows: var(--hora);
  background: none;
}

.hora {
  color: var(--texto-fraco);
  font-size: 0.66rem;
  font-variant-numeric: tabular-nums;
  transform: translateY(-5px);
}

.pedaco {
  position: absolute;
  right: 0;
  left: 0;
  overflow: hidden;
  padding: 2px 4px;
  border: 0;
  border-left: 3px solid var(--cor);
  border-radius: 4px;
  background: color-mix(in srgb, var(--cor) 18%, var(--fundo));
  color: var(--texto);
  font-size: 0.66rem;
  line-height: 1.2;
  text-align: left;
  -webkit-hyphens: auto;
  hyphens: auto;
  cursor: pointer;
}

.pedaco.feito {
  opacity: 0.5;
}

.pedaco.conflito {
  outline: 1.5px solid var(--vermelho);
}

/* Metas */
.meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 11px 0;
}

.meta .nome {
  font-weight: 600;
}

.chips {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.rodape-semana {
  margin-top: 20px;
}

/* Atividades */
.atividade {
  display: grid;
  gap: 8px;
  padding: 12px 0;
}

.linha-nome {
  display: flex;
  align-items: center;
  gap: 10px;
}

.linha-campos {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 16px;
  padding-left: 20px;
}

.cores {
  display: flex;
  gap: 6px;
}

.amostra {
  width: 22px;
  height: 22px;
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--cor);
  cursor: pointer;
}

.amostra.ativa {
  border-color: var(--fundo);
  box-shadow: 0 0 0 2px var(--texto);
}

.numero-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.numero-meta .campo {
  width: 58px;
  min-height: 32px;
  padding: 4px 8px;
}

.prioridade {
  width: auto;
  min-height: 32px;
  padding: 4px 10px;
}

/* Folha */
.veu {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 28%);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
}

.folha {
  display: grid;
  gap: 12px;
  width: min(520px, 100%);
  max-height: 90vh;
  overflow-y: auto;
  padding: 20px 20px max(20px, env(safe-area-inset-bottom));
  border: 1px solid var(--vidro-borda);
  border-radius: var(--raio-g) var(--raio-g) 0 0;
  box-shadow: var(--sombra);
}

.folha h2 {
  margin: 0 0 4px;
  font-size: 1.25rem;
  font-weight: 650;
}

.folha label {
  display: grid;
  gap: 5px;
}

.rotulo-campo {
  color: var(--texto-suave);
  font-size: 0.81rem;
  font-weight: 600;
}

.horas {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.apagar {
  margin-left: auto;
}

.erro-texto {
  margin: 0;
  color: var(--vermelho);
  font-size: 0.85rem;
}

.folha-enter-active,
.folha-leave-active {
  transition: opacity 0.2s ease;
}

.folha-enter-active .folha,
.folha-leave-active .folha {
  transition: transform 0.22s ease;
}

.folha-enter-from,
.folha-leave-to {
  opacity: 0;
}

.folha-enter-from .folha,
.folha-leave-to .folha {
  transform: translateY(24px);
}

@media (min-width: 821px) {
  .veu {
    align-items: center;
  }

  .folha {
    border-radius: var(--raio-g);
  }
}

@media (max-width: 520px) {
  /* No celular, o começo em cima e o fim embaixo, menor. */
  .horario {
    display: grid;
    width: 50px;
    color: var(--texto);
    font-size: 0.9rem;
    font-weight: 600;
  }

  .horario .ate {
    color: var(--texto-fraco);
    font-size: 0.78rem;
    font-weight: 400;
  }

  .duracao {
    display: none;
  }
}
</style>
