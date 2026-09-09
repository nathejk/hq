import { describe, it, expect } from 'vitest'
import {
  allMaps,
  checkpointsWithoutMap,
  extentFromCorners,
  formatLabel,
  groupSelectionState,
  handoutOptions,
  handoutToId,
  handoutToOption,
  HANDOUT_ON_QR_OPTION,
  HANDOUT_ON_QR_LABEL,
  isDegenerate,
  orderPicks,
  sameExtent,
  someMapContainsAll,
  splitCheckgroups,
  teamTypeLabel,
  teamTypeOptions,
  teamTypeToOption,
  teamTypeToValue,
  TEAM_TYPE_NONE_OPTION,
  toggleGroupSelection,
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
  handoutCheckgroupId: '',
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

describe('handoutOptions', () => {
  // The QR rule is the exception, so the checkgroups come first in course order and it comes last —
  // an operator picking the ordinary answer should not have to read past it.
  it('lists the checkgroups in order, with the QR rule last', () => {
    const options = handoutOptions([
      { id: 'cg-1', name: 'Post 1-3' },
      { id: 'cg-2', name: 'Post 4-6' },
    ])

    expect(options).toEqual([
      { value: 'cg-1', label: 'Post 1-3' },
      { value: 'cg-2', label: 'Post 4-6' },
      { value: HANDOUT_ON_QR_OPTION, label: HANDOUT_ON_QR_LABEL },
    ])
  })

  it('offers the QR rule even with no checkgroups', () => {
    expect(handoutOptions()).toEqual([{ value: HANDOUT_ON_QR_OPTION, label: HANDOUT_ON_QR_LABEL }])
  })

  // The one thing that must never regress: the dropdown's QR value cannot be the empty string.
  // PrimeVue's Select decides "is anything selected?" with isNotEmpty(modelValue), and isEmpty('') is
  // true — so an option valued '' renders as the placeholder for ever, and picking it changes
  // nothing, which is how "Udleveret" became unsaveable.
  it('does not use the empty string as a selectable value', () => {
    expect(HANDOUT_ON_QR_OPTION).not.toBe('')
    expect(handoutOptions().every((option) => option.value !== '')).toBe(true)
  })
})

describe('handout value mapping', () => {
  it('shows a stored checkgroup id as itself', () => {
    expect(handoutToOption('cg-1')).toBe('cg-1')
    expect(handoutToId('cg-1')).toBe('cg-1')
  })

  // The API's "" and the dropdown's sentinel are the same fact in two vocabularies, so the round trip
  // has to be lossless in both directions or a save would move a sheet's reveal rule by accident.
  it('round-trips the QR rule between the API and the dropdown', () => {
    expect(handoutToOption('')).toBe(HANDOUT_ON_QR_OPTION)
    expect(handoutToId(HANDOUT_ON_QR_OPTION)).toBe('')
    expect(handoutToId(handoutToOption(''))).toBe('')
    expect(handoutToOption(handoutToId(HANDOUT_ON_QR_OPTION))).toBe(HANDOUT_ON_QR_OPTION)
  })

  // A sheet from an API that predates the field, or a Select cleared to null, must land on the QR
  // rule rather than on undefined — that is what the column's default means.
  it('treats a missing value as the QR rule', () => {
    expect(handoutToOption(undefined)).toBe(HANDOUT_ON_QR_OPTION)
    expect(handoutToOption(null)).toBe(HANDOUT_ON_QR_OPTION)
    expect(handoutToId(null)).toBe('')
  })
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

describe('splitCheckgroups', () => {
  const sheetWith = (id: string, name: string, checkpointIds: string[]): Kort => ({
    ...sheet(id, checkpointIds),
    name,
  })

  // The mistake with a race-day cost: the group reveals as a whole, so the patrol sees a post it has
  // no sheet for.
  it('reports a checkgroup split across two sheets', () => {
    const s = set([sheetWith('k-1', 'Kort 1', ['cp-1']), sheetWith('k-2', 'Kort 2', ['cp-2'])])
    const splits = splitCheckgroups(s, [group('cg-1', ['cp-1', 'cp-2'])])

    expect(splits).toHaveLength(1)
    expect(splits[0].checkgroupName).toBe('cg-1')
    expect(splits[0].sheetNames).toEqual(['Kort 1', 'Kort 2'])
  })

  // Existential, not partitioning: overlap is designed in, so two sheets that each hold the whole
  // group is fine. A partitioning test would fire on the normal case and be ignored.
  it('stays quiet when two overlapping sheets each hold the whole group', () => {
    const s = set([sheetWith('k-1', 'Kort 1', ['cp-1', 'cp-2']), sheetWith('k-2', 'Kort 2', ['cp-1', 'cp-2'])])
    expect(splitCheckgroups(s, [group('cg-1', ['cp-1', 'cp-2'])])).toEqual([])
  })

  it('stays quiet when one sheet holds the whole group', () => {
    const s = set([sheetWith('k-1', 'Kort 1', ['cp-1', 'cp-2', 'cp-3'])])
    expect(splitCheckgroups(s, [group('cg-1', ['cp-1', 'cp-2'])])).toEqual([])
  })

  // Not reported here: that is the unassigned list's job, and saying it twice would train the
  // operator to skim both.
  it('ignores a checkgroup that is on no sheet at all', () => {
    const s = set([sheetWith('k-1', 'Kort 1', ['cp-9'])])
    expect(splitCheckgroups(s, [group('cg-1', ['cp-1', 'cp-2'])])).toEqual([])
  })

  it('ignores an empty checkgroup', () => {
    const s = set([sheetWith('k-1', 'Kort 1', ['cp-1'])])
    expect(splitCheckgroups(s, [group('cg-empty', [])])).toEqual([])
  })

  // Membership, never geometry: a group's checkpoints may legitimately sit in the two different
  // areas of one double-sided sheet, and comparing positions would false-alarm on every one.
  it('accepts a group spread across the two areas of a single sheet', () => {
    const doubleSided: Kort = {
      ...sheetWith('k-1', 'Kort 5', ['cp-1', 'cp-2']),
      extents: [
        { northWest: { latitude: 56, longitude: 9.0 }, southEast: { latitude: 55.5, longitude: 9.4 } },
        { northWest: { latitude: 55, longitude: 10.0 }, southEast: { latitude: 54.5, longitude: 10.4 } },
      ],
    }
    expect(splitCheckgroups(set([doubleSided]), [group('cg-1', ['cp-1', 'cp-2'])])).toEqual([])
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

  // The "no team type" option is the crew set — the commonest answer — so it leads and is labelled
  // rather than left as a blank line an operator reads as "not filled in".
  //
  // Its value is deliberately **not** `null`, which is what the API stores: PrimeVue's Select treats
  // `null` as "nothing selected", so an option valued `null` renders as the placeholder however it is
  // chosen. Pinned here because it looks like harmless indirection and is not.
  it('offers no-team-type first, with a label and a selectable value', () => {
    expect(teamTypeOptions[0]).toEqual({ value: TEAM_TYPE_NONE_OPTION, label: 'Ingen bestemt holdtype' })
    expect(TEAM_TYPE_NONE_OPTION).not.toBeNull()
    expect(teamTypeOptions.every((option) => option.value !== null && option.value !== '')).toBe(true)
  })

  it('round-trips the no-team-type choice between the API and the dropdown', () => {
    expect(teamTypeToOption(null)).toBe(TEAM_TYPE_NONE_OPTION)
    expect(teamTypeToValue(TEAM_TYPE_NONE_OPTION)).toBeNull()
    expect(teamTypeToOption('patrulje')).toBe('patrulje')
    expect(teamTypeToValue('patrulje')).toBe('patrulje')
  })
})
