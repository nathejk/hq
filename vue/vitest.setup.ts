// Global test setup: the plugins and browser stubs a mounted component cannot do without.
//
// Everything here is behind a `window` check and behind dynamic imports, because this file runs for
// *every* spec and most of them are logic-only tests in a `node` environment (see vitest.config.ts).
// Importing @vue/test-utils and PrimeVue unconditionally cost the suite several seconds of startup it
// had no use for.
//
// The plugins installed are deliberately the same set `main.ts` installs, and no more: a component test
// whose app differs from the real one tests something that does not ship — and this suite exists
// precisely because a *library* behaviour (PrimeVue's `Select` treating '' as "nothing selected") got
// past logic-only tests.

if (typeof window !== 'undefined') {
  // jsdom implements neither of the two browser APIs PrimeVue's form components reach for on mount, so
  // without these stubs every mounted component containing a dropdown fails before its first assertion.
  //
  // Both are stubbed as "nothing happens": no media query matches (the desktop case HQ is used on) and
  // nothing is ever observed to resize. Anything that genuinely depends on a resize or a matching query
  // is therefore *not* covered here rather than silently passing — worth knowing before writing a test
  // about responsive behaviour.
  if (!window.matchMedia) {
    window.matchMedia = (query: string): MediaQueryList =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }) as MediaQueryList
  }

  if (!('ResizeObserver' in window)) {
    window.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }

  const { config } = await import('@vue/test-utils')
  const PrimeVue = (await import('primevue/config')).default
  const Aura = (await import('@primevue/themes/aura')).default
  const ToastService = (await import('primevue/toastservice')).default
  const Tooltip = (await import('primevue/tooltip')).default

  config.global.plugins = [
    // The theme is included because PrimeVue 4 components read it from the plugin config; without it
    // some read their pass-through config as undefined and fail on mount.
    [PrimeVue, { locale: { firstDayOfWeek: 1 }, theme: { preset: Aura } }],
    ToastService,
  ]

  config.global.directives = { tooltip: Tooltip }
}
