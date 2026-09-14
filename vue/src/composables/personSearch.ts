// Person search (PRD 014): the parts of the results view that are worth testing without a DOM.
//
// The view itself is a table; everything that could be wrong in a way an operator would act on
// lives here — whether a query is long enough to send, which Danish heading a `kind` belongs
// under, whose number matched, and where a row links to. All pure functions over the payload.

import type { RouteLocationRaw } from 'vue-router'
import { severityTagSeverity, type Severity } from './severity'
import { memberStatusBadge, memberStatusPhrase } from './memberStatus'

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

/**
 * Digits only, in national form, mirroring the server's `normalizePhone`.
 *
 * The two steps have to match the Go side exactly, or the client and the server disagree about
 * what the operator typed. Strip the international access prefix first (`0045 12345678` as
 * dialled), then a Danish country code — and the latter **only** from a 10-digit result, because
 * `45` is a real Danish prefix and `45123456` is somebody's actual number.
 */
export function queryDigits(text: string): string {
  let digits = text.replace(/\D/g, '')
  if (digits.startsWith('00')) digits = digits.slice(2)
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

/**
 * The `tel:` target for a phone field, or null when the field is not one dialable number.
 *
 * A phone value is whatever the register holds, and since task 187 the projection deliberately
 * keeps fields like `mor 22 79 01 52 eller Far 22110715` intact rather than splitting them — the
 * entered text is what tells an operator *which* parent they are looking at. That makes a naive
 * `tel:${phone}` link actively broken: it dials nothing, and looks like it should.
 *
 * So a link is offered only where the field resolves to exactly one number. A two-number field,
 * or a value damaged beyond recognition, renders as text the operator reads themselves — which is
 * the honest outcome, because we cannot know which of two numbers they meant to ring.
 */
export function dialable(phone: string): string | null {
  const digits = queryDigits(phone)
  if (digits.length === 8) return `tel:+45${digits}`
  // A non-Danish number, left as dialled. queryDigits has already removed a Danish country
  // code, so anything still 10+ digits either belongs elsewhere or is two numbers in one field.
  if (digits.length >= 9 && digits.length <= 12 && !/^45/.test(digits)) return `tel:+${digits}`
  return null
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

/**
 * A person's standing, as one badge.
 *
 * # Two vocabularies, and the one that already exists wins
 *
 * `statusKind` says which vocabulary `status` is drawn from, and the two are not comparable:
 * `racing` is a fact about a person, `PAID` is a fact about their team. Member statuses already have
 * badges in HQ (PRD 006), so search reuses them verbatim from `memberStatus.ts` rather than
 * inventing words — a scout who reads "Udgår" on the patrol page must not read something else here,
 * or an operator will take them for two different facts. Team statuses had no vocabulary at all, so
 * one is defined below and coloured through `severity.ts`.
 *
 * # Every row, not only the unusual ones
 *
 * A badge that appeared only when something was wrong would make its *absence* carry meaning the
 * operator has to know how to read. Showing "Aktiv" as plainly as "Udmeldt" means a row can be taken
 * at face value, which is the whole point: a stale hit presented as current is worse than no hit.
 */
export interface StatusBadge {
  label: string
  /** A PrimeVue Tag severity. */
  severity: string
  icon: string
  /**
   * The long form, shown on hover.
   *
   * Colour alone cannot carry the distinction search needs. The established member vocabulary
   * separates *in our care* (`waiting`/`transit` warn, `sheltered` danger) from *somebody else's
   * charge* (`released` secondary, `reunited` info), but it spreads each across two colours, because
   * it was built for the race-night screens rather than for a phone call. Rather than fork it, the
   * unambiguous sentence travels with the badge.
   */
  title: string
}

/**
 * Team signup statuses, in Danish, prefixed with their subject.
 *
 * This answers PRD 014 §11's open question. `PAY` on a person's row reads as though *they* owe
 * money, and they do not — it is a fact about the team they are on. The prefix "Holdet:" makes the
 * subject explicit in the badge itself, which is cheaper than a second column and impossible to
 * misread. `OUT` deliberately reads "ude" and not "udgået", because `sheltered` already owns
 * "Udgået" in the member vocabulary and two states sharing a word is the failure this whole badge
 * exists to prevent.
 */
const TEAM_STATUSES: Record<string, { label: string; severity: Severity; icon: string }> = {
  NEW: { label: 'tilmeldt', severity: 'yellow', icon: 'pi pi-users' },
  HOLD: { label: 'afventer', severity: 'yellow', icon: 'pi pi-pause' },
  PAY: { label: 'mangler betaling', severity: 'yellow', icon: 'pi pi-wallet' },
  SEMIPAID: { label: 'delvist betalt', severity: 'yellow', icon: 'pi pi-wallet' },
  PAID: { label: 'betalt', severity: 'green', icon: 'pi pi-check' },
  STARTED: { label: 'startet', severity: 'green', icon: 'pi pi-directions' },
  OUT: { label: 'ude', severity: 'red', icon: 'pi pi-ban' },
}

/**
 * Unknown is an ordinary answer before the race, and is rendered as unknown.
 *
 * Not dressed up as `registered`: a scout with no `spejderstatus` row has not been registered as
 * anything, and asserting a status the read model does not hold would be the one thing search must
 * never do.
 */
const UNKNOWN_STATUS: StatusBadge = {
  label: 'Ukendt status',
  severity: 'contrast',
  icon: 'pi pi-question-circle',
  title: 'Ingen status registreret — helt normalt før løbet',
}

export function statusBadge(result: PersonResult): StatusBadge {
  const status = result.status ?? ''
  if (!status) return UNKNOWN_STATUS

  if (result.statusKind === 'member') {
    const badge = memberStatusBadge(status)
    return { ...badge, title: memberStatusPhrase(status) }
  }

  if (result.statusKind === 'team') {
    const team = TEAM_STATUSES[status]
    // An unrecognised slug keeps its own name rather than being flattened into "ukendt": that is a
    // deploy skew, and hiding it would look like data loss to the operator reading the screen.
    if (!team) {
      return {
        label: `Holdet: ${status}`,
        severity: 'contrast',
        icon: 'pi pi-users',
        title: `Holdets tilmeldingsstatus: ${status}`,
      }
    }
    return {
      label: `Holdet: ${team.label}`,
      severity: severityTagSeverity(team.severity),
      icon: team.icon,
      title: `Holdets tilmeldingsstatus: ${team.label}`,
    }
  }

  return UNKNOWN_STATUS
}

/**
 * "Udmeldt" — a separate badge, because it is a separate axis.
 *
 * `removed` means taken off a roster; `released` means went home during the night. Both can be true
 * of one person, and collapsing them would have an operator tell a guardian their child never signed
 * up when in fact they were collected at 02:00. Rendered alongside the status badge rather than
 * instead of it, with its own word and its own glyph so it cannot be mistaken for a lifecycle state.
 */
export const REMOVED_BADGE: StatusBadge = {
  label: 'Udmeldt',
  severity: 'danger',
  icon: 'pi pi-user-minus',
  title: 'Meldt ud af holdet — ikke det samme som hentet under løbet',
}

/**
 * The year the request is scoped to — reconstructed, not read.
 *
 * `globalstate.yearSlug` is deliberately '' when the selected year *is* the current calendar year,
 * because that is the condition under which the axios interceptor omits `X-YearSlug` entirely. The
 * server then falls back to `time.Now().Year()` (`app.YearSlug` in `cmd/api/routes.go`), so this
 * expression reproduces the server's own choice rather than guessing at it.
 *
 * It is needed only to tell a cross-year row from a current-year one, and only when the operator has
 * opted into other years — so the cost of it being wrong is a mislabelled group, not a wrong search.
 */
export function activeYearSlug(yearSlug: string): string {
  return yearSlug || String(new Date().getFullYear())
}

/** A row as the table renders it: the payload plus what grouping and ordering need. */
export interface PersonRow extends PersonResult {
  /** Stable key; `(kind, id, year)` is the projection's own key and a person can match twice. */
  rowKey: string
  /** The subheader this row falls under. */
  group: string
  /** Sort field, so PrimeVue's subheader grouping and the heading order agree. */
  order: number
  /** From a year other than the active one, and rendered as such. */
  otherYear: boolean
}

/**
 * Cross-year rows sort after every current-year row, with room to spare.
 *
 * A hit from 2024 sitting among this year's is exactly the stale-data-that-looks-current failure the
 * whole live-updates design exists to avoid, so this is a hard tier and not a sort preference.
 */
const OTHER_YEAR_TIER = 1_000_000

/**
 * Flatten results into one ordered list of rows.
 *
 * One list with subheaders rather than a table per group: it gives the view a single `:loading` to
 * wire `pending` to, and keeps the whole result set in one keyboard tab order.
 *
 * `activeYear` is what the request was scoped to. Passing '' means "treat everything as current",
 * which is correct when the operator has not opted into other years — every row is this year's by
 * definition then, and inventing a year label would be noise.
 */
export function toRows(results: PersonResult[], activeYear = ''): PersonRow[] {
  // Most recent first among the other years, so 2025 precedes 2024. String compare is safe: these
  // are four-digit slugs.
  const otherYears = [...new Set(results.map((r) => r.year))]
    .filter((year) => activeYear && year !== activeYear)
    .sort()
    .reverse()

  return results
    .map((result, index) => {
      const otherYear = Boolean(activeYear) && result.year !== activeYear
      const kind = kindGroup(result.kind)
      const kindIndex = GROUP_ORDER.indexOf(kind)
      const rank = kindIndex === -1 ? GROUP_ORDER.length : kindIndex

      // The year leads the heading for a cross-year group, because the year is the thing that
      // changes how the row should be read — the kind is secondary there.
      const group = otherYear ? `${result.year} · ${kind}` : kind

      // Index within the group order, times a stride that leaves room for the server's own ordering
      // to break ties, so rows stay in the order the query returned them.
      const withinTier = rank * 10_000 + index
      const order = otherYear
        ? OTHER_YEAR_TIER + otherYears.indexOf(result.year) * 100_000 + withinTier
        : withinTier

      return {
        ...result,
        rowKey: `${result.kind}:${result.id}:${result.year}:${index}`,
        group,
        order,
        otherYear,
      }
    })
    .sort((a, b) => a.order - b.order)
}
