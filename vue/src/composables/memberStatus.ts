// The member lifecycle vocabulary (PRD 006): one status, three renderings, in one place.
//
// Extracted from `composables/sos.ts` (task 183, PRD 014 §7) for the same reason and by the same
// pattern as `composables/severity.ts` before it: person search has to show a scout's status, and a
// results page has no business importing the emergency-phone module to learn what `sheltered` means.
// `sos.ts` re-exports these, so the nødtelefon's own call sites are untouched.
//
// The important property is that this is the *only* copy. A second word for `waiting` invented for
// search would mean the same scout reads as "Udgår" on the patrol page and something else here, and
// an operator would reasonably conclude they were two different facts.

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

// The status as a text colour, for places that show an icon rather than a tag: the card's member
// rows and the history timeline in the member dialog.
//
// Derived from the badge's severity rather than mapping slugs a second time — so a status whose
// colour changes changes here too, and a new status gets a sensible colour without being listed.
// Lives here rather than in a component because two components need it, and the copy in each was how
// they were going to drift (task 103).
export const memberStatusColour = (slug: string) => {
  switch (memberStatusBadge(slug).severity) {
    case 'success':
      return 'text-green-600'
    case 'danger':
      return 'text-red-600'
    case 'warn':
      return 'text-amber-600'
    case 'info':
      return 'text-blue-500'
    case 'secondary':
      return 'text-gray-500'
    default:
      return 'text-gray-400'
  }
}
