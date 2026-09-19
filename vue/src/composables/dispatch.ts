// Shared bits of the kørsel UI (PRD 009): the Danish vocabulary, and time formatting for an
// entity that stores unix seconds.
//
// English identifiers, Danish strings — the rule for this whole feature. The labels an operator
// reads are Kørsel, Tur, Opgave, "Ikke planlagt"; nothing in the code is named after them.
//
// # Why its own time helpers
//
// `composables/shelter.ts` formats ISO strings, because that is what the shelter's API sends.
// Kørsel sends unix seconds (`…Uts`), deliberately: every number on this screen is arithmetic —
// waited-for, time-to-deadline, departure plus a leg allowance — and seconds keep a timezone
// question out of each step. Rather than converting at every call site, the same three formats
// are expressed once for seconds. The output is identical, down to the da-DK dot in "21.40".

import { severityLabel, severityTagSeverity, severityOptions } from './severity'

// A task's priority is the nødtelefon's severity vocabulary, shared rather than copied
// (task 112): two race-night desks should not have two words for urgent.
export { severityLabel as priorityLabel, severityTagSeverity as priorityTagSeverity, severityOptions as priorityOptions }

export type TaskKind = 'pickup' | 'transport' | 'collection' | 'delivery' | 'samarit'
export type TaskState = 'queued' | 'planned' | 'underway' | 'done' | 'cancelled'
export type TourState = 'planned' | 'underway' | 'completed' | 'cancelled'
export type PlaceKind = 'checkpoint' | 'lok' | 'hq' | 'text'
export type Role = 'load' | 'unload' | 'action'

export interface Place {
  kind: PlaceKind
  refId?: string
  label: string
}

export interface StopTask {
  taskId: string
  role: Role
}

export interface TourStop {
  stopId: string
  sortOrder: number
  place: Place
  plannedUts: number | null
  override: boolean
  visitedUts: number | null
  tasks: StopTask[]
}

export interface Tour {
  id: string
  year: string
  sectionSlug: string
  departureUts: number | null
  notes?: string
  state: TourState
  createdUts: number
  underwayUts: number | null
  completedUts: number | null
  cancelledUts: number | null
  cancelReason?: string
  stops: TourStop[]
}

/**
 * One appearance of a task on a tour — the load, the unload, or a single action.
 *
 * This is where the answer to "when?" comes from: the planned time of the stop, made by a human
 * who knows the roads. Sent with a task by `GET /api/dispatch/task/:id` and
 * `GET /api/sos/:id/dispatch`; the board matches tasks to stops from the tours it already has.
 */
export interface TaskStop {
  tourId: string
  stopId: string
  role: Role
  sortOrder: number
  place: Place
  plannedUts: number | null
  override: boolean
  visitedUts: number | null
}

export interface Task {
  id: string
  year: string
  kind: TaskKind
  priority?: string
  description: string
  spaceNeeds?: string
  pickup: Place
  dropoff: Place
  state: TaskState
  createdUts: number
  notBeforeUts: number | null
  deadlineUts: number | null
  pickedUpUts: number | null
  doneUts: number | null
  cancelledUts: number | null
  cancelReason?: string
  sosId?: string
  teamId?: string
  memberIds: string[]
  /** Filled in by the single-task and case endpoints; absent on the board payload. */
  stops?: TaskStop[]
}

export interface Vehicle {
  vehicleId: string
  licensePlate: string
  driverUserId: string
  seatCount: number
  brand?: string
  model?: string
}

export interface UnitPerson {
  userId: string
  name: string
  phone?: string
}

export interface Unit {
  sectionSlug: string
  label: string
  vehicles: Vehicle[]
  people: UnitPerson[]
}

/** One window in which a unit is available. Per unit, not per person: the unit is what is
 * available or asleep (PRD 009 §6). */
export interface Duty {
  id: string
  year: string
  sectionSlug: string
  startUts: number
  endUts: number
}

export interface Board {
  tasks: Task[]
  tours: Tour[]
  units: Unit[]
  duty: Duty[]
  kinds: TaskKind[]
  priorities: string[]
  /** The event's span, midnight to midnight in local time, as the roster grid's time axis.
   * Both 0 when the year has no dates set. */
  event?: { startUts: number; endUts: number }
}

