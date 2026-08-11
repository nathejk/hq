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
}

export const activityLabel = (type: string) => activityLabels[type] ?? type

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
}

export const activityIcon = (type: string) => activityIcons[type] ?? 'pi pi-circle'
