import { describe, expect, it } from 'vitest'
import {
  ALLOWANCE_MINUTES,
  DEADLINE_WARNING_MINUTES,
  type Duty,
  type Task,
  type Tour,
  type TourStop,
  deadlineRisk,
  estimateFor,
  formatSpan,
  nextDutyStart,
  planTaskOntoStops,
  unitReadiness,
  unitsEngaged,
  unitsOnDuty,
  untilUts,
  taskRoute,
} from './dispatch'

// The arithmetic behind the two numbers the kørsel board is judged on: how long somebody has
// waited, and when a car might reach them. Worth testing rather than eyeballing, because both are
// read out loud to a patrol standing in the dark and neither is obvious from the screen.

const minute = 60
const nowMs = 1_787_860_000_000
const nowUts = nowMs / 1000

const task = (over: Partial<Task> = {}): Task => ({
  id: 't-1',
  year: '2026',
  kind: 'pickup',
  description: 'spejder ved Post 2B',
  pickup: { kind: 'text', label: 'ved Post 2B' },
  dropoff: { kind: 'hq', label: 'HQ' },
  state: 'queued',
  createdUts: nowUts - 20 * minute,
  notBeforeUts: null,
  deadlineUts: null,
  pickedUpUts: null,
  doneUts: null,
  cancelledUts: null,
  memberIds: [],
  ...over,
})

const window_ = (over: Partial<Duty> = {}): Duty => ({
  id: 'd-1',
  year: '2026',
  sectionSlug: 'bil-2',
  startUts: nowUts - 60 * minute,
  endUts: nowUts + 60 * minute,
  ...over,
})

describe('duty windows', () => {
  it('reports the units on duty at an instant', () => {
    const duty = [window_(), window_({ id: 'd-2', sectionSlug: 'bil-3', startUts: nowUts + 120 * minute, endUts: nowUts + 240 * minute })]
    expect([...unitsOnDuty(duty, nowMs)]).toEqual(['bil-2'])
  })

  // Half-open, matching the server's own Duty.Covers. Two consecutive windows must not both claim
  // the same minute, or one unit reads as being on duty twice.
  it('excludes the instant a window ends', () => {
    const duty = [window_({ startUts: nowUts - 60 * minute, endUts: nowUts })]
    expect(unitsOnDuty(duty, nowMs).size).toBe(0)
  })

  it('finds when the next unit comes on, ignoring windows already past', () => {
    const duty = [
      window_({ id: 'd-old', startUts: nowUts - 200 * minute, endUts: nowUts - 100 * minute }),
      window_({ id: 'd-soon', startUts: nowUts + 30 * minute, endUts: nowUts + 90 * minute }),
      window_({ id: 'd-later', startUts: nowUts + 300 * minute, endUts: nowUts + 400 * minute }),
    ]
    expect(nextDutyStart(duty, nowMs)).toBe(nowUts + 30 * minute)
  })

  it('has no next start when nothing is rostered ahead', () => {
    expect(nextDutyStart([], nowMs)).toBeNull()
  })
})

