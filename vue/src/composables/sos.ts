// Shared bits of the nødtelefon UI: the labels and formatting both the list and
// the detail view need.
//
// A composable rather than a Pinia store, matching the rest of the SPA — and there
// is no shared *state* here at all: case data lives in useLiveResource's module
// cache, which is already shared between the views. Duplicating it in a store is
// how the legacy dims channel ended up with two read models (PRD 004 §2.1).

export type Severity = '' | 'green' | 'yellow' | 'red'

// Danish labels, kept in one place so the list badge and the detail select cannot
// drift apart. The values themselves are the API's (PRD 001 §11 Decisions).
const severityLabels: Record<string, string> = {
  green: 'Grøn',
  yellow: 'Gul',
  red: 'Rød',
}

export const severityLabel = (severity: string) => severityLabels[severity] ?? severity

// PrimeVue Tag severities, which are the theme's own semantic colours rather than
// hardcoded hex — a red case should look like the theme's danger, not like #ff0000.
const severityTags: Record<string, string> = {
  green: 'success',
  yellow: 'warn',
  red: 'danger',
}

export const severityTagSeverity = (severity: string) => severityTags[severity] ?? 'secondary'

export const severityOptions = [
  { value: 'green', label: 'Grøn' },
  { value: 'yellow', label: 'Gul' },
  { value: 'red', label: 'Rød' },
]

// da-DK throughout, matching the rest of the SPA. The API sends UTC with an
// explicit offset, so the browser renders the operator's own clock.
export const formatDateTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('da-DK', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export const formatTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
}

// Timeline entry labels.
//
// Unknown types fall back to the raw type rather than being dropped: PRD 006 adds
// member transitions, whole-team collection and understrength exceptions to this
// same timeline, and an operator seeing an unfamiliar-but-present entry is far
// better than one silently missing from a handover record.
const activityLabels: Record<string, string> = {
  created: 'Sag oprettet',
  'headline.updated': 'Overskrift rettet',
  'description.updated': 'Beskrivelse rettet',
  commented: 'Kommentar',
  'comment.updated': 'Kommentar rettet',
  'severity.specified': 'Prioritet sat',
  assigned: 'Tildelt',
  closed: 'Sag lukket',
  reopened: 'Sag genåbnet',
  'team.associated': 'Patrulje tilknyttet',
  'team.disassociated': 'Patrulje fjernet',
  deleted: 'Sag slettet',

  // The member lifecycle summaries (PRD 006). One entry per *operation*, so
  // "team.collected" is one line however many members it took out of the race.
  'member.status.changed': 'Deltagerstatus ændret',
  'member.moved': 'Deltagere flyttet',
  'team.collected': 'Hele patruljen hentes',
}

export const activityLabel = (type: string) => activityLabels[type] ?? type

// --- the member lifecycle summaries (PRD 006) ---

const memberStatusLabels: Record<string, string> = {
  '': 'ikke startet',
  registered: 'tilmeldt',
  seated: 'har plads',
  racing: 'i løbet',
  finished: 'gennemført',
  waiting: 'venter på at blive hentet',
  transit: 'i bil',
  sheltered: 'på HQ',
  reunited: 'genforenet med patruljen',
  released: 'hentet af forældre',
}

// Lower-case here, unlike the backend's picker labels, because these appear mid-sentence
// ("Ida: i løbet → venter") rather than as a standalone tag.
export const memberStatusPhrase = (slug: string) => memberStatusLabels[slug] ?? slug

// The same lifecycle as a badge: the short label, the theme severity and the glyph that
// goes with it, in one place.
//
// This is a third vocabulary on purpose, and the three are not interchangeable:
//   - the backend's `memberStatuses` are the long, unambiguous forms an operator reads
//     while *choosing* a status ("Venter på at blive hentet"),
//   - `memberStatusPhrase` is the lower-case form for mid-sentence use,
//   - and these are the short forms that fit on a Tag beside a name.
// "Udgår" and "Udgået" only read as different states because they sit in a coloured badge;
// spelled out mid-sentence they would be a riddle.
//
// Label, severity and icon travel together because they describe one thing. Kept apart,
// an amber badge beside a red icon for the same status is not a styling slip — it tells an
// operator two different things about the same member.
export interface MemberStatusBadge {
  label: string
  severity: string
  icon: string
}

