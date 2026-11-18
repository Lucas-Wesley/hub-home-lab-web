import { expect, test } from '@playwright/test'

// Um fluxo só, na ordem em que o Lucas usa: ver o disco → limpar → agendar → servidor → Ajustes.
test.describe.configure({ mode: 'serial' })

test('abre no Início: a página é o campo, com texto, checklist, lista e toggle, e salva sozinha', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Início' })).toHaveAttribute('aria-current', 'page')
  const folha = page.getByRole('textbox', { name: 'Início' })
  await expect(folha).toHaveAttribute('contenteditable', 'true')
  await folha.click()

  // Título ("# "), texto, depois os atalhos do Notion: "[] " checklist e "- " lista.
  await page.keyboard.type('# Casa')
  await page.keyboard.press('Enter')
  await page.keyboard.type('Compras da semana')
  await page.keyboard.press('Enter')
  await page.keyboard.type('[] Pão')
  await page.keyboard.press('Enter')
  await page.keyboard.type('Café')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')
  await page.keyboard.type('- Ideia solta')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')

  // "> " cria o toggle; Enter no título vai para dentro, onde cabe uma lista.
  await page.keyboard.type('> Viagem')
  await page.keyboard.press('Enter')
  await page.keyboard.type('- Passagem')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')

  // O menu do "/" (para o celular, onde os atalhos podem falhar).
  await page.keyboard.type('/check')
  await expect(page.getByRole('option', { name: /Checklist/ })).toBeVisible()
  await page.keyboard.press('Enter')
  await page.keyboard.type('Reservar hotel')

  // Título também pelo menu do "/".
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')
  await page.keyboard.type('/tit')
  await page.keyboard.press('Enter')
  await page.keyboard.type('Trabalho')
  await expect(folha.getByRole('heading', { name: 'Casa' })).toBeVisible()
  await expect(folha.getByRole('heading', { name: 'Trabalho' })).toBeVisible()
  await expect(folha.locator('[data-type="details"]').getByRole('heading')).toHaveCount(0)

  const tarefas = folha.locator('li[data-checked]')
  await expect(tarefas).toHaveCount(3)
  await tarefas.first().getByRole('checkbox').check()
  await expect(tarefas.first().locator('p')).toHaveCSS('text-decoration-line', 'line-through')
  await expect(folha.locator('ul:not([data-type]) li', { hasText: 'Ideia solta' })).toBeVisible()
  const toggle = folha.locator('[data-type="details"]')
  await expect(toggle.locator('summary')).toHaveText('Viagem')
  await expect(toggle.locator('[data-type="detailsContent"] li', { hasText: 'Passagem' })).toBeVisible()
  await expect(toggle.locator('[data-type="detailsContent"] li[data-checked]', { hasText: 'Reservar hotel' })).toBeVisible()

  // Fechar o toggle esconde o que está dentro.
  await page.getByRole('button', { name: 'Fechar o toggle' }).click()
  await expect(toggle.getByText('Passagem')).toBeHidden()

  // Salva sozinho: depois de recarregar, está tudo lá, com o toggle fechado e o Pão riscado.
  await page.waitForResponse((r) => r.url().endsWith('/api/inicio') && r.request().method() === 'PUT')
  await page.waitForTimeout(1000)
  await page.reload()
  await expect(folha.getByText('Compras da semana')).toBeVisible()
  await expect(folha.getByRole('heading', { name: 'Trabalho' })).toBeVisible()
  await expect(folha.locator('li[data-checked="true"]', { hasText: 'Pão' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Abrir o toggle' })).toBeVisible()
  await expect(folha.getByText('Passagem')).toBeHidden()
})

test('Rotina: planeja a semana pelo modelo, remaneja um bloco, marca feito e vê as metas', async ({ page }) => {
  await page.goto('/#/rotina')
  await expect(page.getByRole('heading', { name: 'Rotina', level: 1 })).toBeVisible()

  // A semana ainda não existe: nasce do modelo.
  await expect(page.getByText('Esta semana ainda não foi planejada.')).toBeVisible()
  await page.getByRole('button', { name: 'Planejar a partir do modelo' }).click()
  await expect(page.getByRole('heading', { name: 'Metas da semana' })).toBeVisible()

  // Segunda: inglês às 7h, marcado como feito.
  await page.getByRole('tab', { name: 'Segunda' }).click()
  await page.getByRole('checkbox', { name: 'Inglês feito' }).check()
  await expect(page.locator('.meta', { hasText: 'Inglês' }).getByText('1 feita')).toBeVisible()

  // Remanejar: a programação da segunda vai para domingo.
  await page.locator('.bloco', { hasText: 'Programação' }).locator('.corpo').click()
  const folha = page.getByRole('form', { name: 'Editar bloco' })
  await folha.getByLabel('Dia', { exact: true }).selectOption({ index: 6 })
  await folha.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByRole('heading', { name: /^Domingo/ })).toBeVisible()
  await expect(page.locator('.bloco', { hasText: 'Programação' })).toBeVisible()

  // Nota do dia e dia livre: sexta é feriado.
  await page.getByRole('tab', { name: 'Sexta' }).click()
  await page.getByLabel('Nota do dia').fill('Feriado')
  await page.getByLabel('Nota do dia').press('Enter')
  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Limpar o dia' }).click()
  await expect(page.getByText('Dia livre.')).toBeVisible()
  await expect(page.locator('.meta', { hasText: 'Programação' }).getByText('3 de 3–4')).toBeVisible()

  // Depois de recarregar, tudo continua.
  await page.reload()
  await page.getByRole('tab', { name: 'Sexta' }).click()
  await expect(page.getByLabel('Nota do dia')).toHaveValue('Feriado')

  // A grade da semana inteira e o modelo.
  await page.getByRole('radio', { name: 'Semana inteira' }).click()
  await expect(page.locator('.pedaco').first()).toBeVisible()
  await page.getByRole('tab', { name: 'Modelo' }).click()
  await expect(page.getByRole('heading', { name: 'Metas do modelo' })).toBeVisible()
  await page.getByRole('tab', { name: 'Atividades e metas' }).click()
  await expect(page.getByLabel('Nome da atividade').first()).toHaveValue('Inglês')
})

test('Rotina: adiciona pela fala sem tirar nada, avisa o choque com "!", apaga pela lixeira e não cobra o futebol', async ({ page }) => {
  await page.goto('/#/rotina')
  // O futebol é fixo: mesmo fora da semana, não entra no aviso de falta.
  await page.getByRole('tab', { name: 'Modelo' }).click()
  await page.getByRole('tab', { name: 'Terça' }).click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Limpar o dia' }).click()
  await expect(page.locator('.aviso.erro', { hasText: 'Faltam' })).toBeVisible()
  await expect(page.locator('.aviso.erro', { hasText: 'Futebol' })).toHaveCount(0)
  await expect(page.locator('.meta', { hasText: 'Futebol' }).getByText('0 de 1× · fixa')).toBeVisible()

  // A fala vira um evento na próxima quinta, das 7h às 10h: choca com o inglês e o trabalho.
  await page.getByRole('tab', { name: 'Semana', exact: true }).click()
  await page.getByLabel('Adicionar à rotina').fill('quinta das 7 às 10 encontro com amigos')
  await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
  const resultado = page.getByRole('status').filter({ hasText: 'Adicionado' })
  await expect(resultado).toContainText('Encontro com amigos')
  await expect(resultado).toContainText('No mesmo horário de Inglês, Trabalho')
  await expect(page.getByRole('heading', { name: /^Quinta/ })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Quinta, com choque de horário' })).toBeVisible()
  await expect(page.locator('.bloco.conflito', { hasText: 'Encontro com amigos' })).toBeVisible()

  // A lixeira resolve: some o evento e o "!".
  await page.getByRole('button', { name: 'Apagar Encontro com amigos' }).click()
  await expect(page.locator('.bloco', { hasText: 'Encontro com amigos' })).toHaveCount(0)
  await expect(page.getByRole('tab', { name: 'Quinta', exact: true })).toBeVisible()
})

test('Rotina: mês com evento de data marcada e semanas planejadas pelo modelo', async ({ page }) => {
  await page.goto('/#/rotina')
  await page.getByRole('tab', { name: 'Mês' }).click()
  await page.getByRole('button', { name: 'Próximo mês' }).click()
  await page.getByRole('button', { name: '+ Evento com data' }).click()
  const folha = page.getByRole('form', { name: 'Novo evento' })
  await folha.getByLabel('Nome do evento').fill('Dentista')
  const data = await folha.getByLabel('Data').inputValue()
  await folha.getByLabel('Data').fill(data.slice(0, 8) + '10')
  await folha.getByLabel('Começa').fill('16:00')
  await folha.getByLabel('Termina').fill('17:00')
  await folha.getByRole('button', { name: 'Adicionar' }).click()
  await expect(page.locator('.celula .evento', { hasText: 'Dentista' })).toBeVisible()

  const planejar = page.getByRole('button', { name: /Planejar .*sem plano/ })
  await planejar.click()
  await expect(planejar).toHaveCount(0)

  // Tocar no dia abre a semana nele.
  await page.locator('.celula', { has: page.locator('.evento', { hasText: 'Dentista' }) }).click()
  await expect(page.locator('.bloco', { hasText: 'Dentista' })).toBeVisible()
})

test('Rotina: sugestão da IA antes de salvar, refeita com comentário e usada como modelo', async ({ page }) => {
  await page.goto('/#/rotina')
  await page.getByRole('tab', { name: 'Atividades e metas' }).click()
  await page.getByRole('button', { name: 'Ver a semana sugerida' }).click()
  await expect(page.getByText('Inglês todo dia e programação 1x.')).toBeVisible()
  await expect(page.locator('.sugestao .pedaco')).toHaveCount(8)

  await page.getByPlaceholder(/prioriza mais programação/).fill('prioriza mais programação')
  await page.getByRole('button', { name: 'Refazer com o comentário' }).click()
  await expect(page.getByText('Inglês todo dia e programação 3x.')).toBeVisible()

  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Usar como modelo' }).click()
  await expect(page.getByRole('heading', { name: 'Metas do modelo' })).toBeVisible()
  await expect(page.locator('.meta', { hasText: 'Programação' }).getByText('3 de 3–4×')).toBeVisible()
})

test('Aplicações tem o link de cada app pelo nome', async ({ page }) => {
  await page.goto('/#/aplicacoes')
  await expect(page.getByRole('heading', { name: 'Aplicações' })).toBeVisible()
  await expect(page.getByRole('link', { name: /Flashcards/ })).toHaveAttribute('href', 'https://flashcards.internal')
  await expect(page.getByRole('link', { name: /Biblioteca/ })).toHaveAttribute('href', 'https://biblioteca.internal')
})

test('mostra o disco da VM, cada app e o cache do Docker', async ({ page }) => {
  await page.goto('/#/armazenamento')
  await expect(page.getByRole('heading', { name: 'Armazenamento' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Armazenamento' })).toHaveAttribute('aria-current', 'page')

  // O aviso do cache leva à Limpeza.
  await expect(page.getByText('O Docker guarda 14 GiB que podem ser liberados')).toBeVisible()
  await expect(page.getByText('O Proxmox ainda não está ligado ao painel.')).toBeVisible()

  const legenda = page.locator('.legenda')
  await expect(legenda.getByText('Cache de build do Docker')).toBeVisible()
  await expect(legenda.getByText('Livre')).toBeVisible()

  // A Biblioteca (3 MiB) vem antes do Acervo; abrir mostra o que há dentro.
  const apps = page.locator('.por-app')
  await expect(apps.first()).toContainText('Biblioteca Digital')
  await apps.first().getByRole('button').click()
  await expect(apps.first().getByText('Livros')).toBeVisible()
  await expect(apps.first().getByText('Banco de dados')).toBeVisible()
  await apps.nth(1).getByRole('button').click()
  await expect(apps.nth(1).getByText('Claude: histórico e configuração')).toBeVisible()
})

test('limpa agora e grava no histórico', async ({ page }) => {
  await page.goto('/#/limpeza')
  await expect(page.getByText('Nenhuma limpeza ainda.')).toBeVisible()
  await page.getByRole('button', { name: 'Limpar agora' }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Liberados 14 GiB.' })).toBeVisible()
  await expect(page.locator('.lista .linha').first()).toContainText('Manual')
  await expect(page.locator('.lista .linha').first()).toContainText('14 GiB liberados')

  // Depois de limpar, o aviso some do Armazenamento.
  await page.goto('/#/armazenamento')
  await expect(page.getByRole('heading', { name: 'Armazenamento' })).toBeVisible()
  await expect(page.locator('.legenda')).toBeVisible()
  await expect(page.getByText('podem ser liberados')).toHaveCount(0)
})

test('agenda a limpeza a cada 2 dias às 3h', async ({ page }) => {
  await page.goto('/#/limpeza')
  // Padrão: toda semana às 4h, ligada.
  await expect(page.getByRole('radio', { name: 'Toda semana' })).toHaveAttribute('aria-checked', 'true')
  await expect(page.getByText(/Próxima limpeza: .*04:00/)).toBeVisible()

  await page.getByRole('radio', { name: 'A cada 2 dias' }).click()
  await page.getByLabel('Hora da limpeza').selectOption('3')
  await page.getByLabel('Imagens soltas').uncheck()
  await page.getByRole('button', { name: 'Salvar agenda' }).click()
  await expect(page.getByRole('status').filter({ hasText: /Agenda salva\. Próxima limpeza: .*03:00/ })).toBeVisible()

  await page.reload()
  await expect(page.getByRole('radio', { name: 'A cada 2 dias' })).toHaveAttribute('aria-checked', 'true')
  await expect(page.getByLabel('Hora da limpeza')).toHaveValue('3')
  await expect(page.getByLabel('Imagens soltas')).not.toBeChecked()

  await page.getByLabel('Limpar automaticamente').uncheck()
  await page.getByRole('button', { name: 'Salvar agenda' }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Limpeza automática desligada.' })).toBeVisible()
  await expect(page.getByText('Próxima limpeza')).toHaveCount(0)
})

test('servidor mostra a VM, a memória de cada app, os contêineres e como ligar o Proxmox', async ({ page }) => {
  await page.goto('/#/servidor')
  await expect(page.getByRole('heading', { name: 'Servidor' })).toBeVisible()
  await expect(page.getByText('pveum acl modify / --users hub@pve --roles PVEAuditor')).toBeVisible()
  await expect(page.getByRole('meter', { name: 'Memória' })).toHaveAttribute('aria-valuenow', '19')
  await expect(page.getByText('Up 5 hours')).toBeVisible()

  // Do maior para o menor; o cache de arquivos inativo não conta. O resto do "em uso" é o sistema.
  const apps = page.locator('.consumo')
  await expect(apps.first()).toContainText('Whisper (transcrição)')
  await expect(apps.first()).toContainText('1 GiB')
  await expect(apps.nth(1)).toContainText('Acervo')
  await expect(apps.nth(1)).toContainText('144 MiB')
  await expect(apps.nth(2)).toContainText('Caddy (proxy)')
  await expect(apps.last()).toContainText('Sistema e o resto')
  await expect(page.getByText(/Os apps usam 1,2 GiB dos/)).toBeVisible()
})

test('anota uma sugestão em Ajustes', async ({ page }) => {
  await page.goto('/#/servidor')
  await page.getByRole('button', { name: 'Ajustes' }).click()
  await page.getByRole('link', { name: /Sugestões de melhoria/ }).click()
  await page.getByLabel('Sugestão de melhoria').fill('Mostrar a temperatura do processador')
  await page.getByRole('button', { name: 'Enviar sugestão' }).click()
  await expect(page.getByText('Mostrar a temperatura do processador')).toBeVisible()
  await expect(page.getByText('Nova', { exact: true })).toBeVisible()
})

test('Claude mostra onde deveria estar na janela e quando acaba no ritmo atual', async ({ page }) => {
  // Medido 1 dia e 21 h depois de a semana começar (27% do tempo), com 57% usados: passou do ritmo.
  const medido = new Date('2026-11-18T16:52:00Z')
  const conta = (id, nome, ativa, cinco, semana) => ({
    id, nome, ativa, configurada: true, medido_em: medido.toISOString(),
    gravado_em: '2026-11-10T12:00:00Z', vence_em: '2027-11-10T12:00:00Z',
    uso: { esgotada: false, cinco_horas: { uso: cinco, zera_em: '2026-11-18T17:00:00Z' }, semana: { uso: semana, zera_em: '2026-11-23T20:00:00Z' } },
  })
  await page.route('**/api/claude?medir=1', (r) =>
    r.fulfill({ json: { contas: [conta('pessoal', 'Pessoal', true, 0.81, 0.57), conta('empresa', 'Empresa', false, 0.0, 0.1)] } }),
  )
  await page.goto('/#/claude')
  const pessoal = page.locator('.conta', { hasText: 'Pessoal' })
  await expect(pessoal.locator('.ideal').nth(1)).toHaveAttribute('title', 'Ideal agora: 27%')
  await expect(pessoal.getByText(/Acima do ritmo: o ideal agora seria 27%\. Assim acaba qui\., 19 de nov\., 2\d:\d\d, 4 dias antes de zerar/)).toBeVisible()
  await expect(page.locator('.conta', { hasText: 'Empresa' }).getByText(/Dentro do ritmo: o ideal agora seria 27%\. Assim termina em ~37%/)).toBeVisible()
  await page.screenshot({ path: process.env.SHOT || '/dev/null', fullPage: true })
})
