import { describe, it, expect } from 'vitest'
import {
  allMaps,
  checkpointsWithoutMap,
  extentFromCorners,
  formatLabel,
  gapCells,
  groupSelectionState,
  isDegenerate,
  orderPicks,
  sameExtent,
  someMapContainsAll,
  teamTypeLabel,
  teamTypeOptions,
  toggleGroupSelection,
  unionBounds,
  KORT_DEPENDENCIES,
  type Extent,
  type Kort,
  type Kortsaet,
} from './kort'

const group = (id: string, checkpointIds: string[]) => ({
  id,
  name: id,
  checkpoints: checkpointIds.map((cp) => ({ id: cp })),
})

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

describe('extentFromCorners', () => {
  // Denmark: latitude ~55-57, longitude ~8-13. North is the larger latitude, west the smaller
  // longitude — trivial to write backwards, and a mirrored rectangle is not obviously wrong on
  // screen. It is wrong on the printed sheet, months later.
  it('puts the northern, western corner first whichever way the corners were clicked', () => {
    const topLeftFirst = extentFromCorners({ lat: 56.1, lng: 9.1 }, { lat: 55.8, lng: 9.6 })
    const bottomRightFirst = extentFromCorners({ lat: 55.8, lng: 9.6 }, { lat: 56.1, lng: 9.1 })

    expect(topLeftFirst).toEqual({
      northWest: { latitude: 56.1, longitude: 9.1 },
      southEast: { latitude: 55.8, longitude: 9.6 },
    })
    // Drawing from the other corner must produce the identical rectangle, or re-drawing the same
    // area would look like an edit and publish an event.
    expect(bottomRightFirst).toEqual(topLeftFirst)
  })

  // The two diagonals nobody clicks first but everybody eventually does.
  it('handles the other diagonal', () => {
    expect(extentFromCorners({ lat: 55.8, lng: 9.1 }, { lat: 56.1, lng: 9.6 })).toEqual({
      northWest: { latitude: 56.1, longitude: 9.1 },
      southEast: { latitude: 55.8, longitude: 9.6 },
    })
  })
})

describe('sameExtent', () => {
  const extent: Extent = {
    northWest: { latitude: 56.1, longitude: 9.1 },
    southEast: { latitude: 55.8, longitude: 9.6 },
  }

  it('compares corner for corner', () => {
    expect(sameExtent(extent, { ...extent })).toBe(true)
    expect(sameExtent(extent, { ...extent, southEast: { latitude: 55.9, longitude: 9.6 } })).toBe(false)
  })
})

describe('isDegenerate', () => {
  // Two clicks on the same spot draw nothing, and a saved invisible extent reads as a failed save.
  it('spots a rectangle with no height or no width', () => {
    expect(
      isDegenerate({ northWest: { latitude: 56.1, longitude: 9.1 }, southEast: { latitude: 56.1, longitude: 9.6 } }),
    ).toBe(true)
    expect(
      isDegenerate({ northWest: { latitude: 56.1, longitude: 9.1 }, southEast: { latitude: 55.8, longitude: 9.1 } }),
    ).toBe(true)
  })

  it('accepts a real rectangle', () => {
    expect(
      isDegenerate({ northWest: { latitude: 56.1, longitude: 9.1 }, southEast: { latitude: 55.8, longitude: 9.6 } }),
    ).toBe(false)
  })
})

describe('gapCells', () => {
  const rect = (north: number, west: number, south: number, east: number): Extent => ({
    northWest: { latitude: north, longitude: west },
    southEast: { latitude: south, longitude: east },
  })

  // The failure that matters: a strip of ground no sheet shows, between two sheets that each look
  // fine on their own. A patrol walking into it has nothing in their hands.
  it('finds a vertical seam between two side-by-side sheets', () => {
    const gaps = gapCells([rect(56, 9.0, 55, 9.4), rect(56, 9.5, 55, 9.9)])

    expect(gaps).toHaveLength(1)
    expect(gaps[0]).toEqual(rect(56, 9.4, 55, 9.5))
  })

  // Overlap is designed in, so touching or overlapping sheets must be silent — a check that cried
  // wolf on the normal case would be turned off within a day.
  it('reports nothing for sheets that touch exactly', () => {
    expect(gapCells([rect(56, 9.0, 55, 9.4), rect(56, 9.4, 55, 9.8)])).toEqual([])
  })

  it('reports nothing for overlapping sheets', () => {
    expect(gapCells([rect(56, 9.0, 55, 9.5), rect(56, 9.4, 55, 9.9)])).toEqual([])
  })

  // A thin seam is the dangerous one, and exact decomposition catches it at any width — which
  // sampling on a fixed grid would not.
  it('catches a very thin seam', () => {
    const gaps = gapCells([rect(56, 9.0, 55, 9.4), rect(56, 9.4001, 55, 9.8)])
    expect(gaps).toHaveLength(1)
  })

  // Two sheets stacked with a horizontal band missing between them.
  it('finds a horizontal seam', () => {
    const gaps = gapCells([rect(56.0, 9.0, 55.6, 9.5), rect(55.5, 9.0, 55.0, 9.5)])

    expect(gaps).toHaveLength(1)
    expect(gaps[0]).toEqual(rect(55.6, 9.0, 55.5, 9.5))
  })

  // The L-shape: the bounding box has a corner no sheet covers. That corner is *not* a seam between
  // sheets, but it is genuinely uncovered ground inside the area the set spans, and an operator
  // should see it and decide. Reporting it is the honest answer; the alternative would be guessing
  // which uncovered ground is intentional.
  it('reports the uncovered corner of an L-shaped layout', () => {
    const gaps = gapCells([rect(56, 9.0, 55.5, 9.4), rect(55.5, 9.0, 55.0, 9.4), rect(56, 9.4, 55.5, 9.8)])

    expect(gaps).toHaveLength(1)
    expect(gaps[0]).toEqual(rect(55.5, 9.4, 55.0, 9.8))
  })

  // Merged along the row, so a gap spanning several grid columns is one band rather than a mosaic
  // of squares an operator has to interpret.
  it('merges adjacent gap cells in a row', () => {
    const gaps = gapCells([rect(56, 9.0, 55, 9.2), rect(56, 9.4, 55, 9.6), rect(56, 9.8, 55, 9.9)])

    // Two gaps (9.2–9.4 and 9.6–9.8), not four cells.
    expect(gaps).toHaveLength(2)
  })

  // One sheet cannot have a seam with itself, and no sheets is not a seam either — it is a set
  // nobody has drawn yet.
  it('reports nothing for fewer than two rectangles', () => {
    expect(gapCells([])).toEqual([])
    expect(gapCells([rect(56, 9.0, 55, 9.4)])).toEqual([])
  })
})

