// @vitest-environment jsdom
//
// Component tests for the vagter roster.
//
// The arithmetic is tested in `composables/dutyTimeline.spec.ts`. What is left to get wrong is the
// composition, and it is the part that would fail silently:
//
//   1. A knot carries the id of the window it edits. Drag the wrong one and the screen still looks
//      like a working roster — it has simply moved somebody else's shift. So the drag tests always
//      assert *which* window was saved, never just that something was.
//   2. Writing is emitted, never done here: the board owns the PUT and the `saving` flag that
//      pauses live payloads. A component that grew its own `http.put` would break that discipline
//      without breaking any assertion, so the absence is pinned against the source.
//   3. The two empty states have different fixes — no units is an Organisation problem, no dates is
//      an År problem — and an operator sent to the wrong page stays stuck.
//
// jsdom lays nothing out, so the track's geometry is stubbed: `clientX` only means something
// against a `getBoundingClientRect`, and without the stub every drag would resolve to the same
// instant and the tests would agree with each other about nothing.

import { describe, it, expect } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import DispatchDutyRoster from './DispatchDutyRoster.vue'
import type { Duty, Unit } from '@/composables/dispatch'
import { MIN_PERIOD_MINUTES, NEW_PERIOD_MINUTES } from '@/composables/dutyTimeline'

const local = (y: number, m: number, d: number, hh = 0, mm = 0) =>
  Math.floor(new Date(y, m - 1, d, hh, mm, 0, 0).getTime() / 1000)

// 2026 runs Friday 2026-09-18 to Sunday 2026-09-20: a 72-hour axis.
const event = { startUts: local(2026, 9, 18), endUts: local(2026, 9, 21) }
const HOURS = 72

const units: Unit[] = [
  { sectionSlug: 'bil-1', label: 'Bil 1', vehicles: [], people: [] },
  { sectionSlug: 'bil-2', label: 'Bil 2', vehicles: [], people: [] },
]

const duty = (sectionSlug: string, startUts: number, endUts: number, id: string): Duty => ({
  id,
  year: '2026',
  sectionSlug,
  startUts,
  endUts,
})

/** A track 720px wide starting at x=0: ten pixels per hour of the race, so a test can name an
 *  hour and get a pixel. */
const TRACK_WIDTH = HOURS * 10

const mountRoster = (props: Partial<InstanceType<typeof DispatchDutyRoster>['$props']> = {}) => {
  const wrapper = mount(DispatchDutyRoster, { props: { units, duty: [], event, ...props } })
  for (const track of wrapper.findAll('[data-track]')) {
    ;(track.element as HTMLElement).getBoundingClientRect = () =>
      ({ left: 0, width: TRACK_WIDTH, top: 0, height: 28, right: TRACK_WIDTH, bottom: 28 }) as DOMRect
  }
  return wrapper
}

/** The x of an absolute time on the stubbed track. */
const xOf = (uts: number) => ((uts - event.startUts) / (event.endUts - event.startUts)) * TRACK_WIDTH

/**
 * Fire a pointer event at a time on the axis.
 *
 * Constructed rather than triggered through test-utils: jsdom maps `pointerdown` to a `MouseEvent`,
 * whose `clientX` is a getter, so test-utils' "assign the options onto the event" approach throws.
 * A real MouseEvent takes the coordinate in its constructor.
 */
const fire = async (el: Element, type: string, uts?: number) => {
  el.dispatchEvent(
    new MouseEvent(type, { clientX: uts === undefined ? 0 : xOf(uts), bubbles: true, cancelable: true }),
  )
  await nextTick()
}

/**
 * A unit's position, which is how the three columns are addressed.
 *
 * Names, timeline and add are sibling *columns*, not cells of a row — the middle one has to be the
 * only thing that scrolls — so "the row for Bil 1" is three elements at the same index rather than
 * one element. The tests address them that way on purpose: if the columns ever fall out of step, that
 * is the bug, and a helper that searched each column by name would hide it.
 */
const unitAt = (label: string) => {
  const index = units.findIndex((u) => u.label === label)
  expect(index, `no unit called ${label}`).toBeGreaterThanOrEqual(0)
  return index
}

