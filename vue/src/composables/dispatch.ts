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

export type TaskKind = 'pickup' | 'transport' | 'collection' | 'delivery'
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
}

const kindLabels: Record<string, string> = {
  pickup: 'Hentning',
  transport: 'Transport',
  collection: 'Indsamling',
  delivery: 'Levering',
}

export const kindLabel = (kind: string) => kindLabels[kind] ?? kind

// The four kinds read differently on a board, which is the reason they exist — so they look
// different too. A pickup is people and gets the car; the rest are things.
const kindIcons: Record<string, string> = {
  pickup: 'pi pi-user',
  transport: 'pi pi-arrow-right-arrow-left',
  collection: 'pi pi-download',
  delivery: 'pi pi-upload',
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
