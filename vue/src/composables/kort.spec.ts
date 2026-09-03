import { describe, it, expect } from 'vitest'
import {
  allMaps,
  checkpointsWithoutMap,
  formatLabel,
  someMapContainsAll,
  teamTypeLabel,
  teamTypeOptions,
  KORT_DEPENDENCIES,
  type Kort,
  type Kortsaet,
} from './kort'

const sheet = (id: string, checkpointIds: string[] = []): Kort => ({
  id,
  kortsaetId: 's-1',
  year: '2026',
  version: 0,
  name: id,
  format: 'a4',
  note: '',
  sortOrder: 0,
  checkpointIds,
  extents: [],
})

const set = (kort: Kort[], teamType: Kortsaet['teamType'] = null): Kortsaet => ({
  id: 's-1',
  year: '2026',
  version: 0,
  name: 'Patruljer',
  sortOrder: 0,
  teamType,
  kort,
})

describe('KORT_DEPENDENCIES', () => {
  // The tokens are the event subjects' entities, and a wrong one fails silently — the page looks
  // live and never updates. Pinned because they are not guessable from the UI's own vocabulary.
  it('depends on entity types, including the ones the payload merely joins in', () => {
    expect([...KORT_DEPENDENCIES]).toEqual(['kort', 'kortsaet', 'checkpoint', 'checkgroup'])
  })
})

describe('someMapContainsAll', () => {
  // A checkgroup is revealed as a whole, so the question is whether *some* sheet covers all of it.
  it('is satisfied when one sheet holds the whole checkgroup', () => {
    expect(someMapContainsAll(set([sheet('k-1', ['cp-1', 'cp-2', 'cp-3'])]), ['cp-1', 'cp-2'])).toBe(true)
  })

  // The mistake worth warning about: the patrol would see checkpoints it holds no sheet for.
  it('fails when the checkgroup is split across two sheets', () => {
    const sets = set([sheet('k-1', ['cp-1']), sheet('k-2', ['cp-2'])])
    expect(someMapContainsAll(sets, ['cp-1', 'cp-2'])).toBe(false)
  })

  // Adjacent sheets overlap by design, so a partitioning test would fire constantly. Two sheets
  // that each hold the whole group is fine.
  it('accepts overlapping sheets that both hold the whole group', () => {
    const sets = set([sheet('k-1', ['cp-1', 'cp-2']), sheet('k-2', ['cp-1', 'cp-2'])])
    expect(someMapContainsAll(sets, ['cp-1', 'cp-2'])).toBe(true)
  })

  // A checkgroup with no checkpoints yet is an unfinished course, not a coverage failure.
  it('treats an empty checkgroup as covered', () => {
    expect(someMapContainsAll(set([]), [])).toBe(true)
  })
})

describe('checkpointsWithoutMap', () => {
  it('reports checkpoints on no sheet of the set', () => {
    const sets = set([sheet('k-1', ['cp-1']), sheet('k-2', ['cp-2'])])
    expect(checkpointsWithoutMap(sets, ['cp-1', 'cp-2', 'cp-3'])).toEqual(['cp-3'])
  })

  // Before any set exists, everything is unassigned — which is the true state of a fresh year and
  // must not read as "all covered".
  it('reports everything when there is no set', () => {
    expect(checkpointsWithoutMap(undefined, ['cp-1', 'cp-2'])).toEqual(['cp-1', 'cp-2'])
  })
})

describe('allMaps', () => {
  // Orphans are included, because a sheet whose set is unknown is exactly what an operator opened
  // this screen to find. Dropping it would make the mistake invisible.
  it('includes sheets whose set is unknown', () => {
    const payload = { kortsaet: [set([sheet('k-1')])], orphanKort: [sheet('k-lost')] }
    expect(allMaps(payload).map((m) => m.id)).toEqual(['k-1', 'k-lost'])
  })

  it('survives an absent payload', () => {
    expect(allMaps(undefined)).toEqual([])
  })
})

describe('labels', () => {
  it('labels the formats in Danish', () => {
    expect(formatLabel('a4')).toBe('A4')
    expect(formatLabel('skitse')).toBe('Skitse')
  })

  // A value this build does not know about still reads as something: an unlabelled row looks like
  // missing data, which sends an operator looking for a bug that is not there.
  it('falls back to the raw value', () => {
    expect(formatLabel('a2')).toBe('a2')
    expect(formatLabel('')).toBe('')
  })

  it('labels a team type, and renders an unmarked set as nothing', () => {
    expect(teamTypeLabel('patrulje')).toBe('Patruljer')
    expect(teamTypeLabel(null)).toBe('')
  })

  // The empty option is the crew set — the commonest answer — so it leads and is labelled rather
  // than left as a blank line an operator reads as "not filled in".
  it('offers no-team-type first, with a label', () => {
    expect(teamTypeOptions[0]).toEqual({ value: null, label: 'Ingen bestemt holdtype' })
  })
})