const kindLabels: Record<string, string> = {
  pickup: 'Hentning',
  transport: 'Transport',
  collection: 'Indsamling',
  delivery: 'Levering',
  samarit: 'Samaritter',
}

export const kindLabel = (kind: string) => kindLabels[kind] ?? kind

// The kinds read differently on a board, which is the reason they exist — so they look
// different too. A pickup is people and gets the car; a samaritter call-out is people too but
// brings nobody back; the rest are things.
const kindIcons: Record<string, string> = {
  pickup: 'pi pi-user',
  transport: 'pi pi-arrow-right-arrow-left',
  collection: 'pi pi-download',
  delivery: 'pi pi-upload',
  samarit: 'pi pi-heart-fill',
}

export const kindIcon = (kind: string) => kindIcons[kind] ?? 'pi pi-box'

const stateLabels: Record<string, string> = {
  queued: 'Ikke planlagt',
  planned: 'Lagt i tur',
  underway: 'Undervejs',
  done: 'Færdig',
  cancelled: 'Aflyst',
}

export const stateLabel = (state: string) => stateLabels[state] ?? state

const tourStateLabels: Record<string, string> = {
  planned: 'Planlagt',
  underway: 'Undervejs',
  completed: 'Færdig',
  cancelled: 'Aflyst',
}

export const tourStateLabel = (state: string) => tourStateLabels[state] ?? state

const roleLabels: Record<string, string> = {
  load: 'hentes',
  unload: 'afleveres',
  action: 'udføres',
}

export const roleLabel = (role: string) => roleLabels[role] ?? role

const placeKindLabels: Record<string, string> = {
  checkpoint: 'Post',
  lok: 'Lok',
  hq: 'HQ',
  text: 'Adresse',
}

export const placeKindLabel = (kind: string) => placeKindLabels[kind] ?? kind

/**
 * "lør 21.40" — a unix-seconds timestamp with its weekday.
 *
 * The weekday is there because the race runs through a night: "21.40" alone is ambiguous, and a
 * scout waiting since 21.40 is a very different problem depending on which evening that was.
 * Some ICU versions abbreviate with a trailing period and some do not, so it is normalised away
 * — a format that differs between the dev container and a browser is worse than either.
 */
export const formatUts = (uts?: number | null) => {
  if (!uts) return ''
  const date = new Date(uts * 1000)
  if (Number.isNaN(date.getTime())) return ''
  const weekday = date.toLocaleDateString('da-DK', { weekday: 'short' }).replace(/\.$/, '')
  const time = date.toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
  return `${weekday} ${time}`
}

/** "21.40" — the clock alone, for a column whose header already says which day. */
export const formatUtsTime = (uts?: number | null) => {
  if (!uts) return ''
  const date = new Date(uts * 1000)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
}

/**
 * "2t 14m" — a span, in Danish.
 *
 * Days are deliberately not a unit: this screen exists for one night, and "1d 3t" would mean
 * something had gone very wrong, in which case a large hour count is the clearer alarm. A
 * negative span (clock skew between server and browser) reads as "0m" rather than as minus
 * something: it is not worth explaining a machine's disagreement to a volunteer at 3am.
 */
export const formatSpan = (seconds: number) => {
  const minutes = Math.max(0, Math.floor(seconds / 60))
  if (minutes < 60) return `${minutes}m`
  return `${Math.floor(minutes / 60)}t ${minutes % 60}m`
}

/** How long a task has waited. The number that needs no model and is never wrong. */
export const waitedFor = (task: Task, nowMs: number) => formatSpan(nowMs / 1000 - task.createdUts)

/**
 * Time until a deadline, or how long past it — "om 42m" / "for 8m siden".
 *
 * Signed, in words, because the sign is the whole message: dinner in forty minutes is a plan and
 * dinner eight minutes ago is an apology.
 */
export const untilUts = (uts: number, nowMs: number) => {
  const seconds = uts - nowMs / 1000
  if (seconds >= 0) return `om ${formatSpan(seconds)}`
  return `for ${formatSpan(-seconds)} siden`
}

/** A place as one line: "Post 2A", "Lok 3", "ved skovbrynet". */
export const placeLine = (place?: Place) => {
  if (!place) return ''
  if (place.label) return place.label
  return placeKindLabel(place.kind)
}

