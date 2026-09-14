// @vitest-environment jsdom
//
// Component tests for the person-search results view.
//
// # Why mount this one
//
// Three of its behaviours are not visible from a logic test and each has an operational cost if
// wrong:
//
//  - **The debounce must land before the cache key changes.** Every distinct key is a module-level
//    cache entry that outlives the route, so keystrokes reaching the key leave an entry per prefix
//    of everything ever typed. The only way to see that is to count the requests.
//  - **`pending` must reach the table and nothing else.** It is threaded through a `shallowRef`
//    holding the composable's return value; a plain `ref` would reactive-proxy that object and
//    unwrap the refs inside it, making `pending.value` `undefined` — a loading state that silently
//    never appears. Caught here, invisible to a logic test.
//  - **The three empty states must read differently.** "Nothing typed", "keep typing" and "ingen
//    match" are different answers to a caller, and asserting on the rendered wording is the only
//    check that they have not converged.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import SearchView from './SearchView.vue'
import type { PersonResult, PersonSearchResponse } from '@/composables/personSearch'
import { clearLiveCache } from '@/composables/useLiveResource'

const get = vi.fn()

vi.mock('@/plugins/axios', () => ({
  http: { get: (...args: unknown[]) => get(...args) },
}))

const person = (over: Partial<PersonResult> = {}): PersonResult => ({
  kind: 'spejder',
  id: 'p-1',
  year: '2026',
  name: 'Rakel A. Koch',
  phone: '24450125',
  phoneRole: 'own',
  teamId: 't-1',
  teamName: 'Ørnene',
  teamNumber: '42',
  status: 'racing',
  statusKind: 'member',
  removed: false,
  ...over,
})

const payload = (over: Partial<PersonSearchResponse> = {}): PersonSearchResponse => ({
  results: [],
  truncated: false,
  tooShort: false,
  matchedPhone: false,
  matchedName: false,
  ...over,
})

// Real router, memory history: the query *is* the state under test, so stubbing $route would remove
// the mechanism rather than isolate it.
const makeRouter = (): Router =>
  createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div />' } },
      { path: '/search', name: 'search', component: SearchView },
      { path: '/patrulje/:teamId', name: 'patrulje', component: { template: '<div />' } },
      { path: '/klan', name: 'klaner', component: { template: '<div />' } },
      { path: '/badut', name: 'badutter', component: { template: '<div />' } },
      { path: '/organisation', name: 'organisation', component: { template: '<div />' } },
    ],
  })

let router: Router
let wrapper: VueWrapper

