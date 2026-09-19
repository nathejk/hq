import { describe, it, expect } from 'vitest'
import {
  eventDays,
  percentOf,
  utsAt,
  pairsOn,
  dragKnot,
  dragPair,
  newPeriod,
  rosteredHours,
  uncoveredHours,
  nowPercent,
  zoomBy,
  barLabel,
  spanLabel,
  ZOOM_STEPS,
  STEP_MINUTES,
  MIN_PERIOD_MINUTES,
  NEW_PERIOD_MINUTES,
} from './dutyTimeline'
import type { Duty } from './dispatch'

// A real year edition: 2026 runs Friday 2026-09-18 to Sunday 2026-09-20, so the axis is the three
// days from Friday midnight to Monday midnight. Times are constructed as local dates, because that
// is what the axis works in — a UTC literal here would make the suite pass or fail depending on the
// machine's zone.
const local = (y: number, m: number, d: number, hh = 0, mm = 0) =>
  Math.floor(new Date(y, m - 1, d, hh, mm, 0, 0).getTime() / 1000)

const axis = { startUts: local(2026, 9, 18), endUts: local(2026, 9, 21) }
const HOUR = 3600

const duty = (startUts: number, endUts: number, id = 'd1'): Duty => ({
  id,
  year: '2026',
  sectionSlug: 'bil-1',
  startUts,
  endUts,
})

describe('eventDays', () => {
  it('gives one gridline per calendar day of the race', () => {
    expect(eventDays(axis.startUts, axis.endUts).map((d) => d.label)).toEqual([
      'fre 18/9',
      'lør 19/9',
      'søn 20/9',
    ])
  })

  it('tiles without gaps, so the gridlines meet', () => {
    const days = eventDays(axis.startUts, axis.endUts)
    expect(days[0].endUts).toBe(days[1].startUts)
    expect(days[1].endUts).toBe(days[2].startUts)
  })

  it('is empty for an unset or inverted span, rather than inventing a day', () => {
    expect(eventDays(0, 0)).toEqual([])
    expect(eventDays(axis.startUts, 0)).toEqual([])
    expect(eventDays(axis.endUts, axis.startUts)).toEqual([])
  })

  it('refuses to render an unbounded number of gridlines', () => {
    expect(eventDays(axis.startUts, axis.startUts + 400 * 86400).length).toBeLessThanOrEqual(21)
  })
})

describe('percentOf', () => {
  it('maps the ends of the axis to the ends of the track', () => {
    expect(percentOf(axis.startUts, axis)).toBe(0)
    expect(percentOf(axis.endUts, axis)).toBe(100)
  })

  it('puts Saturday midday halfway through a three-day race', () => {
    expect(percentOf(local(2026, 9, 19, 12), axis)).toBeCloseTo(50, 5)
  })

  it('clamps a window that falls outside the axis to its edge', () => {
    // A roster kept while the year's dates were corrected. Drawn at the edge, not off the track:
    // the shift is real, the axis is what is wrong, and the bar has to stay grabbable.
    expect(percentOf(axis.startUts - 10 * HOUR, axis)).toBe(0)
    expect(percentOf(axis.endUts + 10 * HOUR, axis)).toBe(100)
  })

  it('is zero rather than NaN on a collapsed axis', () => {
    expect(percentOf(axis.startUts, { startUts: axis.startUts, endUts: axis.startUts })).toBe(0)
  })
})

describe('utsAt', () => {
  it('snaps to the nearest step, so a drag lands where the operator meant', () => {
    const down = utsAt(percentOf(local(2026, 9, 19, 12, 7), axis), axis)
    expect(down % (STEP_MINUTES * 60)).toBe(0)
    expect(down).toBe(local(2026, 9, 19, 12, 0))
    expect(utsAt(percentOf(local(2026, 9, 19, 12, 8), axis), axis)).toBe(local(2026, 9, 19, 12, 15))
  })

  it('round-trips a time that is already on the step', () => {
    const t = local(2026, 9, 19, 21, 30)
    expect(utsAt(percentOf(t, axis), axis)).toBe(t)
  })

  it('stays on the axis for a pointer dragged off either end', () => {
    expect(utsAt(-40, axis)).toBe(axis.startUts)
    expect(utsAt(140, axis)).toBe(axis.endUts)
  })
})