/**
 * The one line that says where a task happens.
 *
 * Two places joined by an arrow for everything that moves — and a single place for a samaritter
 * call-out, which moves nothing. Rendering its empty dropoff would put "→ Adresse" on the board,
 * a destination nobody drives to and the operator would have to learn to ignore.
 */
export const taskRoute = (task: Pick<Task, 'kind' | 'pickup' | 'dropoff'>) => {
  const from = placeLine(task.pickup)
  // A place with no label and no kind but `text` is one nobody filled in — which is exactly what
  // a call-out's dropoff is, and what an aborted edit of any other kind leaves behind.
  const unset = !task.dropoff?.label && (task.dropoff?.kind ?? 'text') === 'text'
  if (task.kind === 'samarit' || unset) return from
  return `${from} → ${placeLine(task.dropoff)}`
}

// --- planning a task onto a tour (task 118) ---

/**
 * The identity of a place, for deciding whether two stops are the *same* stop.
 *
 * Known places are identified by what they are, not by what somebody typed: a checkpoint is the
 * same checkpoint however the label was spelled, and there is exactly one HQ. Free text is
 * compared case- and space-insensitively, which is as far as it is honest to go — "ved skovbrynet"
 * and "Slangerupvej" may well be the same gate, and no string comparison can know that.
 *
 * Returns null for a place nobody has filled in. An unnamed place is not equal to another unnamed
 * place: two scouts waiting somewhere unrecorded are two stops, and merging them would lose one.
 */
const placeKey = (place?: Place): string | null => {
  if (!place) return null
  if (place.kind !== 'text') return `${place.kind}:${place.refId ?? ''}`
  const label = place.label?.trim().toLowerCase()
  return label ? `text:${label}` : null
}

/** Whether two places are the same place a car would stop at. */
export const samePlace = (a?: Place, b?: Place) => {
  const key = placeKey(a)
  return key !== null && key === placeKey(b)
}

/**
 * The stops one task needs on a tour.
 *
 * A pickup, transport, collection or delivery becomes **two** stops — where it is loaded and where
 * it is unloaded — because that is what it is: a task that moves something occupies two places,
 * and a single stop would make "when will they be collected" and "when will they arrive" the same
 * number.
 *
 * A samaritter call-out becomes **one** stop with role `action`: it moves nothing, so a second
 * stop would be a place the car never goes and a time the desk would be asked about.
 *
 * A task whose place nobody filled in still gets a stop, labelled with what it is for — the plan
 * is allowed to be vaguer than the map, and the driver is on the phone anyway.
 */
export const stopsForTask = (task: Task): TourStop[] => {
  const stop = (fallback: string, place: Place, role: Role): TourStop => ({
    stopId: '',
    sortOrder: 0,
    place: place?.label ? place : { kind: 'text', label: fallback },
    plannedUts: null,
    override: false,
    visitedUts: null,
    tasks: [{ taskId: task.id, role }],
  })
  if (task.kind === 'samarit') return [stop('Tilses', task.pickup, 'action')]
  return [stop('Hentes', task.pickup, 'load'), stop('Afleveres', task.dropoff, 'unload')]
}

/**
 * Add a task to a tour's stops, reusing the places the tour already visits.
 *
 * **A tour never stops at the same place twice.** Two discontinued scouts collected from two
 * roadsides is one drive home, not two arrivals at HQ; a second HQ row is a stop no driver makes
 * and a time the board would predict for it. So a stop whose place the tour already has joins that
 * stop instead of being appended — which is also what makes "3 opgaver" on one row the normal
 * sight it should be.
 *
 * Merging is only ever done with a stop **not yet visited**: a place the car has already left is
 * history, and hanging new work off it would mark that work done the moment the tour moves on.
 *
 * The load is kept in front of its own unload, because a plan that contradicts itself is refused
 * by the API (`ErrUnloadBeforeLoad`) — so a dropoff the tour already visits pulls the new pickup
 * in ahead of it rather than producing a 422 the operator has to interpret. A transport dropped on
 * a tour whose last stop is where it is going therefore lands *before* that stop; a tour that ends
 * at a dropoff simply waits there for the next task, which is what an idle car does.
 *
 * `afterStopId` is where the operator dropped it, and is respected wherever it does not
 * contradict the above.
 */
