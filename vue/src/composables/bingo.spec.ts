import { describe, expect, it } from 'vitest'
import { areaPath, countAt, hourTicks, niceMax, stepPath, tickStepHours } from './bingo'

const HOUR = 3600
const scale = { x: (uts: number) => uts, y: (count: number) => 100 - count }

describe('niceMax', () => {
  // An axis topping out at the exact maximum makes every gridline a number nobody can
  // divide by eye, and changes the curve's shape whenever one more team signs up.
  it('rounds a count up to something readable', () => {
    expect(niceMax(76)).toBe(100)
    expect(niceMax(12)).toBe(20)
    expect(niceMax(3)).toBe(5)
  })

  // An empty race still needs an axis to draw on, not a division by zero.
  it('never drops below one', () => {
    expect(niceMax(0)).toBe(1)
    expect(niceMax(-5)).toBe(1)
    expect(niceMax(Number.NaN)).toBe(1)
  })
})

describe('tickStepHours', () => {
  it('keeps the label count around eight', () => {
    // Friday 21.00 to Sunday 04.00 is 31 hours.
    expect(tickStepHours(0, 31 * HOUR)).toBe(4)
    expect(tickStepHours(0, 6 * HOUR)).toBe(1)
  })
})

describe('hourTicks', () => {
  // A label reading "20.47" is not a time anybody is working against.
  it('lands on whole hours, not on the start of the window', () => {
    const start = new Date(2026, 6, 10, 20, 47, 0)
    const end = new Date(2026, 6, 10, 23, 30, 0)
    const ticks = hourTicks(start.getTime() / 1000, end.getTime() / 1000, 1)
    expect(ticks.map((t) => new Date(t * 1000).getMinutes())).toEqual([0, 0, 0])
    expect(ticks.map((t) => new Date(t * 1000).getHours())).toEqual([21, 22, 23])
  })

  it('returns nothing for a window with no width', () => {
    expect(hourTicks(1000, 1000, 1)).toEqual([])
    expect(hourTicks(2000, 1000, 1)).toEqual([])
  })
})

describe('stepPath', () => {
  // A diagonal would claim the count changed gradually, and every reading taken off the
  // slope would be a number that never held.
  it('draws along then up, never diagonally', () => {
    expect(
      stepPath(
        [
          { uts: 0, count: 3 },
          { uts: 10, count: 2 }
        ],
        scale
      )
    ).toBe('M 0 97 L 10 97 L 10 98')
  })

  it('is empty for an empty series', () => {
    expect(stepPath([], scale)).toBe('')
    expect(areaPath([], scale, 100)).toBe('')
  })

  it('closes the area down to the baseline', () => {
    expect(areaPath([{ uts: 0, count: 1 }], scale, 100)).toBe('M 0 99 L 0 100 L 0 100 Z')
  })
})

describe('countAt', () => {
  const points = [
    { uts: 100, count: 5 },
    { uts: 200, count: 3 }
  ]

  // The last point at or before the time: that is what a step function means.
  it('holds the previous count until the next step', () => {
    expect(countAt(points, 100)).toBe(5)
    expect(countAt(points, 199)).toBe(5)
    expect(countAt(points, 200)).toBe(3)
    expect(countAt(points, 9999)).toBe(3)
  })

  // The curve makes no claim about a time it does not cover.
  it('is undefined before the series starts', () => {
    expect(countAt(points, 99)).toBeUndefined()
  })
})
