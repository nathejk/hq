// Person search (PRD 014): the parts of the results view that are worth testing without a DOM.
//
// The view itself is a table; everything that could be wrong in a way an operator would act on
// lives here — whether a query is long enough to send, which Danish heading a `kind` belongs
// under, whose number matched, and where a row links to. All pure functions over the payload.

import type { RouteLocationRaw } from 'vue-router'

/** One person as `GET /api/search/person` returns them. Mirrors searchperson.Result. */
export interface PersonResult {
  kind: PersonKind
  id: string
  year: string
  name: string
  /** As entered, because that is what the operator reads back to the caller. */
  phone: string
  phoneRole: PhoneRole
  email?: string
  teamId?: string
  teamName?: string
  teamNumber?: string
  status?: string
  statusKind?: StatusKind
  /** Taken off a roster — "udmeldt". A separate axis from `status`; see statusBadge(). */
  removed: boolean
}

/** The envelope's `search` object. */
export interface PersonSearchResponse {
  results: PersonResult[]
  /** The 50-result cap bit; say so rather than let the operator scroll to a false bottom. */
  truncated: boolean
  /** Nothing was searched — distinct from an empty `results`, which means we looked. */
  tooShort: boolean
  matchedPhone: boolean
  matchedName: boolean
}

/**
 * The seven kinds the projection indexes.
 *
 * There is no `personnel` and no `bandit`: `personnel` is a table name, and a bandit is a senior
 * with an arm number. Both mistakes have been made before on the live tokens below.
 */
export type PersonKind =
  | 'spejder'
  | 'senior'
  | 'gøgler'
  | 'friend'
  | 'crew'
  | 'patruljekontakt'
  | 'klankontakt'

export type PhoneRole = '' | 'own' | 'parent' | 'contact'
export type StatusKind = '' | 'member' | 'team'

/**
 * What invalidates a search result set.
 *
 * Entity **types**, never instances: a new match is by definition a row whose id this client has
 * never seen, so an instance-keyed dependency could not make it appear. The tokens are the event
 * *subject's* entity rather than the projection's name, and this exact set is pinned on the Go
 * side by `searchperson/entities_test.go` — if a source is ever added there, that test and this
 * list change together. A wrong token fails silently: the page looks live and never updates.
 */
export const SEARCH_DEPENDS_ON = [
  'crew',
  'crewmember',
  'friend',
  'gøgler',
  'klan',
  'patrulje',
  'senior',
  'spejder',
] as const

// The server's own minimums (searchperson.MinNameLength / MinPhoneDigits), enforced here as well
// so the first keystroke does not ask for every person in the event. Duplicated deliberately: the
// client needs the rule to decide whether to send at all, and the server needs it because it
// cannot trust a client.
export const MIN_NAME_LENGTH = 3
export const MIN_PHONE_DIGITS = 4

/** Digits only, dropping a Danish country code, mirroring the server's normalisation. */
export function queryDigits(text: string): string {
  const digits = text.replace(/\D/g, '')
  if (digits.length === 10 && digits.startsWith('45')) return digits.slice(2)
  return digits
}

/**
 * Is there enough here to search for?
 *
 * The same two-interpretation rule the server uses: enough digits makes it a number query, and
 * enough letters makes it a name query. Either is enough, which is what lets `tlf. 20 68 00 11`
 * work — it has both.
 */
export function isSearchable(text: string): boolean {
  const trimmed = text.trim()
  if (queryDigits(trimmed).length >= MIN_PHONE_DIGITS) return true
  const hasLetter = /\p{L}/u.test(trimmed)
  return hasLetter && [...trimmed].length >= MIN_NAME_LENGTH
}

/** The three states in which no rows are shown, which must not look alike. */
export type EmptyState =
  /** Nothing typed at all. */
  | 'idle'
  /** Below the minimum, so nothing was searched — "keep typing". */
  | 'tooShort'
  /** We looked and found nobody. */
  | 'noMatch'
  /** Rows to show. */
  | 'results'

export function emptyState(
  text: string,
  response: PersonSearchResponse | undefined,
): EmptyState {
  if (!text.trim()) return 'idle'
  if (!isSearchable(text)) return 'tooShort'
  if (!response) return 'results' // still loading; the table's own loading state covers it
  if (response.tooShort) return 'tooShort'
  return response.results.length ? 'results' : 'noMatch'
}