describe('the queued estimate', () => {
  // max(now, tidligst, next unit on duty) + allowance(kind).
  it('is now plus the kind allowance when a unit is already driving', () => {
    const duty = [window_()]
    expect(estimateFor(task(), duty, nowMs)).toBe(nowUts + ALLOWANCE_MINUTES.pickup * minute)
  })

  it('waits for the next unit when none is on duty', () => {
    const duty = [window_({ startUts: nowUts + 45 * minute, endUts: nowUts + 300 * minute })]
    expect(estimateFor(task(), duty, nowMs)).toBe(
      nowUts + 45 * minute + ALLOWANCE_MINUTES.pickup * minute,
    )
  })

  // A unit already driving *is* capacity. Pushing the estimate out to the next shift would be
  // pessimistic to the point of useless.
  it('does not wait for the next shift while a unit is on duty', () => {
    const duty = [window_(), window_({ id: 'd-2', startUts: nowUts + 600 * minute, endUts: nowUts + 700 * minute })]
    expect(estimateFor(task(), duty, nowMs)).toBe(nowUts + ALLOWANCE_MINUTES.pickup * minute)
  })

  it('respects tidligst', () => {
    const duty = [window_()]
    const t = task({ notBeforeUts: nowUts + 90 * minute })
    expect(estimateFor(t, duty, nowMs)).toBe(
      nowUts + 90 * minute + ALLOWANCE_MINUTES.pickup * minute,
    )
  })

  // The mitigation for the roster going stale (PRD 009 §8): with no duty data at all the estimate
  // degrades to now + allowance rather than to nonsense.
  it('degrades to now plus the allowance with no roster', () => {
    expect(estimateFor(task(), [], nowMs)).toBe(nowUts + ALLOWANCE_MINUTES.pickup * minute)
  })

  it('uses the allowance of the task kind', () => {
    const duty = [window_()]
    expect(estimateFor(task({ kind: 'delivery' }), duty, nowMs)).toBe(
      nowUts + ALLOWANCE_MINUTES.delivery * minute,
    )
  })
})

describe('deadline risk', () => {
  it('is none without a deadline', () => {
    expect(deadlineRisk(task(), null, nowMs)).toBe('none')
  })

  it('is late when the plan lands after the deadline', () => {
    const t = task({ state: 'planned', deadlineUts: nowUts + 30 * minute })
    expect(deadlineRisk(t, nowUts + 45 * minute, nowMs)).toBe('late')
  })

  it('is none when the plan lands inside the deadline', () => {
    const t = task({ state: 'planned', deadlineUts: nowUts + 60 * minute })
    expect(deadlineRisk(t, nowUts + 50 * minute, nowMs)).toBe('none')
  })

  it('is late once the deadline has passed', () => {
    expect(deadlineRisk(task({ deadlineUts: nowUts - minute }), null, nowMs)).toBe('late')
  })

  it('is soon for an unplanned task inside the warning window', () => {
    const t = task({ deadlineUts: nowUts + (DEADLINE_WARNING_MINUTES - 10) * minute })
    expect(deadlineRisk(t, null, nowMs)).toBe('soon')
  })

  it('is none for an unplanned task well before its deadline', () => {
    const t = task({ deadlineUts: nowUts + (DEADLINE_WARNING_MINUTES + 60) * minute })
    expect(deadlineRisk(t, null, nowMs)).toBe('none')
  })

  // A red row for dinner that was delivered on time is how a board teaches its operator to ignore
  // red rows.
  it('is none for finished and cancelled work, however late', () => {
    for (const state of ['done', 'cancelled'] as const) {
      const t = task({ state, deadlineUts: nowUts - 120 * minute })
      expect(deadlineRisk(t, null, nowMs)).toBe('none')
    }
  })
})

describe('engaged units', () => {
  const tour_ = (over: Partial<Tour> = {}): Tour => ({
    id: 'r-1',
    year: '2026',
    sectionSlug: 'bil-2',
    departureUts: null,
    state: 'planned',
    createdUts: nowUts - 10 * minute,
    underwayUts: null,
    completedUts: null,
    cancelledUts: null,
    stops: [],
    ...over,
  })

  it('counts a planned or underway tour as engaging its unit', () => {
    const tours = [tour_(), tour_({ id: 'r-2', sectionSlug: 'bil-3', state: 'underway' })]
    expect([...unitsEngaged(tours)].sort()).toEqual(['bil-2', 'bil-3'])
  })

  // A finished or abandoned run leaves the unit free again, which is the whole point of the split.
  it('leaves a unit free once its tour is completed or cancelled', () => {
    expect(unitsEngaged([tour_({ state: 'completed' })]).size).toBe(0)
    expect(unitsEngaged([tour_({ state: 'cancelled' })]).size).toBe(0)
  })
})

