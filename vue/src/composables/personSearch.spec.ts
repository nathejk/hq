import { describe, expect, it } from 'vitest'
import {
  MIN_NAME_LENGTH,
  MIN_PHONE_DIGITS,
  SEARCH_DEPENDS_ON,
  type PersonResult,
  displayName,
  emptyState,
  isSearchable,
  kindGroup,
  phoneRoleLabel,
  queryDigits,
  resultRoute,
  statusBadge,
  REMOVED_BADGE,
  teamLabel,
  toRows,
} from './personSearch'

const result = (over: Partial<PersonResult> = {}): PersonResult => ({
  kind: 'spejder',
  id: 'p-1',
  year: '2026',
  name: 'Rakel A. Koch',
  phone: '24450125',
  phoneRole: 'own',
  teamId: 't-1',
  teamName: 'Ørnene',
  teamNumber: '42',
  status: 'racing',
  statusKind: 'member',
  removed: false,
  ...over,
})

describe('query classification', () => {
  it('normalises a Danish number to national digits', () => {
    expect(queryDigits('+45 20 30 96 96')).toBe('20309696')
    expect(queryDigits('20 30 96 96')).toBe('20309696')
    expect(queryDigits('tlf. 20 68 00 11')).toBe('20680011')
  })

  it('needs four digits or three letters before it will search', () => {
    expect(MIN_PHONE_DIGITS).toBe(4)
    expect(MIN_NAME_LENGTH).toBe(3)

    expect(isSearchable('')).toBe(false)
    expect(isSearchable('ab')).toBe(false)
    expect(isSearchable('203')).toBe(false)
    expect(isSearchable('2030')).toBe(true)
    expect(isSearchable('Koch')).toBe(true)
    expect(isSearchable('Søren')).toBe(true)
  })

  it('searches a mixed query, because both interpretations run', () => {
    expect(isSearchable('tlf. 20 68 00 11')).toBe(true)
  })
})

describe('empty states', () => {
  const response = (over: Partial<Parameters<typeof emptyState>[1] & object> = {}) => ({
    results: [],
    truncated: false,
    tooShort: false,
    matchedPhone: false,
    matchedName: false,
    ...over,
  })

  it('distinguishes nothing-typed from too-short from no-match', () => {
    expect(emptyState('', undefined)).toBe('idle')
    expect(emptyState('   ', response())).toBe('idle')
    expect(emptyState('ab', undefined)).toBe('tooShort')
    expect(emptyState('notfoundxyz', response())).toBe('noMatch')
    expect(emptyState('Koch', response({ results: [result()] }))).toBe('results')
  })

  it('honours the server saying nothing was searched', () => {
    expect(emptyState('Koch', response({ tooShort: true }))).toBe('tooShort')
  })
})

describe('rows', () => {
  it('groups kinds under Danish headings, contacts together', () => {
    expect(kindGroup('spejder')).toBe('Spejdere')
    expect(kindGroup('senior')).toBe('Seniorer')
    expect(kindGroup('patruljekontakt')).toBe('Kontaktpersoner')
    expect(kindGroup('klankontakt')).toBe('Kontaktpersoner')
    expect(kindGroup('gøgler')).toBe('Gøglere')
    expect(kindGroup('crew')).toBe('Crew')
  })

  it('orders groups and keeps the server order within a group', () => {
    const rows = toRows([
      result({ kind: 'crew', id: 'c-1' }),
      result({ kind: 'spejder', id: 's-2' }),
      result({ kind: 'spejder', id: 's-1' }),
      result({ kind: 'senior', id: 'se-1' }),
    ])
    expect(rows.map((r) => r.id)).toEqual(['s-2', 's-1', 'se-1', 'c-1'])
    expect(rows.map((r) => r.group)).toEqual(['Spejdere', 'Spejdere', 'Seniorer', 'Crew'])
  })

  it('keys rows uniquely when one person matches twice', () => {
    const rows = toRows([result(), result()])
    expect(rows[0].rowKey).not.toBe(rows[1].rowKey)
  })
})

describe('row content', () => {
  it('annotates a number that is not the person’s own, and only then', () => {
    expect(phoneRoleLabel('own')).toBe('')
    expect(phoneRoleLabel('')).toBe('')
    expect(phoneRoleLabel('parent')).toBe('forælders nummer')
    expect(phoneRoleLabel('contact')).toBe('kontaktperson')
  })

  it('renders a missing name and an unnamed team gracefully', () => {
    expect(displayName(result({ name: '   ' }))).toBe('(uden navn)')
    expect(teamLabel(result({ teamName: '', teamNumber: '17' }))).toBe('17')
    expect(teamLabel(result({ teamName: '', teamNumber: '' }))).toBe('(uden navn)')
    expect(teamLabel(result({ teamId: '', teamName: '', teamNumber: '' }))).toBe('')
  })

  it('links a scout to its patrol and everyone else to the list holding them', () => {
    expect(resultRoute(result())).toEqual({ name: 'patrulje', params: { teamId: 't-1' } })
    expect(resultRoute(result({ kind: 'patruljekontakt' }))).toEqual({
      name: 'patrulje',
      params: { teamId: 't-1' },
    })
    expect(resultRoute(result({ kind: 'spejder', teamId: '' }))).toBeNull()
    expect(resultRoute(result({ kind: 'senior' }))).toEqual({ name: 'klaner' })
    expect(resultRoute(result({ kind: 'klankontakt' }))).toEqual({ name: 'klaner' })
    expect(resultRoute(result({ kind: 'gøgler' }))).toEqual({ name: 'badutter' })
    expect(resultRoute(result({ kind: 'friend' }))).toEqual({ name: 'badutter' })
    expect(resultRoute(result({ kind: 'crew' }))).toEqual({ name: 'organisation' })
  })
})