// Danish group headings, per PRD 014 §7.
//
// `friend` gets a heading of its own rather than being folded in with `gøgler`: they are separate
// personnel kinds in hq and the operator's next action differs, so folding them would throw away
// the one word on the row that says which.
const KIND_GROUPS: Record<PersonKind, string> = {
  spejder: 'Spejdere',
  senior: 'Seniorer',
  patruljekontakt: 'Kontaktpersoner',
  klankontakt: 'Kontaktpersoner',
  gøgler: 'Gøglere',
  friend: 'Friends',
  crew: 'Crew',
}

/** Heading order: the two participant populations first, then the adults who answer for them. */
const GROUP_ORDER = ['Spejdere', 'Seniorer', 'Kontaktpersoner', 'Gøglere', 'Friends', 'Crew']

export function kindGroup(kind: PersonKind): string {
  return KIND_GROUPS[kind] ?? 'Øvrige'
}

/**
 * Per-row kind label.
 *
 * Shown because a heading is not always enough: "Kontaktpersoner" holds both a patrol's contact
 * and a klan's, and which one it is decides where the row goes.
 */
const KIND_LABELS: Record<PersonKind, string> = {
  spejder: 'spejder',
  senior: 'senior',
  patruljekontakt: 'patruljekontakt',
  klankontakt: 'klankontakt',
  gøgler: 'gøgler',
  friend: 'friend',
  crew: 'crew',
}

export function kindLabel(kind: PersonKind): string {
  return KIND_LABELS[kind] ?? kind
}

/**
 * Whose number this is, in words, or '' when it is the person's own.
 *
 * Annotated only when it is *not* their own, because that is the case that misleads: an operator
 * who dials without knowing a guardian will answer opens with the wrong sentence. An empty
 * `phone` is its own message — the person has no number at all, and the row still matched by name.
 */
export function phoneRoleLabel(role: PhoneRole): string {
  switch (role) {
    case 'parent':
      return 'forælders nummer'
    case 'contact':
      return 'kontaktperson'
    default:
      return ''
  }
}

/** 44 of 164 indexed 2026 seniorer have no name in the register; a blank cell reads as a bug. */
export function displayName(result: PersonResult): string {
  return result.name.trim() || '(uden navn)'
}

/**
 * The team or section, as one readable string.
 *
 * A patrol that has not been named yet is ordinary, so the number alone is a complete answer and
 * an unnamed *and* unnumbered team says so rather than rendering blank.
 */
export function teamLabel(result: PersonResult): string {
  const parts = [result.teamNumber?.trim(), result.teamName?.trim()].filter(Boolean)
  if (parts.length) return parts.join(' · ')
  return result.teamId ? '(uden navn)' : ''
}

/**
 * Where a row links to.
 *
 * Only the patrol has a detail route, so the others land on the list that holds them — still a
 * shorter path than navigating from scratch. `null` when there is nothing to link to (a personnel
 * row with no section, say), in which case the row is rendered as text rather than as a dead link.
 */
export function resultRoute(result: PersonResult): RouteLocationRaw | null {
  switch (result.kind) {
    case 'spejder':
    case 'patruljekontakt':
      return result.teamId ? { name: 'patrulje', params: { teamId: result.teamId } } : null
    case 'senior':
    case 'klankontakt':
      return { name: 'klaner' }
    case 'gøgler':
    case 'friend':
      return { name: 'badutter' }
    case 'crew':
      return { name: 'organisation' }
    default:
      return null
  }
}

/** A row as the table renders it: the payload plus what grouping and ordering need. */
export interface PersonRow extends PersonResult {
  /** Stable key; `(kind, id, year)` is the projection's own key and a person can match twice. */
  rowKey: string
  /** The subheader this row falls under. */
  group: string
  /** Sort field, so PrimeVue's subheader grouping and the heading order agree. */
  order: number
}

/**
 * Flatten results into one ordered list of rows.
 *
 * One list with subheaders rather than a table per group: it gives the view a single `:loading`
 * to wire `pending` to, and keeps the whole result set in one keyboard tab order.
 */
export function toRows(results: PersonResult[]): PersonRow[] {
  return results
    .map((result, index) => {
      const group = kindGroup(result.kind)
      const groupIndex = GROUP_ORDER.indexOf(group)
      return {
        ...result,
        rowKey: `${result.kind}:${result.id}:${result.year}:${index}`,
        group,
        // Index within the group order, times a stride that leaves room for the server's own
        // ordering to break ties, so rows stay in the order the query returned them.
        order: (groupIndex === -1 ? GROUP_ORDER.length : groupIndex) * 10_000 + index,
      }
    })
    .sort((a, b) => a.order - b.order)
}