describe('unionBounds', () => {
  it('spans every rectangle', () => {
    const bounds = unionBounds([
      { northWest: { latitude: 56, longitude: 9.0 }, southEast: { latitude: 55.5, longitude: 9.4 } },
      { northWest: { latitude: 55.8, longitude: 9.3 }, southEast: { latitude: 55.0, longitude: 9.9 } },
    ])
    expect(bounds).toEqual({
      northWest: { latitude: 56, longitude: 9.0 },
      southEast: { latitude: 55.0, longitude: 9.9 },
    })
  })

  it('is undefined with nothing to bound', () => {
    expect(unionBounds([])).toBeUndefined()
  })
})

describe('groupSelectionState', () => {
  it('reports all, some and none', () => {
    const g = group('cg-1', ['cp-1', 'cp-2'])
    expect(groupSelectionState(g, new Set(['cp-1', 'cp-2']))).toBe('all')
    expect(groupSelectionState(g, new Set(['cp-1']))).toBe('some')
    expect(groupSelectionState(g, new Set())).toBe('none')
  })

  // A half-ticked checkgroup is usually a mistake in the making, since a checkgroup is revealed as
  // a whole — so the third state has to be visible rather than rounded to on or off.
  it('does not round a partial selection to all', () => {
    expect(groupSelectionState(group('cg-1', ['cp-1', 'cp-2', 'cp-3']), new Set(['cp-1', 'cp-2']))).toBe('some')
  })

  it('treats an empty checkgroup as none', () => {
    expect(groupSelectionState(group('cg-1', []), new Set())).toBe('none')
  })
})

describe('toggleGroupSelection', () => {
  it('ticks the whole group when nothing is ticked', () => {
    const next = toggleGroupSelection(group('cg-1', ['cp-1', 'cp-2']), new Set())
    expect([...next]).toEqual(['cp-1', 'cp-2'])
  })

  // With three of four already on, reaching for the group header means "all of them", never "swap
  // them". Inverting each would be the surprising reading.
  it('completes a partial group rather than inverting it', () => {
    const next = toggleGroupSelection(group('cg-1', ['cp-1', 'cp-2', 'cp-3']), new Set(['cp-1']))
    expect([...next].sort()).toEqual(['cp-1', 'cp-2', 'cp-3'])
  })

  it('clears a fully ticked group', () => {
    const next = toggleGroupSelection(group('cg-1', ['cp-1', 'cp-2']), new Set(['cp-1', 'cp-2']))
    expect([...next]).toEqual([])
  })

  // Other sheets' picks are none of this group's business.
  it('leaves checkpoints outside the group alone', () => {
    const next = toggleGroupSelection(group('cg-1', ['cp-1']), new Set(['cp-9']))
    expect(next.has('cp-9')).toBe(true)
  })
})

describe('orderPicks', () => {
  const groups = [group('cg-1', ['cp-1', 'cp-2']), group('cg-2', ['cp-3'])]

  // The API compares the submitted list against the stored one to decide whether anything changed,
  // so the order must depend on the checkgroups and not on the order the boxes were ticked — or
  // every re-save would look like an edit and emit a live signal.
  it('is stable regardless of tick order', () => {
    expect(orderPicks(groups, new Set(['cp-3', 'cp-1']))).toEqual(['cp-1', 'cp-3'])
    expect(orderPicks(groups, new Set(['cp-1', 'cp-3']))).toEqual(['cp-1', 'cp-3'])
  })

  // Can only happen if a checkpoint vanished from the payload mid-edit; dropping it would make the
  // save do something the operator did not ask for.
  it('keeps picked ids that are in no checkgroup', () => {
    expect(orderPicks(groups, new Set(['cp-1', 'cp-gone']))).toEqual(['cp-1', 'cp-gone'])
  })

  it('never repeats an id', () => {
    const duplicated = [group('cg-1', ['cp-1']), group('cg-2', ['cp-1'])]
    expect(orderPicks(duplicated, new Set(['cp-1']))).toEqual(['cp-1'])
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
