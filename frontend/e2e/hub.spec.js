import { expect, test } from '@playwright/test'

// Um fluxo só, na ordem em que o Lucas usa: ver o disco → limpar → agendar → servidor → Ajustes.
test.describe.configure({ mode: 'serial' })

test('mostra o disco da VM, cada app e o cache do Docker', async ({ page }) => {
  await page.goto('/')
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

test('servidor mostra a VM, os contêineres e como ligar o Proxmox', async ({ page }) => {
  await page.goto('/#/servidor')
  await expect(page.getByRole('heading', { name: 'Servidor' })).toBeVisible()
  await expect(page.getByText('pveum acl modify / --users hub@pve --roles PVEAuditor')).toBeVisible()
  await expect(page.getByRole('meter', { name: 'Memória' })).toHaveAttribute('aria-valuenow', '19')
  await expect(page.getByText('acervo-acervo-1')).toBeVisible()
  await expect(page.getByText('Up 5 hours')).toBeVisible()
})

test('anota uma sugestão em Ajustes', async ({ page }) => {
  await page.goto('/#/servidor')
  await page.getByRole('button', { name: 'Ajustes' }).click()
  await page.getByLabel('Sugestão de melhoria').fill('Mostrar a temperatura do processador')
  await page.getByRole('button', { name: 'Enviar sugestão' }).click()
  await expect(page.getByText('Mostrar a temperatura do processador')).toBeVisible()
  await expect(page.getByText('Nova', { exact: true })).toBeVisible()
})
