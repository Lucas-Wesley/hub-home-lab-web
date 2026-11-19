<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import FalarRotina from './FalarRotina.vue'
import GradeSemana from './GradeSemana.vue'
import Icone from './Icone.vue'
import MesRotina from './MesRotina.vue'
import SugestaoSemana from './SugestaoSemana.vue'
import { api } from './api.js'
import {
  CORES,
  DIAS,
  LETRAS,
  PRIORIDADES,
  conflitos,
  dataCurta,
  diaDoMes,
  diasComChoque,
  doDia,
  duracao,
  idDoNome,
  meta,
  minutos,
  nomeDoBloco,
  novoId,
  placar,
  rotuloSemana,
  somarDias,
  sono,
} from './rotina.js'

// Rotina (H14): o modelo da semana, cada semana planejada a partir dele, o mês, as metas de cada
// atividade e o que se adiciona pela voz.
const recado = inject('recado')

const cfg = ref(null) // { atividades, modelo, hoje, dia_de_hoje, semana_atual, ia }
const erro = ref('')
const aba = ref('semana') // semana | mes | modelo | atividades
const visao = ref('dia') // dia | semana
const inicio = ref('') // segunda-feira da semana aberta
const semana = ref(null) // { inicio, planejada, blocos, notas }
const dia = ref(0)
const editando = ref(null) // { modo: 'bloco' | 'evento', novo, b, data, semHorario }
const ocupado = ref(false)
const adicionados = ref(null) // o último resultado da voz ou do "+ Evento"
const versaoDoMes = ref(0)

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
  try {
    semana.value = await api(`/api/rotina/semanas/${inicio.value}`)
  } catch (e) {
    recado(e.message, true)
  }
}

function abrirSemana(novoInicio, novoDia) {
  aba.value = 'semana'
  visao.value = 'dia'
  dia.value = novoDia
  if (novoInicio !== inicio.value) {
    inicio.value = novoInicio
    semana.value = null
  }
  carregarSemana()
}

function mudarSemana(n) {
  const novo = somarDias(inicio.value, 7 * n)
  abrirSemana(novo, novo === cfg.value.semana_atual ? cfg.value.dia_de_hoje : 0)
}

const emModelo = computed(() => aba.value === 'modelo')
const blocos = computed(() => (emModelo.value ? cfg.value?.modelo : semana.value?.blocos) ?? [])
const porId = computed(() => Object.fromEntries((cfg.value?.atividades ?? []).map((a) => [a.id, a])))
const blocosDoDia = computed(() => doDia(blocos.value, dia.value))
const idsEmChoque = computed(() => conflitos(blocos.value))
const diasEmChoque = computed(() => diasComChoque(blocos.value))
const sonoDoDia = computed(() => {
  const seguinte = dia.value < 6 ? doDia(blocos.value, dia.value + 1) : doDia(cfg.value?.modelo ?? [], 0)
  return sono(blocosDoDia.value, seguinte)
})
const metas = computed(() => (cfg.value ? placar(cfg.value.atividades, blocos.value) : []))
const faltando = computed(() => metas.value.filter((m) => m.estado === 'falta'))
const numeros = computed(() => (emModelo.value ? null : LETRAS.map((_, d) => diaDoMes(inicio.value, d))))

const relacao = computed(() => {
  if (!cfg.value) return ''
  const atual = cfg.value.semana_atual
  if (inicio.value === atual) return 'Esta semana'
  if (inicio.value === somarDias(atual, 7)) return 'Próxima semana'
  if (inicio.value === somarDias(atual, -7)) return 'Semana passada'
  return ''
})
const diaDeHoje = computed(() => (!emModelo.value && inicio.value === cfg.value?.semana_atual ? cfg.value.dia_de_hoje : -1))
const nomeDe = (b) => nomeDoBloco(b, porId.value)
const corDe = (b) => porId.value[b.atividade]?.cor ?? (b.titulo ? 'evento' : 'cinza')
const tempo = (b) => (b.inicio ? duracao(minutos(b.fim) - minutos(b.inicio)) : '')

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
  if (refazer && !confirm('Refazer a semana pelo modelo? As mudanças nos blocos do modelo e o que foi marcado como feito se perdem; os eventos adicionados ficam.')) return
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