describe('status badges', () => {
  it('shows a status on ordinary rows too, not only unusual ones', () => {
    expect(statusBadge(result({ status: 'racing', statusKind: 'member' })).label).toBe('Aktiv')
    expect(statusBadge(result({ status: 'PAID', statusKind: 'team' })).label).toBe('Holdet: betalt')
  })

  it('reuses the member vocabulary the rest of HQ already shows', () => {
    // Same words and colours as the patrol page, deliberately: a scout who reads "Udgår" there must
    // not read something else here.
    expect(statusBadge(result({ status: 'waiting', statusKind: 'member' }))).toMatchObject({
      label: 'Udgår',
      severity: 'warn',
    })
    expect(statusBadge(result({ status: 'sheltered', statusKind: 'member' })).label).toBe('Udgået')
  })

  it('styles a departed person differently from a racing one', () => {
    const racing = statusBadge(result({ status: 'racing', statusKind: 'member' }))
    const released = statusBadge(result({ status: 'released', statusKind: 'member' }))
    const sheltered = statusBadge(result({ status: 'sheltered', statusKind: 'member' }))

    expect(released.severity).not.toBe(racing.severity)
    expect(released.label).not.toBe(racing.label)
    expect(sheltered.severity).not.toBe(racing.severity)
  })

  it('keeps “udmeldt” distinguishable from “released”, on a separate axis', () => {
    const released = statusBadge(result({ status: 'released', statusKind: 'member' }))
    expect(REMOVED_BADGE.label).toBe('Udmeldt')
    expect(REMOVED_BADGE.label).not.toBe(released.label)
    expect(REMOVED_BADGE.severity).not.toBe(released.severity)
    expect(REMOVED_BADGE.icon).not.toBe(released.icon)

    // And removal does not replace the status: both facts survive.
    const removedAndReleased = result({ status: 'released', statusKind: 'member', removed: true })
    expect(statusBadge(removedAndReleased).label).toBe(released.label)
  })

  it('renders an unknown status as unknown rather than as registered', () => {
    const unknown = statusBadge(result({ status: '', statusKind: '' }))
    expect(unknown.label).toBe('Ukendt status')
    expect(unknown.label).not.toMatch(/tilmeldt/i)
    // Also when the vocabulary is named but the status is not.
    expect(statusBadge(result({ status: '', statusKind: 'member' })).label).toBe('Ukendt status')
    expect(statusBadge(result({ status: undefined, statusKind: undefined })).label).toBe(
      'Ukendt status',
    )
  })

  it('names a team status as the team’s, so PAY does not read as a debt of the person’s', () => {
    expect(statusBadge(result({ status: 'PAY', statusKind: 'team' })).label).toBe(
      'Holdet: mangler betaling',
    )
    expect(statusBadge(result({ status: 'OUT', statusKind: 'team' })).label).toBe('Holdet: ude')
    // "Udgået" belongs to `sheltered`; two states must not share a word.
    expect(statusBadge(result({ status: 'OUT', statusKind: 'team' })).label).not.toContain('Udgået')
  })

  it('keeps an unrecognised slug visible rather than flattening it', () => {
    expect(statusBadge(result({ status: 'WHATEVER', statusKind: 'team' })).label).toBe(
      'Holdet: WHATEVER',
    )
  })

  it('carries the long form for hovering, because colour cannot say it all', () => {
    expect(statusBadge(result({ status: 'released', statusKind: 'member' })).title).toBe(
      'hentet af forældre',
    )
  })
})

describe('live dependencies', () => {
  // Pinned against go/nathejk/table/searchperson/entities_test.go. A wrong token here fails
  // silently — the page looks live and never updates — so the set is asserted rather than trusted.
  it('names the eight entity tokens the projection can emit, and no others', () => {
    expect([...SEARCH_DEPENDS_ON]).toEqual([
      'crew',
      'crewmember',
      'friend',
      'gøgler',
      'klan',
      'patrulje',
      'senior',
      'spejder',
    ])
  })

  it('does not name the two tokens that do not exist', () => {
    expect(SEARCH_DEPENDS_ON).not.toContain('personnel')
    expect(SEARCH_DEPENDS_ON).not.toContain('bandit')
  })
})
