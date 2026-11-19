<script setup>
// Aplicações (H12): os apps da casa, pelos nomes .internal que o AdGuard resolve. Tocar abre o app; no
// celular, é daqui que se instala cada um como app (PWA) no endereço novo.
const APPS = [
  { nome: 'Biblioteca', endereco: 'https://biblioteca.internal', descricao: 'Os seus livros em PDF e EPUB, com as marcações.', icone: true },
  { nome: 'Acervo', endereco: 'https://acervo.internal', descricao: 'A base de conhecimento: Temas, Referências e resumos.', icone: true },
  { nome: 'Flashcards', endereco: 'https://flashcards.internal', descricao: 'Os baralhos e a revisão do dia.', icone: true },
  { nome: 'Streaming', endereco: 'https://streaming.internal', descricao: 'Canais, filmes e séries, com o celular de controle remoto.', icone: true },
  { nome: 'Trilha de Estudo', endereco: 'https://trilha.internal', descricao: 'Trilhas de estudo em ordem, com lições, meta e progresso.', icone: true },
]

const SERVIDOR = [
  { nome: 'AdGuard', endereco: 'https://adguard.internal', descricao: 'O painel completo do DNS da casa.' },
  { nome: 'ntfy', endereco: 'https://ntfy.internal', descricao: 'Os avisos. O app do celular continua em 192.168.1.20:8451.' },
  { nome: 'Prowlarr', endereco: 'https://prowlarr.internal', descricao: 'Os indexadores de torrents do Streaming.' },
  { nome: 'Proxmox', endereco: 'https://192.168.1.10:8006', descricao: 'O servidor físico e as VMs.' },
]

const curto = (endereco) => endereco.replace('https://', '')
</script>

<template>
  <div class="pagina">
    <h1>Aplicações</h1>
    <p class="suave explicacao">Toque para abrir. Os endereços antigos, com o IP e a porta, continuam valendo.</p>

    <section class="grade">
      <a v-for="app in APPS" :key="app.nome" class="cartao app" :href="app.endereco" target="_blank" rel="noopener">
        <img class="icone-app" :src="`${app.endereco}/icon-192.png`" alt="" width="44" height="44" />
        <span class="texto">
          <strong>{{ app.nome }}</strong>
          <span class="suave">{{ app.descricao }}</span>
          <span class="endereco">{{ curto(app.endereco) }}</span>
        </span>
      </a>
    </section>

    <h2 class="secao">Servidor</h2>
    <section class="cartao servidor">
      <a v-for="s in SERVIDOR" :key="s.nome" class="linha" :href="s.endereco" target="_blank" rel="noopener">
        <strong>{{ s.nome }}</strong>
        <span class="suave">{{ s.descricao }}</span>
      </a>
    </section>

    <h2 class="secao">Instalar no celular</h2>
    <section class="cartao instalar">
      <ol>
        <li>Abra o app pelo Chrome, tocando nele aqui.</li>
        <li>Menu <strong>⋮</strong> → <strong>Instalar app</strong> (ou <strong>Adicionar à tela inicial</strong>).</li>
        <li>Desinstale o app antigo, o do endereço com IP. Para o celular, são dois apps diferentes.</li>
      </ol>
      <p class="suave pequeno">
        Na Biblioteca, os livros guardados para ler sem internet ficam no app antigo: baixe-os de novo no novo. O progresso e as marcações estão
        no servidor e aparecem sozinhos.
      </p>
      <p class="suave pequeno">
        Se um nome não abrir, o celular está ignorando o DNS da casa: desligue o <strong>DNS privado</strong> (Ajustes → Conexões → Mais
        configurações) ou use o endereço com IP.
      </p>
    </section>
  </div>
</template>

<style scoped>
.explicacao {
  margin: 0 0 14px;
  font-size: 14px;
}

h2.secao {
  margin: var(--e6) 0 var(--e3);
}

.app {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.icone-app {
  flex: none;
  border-radius: 11px;
}

.texto {
  display: grid;
  gap: 3px;
  min-width: 0;
  font-size: 14px;
  line-height: 1.4;
}

.texto strong {
  font-size: 16px;
}

.endereco {
  color: var(--link);
  font-size: 12.5px;
}

.servidor {
  display: grid;
  padding: 4px 16px;
}

.linha {
  display: grid;
  gap: 2px;
  padding: 10px 0;
  border-bottom: 1px solid var(--borda);
  color: inherit;
  font-size: 14px;
  text-decoration: none;
}

.linha:last-child {
  border-bottom: 0;
}

.linha:hover strong {
  color: var(--link);
}

.instalar ol {
  margin: 0 0 10px;
  padding-left: 20px;
  font-size: 14px;
  line-height: 1.6;
}

.pequeno {
  margin: 6px 0 0;
  line-height: 1.5;
}
</style>
