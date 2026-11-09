import os from 'node:os'
import path from 'node:path'
import { defineConfig } from '@playwright/test'

const remoto = process.env.E2E_BASE_URL
// O teste local roda num servidor e num banco só dele, com pastas de apps inventadas.
const dados = path.join(os.tmpdir(), 'hub-e2e')
const testdata = path.resolve('../backend/testdata')

export default defineConfig({
  testDir: './e2e',
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: remoto || 'http://localhost:5189',
    ignoreHTTPSErrors: true,
  },
  webServer: remoto
    ? undefined
    : [
        {
          // As pastas dos apps: a Biblioteca com um livro e o banco, o Acervo com o histórico do Claude.
          command: [
            `rm -rf "${dados}"`,
            `mkdir -p "${dados}/apps/biblioteca/books" "${dados}/apps/acervo/home/.claude/projects"`,
            `head -c 3000000 /dev/zero > "${dados}/apps/biblioteca/books/livro.epub"`,
            `head -c 200000 /dev/zero > "${dados}/apps/biblioteca/biblioteca.db"`,
            `head -c 900000 /dev/zero > "${dados}/apps/acervo/home/.claude/projects/a.jsonl"`,
            'go run .',
          ].join(' && '),
          cwd: '../backend',
          // Docker falso; sem token do Proxmox, o painel mostra como ligar.
          env: {
            DATA_DIR: path.join(dados, 'hub'),
            APPS_DIR: path.join(dados, 'apps'),
            PROC_DIR: path.join(testdata, 'proc'),
            DOCKER_URL: 'http://127.0.0.1:8097',
            ADDR: ':8096',
          },
          url: 'http://localhost:8096/api/health',
        },
        {
          command: `python3 "${path.join(testdata, 'docker-falso.py')}" 8097`,
          port: 8097,
        },
        {
          // A versão de produção, com service worker, como no servidor.
          command: 'npm run build && npx vite preview --port 5189 --strictPort',
          env: { API_URL: 'http://localhost:8096' },
          url: 'http://localhost:5189',
          timeout: 180000,
        },
      ],
})
