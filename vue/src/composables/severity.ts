// Grøn / gul / rød: the one urgency vocabulary the race-night screens share.
//
// Extracted from `composables/sos.ts` (task 112, PRD 009 §8). The nødtelefon defined
// these first and kept them in one place "so the list badge and the detail select cannot
// drift apart" — and a dispatch task's priority is deliberately *the same* vocabulary, so
// that a pickup created from a red case can arrive red and two desks working the same
// night do not have two words for urgent.
//
// A neutral module rather than importing `sos.ts` from the kørsel screens: a delivery of
// dinner has no business depending on the emergency-phone module to know what "rød" means.
// `sos.ts` re-exports these, so the nødtelefon's own call sites are untouched.
//
// The Go side makes the mirror-image decision for the mirror-image reason: `dispatch`
// declares its own three-value Priority rather than importing the `sos` package. When `sos`
// is lifted to shared-go (task 055) a shared types.Severity becomes the obvious home for
// both, and that is the moment to converge them.

export type Severity = '' | 'green' | 'yellow' | 'red'

// Danish labels. The values themselves are the API's (PRD 001 §11 Decisions).
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