/** The timeline row for a unit: the track, its bars and its knots. */
const row = (wrapper: VueWrapper, label: string) => {
  const rows = wrapper.findAll('[data-scroller] .duty-row')
  expect(rows.length, 'the scroller must hold one row per unit').toBe(units.length)
  return rows[unitAt(label)]
}

/** The name cell for a unit, in the frozen column. */
const nameCell = (wrapper: VueWrapper, label: string) =>
  wrapper.findAll('[data-names] .duty-row')[unitAt(label)]

/** The add button for a unit, in the column after the timeline. */
const addButton = (wrapper: VueWrapper, label: string) =>
  wrapper.findAll('[data-add] .duty-row')[unitAt(label)].find('button')

const track = (wrapper: VueWrapper, label: string) => row(wrapper, label).find('[data-track]')
const bars = (wrapper: VueWrapper, label: string) => row(wrapper, label).findAll('[data-bar]')
const knots = (wrapper: VueWrapper, label: string) => row(wrapper, label).findAll('[data-knot]')

/** Grab something at `from`, move the pointer to `to`, and let go. */
const dragTo = async (
  wrapper: VueWrapper,
  label: string,
  grab: Element,
  from: number,
  to: number,
) => {
  await fire(grab, 'pointerdown', from)
  await fire(track(wrapper, label).element, 'pointermove', to)
  await fire(track(wrapper, label).element, 'pointerup')
}

