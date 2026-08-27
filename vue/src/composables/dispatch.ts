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