const memberStatusBadges: Record<string, MemberStatusBadge> = {
  registered: { label: 'Tilmeldt', severity: 'secondary', icon: 'pi pi-user' },
  seated: { label: 'Har plads', severity: 'secondary', icon: 'pi pi-ticket' },
  // On the route, under their own steam.
  racing: { label: 'Aktiv', severity: 'success', icon: 'pi pi-directions' },
  finished: { label: 'Gennemført', severity: 'success', icon: 'pi pi-flag' },
  // Has asked to leave the race but is still by the trailside: the intention is recorded,
  // the outcome is not, which is why this warns rather than reading as final.
  waiting: { label: 'Udgår', severity: 'warn', icon: 'pi pi-clock' },
  transit: { label: 'Transit', severity: 'warn', icon: 'pi pi-car' },
  // At HQ: out of the race for good, hence the strongest colour of the set.
  sheltered: { label: 'Udgået', severity: 'danger', icon: 'pi pi-home' },
  reunited: { label: 'Genforenet', severity: 'info', icon: 'pi pi-users' },
  released: { label: 'Afhentet', severity: 'secondary', icon: 'pi pi-sign-out' },
}

// An absent status is not an error: the member is on the roster and the race has not
// claimed them yet.
const notStartedBadge: MemberStatusBadge = {
  label: 'Ikke startet',
  severity: 'contrast',
  icon: 'pi pi-minus-circle',
}

// An unknown slug keeps its own name rather than being flattened into "Ikke startet": a
// status this build has not heard of is a deploy skew, and hiding it would look like data
// loss to the operator reading the screen.
export const memberStatusBadge = (slug: string): MemberStatusBadge =>
  memberStatusBadges[slug] ?? (slug ? { ...notStartedBadge, label: slug } : notStartedBadge)

// What *happened*, as opposed to where the member ended up.
//
// Kept apart from the status labels because the two carry different information and a
// timeline needs both: `racing` reached by carrying on under their own steam and `racing`
// reached by being moved to another patrol are the same status and very different facts.
// Without this the history would render two identical lines.
const memberEventLabels: Record<string, string> = {
  started: 'Startede løbet',
  'withdrawal.requested': 'Ønskede at udgå',
  'withdrawal.cancelled': 'Fortsatte selv',
  'status.overridden': 'Rettet manuelt',
  'team.moved': 'Flyttet til anden patrulje',
  // Published by the car and shelter interfaces, which do not exist yet — named now so a
  // history recorded once they do reads correctly without a frontend change.
  'pickup.accepted': 'Taget op i bil',
  'shelter.accepted': 'Ankommet til HQ',
  'handover.completed': 'Overdraget',
}

export const memberEventPhrase = (event: string) => memberEventLabels[event] ?? event

export interface MemberChangeSummary {
  memberId: string
  name?: string
  from?: string
  to?: string
}

export interface MemberMoveSummary {
  memberId: string
  name?: string
  toTeamId: string
  toTeamName?: string
}

export interface MemberOperationSummary {
  teamId?: string
  teamName?: string
  fromTeamId?: string
  fromTeamName?: string
  members?: (MemberChangeSummary & MemberMoveSummary)[]
  teamStrength?: number
  fromTeamStrength?: number
}

/**
 * Parse a member-operation entry's stored summary.
 *
 * These entries carry JSON rather than a bare string, because one operation can touch
 * several members and the line has to name them. Returns null for anything unparseable
 * so the caller can fall back to the raw value: an entry an operator cannot fully read is
 * much better than one missing from a handover record.
 *
 * The summary is deliberately self-contained — names, statuses and the resulting strength
 * are all stored — so a line never changes meaning as the world moves on. Rendering must
 * therefore use *only* what is in here, never a lookup of a member's current state.
 */
export function parseMemberSummary(value?: string): MemberOperationSummary | null {
  if (!value) return null
  try {
    const parsed = JSON.parse(value)
    return parsed && typeof parsed === 'object' ? (parsed as MemberOperationSummary) : null
  } catch {
    return null
  }
}

const isMemberSummaryType = (type: string) =>
  type === 'member.status.changed' || type === 'member.moved' || type === 'team.collected'

export { isMemberSummaryType }

const activityIcons: Record<string, string> = {
  created: 'pi pi-flag',
  'headline.updated': 'pi pi-pencil',
  'description.updated': 'pi pi-pencil',
  commented: 'pi pi-comment',
  'comment.updated': 'pi pi-pencil',
  'severity.specified': 'pi pi-exclamation-triangle',
  assigned: 'pi pi-user',
  closed: 'pi pi-check-circle',
  reopened: 'pi pi-replay',
  'team.associated': 'pi pi-users',
  'team.disassociated': 'pi pi-user-minus',
  deleted: 'pi pi-trash',
  'member.status.changed': 'pi pi-user-edit',
  'member.moved': 'pi pi-arrow-right-arrow-left',
  'team.collected': 'pi pi-car',
}

export const activityIcon = (type: string) => activityIcons[type] ?? 'pi pi-circle'
