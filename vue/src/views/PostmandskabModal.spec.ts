// @vitest-environment jsdom
//
// Component tests for the postmandskab picker.
//
// Two operator-visible bugs lived in the *composition* of this row's TreeSelect and were invisible
// to logic-only tests, because both were PrimeVue behaviours rather than logic:
//
//   1. Building `:options` / `:modelValue` inline in the template gave them a fresh identity on
//      every render. TreeSelect watches both by identity and resets its own `expandedKeys` in the
//      watcher — state that is read while rendering — so each render caused the next one and the
//      page locked up on selecting a person.
//   2. `expandedKeys` is TreeSelect *state*, not a prop, so binding it did nothing at all — and
//      forcing it onto the instance instead threw from inside `before-show`, which is emitted
//      *before* the overlay is shown, so the dropdown stopped opening at all. The postmandskab are
//      flattened to the top level for that reason.
//
// So the assertions here are deliberately about what the operator does: the overlay opens, the
// names are in it, and a selection lands without the component re-rendering forever.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import PostmandskabModal from './PostmandskabModal.vue'

const get = vi.fn()

vi.mock('@/plugins/axios', () => ({
  http: {
    get: (...args: unknown[]) => get(...args),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: vi.fn().mockResolvedValue({ data: {} }),
    patch: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
  },
}))

const response = {
  status: 200,
  data: {
    checkgroup: { id: 'cg-1', name: 'Postlinje 1' },
    checkpoints: [{ id: 'cp-1', name: 'Post 1', openFrom: '2026-07-10T08:00:00Z', openUntil: '2026-07-10T20:00:00Z' }],
    availablePersonnel: [
      { id: 'u1', name: 'Alma', sectionSlug: 'postmandskab', sectionLabel: 'Postmandskab', priority: true },
      { id: 'u2', name: 'Bo', sectionSlug: 'postmandskab', sectionLabel: 'Postmandskab', priority: true },
      { id: 'u3', name: 'Cecilie', sectionSlug: 'hq', sectionLabel: 'HQ' },
    ],
    assignedPersonnel: [],
    year: { dateStart: '2026-07-10T00:00:00Z' },
  },
}

/** Mount, wait for the load, and add the one pending row every test needs. */
const openWithPendingRow = async (): Promise<VueWrapper> => {
  const wrapper = mount(PostmandskabModal, { props: { checkgroupId: 'cg-1' }, attachTo: document.body })
  await nextTick()
  await nextTick()
  const add = wrapper.findAll('button').find((b: { text: () => string }) => b.text().includes('Tilføj person'))
  expect(add, 'the group header must offer "Tilføj person"').toBeTruthy()
  await add!.trigger('click')
  await nextTick()
  return wrapper
}

/** The TreeSelect's clickable face. */
const picker = (wrapper: VueWrapper) => wrapper.find('.p-treeselect')

describe('PostmandskabModal person picker', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockResolvedValue(response)
  })

  it('opens the overlay on click and offers the crew', async () => {
    const wrapper = await openWithPendingRow()
    await picker(wrapper).trigger('click')
    await nextTick()

    const overlay = document.querySelector('.p-treeselect-overlay')
    expect(overlay, 'clicking the picker must open the overlay').toBeTruthy()
    expect(overlay!.textContent).toContain('HQ (1)')
    wrapper.unmount()
  })

  it('shows the postmandskab without a second gesture, other sections shut', async () => {
    const wrapper = await openWithPendingRow()
    await picker(wrapper).trigger('click')
    await nextTick()

    const overlay = document.querySelector('.p-treeselect-overlay')!
    // The prioritised people are on screen at first click; the rest are one click away rather
    // than one scroll. This is why they are flattened instead of branched — TreeSelect offers no
    // way to open a branch from outside.
    expect(overlay.textContent).toContain('Alma')
    expect(overlay.textContent).toContain('Bo')
    expect(overlay.textContent).not.toContain('Cecilie')
    wrapper.unmount()
  })

  it('records a selection without re-rendering forever', async () => {
    const wrapper = await openWithPendingRow()
    await picker(wrapper).trigger('click')
    await nextTick()

    const label = Array.from(document.querySelectorAll('.p-tree-node-label')).find((el) => el.textContent === 'Alma')
    expect(label, 'the prioritised names must be clickable').toBeTruthy()
    ;(label as HTMLElement).click()
    await nextTick()
    await nextTick()

    // Two things at once: the pick was recorded, and getting here at all means the render settled.
    // The identity bug made this line unreachable — Vue spun until the tab stopped responding.
    expect(wrapper.text()).toContain('Alma')
    expect(picker(wrapper).text()).toContain('Alma')
    wrapper.unmount()
  })
})