async function apagar(b) {
  if (await gravar(blocos.value.filter((x) => x.id !== b.id))) recado(`${nomeDe(b)} apagado.`)
}

// ---------- Voz e eventos com data ----------

function aoAdicionar(r) {
  adicionados.value = r
  versaoDoMes.value++
  if (r.eventos.length === 1 && aba.value !== 'mes') {
    abrirSemana(r.eventos[0].semana, r.eventos[0].dia)
  } else if (r.eventos.some((e) => e.semana === inicio.value)) {
    carregarSemana()
  }
  if (!r.eventos.length) recado(r.observacao || 'Não entendi o que adicionar.', true)
}

// ---------- Editar um bloco ----------

function somarHora(hhmm, min) {
  const t = Math.min(minutos(hhmm) + min, 24 * 60)
  return `${String(Math.floor(t / 60)).padStart(2, '0')}:${String(t % 60).padStart(2, '0')}`
}

function abrirNovo() {
  const ultimo = blocosDoDia.value.filter((b) => b.inicio).at(-1)
  const comeco = ultimo && ultimo.fim < '23:00' ? ultimo.fim : '08:00'
  editando.value = {
    modo: 'bloco',
    novo: true,
    semHorario: false,
    b: { id: novoId(), dia: dia.value, inicio: comeco, fim: somarHora(comeco, 60), atividade: cfg.value.atividades[0].id, titulo: '', nota: '', feito: false, extra: !emModelo.value },
  }
}

function abrirEvento(data) {
  editando.value = {
    modo: 'evento',
    novo: true,
    semHorario: false,
    data,
    b: { id: 'novo', dia: 0, inicio: '19:00', fim: '20:00', atividade: '', titulo: '', nota: '', feito: false, extra: true },
  }
}

function abrirBloco(b) {
  dia.value = b.dia
  editando.value = { modo: 'bloco', novo: false, semHorario: !b.inicio, b: { titulo: '', extra: false, ...b } }
}

const erroDaEdicao = computed(() => {
  const e = editando.value
  if (!e) return ''
  if (!e.b.atividade && !e.b.titulo.trim()) return 'Dê um nome ao evento.'
  if (e.modo === 'evento' && !e.data) return 'Escolha a data.'
  if (e.semHorario) return ''
  if (!e.b.inicio || !e.b.fim) return 'Preencha o começo e o fim.'
  if (e.b.fim <= e.b.inicio) return 'O fim precisa ser depois do começo.'
  return ''
})

async function salvarEdicao() {
  if (erroDaEdicao.value) return
  const e = editando.value
  const b = { ...e.b, titulo: e.b.atividade ? '' : e.b.titulo.trim() }
  if (e.semHorario) b.inicio = b.fim = ''

  if (e.modo === 'evento') {
    ocupado.value = true
    try {
      const r = await api('/api/rotina/eventos', {
        metodo: 'POST',
        corpo: { eventos: [{ data: e.data, inicio: b.inicio, fim: b.fim, atividade: b.atividade, titulo: b.titulo, nota: b.nota }] },
      })
      editando.value = null
      aoAdicionar(r)
    } catch (err) {
      recado(err.message, true)
    } finally {
      ocupado.value = false
    }
    return
  }

  const novos = e.novo ? [...blocos.value, b] : blocos.value.map((x) => (x.id === b.id ? b : x))
  const mudouDeDia = !e.novo && blocos.value.find((x) => x.id === b.id)?.dia !== b.dia
  if (await gravar(novos)) {
    editando.value = null
    if (mudouDeDia) {
      recado(`Movido para ${DIAS[b.dia].toLowerCase()}.`)
      dia.value = b.dia
    }
  }
}