describe('pairsOn', () => {
  it('places a period as a bar, left edge and width', () => {
    const [pair] = pairsOn([duty(local(2026, 9, 19, 12), local(2026, 9, 19, 18))], axis)
    expect(pair.fromPercent).toBeCloseTo(50, 5)
    expect(pair.widthPercent).toBeCloseTo((6 / 72) * 100, 5)
    expect(pair.clipped).toBe(false)
  })

  it('keeps a night shift as one unbroken bar', () => {
    // The whole reason the axis is not split per day: this is the ordinary case on this screen,
    // and as two bars in two cells the seam fell exactly where the race is hardest to staff.
    const [pair] = pairsOn([duty(local(2026, 9, 18, 21, 40), local(2026, 9, 19, 2))], axis)
    expect(pair.widthPercent).toBeCloseTo(((4 + 20 / 60) / 72) * 100, 5)
  })

  it('never renders a bar too thin to grab', () => {
    // Fifteen minutes of a three-day race is 0.3% of the track, which rounds to nothing.
    const [pair] = pairsOn([duty(local(2026, 9, 19, 12), local(2026, 9, 19, 12, 15))], axis)
    expect(pair.widthPercent).toBeGreaterThanOrEqual(0.4)
  })

  it('preserves the stored order instead of sorting by time', () => {
    // Load-bearing: a knot carries the id of the window it edits, so a slider that reordered its
    // handles when one was dragged past another would move a different shift, silently.
    const late = duty(local(2026, 9, 20, 8), local(2026, 9, 20, 12), 'late')
    const early = duty(local(2026, 9, 18, 8), local(2026, 9, 18, 12), 'early')
    expect(pairsOn([late, early], axis).map((p) => p.duty.id)).toEqual(['late', 'early'])
  })

  it('marks a period that hangs off the axis', () => {
    expect(pairsOn([duty(axis.startUts - HOUR, axis.startUts + HOUR)], axis)[0].clipped).toBe(true)
    expect(pairsOn([duty(axis.endUts - HOUR, axis.endUts + HOUR)], axis)[0].clipped).toBe(true)
  })
})

describe('dragKnot', () => {
  const window = duty(local(2026, 9, 19, 8), local(2026, 9, 19, 16))

  it('moves only the edge being dragged', () => {
    // Adjusting the end of a shift must not quietly move its start.
    expect(dragKnot(window, 'end', local(2026, 9, 19, 20), axis)).toEqual({
      startUts: window.startUts,
      endUts: local(2026, 9, 19, 20),
    })
    expect(dragKnot(window, 'start', local(2026, 9, 19, 6), axis)).toEqual({
      startUts: local(2026, 9, 19, 6),
      endUts: window.endUts,
    })
  })

  it('stops an edge against its partner instead of turning the window inside out', () => {
    // Inverted, a window reads as a completely different shift — and the API rejects it anyway.
    const dragged = dragKnot(window, 'start', local(2026, 9, 19, 20), axis)
    expect(dragged.endUts - dragged.startUts).toBe(MIN_PERIOD_MINUTES * 60)
    expect(dragged.endUts).toBe(window.endUts)

    const other = dragKnot(window, 'end', local(2026, 9, 19, 4), axis)
    expect(other.endUts - other.startUts).toBe(MIN_PERIOD_MINUTES * 60)
    expect(other.startUts).toBe(window.startUts)
  })

  it('keeps the edge on the axis', () => {
    expect(dragKnot(window, 'start', axis.startUts - 5 * HOUR, axis).startUts).toBe(axis.startUts)
    expect(dragKnot(window, 'end', axis.endUts + 5 * HOUR, axis).endUts).toBe(axis.endUts)
  })
})