describe('DispatchDutyRoster layout', () => {
  it('costs one line per unit: a name, a timeline and an add button', () => {
    // The point of the condensed layout. The value of this screen is seeing *many* units at once, so
    // a unit that wanted two lines would halve how much of the race is visible.
    const wrapper = mountRoster()
    expect(wrapper.findAll('[data-scroller] .duty-row')).toHaveLength(2)
    expect(wrapper.findAll('[data-names] .duty-row')).toHaveLength(2)
    expect(wrapper.findAll('[data-add] .duty-row')).toHaveLength(2)
    expect(bars(wrapper, 'Bil 1')).toHaveLength(0)
    wrapper.unmount()
  })

  it('scrolls the timeline and nothing else, in one container for every row', () => {
    // One scroller, not one per row: two units scrolled to different hours cannot be compared, and
    // comparison is the only reason this screen exists. The name and the add button stay put, so
    // neither can scroll away from the row it belongs to.
    const wrapper = mountRoster()
    const scrollers = wrapper.findAll('[data-scroller]')
    expect(scrollers).toHaveLength(1)
    expect(scrollers[0].classes()).toContain('overflow-x-auto')
    expect(scrollers[0].findAll('[data-track]')).toHaveLength(2)
    wrapper.unmount()
  })

  it('keeps a night shift as one bar, not two', () => {
    // The whole reason the axis is not a column per day: as two bars the seam fell exactly where
    // the race is hardest to staff.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 18, 21, 40), local(2026, 9, 19, 2), 'night')],
    })
    expect(bars(wrapper, 'Bil 1')).toHaveLength(1)
    wrapper.unmount()
  })

  it('draws two knots per period, and the bar between them', () => {
    const wrapper = mountRoster({
      duty: [
        duty('bil-1', local(2026, 9, 18, 20), local(2026, 9, 19, 2), 'a'),
        duty('bil-1', local(2026, 9, 19, 20), local(2026, 9, 20, 2), 'b'),
      ],
    })
    expect(bars(wrapper, 'Bil 1')).toHaveLength(2)
    expect(knots(wrapper, 'Bil 1')).toHaveLength(4)
    wrapper.unmount()
  })

  it('places a bar where its hours are on the axis', () => {
    // Saturday midday is halfway through a three-day race, and six hours is a twelfth of it.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 19, 12), local(2026, 9, 19, 18), 'a')],
    })
    const style = bars(wrapper, 'Bil 1')[0].attributes('style') ?? ''
    expect(style).toContain('left: 50%')
    expect(Number(/width: ([\d.]+)%/.exec(style)?.[1])).toBeCloseTo((6 / HOURS) * 100, 6)
    wrapper.unmount()
  })

  it('labels the days once for the whole roster, not once per unit', () => {
    // Repeating the scale per row was a third of the vertical space and said nothing new.
    const wrapper = mountRoster()
    const labels = wrapper.findAll('[data-scroller] .duty-scale span').map((s) => s.text())
    expect(labels).toEqual(['fre 18/9', 'lør 19/9', 'søn 20/9'])
    expect(row(wrapper, 'Bil 1').text()).not.toContain('fre 18/9')
    wrapper.unmount()
  })

  it('writes the times inside the bar', () => {
    // Condensed: the reading of a shift is on the shift, not in a list underneath it, which is what
    // buys the second line back.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 18, 21, 40), local(2026, 9, 19, 2), 'night')],
    })
    expect(bars(wrapper, 'Bil 1')[0].find('[data-bar-label]').text()).toBe('21.40–02.00')
    wrapper.unmount()
  })

  it('clips the in-bar times rather than letting them escape a short shift', () => {
    // A fifteen-minute window is 0.3% of a three-day axis. Without the clip its label would spill
    // across the neighbours and read as belonging to them.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 19, 12), local(2026, 9, 19, 12, 15), 'short')],
    })
    expect(bars(wrapper, 'Bil 1')[0].classes()).toContain('overflow-hidden')
    wrapper.unmount()
  })

  it('shows the full times on hover, outside the bar where nothing clips them', async () => {
    // An operator who cannot read a time cannot check a roster, and a two-hour bar cannot hold one.
    // The hover card also carries the weekday, which the in-bar label drops.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 18, 21, 40), local(2026, 9, 19, 2), 'night')],
    })
    expect(row(wrapper, 'Bil 1').find('[data-bar-card]').exists()).toBe(false)

    await fire(bars(wrapper, 'Bil 1')[0].element, 'pointerenter')
    expect(row(wrapper, 'Bil 1').find('[data-bar-card]').text()).toContain('fre 21.40–lør 02.00')

    await fire(bars(wrapper, 'Bil 1')[0].element, 'pointerleave')
    expect(row(wrapper, 'Bil 1').find('[data-bar-card]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('says how many hours a unit is rostered, or that it is not', () => {
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a')],
    })
    expect(nameCell(wrapper, 'Bil 1').text()).toContain('8t')
    expect(nameCell(wrapper, 'Bil 2').text()).toContain('ingen')
    wrapper.unmount()
  })

  it('reports the hours of the race nobody covers', () => {
    // The number the screen exists to drive to zero, and it is the sum across units — no single
    // row can show it.
    const wrapper = mountRoster({
      duty: [
        duty('bil-1', event.startUts, local(2026, 9, 19, 2), 'a'),
        duty('bil-2', local(2026, 9, 19, 6), event.endUts, 'b'),
      ],
    })
    expect(wrapper.text()).toContain('4 timer af løbet er ingen enhed på vagt')
    wrapper.unmount()
  })

  it('says so when the race is covered end to end', () => {
    const wrapper = mountRoster({ duty: [duty('bil-1', event.startUts, event.endUts, 'a')] })
    expect(wrapper.text()).toContain('Hele løbet er dækket')
    wrapper.unmount()
  })
})