async function apagarDaFolha() {
  const b = editando.value.b
  editando.value = null
  await apagar(b)
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

// Troca o dia aberto pelos blocos do modelo de outro dia (ou do mesmo): "a sexta de feriado como um
// sábado". Os eventos adicionados no dia ficam.
const copiarDe = ref('')
async function copiarDoModelo(origem) {
  const vindos = doDia(cfg.value.modelo, origem).map((b) => ({ ...b, id: novoId(), dia: dia.value, feito: false }))
  const ficam = blocos.value.filter((b) => b.dia !== dia.value || (!emModelo.value && b.extra))
  if (await gravar([...ficam, ...vindos])) {
    recado(origem === dia.value ? 'O dia voltou ao modelo.' : `${DIAS[dia.value]} agora segue o modelo de ${DIAS[origem].toLowerCase()}.`)
  }
  copiarDe.value = ''
}

watch(copiarDe, (v) => {
  if (v !== '') copiarDoModelo(Number(v))
})

// ---------- Atividades ----------

const atividadesEdit = ref([])
watch(aba, (v) => {
  if (v === 'atividades') atividadesEdit.value = cfg.value.atividades.map((a) => ({ fixa: false, ...a }))
})
const atividadesMudaram = computed(() => JSON.stringify(atividadesEdit.value) !== JSON.stringify(cfg.value?.atividades.map((a) => ({ fixa: false, ...a }))))
const noModelo = computed(() => new Set((cfg.value?.modelo ?? []).map((b) => b.atividade)))

function novaAtividade() {
  atividadesEdit.value.push({ id: '', nome: '', cor: 'azul', meta_min: 0, meta_max: 0, prioridade: 2, fixa: false })
}

// As atividades da tela prontas para o servidor: números, meta coerente e id para as novas.
const atividadesNormais = computed(() => {
  const usados = new Set(atividadesEdit.value.map((a) => a.id).filter(Boolean))
  return atividadesEdit.value
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
})

async function salvarConfig(modelo, mensagem) {
  ocupado.value = true
  try {
    const salvo = await api('/api/rotina', { metodo: 'PUT', corpo: { atividades: atividadesNormais.value, modelo } })
    cfg.value.atividades = salvo.atividades
    cfg.value.modelo = salvo.modelo
    atividadesEdit.value = salvo.atividades.map((a) => ({ fixa: false, ...a }))
    recado(mensagem)
    return true
  } catch (e) {
    recado(e.message, true)
    return false
  } finally {
    ocupado.value = false
  }
}

function salvarAtividades() {
  salvarConfig(cfg.value.modelo, 'Atividades salvas.')
}

async function usarSugestao(sugeridos) {
  if (!confirm('Trocar o modelo pela semana sugerida? As atividades da tela são salvas junto. As semanas já planejadas não mudam.')) return
  if (await salvarConfig(sugeridos, 'A sugestão virou o modelo.')) aba.value = 'modelo'
}

onMounted(carregar)
</script>

<template>
  <div class="pagina rotina">
    <div class="topo-pagina">
      <h1>Rotina</h1>
      <FalarRotina v-if="cfg?.ia && (aba === 'semana' || aba === 'mes')" class="meio" @adicionado="aoAdicionar" />
    </div>

    <div v-if="erro" class="aviso">{{ erro }}</div>

    <template v-if="cfg">
      <div v-if="adicionados?.eventos.length" class="adicionados cartao" role="status">
        <button class="botao icone fechar" aria-label="Fechar" @click="adicionados = null"><Icone nome="fechar" /></button>
        <p class="titulo-adicionados">{{ adicionados.eventos.length === 1 ? 'Adicionado' : `${adicionados.eventos.length} adicionados` }}</p>
        <ul>
          <li v-for="ev in adicionados.eventos" :key="ev.id">
            <button class="link" @click="abrirSemana(ev.semana, ev.dia)">
              <strong>{{ ev.nome }}</strong> · {{ dataCurta(ev.data) }}{{ ev.inicio ? `, ${ev.inicio}–${ev.fim}` : ', sem horário' }}
            </button>
            <span v-if="ev.choca_com.length" class="choque"><Icone nome="alerta" /> No mesmo horário de {{ ev.choca_com.join(', ') }}: resolva no dia.</span>
            <span v-if="ev.semana_nova" class="suave pequeno"> A semana ainda não tinha plano e nasceu do modelo.</span>
          </li>
        </ul>
        <p v-if="adicionados.observacao" class="suave pequeno observacao">{{ adicionados.observacao }}</p>
      </div>

      <div class="ferramentas">
      <div class="abas" role="tablist" aria-label="Partes da rotina">
        <button
          v-for="[id, nome] in [
            ['semana', 'Semana'],
            ['mes', 'Mês'],
            ['modelo', 'Modelo'],
            ['atividades', 'Atividades e metas'],
          ]"
          :key="id"
          role="tab"
          :class="{ ativa: aba === id }"
          :aria-selected="aba === id"
          @click="aba = id"
        >
          {{ nome }}
        </button>
      </div>

      <div v-if="aba === 'semana' || aba === 'modelo'" class="direita">
        <div v-if="aba === 'semana'" class="navegar">
          <button class="botao icone discreto" aria-label="Semana anterior" @click="mudarSemana(-1)"><Icone nome="voltar" /></button>
          <strong class="qual">{{ rotuloSemana(inicio) }}</strong>
          <button class="botao icone discreto" aria-label="Próxima semana" @click="mudarSemana(1)"><Icone nome="avancar" /></button>
          <span v-if="relacao" class="etiqueta">{{ relacao }}</span>
          <button v-else class="botao discreto" @click="abrirSemana(cfg.semana_atual, cfg.dia_de_hoje)">Ir para hoje</button>
        </div>
        <div v-if="emModelo || semana?.planejada" class="abas" role="radiogroup" aria-label="Visão">
          <button role="radio" :class="{ ativa: visao === 'dia' }" :aria-checked="visao === 'dia'" @click="visao = 'dia'">Dia</button>
          <button role="radio" :class="{ ativa: visao === 'semana' }" :aria-checked="visao === 'semana'" @click="visao = 'semana'">Semana inteira</button>
        </div>
      </div>
      </div>

      <!-- ===== Mês ===== -->
      <MesRotina v-if="aba === 'mes'" :cfg="cfg" :por-id="porId" :versao="versaoDoMes" @abrir="(x) => abrirSemana(x.inicio, x.dia)" @novo="abrirEvento" />

      <!-- ===== Semana e modelo ===== -->
      <template v-else-if="aba === 'semana' || aba === 'modelo'">
        <p v-if="aba === 'modelo'" class="explicacao">
          A semana padrão. Cada semana nova nasce como cópia dele; mudar o modelo não mexe nas semanas já planejadas.
        </p>

        <div v-if="aba === 'semana' && semana && !semana.planejada" class="vazio cartao cheio">
          <p>Esta semana ainda não foi planejada.</p>
          <p class="pequeno">Ela começa como o modelo; depois é só ajustar o que for diferente (feriado, evento, uma noite livre).</p>
          <button class="botao primario" :disabled="ocupado" @click="planejar()">Planejar a partir do modelo</button>
        </div>

        <div v-else-if="emModelo || semana" class="duas-colunas">
          <div class="principal">
          <!-- Dias: S T Q Q S S D -->
          <div v-if="visao === 'dia'" class="dias" role="tablist" aria-label="Dia da semana">
            <button
              v-for="(letra, d) in LETRAS"
              :key="d"
              role="tab"
              class="dia"
              :class="{ ativo: dia === d, hoje: diaDeHoje === d }"
              :aria-selected="dia === d"
              :aria-label="DIAS[d] + (diasEmChoque.has(d) ? ', com choque de horário' : '')"
              @click="dia = d"
            >
              <span class="letra">{{ letra }}</span>
              <span v-if="numeros" class="numero">{{ numeros[d] }}</span>
              <span v-if="diasEmChoque.has(d)" class="alerta" aria-hidden="true">!</span>
              <span v-else-if="!emModelo && semana.notas[d]" class="ponto" aria-hidden="true" />
            </button>
          </div>

          <!-- ===== Um dia ===== -->
          <section v-if="visao === 'dia'" class="um-dia">
            <div class="cabeca-dia">
              <h2>{{ DIAS[dia] }}<span v-if="numeros">, {{ numeros[dia] }}</span></h2>
              <span v-if="diaDeHoje === dia" class="etiqueta azul">Hoje</span>
              <input
                v-if="!emModelo"
                v-model.lazy="notaDoDia"
                class="campo nota-dia"
                placeholder="Nota do dia: feriado, aniversário, viagem…"
                aria-label="Nota do dia"
                maxlength="120"
                @change="salvarNota"
              />
            </div>
            <p v-if="diasEmChoque.has(dia)" class="aviso choque-dia">
              <Icone nome="alerta" /> Tem coisas no mesmo horário (em vermelho). Mova, encurte ou apague uma delas.
            </p>

            <ul v-if="blocosDoDia.length" class="lista blocos">
              <li v-for="b in blocosDoDia" :key="b.id" class="bloco" :class="[`cor-${corDe(b)}`, { feito: b.feito, conflito: idsEmChoque.has(b.id) }]">
                <label v-if="!emModelo" class="marcar" :title="b.feito ? 'Desmarcar' : 'Marcar como feito'">
                  <input type="checkbox" :checked="b.feito" :aria-label="`${nomeDe(b)} feito`" @change="alternarFeito(b)" />
                </label>
                <button class="corpo" @click="abrirBloco(b)">
                  <span v-if="b.inicio" class="horario"><span>{{ b.inicio }}</span><span class="ate">–{{ b.fim }}</span></span>
                  <span v-else class="horario tarefa">Tarefa</span>
                  <span class="texto">
                    <span class="nome">{{ nomeDe(b) }}</span>
                    <span v-if="b.nota" class="nota">{{ b.nota }}</span>
                  </span>
                  <span class="duracao">{{ tempo(b) }}</span>
                </button>
                <button class="botao icone lixeira" :aria-label="`Apagar ${nomeDe(b)}`" title="Apagar" :disabled="ocupado" @click="apagar(b)">
                  <Icone nome="lixeira" />
                </button>
              </li>
            </ul>
            <p v-else class="suave livre">Dia livre.</p>

            <p v-if="sonoDoDia" class="sono" :class="{ pouco: sonoDoDia.minutos < 360 }">
              Livre das {{ sonoDoDia.de }} às {{ sonoDoDia.ate }} do dia seguinte: até {{ duracao(sonoDoDia.minutos) }} de sono.
            </p>

            <div class="linha-acoes acoes-dia">
              <button class="botao" @click="abrirNovo">+ Bloco</button>
              <select v-model="copiarDe" class="campo compacto" aria-label="Usar o modelo de um dia">
                <option value="" disabled>{{ emModelo ? 'Copiar outro dia…' : 'Usar o modelo de…' }}</option>
                <option v-for="(nome, d) in DIAS" :key="d" :value="String(d)">{{ !emModelo && d === dia ? `${nome} (voltar ao modelo)` : nome }}</option>
              </select>
              <button v-if="blocosDoDia.length" class="botao perigo" @click="limparDia">Limpar o dia</button>
            </div>
          </section>

          <!-- ===== Semana inteira ===== -->
          <GradeSemana
            v-else
            :blocos="blocos"
            :por-id="porId"
            :numeros="numeros"
            :notas="emModelo ? {} : semana.notas"
            :hoje="diaDeHoje"
            @bloco="abrirBloco"
            @dia="(d) => ((dia = d), (visao = 'dia'))"
          />
          </div>

          <!-- ===== Metas (painel ao lado) ===== -->
          <aside class="lateral">
          <section class="quadro">
          <div class="quadro-topo">
            <h2 class="rotulo-secao">Metas {{ emModelo ? 'do modelo' : 'da semana' }}</h2>
            <button v-if="!emModelo" class="link-acao" :disabled="ocupado" @click="planejar(true)">Refazer pelo modelo</button>
          </div>
          <p v-if="faltando.length" class="aviso">
            Faltam na {{ emModelo ? 'semana padrão' : 'semana' }}: {{ faltando.map((m) => `${m.atividade.nome} (${m.falta})`).join(', ') }}.
          </p>
          <ul class="lista metas">
            <li v-for="m in metas" :key="m.atividade.id" class="meta" :class="`cor-${m.atividade.cor}`">
              <span class="ponto-cor" aria-hidden="true" />
              <span class="nome">{{ m.atividade.nome }}</span>
              <span class="fraco tempo-meta">{{ duracao(m.tempo || 0) }}</span>
              <span class="etiquetas">
                <span class="etiqueta" :class="{ vermelho: m.estado === 'falta', verde: m.estado === 'ok' }">{{ m.planejados }} de {{ meta(m.atividade) }}{{ m.estado === 'fixa' ? ' · fixa' : '' }}</span>
                <span v-if="!emModelo && m.feitos" class="etiqueta verde">{{ m.feitos }} {{ m.feitos === 1 ? 'feita' : 'feitas' }}</span>
              </span>
            </li>
          </ul>
          </section>
          </aside>
        </div>
      </template>

      <!-- ===== Atividades ===== -->
      <template v-else>
        <p class="explicacao">
          A meta é quantas vezes por semana. A prioridade diz o que fica quando a semana aperta: alta primeiro. Fixa é o que não dá para remarcar (o futebol de
          terça): se não acontecer, não aparece como falta.
        </p>
        <ul class="lista atividades">
          <li v-for="(a, i) in atividadesEdit" :key="a.id || `nova-${i}`" class="atividade" :class="`cor-${a.cor}`">
            <div class="linha-nome">
              <span class="ponto-cor" aria-hidden="true" />
              <input v-model="a.nome" class="campo" placeholder="Nome" aria-label="Nome da atividade" maxlength="60" />
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
              <label class="marcar-fixa"><input v-model="a.fixa" type="checkbox" /> Fixa</label>
            </div>
              <button
                class="botao icone discreto apagar-atividade"
                :disabled="noModelo.has(a.id)"
                :title="noModelo.has(a.id) ? 'Está no modelo: tire os blocos dela antes' : 'Apagar'"
                aria-label="Apagar atividade"
                @click="atividadesEdit.splice(i, 1)"
              >
                <Icone nome="lixeira" />
              </button>
          </li>
        </ul>
        <div class="linha-acoes rodape-semana">
          <button class="botao" @click="novaAtividade">+ Atividade</button>
          <button class="botao primario" :disabled="!atividadesMudaram || ocupado" @click="salvarAtividades">Salvar</button>
        </div>

        <SugestaoSemana :atividades="atividadesNormais" :ligada="cfg.ia" @usar="usarSugestao" />
      </template>
    </template>

    <!-- ===== Folha: editar um bloco ou criar um evento com data ===== -->
    <Transition name="folha">
      <div v-if="editando" class="veu" @click.self="editando = null">
        <form
          class="folha"
          :aria-label="editando.modo === 'evento' ? 'Novo evento' : editando.novo ? 'Novo bloco' : 'Editar bloco'"
          @submit.prevent="salvarEdicao"
        >
          <h2>{{ editando.modo === 'evento' ? 'Novo evento' : editando.novo ? 'Novo bloco' : 'Editar bloco' }}</h2>
          <label>
            <span class="rotulo">O quê</span>
            <select v-model="editando.b.atividade" class="campo" aria-label="Atividade">
              <option v-if="!emModelo" value="">Outro: um evento com nome</option>
              <option v-for="a in cfg.atividades" :key="a.id" :value="a.id">{{ a.nome }}</option>
              <option v-if="editando.b.atividade && !porId[editando.b.atividade]" :value="editando.b.atividade">Atividade apagada</option>
            </select>
          </label>
          <label v-if="!editando.b.atividade">
            <span class="rotulo">Nome</span>
            <input v-model="editando.b.titulo" class="campo" maxlength="80" placeholder="Encontro com amigos" aria-label="Nome do evento" />
          </label>
          <label v-if="editando.modo === 'evento'">
            <span class="rotulo">Data</span>
            <input v-model="editando.data" type="date" class="campo" aria-label="Data" required />
          </label>
          <label v-else>
            <span class="rotulo">Dia</span>
            <select v-model.number="editando.b.dia" class="campo" aria-label="Dia">
              <option v-for="(nome, d) in DIAS" :key="d" :value="d">{{ nome }}<template v-if="numeros">, {{ numeros[d] }}</template></option>
            </select>
          </label>
          <label v-if="!emModelo" class="marcar-fixa">
            <input v-model="editando.semHorario" type="checkbox" /> Sem horário (tarefa do dia)
          </label>
          <div v-if="!editando.semHorario" class="horas">
            <label>
              <span class="rotulo">Começa</span>
              <input v-model="editando.b.inicio" type="time" step="300" class="campo" aria-label="Começa" />
            </label>
            <label>
              <span class="rotulo">Termina</span>
              <input v-model="editando.b.fim" type="time" step="300" class="campo" aria-label="Termina" />
            </label>
          </div>
          <label>
            <span class="rotulo">Nota</span>
            <input v-model="editando.b.nota" class="campo" maxlength="300" placeholder="Opcional" aria-label="Nota" />
          </label>
          <p v-if="erroDaEdicao" class="erro-texto">{{ erroDaEdicao }}</p>
          <div class="linha-acoes">
            <button class="botao primario" :disabled="!!erroDaEdicao || ocupado">{{ editando.modo === 'evento' ? 'Adicionar' : 'Salvar' }}</button>
            <button type="button" class="botao" @click="editando = null">Cancelar</button>
            <button v-if="!editando.novo" type="button" class="botao perigo apagar" :disabled="ocupado" @click="apagarDaFolha">Apagar</button>
          </div>
        </form>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.explicacao {
  margin-top: calc(-1 * var(--e2));
}

