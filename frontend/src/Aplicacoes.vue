<script setup>
// Aplicações (H12): os apps da casa, pelos nomes .internal que o AdGuard resolve. Tocar abre o app; no
// celular, é daqui que se instala cada um como app (PWA) no endereço novo.
const APPS = [
  { nome: 'Biblioteca', endereco: 'https://biblioteca.internal', descricao: 'Os seus livros em PDF e EPUB, com as marcações.', icone: true },
  { nome: 'Acervo', endereco: 'https://acervo.internal', descricao: 'A base de conhecimento: Temas, Referências e resumos.', icone: true },
  { nome: 'Flashcards', endereco: 'https://flashcards.internal', descricao: 'Os baralhos e a revisão do dia.', icone: true },
  { nome: 'Streaming', endereco: 'https://streaming.internal', descricao: 'Canais, filmes e séries, com o celular de controle remoto.', icone: true },
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

    <section class="lista">
      <a v-for="app in APPS" :key="app.nome" class="cartao app" :href="app.endereco" target="_blank" rel="noopener">
        <img class="icone-app" :src="`${app.endereco}/icon-192.png`" alt="" width="44" height="44" />
        <span class="texto">
          <strong>{{ app.nome }}</strong>
          <span class="suave">{{ app.descricao }}</span>
          <span class="endereco">{{ curto(app.endereco) }}</span>
        </span>
      </a>
    </section>

    <h2>Servidor</h2>
    <section class="cartao servidor">
      <a v-for="s in SERVIDOR" :key="s.nome" class="linha" :href="s.endereco" target="_blank" rel="noopener">
        <strong>{{ s.nome }}</strong>
        <span class="suave">{{ s.descricao }}</span>
      </a>
    </section>

    <h2>Instalar no celular</h2>
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
  font-size: 0.88rem;
}

h2 {
  margin: 26px 0 10px;
  font-size: 1.05rem;
}

.lista {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}

.app {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  color: inherit;
  text-decoration: none;
  transition: border-color 0.15s ease;
}

.app:hover {
  border-color: var(--marca);
}

.icone-app {
  flex: none;
  border-radius: 11px;
}

.texto {
  display: grid;
  gap: 3px;
  min-width: 0;
  font-size: 0.88rem;
  line-height: 1.4;
}

.texto strong {
  font-size: 1rem;
}

.endereco {
  color: var(--link);
  font-size: 0.8rem;
}

.servidor {
  display: grid;
  padding: 4px 16px;
}

.linha {
  display: grid;
  gap: 2px;
  padding: 10px 0;
  border-bottom: 1px solid var(--linha);
  color: inherit;
  font-size: 0.88rem;
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
  font-size: 0.9rem;
  line-height: 1.6;
}

.pequeno {
  margin: 6px 0 0;
  font-size: 0.82rem;
  line-height: 1.5;
}
</style>
