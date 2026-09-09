// @vitest-environment jsdom
//
// Component tests for the map-sheet editor.
//
// # Why this component and not the map
//
// This is where the state is: five dirty flags, a save that composes two requests, and a buffer that
// must follow the selected sheet without being overwritten by live payloads. Two of the three
// regressions that reached an operator recently lived here, and both were invisible to logic-only tests
// — one because it needed a real PrimeVue `Select`, the other because it emerged from *composing*
// reactive pieces that were each individually correct.
//
// `KortView` is deliberately not tested this way: Leaflet in jsdom has no layout, so every element is
// 0x0 and any assertion about the map is really an assertion about a mock. The tile regression proved
// that class of bug needs a real browser, not a heavier jsdom setup.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import KortSettingsDialog from './KortSettingsDialog.vue'
import { HANDOUT_ON_QR_LABEL, type Kort, type KortPayload } from '@/composables/kort'

const put = vi.fn().mockResolvedValue({ data: {} })
const post = vi.fn().mockResolvedValue({ data: { kortId: 'kort-new' } })
const del = vi.fn().mockResolvedValue({ data: {} })

vi.mock('@/plugins/axios', () => ({
  http: {
    get: vi.fn().mockResolvedValue({ data: {} }),
    post: (...args: unknown[]) => post(...args),
    put: (...args: unknown[]) => put(...args),
    delete: (...args: unknown[]) => del(...args),
  },
}))

const sheet = (over: Partial<Kort> = {}): Kort => ({
  id: 'kort-1',
  kortsaetId: 'set-1',
  year: '2026',
  version: 0,
  name: 'Kort 1',
  format: 'a4',
  note: '',
  sortOrder: 0,
  checkpointIds: [],
  extents: [],
  handoutCheckgroupId: '',
  ...over,
})

const payload = (kort: Kort[]): KortPayload => ({
  kortsaet: [{ id: 'set-1', year: '2026', version: 0, name: 'Deltagerkort', sortOrder: 0, teamType: 'patrulje', kort }],
  orphanKort: [],
})

const checkgroups = [
  { id: 'cg-1', name: 'Postlinje 1', checkpoints: [{ id: 'cp-1', name: 'Post 1', latitude: 55.8, longitude: 12.1 }] },
  { id: 'cg-2', name: 'Postlinje 2', checkpoints: [{ id: 'cp-2', name: 'Post 2', latitude: 55.9, longitude: 12.2 }] },
]

/**
 * Mount with the sheet selected, the way the view does it: `selectedId` is the view's state, so the
 * dialog is told what is selected rather than deciding for itself.
 */
const open = async (sheets: Kort[], selectedId?: string) => {
  const wrapper = mount(KortSettingsDialog, {
    props: { visible: true, payload: payload(sheets), checkgroups, selectedId },
    global: {
      // The only stub: PrimeVue's Dialog teleports to body, which turns every query in these tests into
      // a document search for no gain — nothing here is testing the dialog chrome.
      stubs: { Dialog: { template: '<div><slot /></div>' } },
    },
  })
  await nextTick()
  return wrapper
}

/** Select a sheet the way an operator does: by clicking its row in the list. */
const clickSheet = async (wrapper: VueWrapper, name: string) => {
  const row = wrapper.findAll('button').find((b) => b.text() === name)
  if (!row) throw new Error(`no row for ${name}`)
  await row.trigger('click')
  await nextTick()
}

/** Re-render with a new selection, standing in for the view's `v-model:selectedId`. */
const select = async (wrapper: VueWrapper, selectedId: string | undefined) => {
  await wrapper.setProps({ selectedId })
  await nextTick()
}

const nameField = (wrapper: VueWrapper) => wrapper.find<HTMLInputElement>('input[type="text"]')
const button = (wrapper: VueWrapper, label: string) =>
  wrapper.findAll('button').find((b) => b.text().includes(label))

beforeEach(() => {
  put.mockClear()
  post.mockClear()
  del.mockClear()
})