export const planTaskOntoStops = (
  stops: TourStop[],
  task: Task,
  afterStopId?: string,
): TourStop[] => {
  const next = stops.map((s) => ({ ...s, tasks: [...s.tasks] }))
  const at =
    afterStopId && next.some((s) => s.stopId === afterStopId)
      ? next.findIndex((s) => s.stopId === afterStopId) + 1
      : next.length

  // A stop the tour already makes at this place, and has not yet made. `before` bounds the search
  // for a load, so it cannot merge into a stop that comes after its own unload.
  const mergeable = (part: TourStop, before?: number) =>
    next.findIndex(
      (s, i) =>
        (before === undefined || i < before) && !s.visitedUts && samePlace(s.place, part.place),
    )

  const attach = (stop: TourStop, part: TourStop) => {
    for (const t of part.tasks) {
      if (!stop.tasks.some((existing) => existing.taskId === t.taskId && existing.role === t.role)) {
        stop.tasks.push(t)
      }
    }
  }

  const parts = stopsForTask(task)

  // Whether a part may join a stop the tour already makes is decided from the *task's* places, not
  // from the stop we just built: an unfilled place is given a label saying what it is for
  // ("Hentes"), and two stops labelled that way are two unrecorded roadsides, not one place.
  const named = (place?: Place) => !!place?.label

  // A call-out is one stop with nothing to order it against.
  if (parts.length === 1) {
    const i = named(task.pickup) ? mergeable(parts[0]) : -1
    if (i >= 0) attach(next[i], parts[0])
    else next.splice(at, 0, parts[0])
    return next
  }

  const [load, unload] = parts
  let unloadIdx = named(task.dropoff) ? mergeable(unload) : -1
  const loadIdx = named(task.pickup)
    ? mergeable(load, unloadIdx >= 0 ? unloadIdx : undefined)
    : -1

  let loadAt: number
  if (loadIdx >= 0) {
    attach(next[loadIdx], load)
    loadAt = loadIdx
  } else {
    loadAt = unloadIdx >= 0 ? Math.min(at, unloadIdx) : at
    next.splice(loadAt, 0, load)
    if (unloadIdx >= loadAt) unloadIdx += 1
  }

  if (unloadIdx >= 0) attach(next[unloadIdx], unload)
  else next.splice(Math.max(at, loadAt + 1), 0, unload)

  return next
}

// --- capacity, and the answer to "when?" for a task nobody has planned (task 116) ---

/**
 * How long a kind of task is assumed to take once somebody sets off.
 *
 * Minutes, and deliberately crude: it ignores distance, traffic and where the car actually is,
 * because there are no vehicle positions to derive any of that from (PRD 009 §8). **One set of
 * numbers for every vehicle**, per open question 10 — a minibus and an estate do not drive alike
 * and the difference between them is far smaller than the error in the estimate itself.
 *
 * That crudeness is honesty about the inputs, not laziness. An estimate that looks precise gets
 * quoted down a phone to a patrol in the dark, who then stop making their own plans — so it is
 * coarse, always labelled *anslået*, and always shown beside the fact that needs no model at all:
 * how long they have already waited.
 */
export const ALLOWANCE_MINUTES: Record<TaskKind, number> = {
  pickup: 30,
  transport: 20,
  collection: 20,
  delivery: 20,
  // A call-out is the drive plus looking somebody over, which is not a stop you tick off in
  // passing: a blister gets cleaned and taped where the scout is standing.
  samarit: 30,
}

/** Units on duty at an instant. */
export const unitsOnDuty = (duty: Duty[], nowMs: number) => {
  const uts = nowMs / 1000
  const slugs = new Set<string>()
  for (const window of duty) {
    // Half-open, as the server's own Covers is: two consecutive windows must not both claim the
    // same minute, or one unit reads as being on duty twice.
    if (window.startUts <= uts && uts < window.endUts) slugs.add(window.sectionSlug)
  }
  return slugs
}

/**
 * Units that are spoken for: they have a tour planned or underway.
 *
 * A unit on duty is capacity; a unit on duty *without* a tour is capacity somebody can hand the
 * next task to right now, and that is the distinction the desk actually acts on. A planned tour
 * counts as engaged even before it departs — the run has been built for that unit.
 */