describe('dragPair', () => {
  const window = duty(local(2026, 9, 19, 8), local(2026, 9, 19, 16))

  it('moves the whole period without changing its length', () => {
    // "They start two hours later" is not "they work two hours more".
    const moved = dragPair(window, local(2026, 9, 19, 10), axis)
    expect(moved).toEqual({ startUts: local(2026, 9, 19, 10), endUts: local(2026, 9, 19, 18) })
  })

  it('shifts back inside the axis rather than squashing against the end', () => {
    // At the edges the length is what has to survive: a shift dragged past the end of the race is
    // still eight hours long, it just cannot be there.
    const moved = dragPair(window, axis.endUts - HOUR, axis)
    expect(moved.endUts).toBe(axis.endUts)
    expect(moved.endUts - moved.startUts).toBe(window.endUts - window.startUts)

    const back = dragPair(window, axis.startUts - HOUR, axis)
    expect(back.startUts).toBe(axis.startUts)
    expect(back.endUts - back.startUts).toBe(window.endUts - window.startUts)
  })
})

describe('newPeriod', () => {
  it('lands at the end of the timeline for a unit with nothing rostered', () => {
    expect(newPeriod([], axis)).toEqual({
      startUts: axis.endUts - NEW_PERIOD_MINUTES * 60,
      endUts: axis.endUts,
    })
  })

  it('lands at the end of the timeline, not after the last window', () => {
    // Appended to the list *and* to the axis: anywhere else and the visual order would disagree
    // with the stored order, and it would be unclear which bar was just created.
    const existing = [duty(local(2026, 9, 18, 20), local(2026, 9, 19, 2), 'night')]
    expect(newPeriod(existing, axis).endUts).toBe(axis.endUts)
  })

  it('never overlaps what is already there', () => {
    // No doubt about which bar is the new one under the cursor.
    const existing = [duty(local(2026, 9, 20, 23), axis.endUts - 30 * 60, 'late')]
    expect(newPeriod(existing, axis).startUts).toBeGreaterThanOrEqual(existing[0].endUts)
  })

  it('shrinks to a sliver rather than refusing, for a unit rostered to the very end', () => {
    // A button that does nothing is worse than a short bar to drag out.
    const full = [duty(axis.startUts, axis.endUts, 'all')]
    const period = newPeriod(full, axis)
    expect(period.endUts - period.startUts).toBe(MIN_PERIOD_MINUTES * 60)
    expect(period.endUts).toBe(axis.endUts)
  })

  it('is always a period the API will accept', () => {
    for (const windows of [[], [duty(axis.startUts, axis.endUts, 'all')]]) {
      const period = newPeriod(windows, axis)
      expect(period.endUts).toBeGreaterThan(period.startUts)
      expect(period.startUts).toBeGreaterThanOrEqual(axis.startUts)
    }
  })
})

describe('nowPercent', () => {
  it('places now on the axis', () => {
    expect(nowPercent(local(2026, 9, 19, 12) * 1000, axis)).toBeCloseTo(50, 5)
  })

  it('is null before and after the race, rather than pinned to an edge', () => {
    // A line at the left edge in the week before the race does not read as "not yet" — it reads as
    // "the race started at midnight and it is now", which is a lie an operator would plan against.
    expect(nowPercent((axis.startUts - 3600) * 1000, axis)).toBeNull()
    expect(nowPercent((axis.endUts + 3600) * 1000, axis)).toBeNull()
  })

  it('is null when the race has no dates', () => {
    expect(nowPercent(Date.now(), { startUts: 0, endUts: 0 })).toBeNull()
  })

  it('includes both ends of the race', () => {
    expect(nowPercent(axis.startUts * 1000, axis)).toBe(0)
    expect(nowPercent(axis.endUts * 1000, axis)).toBe(100)
  })
})