describe('unit readiness', () => {
  const unit = (vehicles: number, people: number) => ({
    sectionSlug: 'bil-2',
    label: 'Bil 2',
    vehicles: Array.from({ length: vehicles }, (_, i) => ({
      vehicleId: `v-${i}`,
      licensePlate: 'DK+AB12345',
      driverUserId: '',
      seatCount: 4,
    })),
    people: Array.from({ length: people }, (_, i) => ({ userId: `u-${i}`, name: `Ib ${i}` })),
  })

  it('is ready with a car and a crew', () => {
    expect(unitReadiness(unit(1, 2)).ready).toBe(true)
  })

  it('says which half is missing', () => {
    expect(unitReadiness(unit(0, 2)).missing).toEqual(['intet køretøj'])
    expect(unitReadiness(unit(1, 0)).missing).toEqual(['ingen mandskab'])
    expect(unitReadiness(unit(0, 0)).missing).toHaveLength(2)
  })

  // Flagged, not forbidden: the desk can still work, and the Organisation page is where it is
  // fixed.
  it('flags more than one vehicle without calling the unit unready', () => {
    const readiness = unitReadiness(unit(2, 1))
    expect(readiness.tooManyVehicles).toBe(true)
    expect(readiness.ready).toBe(true)
  })
})

describe('spans', () => {
  it('reads minutes below the hour and hours above it', () => {
    expect(formatSpan(42 * minute)).toBe('42m')
    expect(formatSpan(134 * minute)).toBe('2t 14m')
  })

  // Clock skew between server and browser is not worth explaining to a volunteer at 3am.
  it('never reads negative', () => {
    expect(formatSpan(-500)).toBe('0m')
  })

  // The sign is the whole message: dinner in forty minutes is a plan, dinner eight minutes ago is
  // an apology.
  it('says whether a deadline is ahead or behind', () => {
    expect(untilUts(nowUts + 42 * minute, nowMs)).toBe('om 42m')
    expect(untilUts(nowUts - 8 * minute, nowMs)).toBe('for 8m siden')
  })
})

describe('the route line', () => {
  it('joins both ends of something that moves', () => {
    expect(taskRoute(task())).toBe('ved Post 2B → HQ')
  })

  it('is one place for a samaritter call-out, which moves nothing', () => {
    // A call-out has no destination, and "→ Adresse" is a place nobody drives to.
    expect(
      taskRoute(task({ kind: 'samarit', dropoff: { kind: 'text', label: '' } })),
    ).toBe('ved Post 2B')
  })
})

