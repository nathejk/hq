import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import type { UserConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { PrimeVueResolver } from '@primevue/auto-import-resolver'

// Standalone from vite.config.ts on purpose: this file exists to run tests, and the dev server's
// proxy, host and port have nothing to do with that.
//
// # Two kinds of test live here
//
// Most specs cover plain reactive logic and need no DOM. A few mount a component, and those need the
// Vue SFC plugin, a DOM, and PrimeVue — because the bug that made component tests worth having was a
// PrimeVue behaviour (`Select` treats '' as "nothing selected", so an option valued '' can never
// render as chosen). Testing around the library would have missed it entirely.
//
// `unplugin-vue-components` is the same plugin, with the same resolver, as the app build. Registering
// PrimeVue components by hand in a setup file instead would mean the test tree and the real tree could
// differ — which is the one thing a component test must not allow.
//
// jsdom for everything would be the simple choice and it is the wrong one: it took the suite from 1.3 s
// to about 12 s, because every logic-only spec paid for a DOM and for PrimeVue's setup. Fast tests get
// run; slow ones get skipped. So `node` stays the default and the few specs that mount a component opt
// in with a `@vitest-environment jsdom` docblock at the top of the file — see
// `src/components/kort/KortSettingsDialog.spec.ts`.
export default defineConfig({
  // Cast because vitest 2 ships its own pinned copy of Vite, so `@vitejs/plugin-vue` (built against the
  // project's Vite 6) and `vitest/config` disagree about the `Plugin` type even though they are the same
  // plugin at runtime. Contained to this one line, and removable when vitest is upgraded to a version
  // that shares the project's Vite.
  plugins: [
    vue(),
    Components({
      resolvers: [PrimeVueResolver()],
      dts: false,
    }),
  ] as unknown as UserConfig['plugins'],
  test: {
    environment: 'node',
    include: ['src/**/*.spec.ts'],
    setupFiles: ['./vitest.setup.ts'],
    // No watch by default: a watching process would hang a container run.
    watch: false,
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
