// @vitest-environment jsdom
//
// The one behaviour of the Kørsel board that is invisible from the roster's own tests: **a saved
// vagt has to appear while the dialog is still open.**
//
// This is a regression test with a real bug behind it. The roster saved correctly, the board
// refetched, and the bar snapped back to where it had been — because `dutyDialog` was one of the
// conditions in `paused`, so `useDeferredApply` held the fresh payload back for as long as the
// editor was open. The server was right, the screen was wrong, and it only looked right after a
// reload. Nothing in `DispatchDutyRoster.spec.ts` could see it: the component emits, and it was the
// board that then declined to listen.
//
// So the assertions here are deliberately about the seam between the two — the dialog, the write and
// what the row says afterwards — and not about slider mechanics, which are tested next door.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import DispatchView from './DispatchView.vue'
import { clearLiveCache } from '@/composables/useLiveResource'
import type { Board, Duty } from '@/composables/dispatch'

const get = vi.fn()
const put = vi.fn()

vi.mock('@/plugins/axios', () => ({
  http: {
    get: (...args: unknown[]) => get(...args),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: (...args: unknown[]) => put(...args),
    patch: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
  },
}))

const local = (y: number, m: number, d: number, hh = 0, mm = 0) =>
  Math.floor(new Date(y, m - 1, d, hh, mm, 0, 0).getTime() / 1000)

const event = { startUts: local(2026, 9, 18), endUts: local(2026, 9, 21) }

const shift: Duty = {
  id: 'duty-1',
  year: '2026',
  sectionSlug: 'bil-1',
  startUts: local(2026, 9, 19, 8),
  endUts: local(2026, 9, 19, 16),
}

const board = (duty: Duty[]): Board => ({
  tasks: [],
  tours: [],
  units: [{ sectionSlug: 'bil-1', label: 'Bil 1', vehicles: [], people: [] }],
  duty,
  kinds: [],
  priorities: [],
  event,
})

let wrapper: VueWrapper | undefined

beforeEach(() => {
  // The live cache is module-level and outlives a mount, which is its purpose — so it has to be
  // discarded between tests or one test would render another's board with no request.
  clearLiveCache()
  get.mockReset()
  put.mockReset()
  get.mockResolvedValue({ data: board([shift]) })
  put.mockResolvedValue({ data: { dutyId: shift.id } })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

/** Mount the board and open the Vagter dialog, as an operator would. */
const openRoster = async () => {
  wrapper = mount(DispatchView, { attachTo: document.body })
  await flushPromises()
  const button = wrapper.findAll('button').find((b) => b.text().includes('Vagter'))
  expect(button, 'the board must offer a Vagter button').toBeTruthy()
  await button!.trigger('click')
  await flushPromises()
  return wrapper
}

const roster = (w: VueWrapper) => w.findComponent({ name: 'DispatchDutyRoster' })

describe('the Vagter dialog', () => {
  it('opens on the board’s own button', async () => {
    const w = await openRoster()
    expect(roster(w).exists()).toBe(true)
    expect(roster(w).text()).toContain('Bil 1')
  })

  it('shows a saved vagt without the dialog being closed', async () => {
    // The bug. The roster emits, the board writes, the board refetches — and the operator has to see
    // the result *here*, not after a reload. Asserted on the label inside the bar, which is where a
    // shift's times live in the condensed layout.
    const w = await openRoster()
    expect(roster(w).text()).toContain('08.00–16.00')

    const moved = { ...shift, endUts: local(2026, 9, 19, 20) }
    get.mockResolvedValue({ data: board([moved]) })
    roster(w).vm.$emit('save', {
      dutyId: shift.id,
      sectionSlug: shift.sectionSlug,
      startUts: moved.startUts,
      endUts: moved.endUts,
    })
    await flushPromises()

    expect(put).toHaveBeenCalledWith('/dispatchduty', {
      dutyId: shift.id,
      sectionSlug: shift.sectionSlug,
      startUts: moved.startUts,
      endUts: moved.endUts,
    })
    expect(roster(w).text()).toContain('08.00–20.00')
    expect(roster(w).text()).not.toContain('08.00–16.00')
  })

  it('shows a removed vagt as gone, likewise', async () => {
    const w = await openRoster()
    get.mockResolvedValue({ data: board([]) })
    roster(w).vm.$emit('remove', shift.id)
    await flushPromises()

    expect(roster(w).text()).toContain('ingen')
  })

  it('shows an added vagt, likewise', async () => {
    const w = await openRoster()
    const added: Duty = { ...shift, id: 'duty-2', startUts: local(2026, 9, 20, 22), endUts: event.endUts }
    get.mockResolvedValue({ data: board([shift, added]) })
    roster(w).vm.$emit('save', {
      sectionSlug: 'bil-1',
      startUts: added.startUts,
      endUts: added.endUts,
    })
    await flushPromises()

    expect(roster(w).text()).toContain('22.00–00.00')
  })

  it('holds updates back while a knot is actually held, and applies them on release', async () => {
    // The pause rule as it should have been written: the *gesture* is the unsaved state, not the
    // dialog. A payload landing mid-drag would move the bar out from under the cursor.
    const w = await openRoster()
    roster(w).vm.$emit('dragging', true)
    await flushPromises()

    get.mockResolvedValue({ data: board([{ ...shift, endUts: local(2026, 9, 19, 23) }]) })
    // Something else on the board changed — another operator, or a scan — and the board refetched.
    await (w.vm as unknown as { refresh: () => Promise<void> }).refresh?.()
    await flushPromises()
    expect(roster(w).text()).toContain('08.00–16.00')

    roster(w).vm.$emit('dragging', false)
    await flushPromises()
    expect(roster(w).text()).toContain('08.00–23.00')
  })
})
