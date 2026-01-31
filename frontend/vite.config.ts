import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Strip the crossorigin attribute from HTML tags. Vite adds it by default
// for ES modules, but it causes CORS enforcement issues when serving
// same-origin assets through Cloudflare (Chrome 144+ rejects stylesheets
// loaded in CORS mode without Access-Control-Allow-Origin headers).
function stripCrossorigin(): Plugin {
  return {
    name: 'strip-crossorigin',
    transformIndexHtml(html) {
      return html.replace(/ crossorigin/g, '')
    },
  }
}

export default defineConfig({
  plugins: [vue(), tailwindcss(), stripCrossorigin()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