export const unitsEngaged = (tours: Tour[]) => {
  const slugs = new Set<string>()
  for (const tour of tours ?? []) {
    if (tour.state === 'planned' || tour.state === 'underway') slugs.add(tour.sectionSlug)
  }
  return slugs
}

/** When the next unit comes on duty, or null if none is rostered ahead. */
export const nextDutyStart = (duty: Duty[], nowMs: number): number | null => {
  const uts = nowMs / 1000
  let next: number | null = null
  for (const window of duty) {
    if (window.startUts <= uts) continue
    if (next === null || window.startUts < next) next = window.startUts
  }
  return next
}

/**
 * When a queued task might be dealt with: `max(now, tidligst, next unit on duty) + allowance`.
 *
 * Returns unix seconds, and the caller must label it *anslået*. A tour's planned time beats this
 * whenever one exists — a dispatcher who has built a run knows more than the queue does.
 *
 * With no roster at all the estimate degrades to `now + allowance` rather than to nonsense, which
 * is the mitigation for the roster going stale (PRD 009 §8): a stale roster makes the number
 * optimistic, a missing one makes it merely crude.
 */
export const estimateFor = (task: Task, duty: Duty[], nowMs: number): number => {
  const nowUts = Math.floor(nowMs / 1000)
  let from = nowUts
  if (task.notBeforeUts && task.notBeforeUts > from) from = task.notBeforeUts
  // Only wait for the next unit if none is on duty now. A unit already driving is capacity, and
  // pushing the estimate out to the *next* shift would be pessimistic to the point of useless.
  if (unitsOnDuty(duty, nowMs).size === 0) {
    const next = nextDutyStart(duty, nowMs)
    if (next && next > from) from = next
  }
  return from + (ALLOWANCE_MINUTES[task.kind] ?? 20) * 60
}

/**
 * A unit's readiness: a dispatchable subsection missing a vehicle or a crew is not capacity, and
 * the board says so rather than silently offering it (PRD 009 §6).
 *
 * More than one vehicle is a **configuration mistake, flagged not forbidden** — the desk can still
 * work, and the Organisation page is where it gets fixed.
 */
export const unitReadiness = (unit: Unit) => {
  const missing: string[] = []
  if ((unit.vehicles ?? []).length === 0) missing.push('intet køretøj')
  if ((unit.people ?? []).length === 0) missing.push('ingen mandskab')
  return {
    ready: missing.length === 0,
    missing,
    tooManyVehicles: (unit.vehicles ?? []).length > 1,
  }
}

// --- deadlines (task 117) ---

/**
 * How close to a deadline a task must be before the board shouts about it.
 *
 * One constant in one place, so adopting a different value is an edit here and nowhere else. An
 * hour is chosen to be useful rather than correct: it catches the dinner run while there is still
 * time to send a second car, without lighting up every delivery entered in the afternoon.
 */
export const DEADLINE_WARNING_MINUTES = 60

export type DeadlineRisk = 'none' | 'soon' | 'late'

/**
 * Whether a deadline task is at risk, and why.
 *
 * Two independent causes, and both matter because they call for different actions:
 *   - `late` — the plan itself lands after the deadline, or the deadline has simply passed. The
 *     desk needs another car, and it can know that at 16:00 rather than at 19:20, which PRD 009
 *     §5 calls "the entire point".
 *   - `soon` — still unplanned with the deadline inside the warning window. Nothing is wrong yet;
 *     nothing is happening either.
 *
 * A finished or cancelled task is never at risk: it is history, and a red row for dinner that was
 * delivered on time is how a board teaches its operator to ignore red rows.
 */
export const deadlineRisk = (task: Task, plannedUts: number | null, nowMs: number): DeadlineRisk => {
  if (!task.deadlineUts) return 'none'
  if (task.state === 'done' || task.state === 'cancelled') return 'none'
  const nowUts = nowMs / 1000
  if (task.deadlineUts < nowUts) return 'late'
  if (plannedUts && plannedUts > task.deadlineUts) return 'late'
  if (task.state === 'queued' && task.deadlineUts - nowUts < DEADLINE_WARNING_MINUTES * 60) {
    return 'soon'
  }
  return 'none'
}
