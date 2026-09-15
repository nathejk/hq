import { describe, it, expect } from 'vitest'
import {
  LABEL_NO_SECTION,
  buildPersonnelTree,
  groupPersonnel,
  selectionKeysFor,
  userIdFromSelection,
  type AssignablePerson,
} from './personnelTree'

// The order here is the order the API serves: postmandskab first, then sections by label, then
// the signed-up helpers. The client's job is to preserve it, not to reproduce it.
const offered: AssignablePerson[] = [
  { id: 'u1', name: 'Alma', sectionSlug: 'postmandskab', sectionLabel: 'Postmandskab', priority: true },
  { id: 'u2', name: 'Bo', sectionSlug: 'postmandskab', sectionLabel: 'Postmandskab', priority: true },
  { id: 'u3', name: 'Cecilie', sectionSlug: 'goeglerledelse', sectionLabel: 'Gøglerledelse' },
  { id: 'u4', name: 'Ditte', sectionSlug: 'hq', sectionLabel: 'HQ' },
  { id: 'u5', name: 'Erik', sectionLabel: 'Tilmeldte hjælpere' },
]

describe('groupPersonnel', () => {
  it('groups by section without reordering what the server sent', () => {
    const groups = groupPersonnel(offered)
    expect(groups.map((g) => g.label)).toEqual(['Postmandskab', 'Gøglerledelse', 'HQ', 'Tilmeldte hjælpere'])
    expect(groups[0].items.map((p) => p.name)).toEqual(['Alma', 'Bo'])
  })

  it('marks the postmandskab group as prioritised', () => {
    const groups = groupPersonnel(offered)
    expect(groups[0].priority).toBe(true)
    expect(groups.slice(1).every((g) => !g.priority)).toBe(true)
  })

  // The flag travels per person, so a section whose first row happens to lack it must not lose
  // its place in the tree.
  it('prioritises a group if any of its people are prioritised', () => {
    const groups = groupPersonnel([
      { id: 'a', name: 'A', sectionSlug: 'postmand', sectionLabel: 'Postmand' },
      { id: 'b', name: 'B', sectionSlug: 'postmand', sectionLabel: 'Postmand', priority: true },
    ])
    expect(groups).toHaveLength(1)
    expect(groups[0].priority).toBe(true)
  })

  it('labels people with no section rather than making an empty heading', () => {
    const groups = groupPersonnel([{ id: 'a', name: 'A' }])
    expect(groups[0].label).toBe(LABEL_NO_SECTION)
  })

  it('handles an empty organisation', () => {
    expect(groupPersonnel([])).toEqual([])
  })

  // Two people from the same section arriving either side of another section stay in two
  // groups: the list is pre-sorted, and quietly merging them would hide a sorting bug.
  it('does not merge non-adjacent runs of the same section', () => {
    const groups = groupPersonnel([
      { id: 'a', name: 'A', sectionLabel: 'HQ' },
      { id: 'b', name: 'B', sectionLabel: 'Team' },
      { id: 'c', name: 'C', sectionLabel: 'HQ' },
    ])
    expect(groups.map((g) => g.label)).toEqual(['HQ', 'Team', 'HQ'])
  })
})

describe('buildPersonnelTree', () => {
  const tree = buildPersonnelTree(groupPersonnel(offered))

  // Flat, not behind a branch: TreeSelect cannot be told to open a branch (its `expandedKeys` is
  // private state it resets whenever options or value change), so the only way the postmandskab
  // are on screen at first click is to not nest them.
  it('puts the prioritised people at the top level', () => {
    expect(tree.slice(0, 2).map((n) => n.label)).toEqual(['Alma', 'Bo'])
    expect(tree.slice(0, 2).every((n) => n.leaf)).toBe(true)
  })

  it('puts everyone else under their section, with a count', () => {
    expect(tree.slice(2).map((n) => n.label)).toEqual([
      'Gøglerledelse (1)',
      'HQ (1)',
      'Tilmeldte hjælpere (1)',
    ])
    expect(tree[2].children?.map((c) => c.label)).toEqual(['Cecilie'])
  })

  // A section cannot staff a post. If it were selectable, picking one would save a shift
  // assigned to a section slug — a row belonging to nobody.
  it('makes sections unselectable and people selectable', () => {
    expect(tree[2].selectable).toBe(false)
    expect(tree[2].children?.[0].leaf).toBe(true)
  })

  it('keys people by userId, which is what gets saved', () => {
    expect(tree.slice(0, 2).map((n) => n.key)).toEqual(['u1', 'u2'])
    expect(tree[2].children?.map((c) => c.key)).toEqual(['u3'])
  })

  // Branch keys and person keys share one namespace in a TreeSelect, so they must not be able
  // to collide.
  it('namespaces section keys away from userIds', () => {
    expect(tree.slice(2).every((node) => node.key.startsWith('sec:'))).toBe(true)
    expect(tree.slice(0, 2).some((node) => node.key.startsWith('sec:'))).toBe(false)
  })

  it('keys a section by slug so it survives a label rename', () => {
    expect(tree[2].key).toBe('sec:goeglerledelse')
  })

  // The year in this database has nobody in postmandskab; the picker must still work, just
  // entirely sectioned.
  it('is all branches when nobody is prioritised', () => {
    const tree = buildPersonnelTree(groupPersonnel([{ id: 'a', name: 'A', sectionLabel: 'HQ' }]))
    expect(tree.map((n) => n.label)).toEqual(['HQ (1)'])
  })
})

describe('selectionKeysFor', () => {
  it('wraps a userId in the shape TreeSelect expects', () => {
    expect(selectionKeysFor('u1')).toEqual({ u1: true })
  })

  // Not `{}`: an empty object is truthy, and TreeSelect shows its placeholder only for a falsy
  // value — so an unselected row would render as blank-but-filled.
  it('is null when nothing is selected', () => {
    expect(selectionKeysFor(undefined)).toBeNull()
    expect(selectionKeysFor('')).toBeNull()
  })
})

describe('userIdFromSelection', () => {
  it('reads the selected userId back out', () => {
    expect(userIdFromSelection({ u1: true })).toBe('u1')
  })

  it('is empty when the operator clears the field', () => {
    expect(userIdFromSelection(null)).toBe('')
    expect(userIdFromSelection({})).toBe('')
  })

  // Defence in depth: `selectable: false` should stop this, but a section key saved as a userId
  // would create a shift assigned to nobody, which nothing downstream would flag.
  it('never returns a section key', () => {
    expect(userIdFromSelection({ 'sec:postmandskab': true })).toBe('')
  })

  it('ignores keys that are explicitly false', () => {
    expect(userIdFromSelection({ u1: false, u2: true })).toBe('u2')
  })

  it('round-trips with selectionKeysFor', () => {
    expect(userIdFromSelection(selectionKeysFor('u9'))).toBe('u9')
  })
})