describe('planning a task onto a tour', () => {
  const stop = (label: string, over: Partial<TourStop> = {}): TourStop => ({
    stopId: `s-${label}`,
    sortOrder: 0,
    place: { kind: 'text', label },
    plannedUts: null,
    override: false,
    visitedUts: null,
    tasks: [],
    ...over,
  })
  const hqStop = (over: Partial<TourStop> = {}) =>
    stop('HQ', { stopId: 's-hq', place: { kind: 'hq', label: 'HQ' }, ...over })
  const pickupTask = (id: string, from: string): Task =>
    task({ id, kind: 'pickup', pickup: { kind: 'text', label: from }, dropoff: { kind: 'hq', label: 'HQ' } })

  it('is a load and an unload for something that moves', () => {
    const stops = planTaskOntoStops([], pickupTask('t-1', 'ved Post 2B'))
    expect(stops.map((s) => s.place.label)).toEqual(['ved Post 2B', 'HQ'])
    expect(stops.map((s) => s.tasks[0].role)).toEqual(['load', 'unload'])
  })

  it('never returns to HQ twice: a second pickup joins the drive home', () => {
    // Two discontinued scouts from two roadsides is one arrival at HQ, with two scouts in the car.
    let stops = planTaskOntoStops([], pickupTask('t-1', 'ved Post 2B'))
    stops = planTaskOntoStops(stops, pickupTask('t-2', 'ved Lok 3'))
    expect(stops.map((s) => s.place.label)).toEqual(['ved Post 2B', 'ved Lok 3', 'HQ'])
    expect(stops[2].tasks).toEqual([
      { taskId: 't-1', role: 'unload' },
      { taskId: 't-2', role: 'unload' },
    ])
  })

  it('pulls a new pickup in front of the HQ stop it shares', () => {
    // The load must not come after its own unload — the API refuses that plan outright.
    const stops = planTaskOntoStops([hqStop({ tasks: [{ taskId: 't-1', role: 'unload' }] })], pickupTask('t-2', 'ved Lok 3'))
    expect(stops.map((s) => s.place.label)).toEqual(['ved Lok 3', 'HQ'])
    expect(stops[1].tasks.map((t) => t.taskId)).toEqual(['t-1', 't-2'])
  })

  it('leaves a transport idle at its dropoff rather than sending it home', () => {
    const stops = planTaskOntoStops(
      [],
      task({
        id: 't-3',
        kind: 'transport',
        pickup: { kind: 'checkpoint', refId: 'cp-2a', label: 'Post 2A' },
        dropoff: { kind: 'checkpoint', refId: 'cp-2b', label: 'Post 2B' },
      }),
    )
    expect(stops.map((s) => s.place.label)).toEqual(['Post 2A', 'Post 2B'])
  })

  it('will not hang new work off a stop the car has already left', () => {
    // Merging into history would mark the new task done the moment the tour moves on.
    const stops = planTaskOntoStops([hqStop({ visitedUts: 9000 })], pickupTask('t-2', 'ved Lok 3'))
    expect(stops.map((s) => s.place.label)).toEqual(['HQ', 'ved Lok 3', 'HQ'])
  })

  it('keeps two places nobody named apart', () => {
    // Two scouts waiting somewhere unrecorded are two stops; merging them would lose one.
    let stops = planTaskOntoStops([], task({ id: 't-1', pickup: { kind: 'text', label: '' } }))
    stops = planTaskOntoStops(stops, task({ id: 't-2', pickup: { kind: 'text', label: '' } }))
    expect(stops.filter((s) => s.place.label === 'Hentes')).toHaveLength(2)
  })

  it('gives a Guides drive one stop too — the crew is set down and nothing comes back', () => {
    const drive = task({
      id: 't-5',
      kind: 'guides',
      pickup: { kind: 'checkpoint', refId: 'cp-3', label: 'Post 3' },
      dropoff: { kind: 'text', label: '' },
    })
    const stops = planTaskOntoStops([hqStop()], drive)
    expect(stops.map((s) => s.place.label)).toEqual(['HQ', 'Post 3'])
    expect(stops[1].tasks).toEqual([{ taskId: 't-5', role: 'action' }])
  })

  it('gives a samaritter call-out one stop, and reuses a place already on the tour', () => {
    const callout = task({
      id: 't-4',
      kind: 'samarit',
      pickup: { kind: 'checkpoint', refId: 'cp-2a', label: 'Post 2A' },
      dropoff: { kind: 'text', label: '' },
    })
    const existing = stop('Post 2A', { place: { kind: 'checkpoint', refId: 'cp-2a', label: 'Post 2A' } })
    const stops = planTaskOntoStops([existing], callout)
    expect(stops).toHaveLength(1)
    expect(stops[0].tasks).toEqual([{ taskId: 't-4', role: 'action' }])
  })

  it('drops the task where the operator dropped it', () => {
    const stops = planTaskOntoStops(
      [stop('Post 2A'), stop('Post 2C')],
      task({
        id: 't-5',
        kind: 'transport',
        pickup: { kind: 'text', label: 'Post 2B' },
        dropoff: { kind: 'text', label: 'Post 2D' },
      }),
      's-Post 2A',
    )
    expect(stops.map((s) => s.place.label)).toEqual(['Post 2A', 'Post 2B', 'Post 2D', 'Post 2C'])
  })
})