/* Resultado da voz */
.adicionados {
  position: relative;
  margin-bottom: 16px;
  padding-right: 44px;
}

.adicionados .fechar {
  position: absolute;
  top: 6px;
  right: 6px;
}

.titulo-adicionados {
  margin: 0 0 6px;
  font-weight: 650;
}

.adicionados ul {
  display: grid;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--texto);
  text-align: left;
  cursor: pointer;
}

.link:hover strong {
  color: var(--link);
}

.choque {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  color: var(--vermelho);
  font-size: 13.5px;
}

.choque svg,
.choque-dia svg {
  flex-shrink: 0;
  width: 17px;
  height: 17px;
}

.observacao {
  margin: 8px 0 0;
}

/* Navegar entre semanas: ‹ 16 – 22 de nov. › Esta semana, na barra de ferramentas */
.navegar {
  display: flex;
  align-items: center;
  gap: 2px;
}

.qual {
  min-width: 118px;
  font-size: 14px;
  font-weight: 600;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.navegar .etiqueta,
.navegar > .botao:last-child:not(.icone) {
  margin-left: 6px;
}

.vazio p {
  margin: 0 0 10px;
}

.vazio .botao {
  margin-top: 6px;
}

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
  margin: 0 0 var(--e4);
  padding: 4px;
  border: 1px solid var(--borda);
  border-radius: var(--raio);
}

