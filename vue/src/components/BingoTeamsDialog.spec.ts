// @vitest-environment jsdom
//
// Component tests for the bingo table.
//
// # Why this is a component test
//
// The thing worth protecting here is not arithmetic — that is covered in Go, where the verdicts are
// computed — but the *rendering* of a distinction the eye depends on: a cross means somebody should do
// something, a middot means nothing has happened yet. Get those the wrong way round and the table is
// actively misleading while looking perfectly healthy, which no logic-only test would notice.
//
// Mounted rather than shallow-rendered because the cells live inside PrimeVue's DataTable, and its row
// slot is exactly the part that has to work.

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { clearLiveCache } from '@/composables/useLiveResource'

const get = vi.fn()

vi.mock('@/plugins/axios', () => ({
  http: { get: (...args: unknown[]) => get(...args) }
}))

import BingoTeamsDialog from './BingoTeamsDialog.vue'

type Cell = { status: string; scannedAtUts?: number; deadlineUts?: number }

const row = (over: Record<string, unknown> = {}) => ({
  teamId: 't1',
  teamNumber: '7',
  name: 'Remuladerøverne',
  group: '2 Randers',
  activeMemberCount: 4,
  cells: [] as Cell[],
  catchCount: 0,
  bingo: false,
  ...over
})

const payload = (over: Record<string, unknown> = {}) => ({
  lines: [{ checkgroupId: 'cg1', name: 'Postlinje1' }],
  teams: [row()],
  bingoCount: 0,
  ...over
})

async function mountWith(data: Record<string, unknown>) {
  get.mockResolvedValue({ data })
  const wrapper = mount(BingoTeamsDialog, {
    global: {
      stubs: {
        // PrimeVue's Dialog teleports to body, which turns every query below into a document
        // search for no gain — nothing here is testing the dialog chrome. Same stub, for the same
        // reason, as KortSettingsDialog.spec.ts.
        Dialog: { template: '<div><slot /><slot name="header" /><slot name="footer" /></div>' },
        'router-link': { template: '<a><slot /></a>' }
      }
    }
  })
  // One tick for the fetch, one for the render it causes.
  await new Promise((r) => setTimeout(r, 0))
  await wrapper.vm.$nextTick()
  return wrapper
}

beforeEach(() => {
  // The resource is cached at module level by key, so without this the second test renders the
  // first test's payload — and passes for the wrong reason.
  clearLiveCache()
  get.mockReset()
})

/** The cells of the first (and only) postlinje column, as rendered. */
function lineCell(wrapper: ReturnType<typeof mount> extends never ? never : any) {
  // The two frozen columns come first, so the postlinje cell is the third in the row.
  return wrapper.findAll('tbody td')[2]
}

/** The hover/screen-reader sentence for that cell, which sits on the glyph's own element. */
function lineCellLabel(wrapper: ReturnType<typeof mount> extends never ? never : any) {
  return lineCell(wrapper).find('[aria-label]').attributes('aria-label')
}

describe('a postlinje the team has not reached', () => {
  // The requirement: at 23.00 a patrol that has not reached a post closing at 03.00 has done
  // nothing wrong, and must not be marked as having missed it.
  it('is a grey middot, not a cross', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ cells: [{ status: 'pending', deadlineUts: 1789869600 }] })] }))
    const cell = lineCell(wrapper)
    expect(cell.text()).toContain('·')
    expect(cell.find('.pi-times').exists()).toBe(false)
    expect(cell.html()).toContain('text-gray-400')
  })

  // The hover still has to carry the deadline: "not yet — due at 03.00" is the useful sentence,
  // and a dot with no explanation is just a dot.
  it('says when it is due, on hover', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ cells: [{ status: 'pending', deadlineUts: 1789869600 }] })] }))
    expect(lineCellLabel(wrapper)).toMatch(/Ikke lukket endnu.*frist/)
  })
})

describe('a postlinje that has closed', () => {
  it('is a red cross when the team was never seen', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ cells: [{ status: 'missing', deadlineUts: 1789763400 }] })] }))
    const cell = lineCell(wrapper)
    expect(cell.find('.pi-times').exists()).toBe(true)
    expect(cell.html()).toContain('text-red-600')
  })

  // A late scan is a cross too, but hovers as a different sentence and carries both times.
  it('distinguishes a late arrival from never arriving', async () => {
    const wrapper = await mountWith(
      payload({
        teams: [row({ cells: [{ status: 'late', scannedAtUts: 1789763999, deadlineUts: 1789763400 }] })]
      })
    )
    const label = lineCellLabel(wrapper)
    expect(label).toMatch(/For sent/)
    expect(label).toMatch(/scannet/)
  })

  it('is a green tick when the team was on time', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ cells: [{ status: 'onTime', scannedAtUts: 1789760000 }] })] }))
    const cell = lineCell(wrapper)
    expect(cell.find('.pi-check').exists()).toBe(true)
    expect(cell.html()).toContain('text-green-600')
  })
})

describe('the bingo highlight', () => {
  it('paints a qualifying patrol green', async () => {
    const wrapper = await mountWith(
      payload({
        teams: [row({ bingo: true, cells: [{ status: 'onTime', scannedAtUts: 1 }] })],
        bingoCount: 1
      })
    )
    expect(wrapper.find('tbody tr').classes()).toContain('bg-green-50')
  })

  it('leaves everyone else alone', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ cells: [{ status: 'pending' }] })] }))
    expect(wrapper.find('tbody tr').classes()).not.toContain('bg-green-50')
  })
})

// The catch columns are the other half of the verdict, and zero is the common case: it reads as a
// muted dash-like zero rather than as an alarming red one.
describe('the catch columns', () => {
  it('shows a catch count and the first catch', async () => {
    const wrapper = await mountWith(
      payload({
        teams: [row({ catchCount: 3, firstCatchUts: 1758320224, cells: [{ status: 'pending' }] })]
      })
    )
    const text = wrapper.find('tbody tr').text()
    expect(text).toContain('3')
    expect(wrapper.find('tbody tr').html()).toContain('text-red-600')
  })

  it('shows a dash where a patrol has never been caught', async () => {
    const wrapper = await mountWith(payload({ teams: [row({ catchCount: 0, cells: [{ status: 'pending' }] })] }))
    expect(wrapper.find('tbody tr').text()).toContain('—')
  })
})

// A route with nothing flagged obligatorisk has no card to complete, so the table says so rather
// than rendering a row of nothing and letting it look like a data problem.
describe('no obligatoriske postlinjer', () => {
  it('explains itself', async () => {
    const wrapper = await mountWith(payload({ lines: [], teams: [row({ cells: [] })] }))
    expect(wrapper.text()).toContain('Ingen postlinjer er markeret obligatoriske')
  })
})