describe('zoomBy', () => {
  it('steps through the levels', () => {
    expect(zoomBy(1, 1)).toBe(2)
    expect(zoomBy(2, 1)).toBe(4)
    expect(zoomBy(4, -1)).toBe(2)
  })

  it('clamps at both ends rather than wrapping', () => {
    // Wrapping from 8× back to 1× on a click of "+" would throw away the operator's place on the
    // axis with no way to tell it happened.
    expect(zoomBy(ZOOM_STEPS[0], -1)).toBe(ZOOM_STEPS[0])
    expect(zoomBy(ZOOM_STEPS[ZOOM_STEPS.length - 1], 1)).toBe(ZOOM_STEPS[ZOOM_STEPS.length - 1])
  })

  it('recovers from a level that is not on the scale', () => {
    expect(ZOOM_STEPS).toContain(zoomBy(3, 1))
  })

  it('starts at “it fits”, so the default needs no scrolling', () => {
    expect(ZOOM_STEPS[0]).toBe(1)
  })
})

describe('barLabel', () => {
  it('is the clock alone, because the bar’s position already says the day', () => {
    // And because dropping the weekday is most of what makes the label fit inside the bar, which is
    // the whole point of putting it there.
    expect(barLabel({ startUts: local(2026, 9, 18, 21, 40), endUts: local(2026, 9, 19, 2) })).toBe(
      '21.40–02.00',
    )
  })
})

describe('spanLabel', () => {
  it('says the day once for a shift inside one day', () => {
    expect(spanLabel({ startUts: local(2026, 9, 19, 8), endUts: local(2026, 9, 19, 16) })).toBe(
      'lør 08.00–16.00',
    )
  })

  it('says both days for a shift through the night', () => {
    // "21.40 til 02.00" does not say which night, and this screen is mostly night shifts.
    expect(spanLabel({ startUts: local(2026, 9, 18, 21, 40), endUts: local(2026, 9, 19, 2) })).toBe(
      'fre 21.40–lør 02.00',
    )
  })
})

describe('rosteredHours', () => {
  it('sums a unit’s windows', () => {
    expect(
      rosteredHours([
        duty(local(2026, 9, 19, 8), local(2026, 9, 19, 16), 'a'),
        duty(local(2026, 9, 19, 20), local(2026, 9, 19, 23, 30), 'b'),
      ]),
    ).toBe(11.5)
  })

  it('counts an overlap once, because it is not two hours of cover', () => {
    expect(
      rosteredHours([
        duty(local(2026, 9, 19, 8), local(2026, 9, 19, 12), 'a'),
        duty(local(2026, 9, 19, 10), local(2026, 9, 19, 14), 'b'),
      ]),
    ).toBe(6)
  })

  it('is zero for a unit with no roster', () => {
    expect(rosteredHours([])).toBe(0)
  })
})

describe('uncoveredHours', () => {
  it('is the whole race when nobody is rostered', () => {
    expect(uncoveredHours([], axis)).toBe(72)
  })

  it('is zero when the race is covered end to end', () => {
    expect(uncoveredHours([duty(axis.startUts, axis.endUts, 'all')], axis)).toBe(0)
  })

  it('counts the gap between two units’ shifts, which is the number that matters', () => {
    // The hole in the night, across units: this is what the screen exists to drive to zero.
    expect(
      uncoveredHours(
        [
          duty(axis.startUts, local(2026, 9, 19, 2), 'a'),
          duty(local(2026, 9, 19, 6), axis.endUts, 'b'),
        ],
        axis,
      ),
    ).toBe(4)
  })

  it('does not credit cover from outside the race', () => {
    // A window hanging off the axis covers the hours inside it and no more, or a stale roster
    // would report a night as staffed by a shift that ended before the race began.
    expect(uncoveredHours([duty(axis.startUts - 48 * HOUR, axis.startUts, 'before')], axis)).toBe(72)
  })

  it('counts overlapping cover once', () => {
    expect(
      uncoveredHours(
        [
          duty(axis.startUts, local(2026, 9, 19, 12), 'a'),
          duty(local(2026, 9, 19, 6), axis.endUts, 'b'),
        ],
        axis,
      ),
    ).toBe(0)
  })
})