.dia {
  position: relative;
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 6px;
  padding: 6px 0;
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
  font-size: 12px;
  font-weight: 600;
}

.dia .numero {
  font-size: 15px;
  font-variant-numeric: tabular-nums;
  color: var(--texto);
}

.dia.hoje .letra,
.dia.hoje .numero {
  color: var(--azul);
}

.dia.ativo {
  border-color: var(--borda-forte);
  background: var(--ativo);
  color: var(--texto);
}

.dia .ponto {
  position: absolute;
  top: 5px;
  right: 6px;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--texto-fraco);
}


.dia .alerta {
  position: absolute;
  top: -8px;
  right: -4px;
  display: grid;
  place-items: center;
  width: 13px;
  height: 13px;
  border: 2px solid var(--fundo);
  border-radius: 999px;
  background: var(--vermelho);
  color: var(--fundo);
  font-size: 10px;
  font-weight: 800;
  box-sizing: content-box;
}

/* Um dia */
.cabeca-dia {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e2) var(--e3);
  margin: 0 0 var(--e3);
}

.cabeca-dia h2 {
  margin: 0;
}

.nota-dia {
  flex: 1 1 220px;
  max-width: 380px;
  margin-left: auto;
}

.choque-dia {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.bloco {
  display: flex;
  align-items: stretch;
  gap: 2px;
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
  accent-color: var(--azul);
}

.corpo {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 9px 4px 9px 10px;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.bloco > .corpo:hover .nome {
  color: var(--link);
}

.horario {
  flex-shrink: 0;
  width: 88px;
  color: var(--texto-suave);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.horario.tarefa {
  color: var(--azul);
  font-size: 12.5px;
  font-weight: 600;
}

.texto {
  display: grid;
  flex: 1;
  min-width: 0;
}

.texto .nome {
  font-size: 14.5px;
  font-weight: 600;
  transition: color var(--transicao);
}

.texto .nota {
  overflow: hidden;
  color: var(--texto-suave);
  font-size: 13px;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.duracao {
  flex-shrink: 0;
  color: var(--texto-fraco);
  font-size: 12.5px;
}

.lixeira {
  flex-shrink: 0;
  align-self: center;
  color: var(--texto-fraco);
}

.lixeira:hover {
  color: var(--vermelho);
}

.lixeira svg {
  width: 17px;
  height: 17px;
}

.bloco.feito .nome {
  color: var(--texto-suave);
  text-decoration: line-through;
}

.bloco.conflito {
  background: var(--vermelho-fundo);
}

.livre {
  padding: 14px 0;
  border-top: 1px solid var(--borda);
  border-bottom: 1px solid var(--borda);
}

.sono {
  margin: 10px 0 0;
  color: var(--texto-fraco);
  font-size: 12.5px;
}

.sono.pouco {
  color: var(--vermelho);
}

.acoes-dia {
  margin-top: var(--e3);
}

/* Metas (painel ao lado): uma linha compacta por atividade */
.quadro .aviso {
  margin: 0 0 var(--e2);
}

.meta {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 0;
  font-size: 13.5px;
}

.meta .nome {
  overflow: hidden;
  font-weight: 500;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.tempo-meta {
  flex-shrink: 0;
  font-size: 12px;
}

.etiquetas {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.rodape-semana {
  margin-top: var(--e4);
}

/* Atividades: uma linha por atividade na tela larga (nome, cor, meta, prioridade, fixa) */
.atividade {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 20px;
  padding: 8px 0;
}

.linha-nome {
  display: flex;
  flex: 1 1 220px;
  max-width: 380px;
  align-items: center;
  gap: 10px;
}

.linha-campos {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 16px;
}

@media (max-width: 700px) {
  .linha-campos {
    padding-left: 20px;
  }
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

.apagar-atividade {
  margin-left: auto;
}

.numero-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.numero-meta .campo {
  width: 58px;
  padding: 0 8px;
}

.prioridade {
  width: auto;
}

.marcar-fixa {
  display: flex !important;
  align-items: center;
  gap: 6px;
  font-size: 13.5px;
  cursor: pointer;
}

.marcar-fixa input {
  width: 16px;
  height: 16px;
  accent-color: var(--azul);
}

/* Folha */
.veu {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 55%);
}

.folha {
  display: grid;
  gap: 12px;
  width: min(520px, 100%);
  max-height: 90vh;
  overflow-y: auto;
  padding: 20px 20px max(20px, env(safe-area-inset-bottom));
  background: var(--superficie);
  border: 1px solid var(--borda);
  border-radius: var(--raio-g) var(--raio-g) 0 0;
}

.folha h2 {
  margin: 0 0 4px;
}

.folha label {
  display: grid;
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
  font-size: 13.5px;
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
    font-size: 14px;
    font-weight: 600;
  }

  .horario .ate {
    color: var(--texto-fraco);
    font-size: 12.5px;
    font-weight: 400;
  }

  .duracao {
    display: none;
  }
}
</style>