describe('DispatchDutyRoster now marker', () => {
  const nowMs = local(2026, 9, 19, 12) * 1000

  it('marks now in every unit’s row, at the same instant', () => {
    // One straight edge through the roster: "who covers 02:00?" is asked about *now* more often than
    // about any other moment, and an edge per row computed from its own clock would be ragged.
    const wrapper = mountRoster({ nowMs })
    const markers = wrapper.findAll('[data-now]')
    expect(markers).toHaveLength(units.length)
    for (const marker of markers) {
      expect(marker.attributes('style')).toContain('left: 50%')
    }
    wrapper.unmount()
  })

  it('draws it under the shifts and over the track', async () => {
    // Asked for, and right: the marker must not obscure the thing being read. The order is five
    // explicit layers in one parent — stripe, gridline, now, bar, knot — because anything subtler can
    // be undone by whatever an ancestor happens to paint.
    //
    // Asserted against the stylesheet in the source: jsdom neither applies scoped styles nor even
    // injects them, so there is nothing in the document to measure. The declared order *is* the
    // invariant, and it is the thing a later edit would break.
    const { readFileSync } = await import('node:fs')
    const css = readFileSync('src/components/DutyRangeSlider.vue', 'utf8')
    const zOf = (className: string) => {
      const rule = new RegExp(`\\.${className}\\s*{[^}]*?z-index:\\s*(-?\\d+)`).exec(css)
      expect(rule, `.${className} declares no z-index`).not.toBeNull()
      return Number(rule![1])
    }
    expect(zOf('duty-bg')).toBeLessThan(zOf('duty-now'))
    expect(zOf('duty-now')).toBeLessThan(zOf('duty-bar'))
    expect(zOf('duty-bar')).toBeLessThan(zOf('duty-knot-hit'))

    // And the elements the layers refer to are really in one parent, which is what makes the order
    // mean anything: z-index only orders siblings.
    const wrapper = mountRoster({
      nowMs,
      duty: [duty('bil-1', local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a')],
    })
    const parent = track(wrapper, 'Bil 1').element
    for (const selector of ['.duty-bg', '[data-now]', '[data-bar]', '[data-knot]']) {
      expect(parent.querySelector(selector)?.parentElement, selector).toBe(parent)
    }
    wrapper.unmount()
  })

  it('is absent outside the race, rather than pinned to an edge', () => {
    // A line at the left edge in the week before the race reads as "the race started at midnight and
    // it is now", which is a lie an operator would plan against.
    const before = mountRoster({ nowMs: (event.startUts - 3600) * 1000 })
    expect(before.findAll('[data-now]')).toHaveLength(0)
    before.unmount()

    const after = mountRoster({ nowMs: (event.endUts + 3600) * 1000 })
    expect(after.findAll('[data-now]')).toHaveLength(0)
    after.unmount()
  })

  it('is absent when the board passes no clock', () => {
    expect(mountRoster().findAll('[data-now]')).toHaveLength(0)
  })

  it('offers to scroll to now, and only while now is in the race', () => {
    // Zoomed in, "where are we" is the first question and the scrollbar is a poor way to answer it.
    const during = mountRoster({ nowMs })
    expect(during.findAll('button').some((b) => b.text().includes('Nu'))).toBe(true)
    during.unmount()

    const outside = mountRoster({ nowMs: (event.endUts + 3600) * 1000 })
    expect(outside.findAll('button').some((b) => b.text().includes('Nu'))).toBe(false)
    outside.unmount()
  })
})

describe('DispatchDutyRoster zoom', () => {
  const zoomLabel = (wrapper: VueWrapper) =>
    wrapper.findAll('span').find((s) => /^\d+×$/.test(s.text()))?.text()

  const zoomButton = (wrapper: VueWrapper, direction: 'in' | 'out') =>
    wrapper
      .findAll('button')
      .find((b) => b.attributes('aria-label') === (direction === 'in' ? 'Zoom ind' : 'Zoom ud'))

  it('starts at “the whole race fits”, so nothing has to be scrolled to be seen', () => {
    const wrapper = mountRoster()
    expect(zoomLabel(wrapper)).toBe('1×')
    expect(wrapper.find('[data-scroller] > div').attributes('style')).toContain('width: 100%')
    wrapper.unmount()
  })

  it('widens the canvas as a multiple of the container', async () => {
    // A multiple rather than pixels per hour: 1× then means "it fits" on a laptop in a tent and on
    // the big screen in HQ alike, with no breakpoints.
    const wrapper = mountRoster()
    await zoomButton(wrapper, 'in')!.trigger('click')
    expect(wrapper.find('[data-scroller] > div').attributes('style')).toContain('width: 200%')
    wrapper.unmount()
  })

  it('stops at both ends instead of wrapping', async () => {
    // Wrapping from 8× back to 1× on a click of “+” would throw away the operator’s place with no
    // way to tell it had happened.
    const wrapper = mountRoster()
    expect(zoomButton(wrapper, 'out')!.attributes('disabled')).toBeDefined()
    for (let i = 0; i < 6; i++) await zoomButton(wrapper, 'in')!.trigger('click')
    expect(zoomLabel(wrapper)).toBe('8×')
    expect(zoomButton(wrapper, 'in')!.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('keeps the bars positioned in percentages, so zoom needs no arithmetic of its own', async () => {
    // The axis is the canvas, and the canvas is what gets wider. A bar placed in pixels would have to
    // be recomputed on every zoom, and the drag maths with it.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 19, 12), local(2026, 9, 19, 18), 'a')],
    })
    const before = bars(wrapper, 'Bil 1')[0].attributes('style')
    await zoomButton(wrapper, 'in')!.trigger('click')
    expect(bars(wrapper, 'Bil 1')[0].attributes('style')).toBe(before)
    wrapper.unmount()
  })
})

describe('DispatchDutyRoster adding and removing', () => {
  it('puts the add button after the timeline, where it cannot scroll out of reach', () => {
    // In its own column rather than inside the scroller: at 8× a button on the canvas would be three
    // screens to the right of the row it belongs to.
    const wrapper = mountRoster()
    expect(addButton(wrapper, 'Bil 1').text()).toContain('Vagt')
    expect(wrapper.find('[data-scroller]').findAll('button')).toHaveLength(0)
    wrapper.unmount()
  })

  it('adds a pair at the end of the timeline', () => {
    // Appended to the list *and* to the axis: anywhere else and the visual order would disagree
    // with the stored order, and it would be unclear which bar was just created.
    const wrapper = mountRoster()
    addButton(wrapper, 'Bil 2').trigger('click')

    expect(wrapper.emitted('save')).toEqual([
      [
        {
          sectionSlug: 'bil-2',
          startUts: event.endUts - NEW_PERIOD_MINUTES * 60,
          endUts: event.endUts,
        },
      ],
    ])
    wrapper.unmount()
  })

  it('adds without an id, so the API creates rather than edits', () => {
    const wrapper = mountRoster()
    addButton(wrapper, 'Bil 1').trigger('click')
    expect(wrapper.emitted('save')![0][0]).not.toHaveProperty('dutyId')
    wrapper.unmount()
  })

  it('never puts a new pair on top of an existing one', () => {
    const existing = duty('bil-1', local(2026, 9, 20, 23), event.endUts - 30 * 60, 'late')
    const wrapper = mountRoster({ duty: [existing] })
    addButton(wrapper, 'Bil 1').trigger('click')

    const added = wrapper.emitted('save')![0][0] as { startUts: number }
    expect(added.startUts).toBeGreaterThanOrEqual(existing.endUts)
    wrapper.unmount()
  })

  it('still adds something for a unit rostered to the very end', () => {
    // A button that does nothing is worse than a sliver to drag out.
    const wrapper = mountRoster({ duty: [duty('bil-1', event.startUts, event.endUts, 'all')] })
    addButton(wrapper, 'Bil 1').trigger('click')

    const added = wrapper.emitted('save')![0][0] as { startUts: number; endUts: number }
    expect(added.endUts - added.startUts).toBe(MIN_PERIOD_MINUTES * 60)
    wrapper.unmount()
  })

  it('removes the period that was hovered, not the row', async () => {
    // Removal lives on the hover card, which is where it went when the chips did. It is the only
    // gesture that needs a target bigger than a bar, and hovering is what says *which* bar.
    const wrapper = mountRoster({
      duty: [
        duty('bil-1', local(2026, 9, 18, 20), local(2026, 9, 19, 2), 'first'),
        duty('bil-1', local(2026, 9, 19, 20), local(2026, 9, 20, 2), 'second'),
      ],
    })
    // Nothing to click before hovering: one card at a time, for the bar under the cursor.
    expect(row(wrapper, 'Bil 1').findAll('[data-remove]')).toHaveLength(0)

    await fire(bars(wrapper, 'Bil 1')[1].element, 'pointerenter')
    const removes = row(wrapper, 'Bil 1').findAll('[data-remove]')
    expect(removes).toHaveLength(1)
    await removes[0].trigger('click')
    expect(wrapper.emitted('remove')).toEqual([['second']])
    wrapper.unmount()
  })

  it('does not start a drag when the remove button is pressed', async () => {
    // The button sits on top of the bar, so its pointerdown would otherwise grab the shift and the
    // click would both remove it and save a move of it.
    const wrapper = mountRoster({ duty: [duty('bil-1', local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a')] })
    await fire(bars(wrapper, 'Bil 1')[0].element, 'pointerenter')
    await fire(row(wrapper, 'Bil 1').find('[data-remove]').element, 'pointerdown', local(2026, 9, 19, 12))
    expect(wrapper.emitted('dragging')).toBeUndefined()
    wrapper.unmount()
  })
})

describe('DispatchDutyRoster dragging', () => {
  const night = duty('bil-1', local(2026, 9, 18, 21), local(2026, 9, 19, 3), 'night')

  it('moves only the knot that was grabbed', async () => {
    const wrapper = mountRoster({ duty: [night] })
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[1].element, night.endUts, local(2026, 9, 19, 6))

    expect(wrapper.emitted('save')).toEqual([
      [
        {
          dutyId: 'night',
          sectionSlug: 'bil-1',
          startUts: night.startUts,
          endUts: local(2026, 9, 19, 6),
        },
      ],
    ])
    wrapper.unmount()
  })

  it('moves the whole period when the bar is dragged, keeping its length', async () => {
    // "They start two hours later" is not "they work two hours more".
    const wrapper = mountRoster({ duty: [night] })
    // Grabbed at its own start, so the offset within the bar is zero and the move is exact.
    await dragTo(
      wrapper,
      'Bil 1',
      bars(wrapper, 'Bil 1')[0].element,
      night.startUts,
      local(2026, 9, 19, 21),
    )

    expect(wrapper.emitted('save')![0][0]).toEqual({
      dutyId: 'night',
      sectionSlug: 'bil-1',
      startUts: local(2026, 9, 19, 21),
      endUts: local(2026, 9, 20, 3),
    })
    wrapper.unmount()
  })

  it('holds a bar where it was grabbed, rather than jumping it under the cursor', async () => {
    // Grabbed two hours into the bar and dropped at Saturday 12:00, the shift starts at 10:00. The
    // other behaviour throws a shift hours off on an axis this wide.
    const wrapper = mountRoster({ duty: [night] })
    await dragTo(
      wrapper,
      'Bil 1',
      bars(wrapper, 'Bil 1')[0].element,
      night.startUts + 2 * 3600,
      local(2026, 9, 19, 12),
    )

    expect(wrapper.emitted('save')![0][0]).toMatchObject({ startUts: local(2026, 9, 19, 10) })
    wrapper.unmount()
  })

  it('shows the time being chosen while the knot is held', async () => {
    const wrapper = mountRoster({ duty: [night] })
    await fire(knots(wrapper, 'Bil 1')[1].element, 'pointerdown', night.endUts)
    await fire(track(wrapper, 'Bil 1').element, 'pointermove', local(2026, 9, 19, 6))

    expect(row(wrapper, 'Bil 1').text()).toContain('lør 06.00')
    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })

  it('commits once, on release, not on every step', async () => {
    // One PUT per quarter-hour crossed would be dozens of writes for one drag, each a projection, a
    // live signal and a revalidation of every open board.
    const wrapper = mountRoster({ duty: [night] })
    await fire(knots(wrapper, 'Bil 1')[1].element, 'pointerdown', night.endUts)
    for (const hour of [4, 5, 6]) {
      await fire(track(wrapper, 'Bil 1').element, 'pointermove', local(2026, 9, 19, hour))
    }
    expect(wrapper.emitted('save')).toBeUndefined()

    await fire(track(wrapper, 'Bil 1').element, 'pointerup')
    expect(wrapper.emitted('save')).toHaveLength(1)
    wrapper.unmount()
  })

  it('does not write when nothing moved', async () => {
    // A click on a bar, or a drag and a change of mind. A no-op PUT still costs every open board a
    // revalidation.
    const wrapper = mountRoster({ duty: [night] })
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[1].element, night.endUts, night.endUts)

    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })

  it('drops the draft on release, so the row shows what was saved', async () => {
    const wrapper = mountRoster({ duty: [night] })
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[1].element, night.endUts, local(2026, 9, 19, 6))
    // The payload has not changed — it arrives later, by revalidation — so the bar must be back to
    // what the server holds rather than staying where it was dropped.
    expect(bars(wrapper, 'Bil 1')[0].find('[data-bar-label]').text()).toBe('21.00–03.00')
    wrapper.unmount()
  })

  it('edits the window whose knot was grabbed when a unit has several', async () => {
    // The silent failure this file exists for: with the pairs sorted, or the index mistaken for a
    // position, this moves the wrong shift and still looks right.
    const late = duty('bil-1', local(2026, 9, 20, 8), local(2026, 9, 20, 12), 'late')
    const wrapper = mountRoster({
      duty: [late, duty('bil-1', local(2026, 9, 18, 8), local(2026, 9, 18, 12), 'early')],
    })
    // Knot 0 is the first *stored* window's start — 'late' — not the earliest on the axis.
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[0].element, late.startUts, local(2026, 9, 20, 6))
    expect(wrapper.emitted('save')![0][0]).toMatchObject({ dutyId: 'late' })
    wrapper.unmount()
  })

  it('stops a knot against its partner instead of inverting the period', async () => {
    const wrapper = mountRoster({ duty: [night] })
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[0].element, night.startUts, local(2026, 9, 19, 20))

    const saved = wrapper.emitted('save')![0][0] as { startUts: number; endUts: number }
    expect(saved.endUts - saved.startUts).toBe(MIN_PERIOD_MINUTES * 60)
    wrapper.unmount()
  })

  it('reports the gesture upward, so the board can stop redrawing under it', async () => {
    // The board's pause reason is the *gesture*, not the dialog being open — see DispatchView.spec.ts
    // for the bug that taught us the difference.
    const wrapper = mountRoster({ duty: [night] })
    await fire(knots(wrapper, 'Bil 1')[1].element, 'pointerdown', night.endUts)
    expect(wrapper.emitted('dragging')).toEqual([[true]])

    await fire(track(wrapper, 'Bil 1').element, 'pointermove', local(2026, 9, 19, 6))
    await fire(track(wrapper, 'Bil 1').element, 'pointerup')
    expect(wrapper.emitted('dragging')).toEqual([[true], [false]])
    wrapper.unmount()
  })

  it('ignores a drag while a write is in flight', async () => {
    const wrapper = mountRoster({ duty: [night], busy: true })
    await dragTo(wrapper, 'Bil 1', knots(wrapper, 'Bil 1')[1].element, night.endUts, local(2026, 9, 19, 6))
    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })

  it('disables the row’s buttons while a write is in flight', () => {
    const wrapper = mountRoster({ duty: [night], busy: true })
    expect(
      row(wrapper, 'Bil 1')
        .findAll('button')
        .every((b) => b.attributes('disabled') !== undefined),
    ).toBe(true)
    wrapper.unmount()
  })
})

describe('DispatchDutyRoster keyboard', () => {
  const shift = duty('bil-1', local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a')

  it('moves a knot by a step with the arrow keys', async () => {
    // Not a nicety here: the operator is often on a laptop with no room for a mouse, and a
    // quarter-hour by key is more precise than aiming at two pixels.
    const wrapper = mountRoster({ duty: [shift] })
    await knots(wrapper, 'Bil 1')[1].trigger('keydown', { key: 'ArrowRight' })

    expect(wrapper.emitted('save')![0][0]).toEqual({
      dutyId: 'a',
      sectionSlug: 'bil-1',
      startUts: shift.startUts,
      endUts: shift.endUts + 15 * 60,
    })
    wrapper.unmount()
  })

  it('moves by an hour with shift held', async () => {
    const wrapper = mountRoster({ duty: [shift] })
    await knots(wrapper, 'Bil 1')[0].trigger('keydown', { key: 'ArrowLeft', shiftKey: true })
    expect(wrapper.emitted('save')![0][0]).toMatchObject({ startUts: shift.startUts - 3600 })
    wrapper.unmount()
  })

  it('announces which edge of which shift a knot is', () => {
    const wrapper = mountRoster({ duty: [shift] })
    expect(knots(wrapper, 'Bil 1')[0].attributes('aria-valuetext')).toBe('Fra lør 08.00')
    expect(knots(wrapper, 'Bil 1')[1].attributes('aria-valuetext')).toBe('Til lør 16.00')
    wrapper.unmount()
  })

  it('ignores keys while a write is in flight', async () => {
    const wrapper = mountRoster({ duty: [shift], busy: true })
    await knots(wrapper, 'Bil 1')[1].trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })
})

describe('DispatchDutyRoster empty states', () => {
  it('sends the operator to Organisation when there are no units', () => {
    const wrapper = mountRoster({ units: [] })
    expect(wrapper.text()).toContain('Organisation')
    expect(wrapper.find('table').exists()).toBe(false)
    wrapper.unmount()
  })

  it('sends them to År when the race has no dates, rather than drawing a guessed axis', () => {
    const wrapper = mountRoster({ event: { startUts: 0, endUts: 0 } })
    expect(wrapper.text()).toContain('datoer')
    expect(wrapper.find('table').exists()).toBe(false)
    wrapper.unmount()
  })

  it('survives a board payload with no event at all', () => {
    const wrapper = mountRoster({ event: undefined })
    expect(wrapper.find('table').exists()).toBe(false)
    wrapper.unmount()
  })

  it('never writes on its own: the board owns the PUT and the pause it implies', async () => {
    const { readFileSync } = await import('node:fs')
    for (const file of ['DispatchDutyRoster.vue', 'DutyRangeSlider.vue']) {
      expect(readFileSync(`src/components/${file}`, 'utf8')).not.toContain('plugins/axios')
    }
  })
})

describe('DispatchDutyRoster is visible at all', () => {
  // A section of its own, because "the operator can see it" turned out to be a property with no
  // test — and the one that actually failed. Everything below the surface was right.

  it('takes its colours from variables the theme really defines', async () => {
    // The naming, verified against the theme PrimeVue injects rather than against a memory of it:
    // Aura emits `--p-`-prefixed tokens, and the unprefixed names Tailwind is configured with do not
    // exist. If a PrimeVue upgrade changes the prefix, this fails here instead of on race night.
    const wrapper = mountRoster({
      duty: [duty('bil-1', local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a')],
    })
    const themeCss = Array.from(document.querySelectorAll('style'))
      .map((tag) => tag.textContent ?? '')
      .join('\n')
    expect(themeCss, 'no PrimeVue theme was injected; this test proves nothing').toContain(
      '--p-primary-400',
    )

    const { readFileSync } = await import('node:fs')
    const referenced = new Set<string>()
    for (const file of ['DispatchDutyRoster.vue', 'DutyRangeSlider.vue']) {
      for (const match of readFileSync(`src/components/${file}`, 'utf8').matchAll(
        /var\((--p-[\w-]+)/g,
      )) {
        referenced.add(match[1])
      }
    }
    expect(referenced.size, 'the slider must take its colours from the theme').toBeGreaterThan(0)
    for (const name of referenced) {
      expect(themeCss, `${name} is not a variable this theme defines`).toContain(`${name}:`)
    }
    wrapper.unmount()
  })

  it('gives every theme colour a literal fallback', async () => {
    // So that getting the variable name wrong — the bug this section exists for — degrades to a
    // visible slider in the wrong green, rather than an invisible one.
    const { readFileSync } = await import('node:fs')
    const source = readFileSync('src/components/DutyRangeSlider.vue', 'utf8')
    for (const match of source.matchAll(/var\(--p-[\w-]+([^)]*)\)/g)) {
      expect(match[1].trim(), `${match[0]} has no fallback`).toMatch(/^,\s*\S/)
    }
  })

  it('uses no colour token that this app leaves undefined', async () => {
    // This control was once invisible in the browser while every test here passed. Tailwind is
    // configured with `--surface-200` / `--primary-400` — PrimeVue *3* token names — while PrimeVue 4
    // emits them prefixed (`--p-surface-200`). So a utility class like bg-surface-* compiles against
    // a variable nothing defines and resolves to transparent: the track, the bars and both knots
    // rendered correctly and could not be seen.
    //
    // Asserted against the source because there is nothing to assert at runtime — jsdom computes no
    // styles, and a class that resolves to nothing is still present in the DOM. Anything needing
    // theme colour goes through `var(--p-*, literal)` in a style block instead.
    const { readFileSync } = await import('node:fs')
    for (const file of ['DispatchDutyRoster.vue', 'DutyRangeSlider.vue']) {
      const source = readFileSync(`src/components/${file}`, 'utf8')
      const dead = source.match(/(?<![-\w])(?:bg|text|border|ring|outline|from|to|via)-(?:surface|primary)-\d{2,3}/g)
      expect(dead, `${file} uses Tailwind tokens this app does not define`).toBeNull()
    }
  })
})
