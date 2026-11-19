<script setup>
import { ref } from 'vue'
import Icone from './Icone.vue'
import Medidor from './Medidor.vue'

// Vitrine viva do design system (docs/DESIGN-SYSTEM.md). Fica só no endereço #/design, sem aba.
// O Hub é a fonte: as outras apps copiam daqui. Mudou o design? Atualize o documento e esta tela.
const NEUTROS = ['fundo', 'superficie', 'superficie-2', 'ativo', 'borda', 'borda-forte', 'texto', 'texto-suave', 'texto-fraco', 'primario', 'link']
const ESTADOS = ['azul', 'verde', 'vermelho', 'ambar', 'cinza']
const ATIVIDADES = ['azul', 'violeta', 'verde', 'laranja', 'amarelo', 'rosa', 'turquesa', 'cinza']
const aba = ref('um')
const visao = ref('dia')
</script>

<template>
  <div class="pagina vitrine">
    <div class="topo-pagina">
      <h1>Design system</h1>
      <div class="meio busca">
        <Icone nome="lupa" />
        <input class="campo" placeholder="A ferramenta principal fica na linha do título" aria-label="Exemplo de busca" />
      </div>
      <div class="acoes">
        <button class="botao discreto icone" aria-label="Atualizar"><Icone nome="atualizar" /></button>
        <button class="botao primario">Ação principal</button>
      </div>
    </div>
    <p class="explicacao">
      A referência viva do <code>docs/DESIGN-SYSTEM.md</code>. O Hub é a fonte; as outras apps copiam os tokens e as classes do <code>style.css</code> daqui.
    </p>

    <!-- Espaço -->
    <div class="ferramentas">
      <div class="abas" role="tablist" aria-label="Exemplo de abas">
        <button v-for="[id, nome] in [['um', 'Semana'], ['dois', 'Mês'], ['tres', 'Modelo']]" :key="id" role="tab" :class="{ ativa: aba === id }" @click="aba = id">{{ nome }}</button>
      </div>
      <div class="direita">
        <button class="botao icone discreto" aria-label="Anterior"><Icone nome="voltar" /></button>
        <strong>16 – 22 de nov.</strong>
        <button class="botao icone discreto" aria-label="Próxima"><Icone nome="avancar" /></button>
        <div class="abas">
          <button :class="{ ativa: visao === 'dia' }" @click="visao = 'dia'">Dia</button>
          <button :class="{ ativa: visao === 'semana' }" @click="visao = 'semana'">Semana inteira</button>
        </div>
      </div>
    </div>

    <div class="numeros-quadro">
      <div v-for="[r, v, d] in [['Processador', '3%', 'de 12 threads'], ['Memória', '61%', '9,5 GiB de 15 GiB'], ['Disco', '9%', '6,3 GiB de 68 GiB'], ['Máquinas', '1', 'de 1 ligada']]" :key="r" class="numero-quadro">
        <p class="rotulo-secao">{{ r }}</p>
        <span class="valor-grande">{{ v }}</span>
        <span class="pequeno">{{ d }}</span>
      </div>
    </div>

    <div class="duas-colunas">
      <div class="principal">
        <div class="quadros">
          <section class="quadro">
            <div class="quadro-topo">
              <h2 class="rotulo-secao">Quadro com lista</h2>
              <a href="#/design">Ver tudo</a>
            </div>
            <ul class="lista">
              <li v-for="n in ['Inglês', 'Programação', 'Muay Thai']" :key="n" class="linha-exemplo">
                <span>{{ n }}</span><span class="etiqueta verde">ok</span>
              </li>
            </ul>
          </section>
          <section class="quadro">
            <div class="quadro-topo"><h2 class="rotulo-secao">Medidor de uso</h2></div>
            <div class="lista">
              <Medidor nome="Memória" :usado="6.1e9" :total="8e9" />
              <Medidor nome="Disco" :usado="7.4e10" :total="8e10" />
            </div>
          </section>
        </div>
      </div>
      <aside class="lateral">
        <section class="quadro">
          <div class="quadro-topo"><h2 class="rotulo-secao">Painel lateral</h2><span class="pequeno">4 / 34</span></div>
          <div class="progresso"><span style="width: 42%" /></div>
          <p class="pequeno nota-lateral">Resumo, progresso, metas: o que acompanha o conteúdo principal.</p>
        </section>
      </aside>
    </div>

    <!-- Cor -->
    <h2 class="secao">Cor</h2>
    <div class="amostras">
      <div v-for="t in NEUTROS" :key="t" class="amostra">
        <span class="cor" :style="{ background: `var(--${t})` }" />
        <code>--{{ t }}</code>
      </div>
    </div>
    <div class="linha estados">
      <span v-for="e in ESTADOS" :key="e" class="etiqueta" :class="e">{{ e }}</span>
    </div>
    <p class="rotulo-secao subtitulo">Tokens próprios do Hub (Rotina)</p>
    <div class="linha">
      <span v-for="c in ATIVIDADES" :key="c" class="atividade" :class="`cor-${c}`">{{ c }}</span>
    </div>

    <!-- Tipografia -->
    <h2 class="secao">Tipografia</h2>
    <div class="tipos">
      <h1>Título de página · 26</h1>
      <h2>Seção · 20</h2>
      <h3>Subseção · 16</h3>
      <p class="rotulo-secao">Rótulo de seção · 12, caixa alta</p>
      <p class="lead">Destaque · 15, em texto suave.</p>
      <p>Interface · 14. O texto de botões, listas e campos.</p>
      <p class="pequeno">Pequeno · 12,5. Rótulos, datas e valores secundários.</p>
      <p><code>código · mono 13,5</code></p>
    </div>

    <!-- Componentes -->
    <h2 class="secao">Componentes</h2>
    <div class="linha">
      <button class="botao primario">Principal</button>
      <button class="botao">Secundário</button>
      <button class="botao discreto">Discreto</button>
      <button class="botao perigo">Perigo</button>
      <button class="botao icone" aria-label="Ícone"><Icone nome="mais" /></button>
      <span class="botao-dividido"><button class="botao">Copiar</button><button class="botao" aria-label="Mais"><Icone nome="seta-baixo" /></button></span>
    </div>
    <div class="campos exemplo-campos">
      <label><span class="rotulo">Campo</span><input class="campo" placeholder="Escreva aqui" /></label>
      <label><span class="rotulo">Seleção</span><select class="campo"><option>Uma opção</option></select></label>
      <label class="busca"><Icone nome="lupa" /><input class="campo" placeholder="Buscar" aria-label="Buscar" /><kbd>Ctrl K</kbd></label>
    </div>
    <div class="pilha avisos">
      <p class="aviso">Aviso de erro: o que deu errado e o que fazer.</p>
      <p class="aviso neutro">Aviso neutro: uma informação que não é erro.</p>
    </div>
    <div class="grade">
      <a class="cartao" href="#/design"><h3>Cartão</h3><p>Título 16/600 e descrição em duas linhas, no máximo.</p></a>
      <div class="cartao cheio"><h3>Cartão cheio</h3><p>Com fundo de superfície.</p></div>
    </div>
  </div>
</template>

<style scoped>
.linha-exemplo {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}

.nota-lateral {
  margin: var(--e2) 0 0;
}

.amostras {
  display: grid;
  gap: var(--e2);
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  margin-bottom: var(--e3);
}

.amostra {
  display: flex;
  align-items: center;
  gap: var(--e2);
  font-size: 12.5px;
}

.cor {
  flex: none;
  width: 28px;
  height: 28px;
  border: 1px solid var(--borda-forte);
  border-radius: var(--raio-p);
}

.estados {
  margin-bottom: var(--e4);
}

.subtitulo {
  margin-bottom: var(--e2);
}

.atividade {
  padding: 2px 8px;
  border-left: 3px solid var(--cor);
  border-radius: var(--raio-e);
  background: color-mix(in srgb, var(--cor) 18%, var(--fundo));
  font-size: 12.5px;
}

.tipos {
  display: grid;
  gap: var(--e2);
}

.tipos p {
  margin: 0;
}

.exemplo-campos {
  margin: var(--e4) 0;
}

.exemplo-campos .busca {
  align-self: end;
}

.avisos {
  gap: var(--e2);
  margin-bottom: var(--e4);
}

.avisos .aviso {
  margin: 0;
}
</style>