describe('KortSettingsDialog', () => {
  // The regression that reached an operator: the editor opened with blank fields, and every other sheet
  // was then refused with "gem eller annullér først" with nothing to save. The cause was a dirty check
  // gating the load of the very value it compared against.
  it('loads the selected sheet into the form, and switches sheets without demanding a save', async () => {
    const wrapper = await open([sheet(), sheet({ id: 'kort-2', name: 'Kort 2' })], 'kort-1')

    expect(nameField(wrapper).element.value).toBe('Kort 1')
    // Nothing unsaved, so the view is not told to pause live updates.
    expect(wrapper.emitted('update:dirty')?.flat()).not.toContain(true)

    await select(wrapper, 'kort-2')
    expect(nameField(wrapper).element.value).toBe('Kort 2')
  })

  it('lets the operator click from one sheet to another', async () => {
    const wrapper = await open([sheet(), sheet({ id: 'kort-2', name: 'Kort 2' })], 'kort-1')

    await clickSheet(wrapper, 'Kort 2')

    // The dialog does not own the selection; it asks for it. A refusal would emit nothing.
    expect(wrapper.emitted('update:selectedId')?.at(-1)).toEqual(['kort-2'])
  })

  // The other regression, and the reason a *component* test was needed: PrimeVue's Select decides
  // whether anything is selected with isNotEmpty(modelValue), and isEmpty('') is true. With '' as the
  // QR option's value the field showed its placeholder for ever and picking QR changed nothing, so
  // "Udleveret" could not be saved. Logic-only tests cannot see this.
  it('shows the QR handout rule as a chosen value, not as a placeholder', async () => {
    const wrapper = await open([sheet({ handoutCheckgroupId: '' })], 'kort-1')

    expect(wrapper.text()).toContain(HANDOUT_ON_QR_LABEL)
  })

  it('saves a chosen checkgroup as its id, and the QR rule as an empty string', async () => {
    const wrapper = await open([sheet({ handoutCheckgroupId: '' })], 'kort-1')
    const selects = wrapper.findAllComponents({ name: 'Select' })
    // Format first, then Udleveret — the order they appear in the form.
    const handout = selects[1]

    handout.vm.$emit('update:modelValue', 'cg-2')
    await nextTick()
    await button(wrapper, 'Gem kort')!.trigger('click')
    expect(put.mock.calls[0][1]).toMatchObject({ handoutCheckgroupId: 'cg-2' })

    put.mockClear()
    const back = await open([sheet({ handoutCheckgroupId: 'cg-2' })], 'kort-1')
    back.findAllComponents({ name: 'Select' })[1].vm.$emit('update:modelValue', 'qr')
    await nextTick()
    await button(back, 'Gem kort')!.trigger('click')
    expect(put.mock.calls[0][1]).toMatchObject({ handoutCheckgroupId: '' })
  })

  // One button, one save. The three-button version let an operator save the name and walk away with the
  // checkpoints still unsaved — and the checkpoints are the part a consuming app actually needs.
  it('sends the description, the areas and the checkpoints in one gesture', async () => {
    const wrapper = await open([sheet({ extents: [] })], 'kort-1')

    await nameField(wrapper).setValue('Kort 1 — Start til Post 2')
    wrapper.findAllComponents({ name: 'TreeSelect' })[0].vm.$emit('update:modelValue', {
      'cp-1': { checked: true, partialChecked: false },
    })
    await nextTick()

    await button(wrapper, 'Gem kort')!.trigger('click')
    await nextTick()

    expect(put).toHaveBeenCalledTimes(2)
    expect(put.mock.calls[0][0]).toBe('/kort/kort-1')
    expect(put.mock.calls[0][1]).toMatchObject({ name: 'Kort 1 — Start til Post 2', extents: [] })
    // The checkpoints are a second endpoint — a replace, not a patch — and go last, so a failure there
    // leaves the sheet described but not re-pointed.
    expect(put.mock.calls[1][0]).toBe('/kort/kort-1/checkpoints')
    expect(put.mock.calls[1][1]).toEqual({ checkpointIds: ['cp-1'] })
  })

  it('reverts every part of the sheet on Annullér, and reports itself clean again', async () => {
    const wrapper = await open([sheet({ name: 'Kort 1', checkpointIds: ['cp-1'] })], 'kort-1')

    await nameField(wrapper).setValue('Noget andet')
    wrapper.findAllComponents({ name: 'TreeSelect' })[0].vm.$emit('update:modelValue', {})
    await nextTick()
    expect(wrapper.emitted('update:dirty')?.flat()).toContain(true)

    await button(wrapper, 'Annullér')!.trigger('click')
    await nextTick()

    expect(nameField(wrapper).element.value).toBe('Kort 1')
    // Back to clean, which is what un-pauses live updates on the map behind the dialog.
    expect(wrapper.emitted('update:dirty')?.at(-1)).toEqual([false])
    expect(put).not.toHaveBeenCalled()
  })

  // The contract with the map for drag-to-move and drag-to-resize. The map owns the gesture
  // and reports a finished rectangle by index; the dialog owns the draft. Testable without a map, which
  // is the point of the split — and otherwise this contract has no coverage at all.
  it('takes a dragged area from the map into the same unsaved draft', async () => {
    const original = {
      northWest: { latitude: 55.9, longitude: 12.0 },
      southEast: { latitude: 55.8, longitude: 12.2 },
    }
    const dragged = {
      northWest: { latitude: 55.95, longitude: 12.05 },
      southEast: { latitude: 55.85, longitude: 12.25 },
    }
    const wrapper = await open([sheet({ extents: [original] })], 'kort-1')

    await wrapper.setProps({ extentEdit: { index: 0, extent: dragged, seq: 1 } })
    await nextTick()

    // Unsaved, so the map pauses live updates rather than redrawing over the operator's drag.
    expect(wrapper.emitted('update:dirty')?.at(-1)).toEqual([true])

    await button(wrapper, 'Gem kort')!.trigger('click')
    await nextTick()
    expect(put.mock.calls[0][1]).toMatchObject({ extents: [dragged] })
  })

  // An index the dialog never handed out would mean the two had drifted apart; appending a rectangle
  // nobody drew would be a strange way to discover that.
  it('ignores a dragged area for an index that does not exist', async () => {
    const wrapper = await open([sheet({ extents: [] })], 'kort-1')

    await wrapper.setProps({
      extentEdit: {
        index: 3,
        extent: { northWest: { latitude: 56, longitude: 12 }, southEast: { latitude: 55, longitude: 13 } },
        seq: 1,
      },
    })
    await nextTick()

    expect(wrapper.emitted('update:dirty')?.flat()).not.toContain(true)
  })

  // The same PrimeVue trap as the QR handout option, one field over: the API stores `null` for "no
  // particular team type", and `Select` renders a `null`-valued option as its placeholder — so the
  // commonest answer looked like an unfilled field.
  it('shows a set with no team type as a chosen value, and saves it back as null', async () => {
    const wrapper = await open([sheet()], undefined)

    await button(wrapper, 'Nyt sæt')!.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('Ingen bestemt holdtype')

    await wrapper.find('input[type="text"]').setValue('Crew')
    await button(wrapper, 'Opret')!.trigger('click')
    await nextTick()

    expect(post.mock.calls[0][0]).toBe('/kortsaet')
    expect(post.mock.calls[0][1]).toEqual({ name: 'Crew', teamType: null })
  })
})