const openSearch = async (query: Record<string, string> = {}) => {
  router = makeRouter()
  await router.push({ name: 'search', query })
  await router.isReady()
  wrapper = mount(SearchView, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

const type = async (value: string) => {
  await wrapper.find('input').setValue(value)
}

beforeEach(() => {
  // The live cache is module-level and survives a mount, which is the point of it — so it has to be
  // discarded between tests or a later spec would render an earlier spec's results with no request.
  clearLiveCache()
  get.mockReset()
  get.mockResolvedValue({ data: { search: payload() } })
  vi.useFakeTimers({ shouldAdvanceTime: true })
})

afterEach(() => {
  vi.useRealTimers()
  wrapper?.unmount()
})

describe('the query lives in the URL', () => {
  it('searches what the URL carries on arrival, so a link and a reload both work', async () => {
    get.mockResolvedValue({ data: { search: payload({ results: [person()] }) } })
    await openSearch({ q: 'Koch' })

    expect(get).toHaveBeenCalledWith('/search/person', { params: { q: 'Koch' } })
    expect(wrapper.find('input').element.value).toBe('Koch')
    expect(wrapper.text()).toContain('Rakel A. Koch')
  })

  it('writes the debounced value to the URL', async () => {
    await openSearch()
    await type('Koch')
    expect(router.currentRoute.value.query.q).toBeUndefined()

    await vi.advanceTimersByTimeAsync(300)
    expect(router.currentRoute.value.query.q).toBe('Koch')
  })
})

describe('debounce lands before the cache key', () => {
  it('asks once for the settled query, not once per prefix', async () => {
    await openSearch()

    // Four keystrokes that each pass the minimum length on their own: without a debounce ahead of
    // the key, this is four cache entries and four requests.
    for (const value of ['2030', '20309', '203096', '20309696']) {
      await type(value)
      await vi.advanceTimersByTimeAsync(50)
    }
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()

    expect(get).toHaveBeenCalledTimes(1)
    expect(get).toHaveBeenCalledWith('/search/person', { params: { q: '20309696' } })
  })

  it('never asks below the minimum length', async () => {
    await openSearch()
    await type('ab')
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()

    expect(get).not.toHaveBeenCalled()
  })
})

describe('three empty states', () => {
  it('says nothing has been typed', async () => {
    await openSearch()
    expect(wrapper.text()).toContain('Skriv et telefonnummer eller et navn')
    expect(wrapper.text()).not.toContain('Ingen match på')
  })

  it('says the query is too short to have been searched', async () => {
    await openSearch({ q: 'ab' })
    expect(wrapper.text()).toContain('der er ikke søgt endnu')
    expect(wrapper.text()).not.toContain('Ingen match på')
  })

  it('says we looked and found nobody', async () => {
    await openSearch({ q: 'notfoundxyz' })
    expect(wrapper.text()).toContain('Ingen match på “notfoundxyz”')
    expect(wrapper.text()).not.toContain('Skriv et telefonnummer eller et navn')
  })
})

describe('the row', () => {
  it('links a scout row to its patrol', async () => {
    get.mockResolvedValue({ data: { search: payload({ results: [person()] }) } })
    await openSearch({ q: 'Koch' })

    const link = wrapper.find('a[href="/patrulje/t-1"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toContain('Rakel A. Koch')
  })

  it('navigates on a click anywhere in the row, not only on the name', async () => {
    get.mockResolvedValue({ data: { search: payload({ results: [person()] }) } })
    await openSearch({ q: 'Koch' })

    // The Hold cell — nothing anchored in it.
    await wrapper.findAll('tbody td')[2].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/patrulje/t-1')
  })

  it('does nothing when a row has nowhere to go', async () => {
    get.mockResolvedValue({
      data: { search: payload({ results: [person({ teamId: '' })] }) },
    })
    await openSearch({ q: 'Koch' })

    await wrapper.findAll('tbody td')[2].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.name).toBe('search')
  })

  it('says whose number matched when it is not the person’s own', async () => {
    get.mockResolvedValue({
      data: { search: payload({ results: [person({ phoneRole: 'parent' })] }) },
    })
    await openSearch({ q: '24450125' })
    expect(wrapper.text()).toContain('forælders nummer')
  })

  it('does not annotate a person’s own number', async () => {
    get.mockResolvedValue({ data: { search: payload({ results: [person()] }) } })
    await openSearch({ q: '24450125' })
    expect(wrapper.text()).not.toContain('forælders nummer')
    expect(wrapper.text()).not.toContain('kontaktperson')
  })

  it('renders a nameless person and an unnamed team without looking broken', async () => {
    get.mockResolvedValue({
      data: {
        search: payload({
          results: [person({ name: '', teamName: '', teamNumber: '' })],
        }),
      },
    })
    await openSearch({ q: 'Koch' })
    expect(wrapper.text()).toContain('(uden navn)')
  })

  it('groups by kind under Danish headings', async () => {
    get.mockResolvedValue({
      data: {
        search: payload({
          results: [
            person({ kind: 'klankontakt', id: 'k-1' }),
            person({ kind: 'spejder', id: 's-1' }),
          ],
        }),
      },
    })
    await openSearch({ q: 'Koch' })
    const text = wrapper.text()
    expect(text).toContain('Spejdere')
    expect(text).toContain('Kontaktpersoner')
    // Participants before the adults who answer for them.
    expect(text.indexOf('Spejdere')).toBeLessThan(text.indexOf('Kontaktpersoner'))
  })
})

describe('status badges', () => {
  const twoRows = () =>
    payload({
      results: [
        person({ id: 'racing-1', name: 'Aktiv Andersen', status: 'racing', statusKind: 'member' }),
        person({
          id: 'gone-1',
          name: 'Hjemme Hansen',
          status: 'released',
          statusKind: 'member',
        }),
      ],
    })

  // The criterion this task exists for: a departed person must not look like a current one. Asserted
  // on rendered markup rather than on the mapping (which `personSearch.spec.ts` covers), because the
  // thing that can break is the wiring — a `severity` that never reaches the Tag renders every row
  // identically while every unit test still passes.
  it('renders a departed row differently from a racing one', async () => {
    get.mockResolvedValue({ data: { search: twoRows() } })
    await openSearch({ q: 'Koch' })

    const tags = wrapper.findAll('.p-tag')
    expect(tags).toHaveLength(2)

    const signature = (i: number) => {
      const el = tags[i].element as HTMLElement
      return `${el.className}|${el.getAttribute('data-p') ?? ''}`
    }

    expect(tags[0].text()).toBe('Aktiv')
    expect(tags[1].text()).toBe('Afhentet')
    expect(signature(0)).not.toBe(signature(1))
  })

  it('shows a status on every row, including the ordinary one', async () => {
    get.mockResolvedValue({ data: { search: twoRows() } })
    await openSearch({ q: 'Koch' })
    // Two rows, two badges: the ordinary row is badged too, so an absent badge never has to be read
    // as meaning something.
    expect(wrapper.findAll('.p-tag')).toHaveLength(2)
  })

  it('shows udmeldt beside the status, not instead of it', async () => {
    get.mockResolvedValue({
      data: {
        search: payload({
          results: [person({ status: 'released', statusKind: 'member', removed: true })],
        }),
      },
    })
    await openSearch({ q: 'Koch' })

    const labels = wrapper.findAll('.p-tag').map((t) => t.text())
    expect(labels).toContain('Afhentet')
    expect(labels).toContain('Udmeldt')
  })

  it('badges an unknown status as unknown', async () => {
    get.mockResolvedValue({
      data: {
        search: payload({ results: [person({ status: undefined, statusKind: undefined })] }),
      },
    })
    await openSearch({ q: 'Koch' })
    expect(wrapper.find('.p-tag').text()).toBe('Ukendt status')
  })
})

describe('the cap is stated', () => {
  it('says there are more matches than shown', async () => {
    get.mockResolvedValue({
      data: { search: payload({ results: [person()], truncated: true }) },
    })
    await openSearch({ q: 'Anders' })
    expect(wrapper.text()).toContain('Indsnævr søgningen')
  })
})

describe('loading', () => {
  it('wires pending to the table and adds no spinner of its own', async () => {
    // A request that has not resolved: the table must be in its loading state, and that state must
    // come from the resource — a `ref` instead of a `shallowRef` around it would unwrap `pending`
    // and leave this false forever.
    get.mockImplementation(() => new Promise(() => {}))
    router = makeRouter()
    await router.push({ name: 'search', query: { q: 'Koch' } })
    wrapper = mount(SearchView, { global: { plugins: [router] } })
    await flushPromises()

    // Asserted on the prop rather than on a themed class: the criterion is that `pending` reaches
    // the table's own loading state, and PrimeVue's overlay markup is not ours to depend on.
    const table = wrapper.findComponent({ name: 'DataTable' })
    expect(table.props('loading')).toBe(true)

    // And no spinner of our own beside it.
    expect(wrapper.findComponent({ name: 'ProgressSpinner' }).exists()).toBe(false)
  })

  it('does not raise loading again for a query already in the cache', async () => {
    get.mockResolvedValue({ data: { search: payload({ results: [person()] }) } })
    await openSearch({ q: 'Koch' })
    expect(wrapper.findComponent({ name: 'DataTable' }).props('loading')).toBe(false)

    // Leave and come back: the entry survives, so there is nothing to show a spinner for.
    wrapper.unmount()
    get.mockImplementation(() => new Promise(() => {}))
    router = makeRouter()
    await router.push({ name: 'search', query: { q: 'Koch' } })
    wrapper = mount(SearchView, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.findComponent({ name: 'DataTable' }).props('loading')).toBe(false)
    expect(wrapper.text()).toContain('Rakel A. Koch')
  })
})
