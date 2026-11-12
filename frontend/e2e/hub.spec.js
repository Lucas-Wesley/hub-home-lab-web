import { expect, test } from '@playwright/test'

// Um fluxo só, na ordem em que o Lucas usa: ver o disco → limpar → agendar → servidor → Ajustes.
test.describe.configure({ mode: 'serial' })

test('abre no Início: a página é o campo, com texto, checklist, lista e toggle, e salva sozinha', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Início' })).toHaveAttribute('aria-current', 'page')
  const folha = page.getByRole('textbox', { name: 'Início' })
  await expect(folha).toHaveAttribute('contenteditable', 'true')
  await folha.click()

  // Texto, depois os atalhos do Notion: "[] " checklist e "- " lista.
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
  await expect(folha.locator('li[data-checked="true"]', { hasText: 'Pão' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Abrir o toggle' })).toBeVisible()
  await expect(folha.getByText('Passagem')).toBeHidden()
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
