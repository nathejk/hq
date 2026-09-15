/**
 * The person picker on the postmandskab screen: who is offered, grouped how, and open or shut.
 *
 * Extracted from `PostmandskabModal.vue` because the awkward part is not the markup. PrimeVue's
 * TreeSelect speaks selection *keys* rather than values, section branches and people share one
 * key namespace, and "postmandskab expanded, the rest collapsed" is derived from the data rather
 * than written down — three things that are wrong in silent ways and cannot be tested inside a
 * single-file component, since this project's vitest setup is node-only with no DOM.
 */

/** A person the API offers for a post. Shaped by `assignablePersonnel` in the Go API. */
export type AssignablePerson = {
  id: string
  name: string
  sectionSlug?: string
  sectionLabel?: string
  /** True for the postmandskab, who are offered first and expanded. */
  priority?: boolean
}

export type PersonnelGroup = {
  key: string
  label: string
  priority: boolean
  items: AssignablePerson[]
}

export type PersonnelNode = {
  key: string
  label: string
  selectable?: boolean
  leaf?: boolean
  children?: PersonnelNode[]
}

/** Shown for anyone with no section, rather than an empty group heading. */
export const LABEL_NO_SECTION = 'Uden sektion'

/**
 * Cut the offered people into section groups, preserving the order given.
 *
 * The server has already sorted them — postmandskab first, then sections by label, then the
 * signed-up helpers — so this walks the list rather than sorting again. Re-sorting here is how
 * the client and the server would come to disagree about which section is prioritised.
 */
export function groupPersonnel(people: AssignablePerson[]): PersonnelGroup[] {
  const groups: PersonnelGroup[] = []
  for (const person of people) {
    const label = person.sectionLabel || LABEL_NO_SECTION
    const last = groups[groups.length - 1]
    if (last && last.label === label) {
      last.items.push(person)
      // A group is prioritised if anyone in it is: the flag travels per person, and a
      // section whose first row happened to lack it must not lose its place.
      last.priority = last.priority || !!person.priority
      continue
    }
    groups.push({ key: sectionKey(person), label, priority: !!person.priority, items: [person] })
  }
  return groups
}

/**
 * The key for a section branch.
 *
 * Prefixed because branches and people share one key namespace in a TreeSelect, and an
 * unprefixed section key could in principle collide with a userId — which would make selecting
 * a person silently select a section, or vice versa. Keyed by slug where there is one so a
 * section's expanded state survives its label being renamed.
 */
function sectionKey(person: AssignablePerson): string {
  return 'sec:' + (person.sectionSlug || person.sectionLabel || LABEL_NO_SECTION)
}

/**
 * The tree: prioritised people at the top level, everyone else behind a section branch.
 *
 * The postmandskab are *not* wrapped in a branch, and that is the whole trick. They normally staff
 * the posts, so their names must be on screen the moment the picker opens — and TreeSelect keeps
 * `expandedKeys` as internal state rather than a prop, resetting it whenever its options or value
 * change, so "open this branch by default" is not something a caller can ask for. Reaching into the
 * instance to force it was tried and is what broke the dropdown: the write throws on the public
 * proxy, from inside `before-show`, before the overlay is ever shown. Flattening the prioritised
 * section says the same thing using only the component's own API.
 *
 * Other sections stay branches, collapsed by TreeSelect's default, which is what we want: one
 * click rather than one long scroll. Sections are `selectable: false` — a section cannot staff a
 * post, and offering one would save a shift whose userId is a section slug.
 */
export function buildPersonnelTree(groups: PersonnelGroup[]): PersonnelNode[] {
  const nodes: PersonnelNode[] = []
  for (const group of groups) {
    if (group.priority) {
      nodes.push(...group.items.map(personNode))
      continue
    }
    nodes.push({
      key: group.key,
      // The count is what makes a shut branch worth shutting: "Hønsegård (8)" says whether it is
      // worth opening.
      label: `${group.label} (${group.items.length})`,
      selectable: false,
      children: group.items.map(personNode),
    })
  }
  return nodes
}

function personNode(person: AssignablePerson): PersonnelNode {
  return { key: person.id, label: person.name, leaf: true }
}

/**
 * A userId as TreeSelect's `modelValue`.
 *
 * Returns null rather than `{}` for "nothing selected": an empty object is truthy, and
 * TreeSelect renders its placeholder only for a falsy value.
 */
export function selectionKeysFor(userId?: string | null): Record<string, boolean> | null {
  return userId ? { [userId]: true } : null
}

/**
 * The userId out of a TreeSelect selection.
 *
 * Tolerates null (emitted when the operator clears the field) and, defensively, a section key —
 * which `selectable: false` should already prevent, but which would otherwise be saved as a
 * userId and produce a shift assigned to nobody.
 */
export function userIdFromSelection(keys: Record<string, boolean> | null | undefined): string {
  if (!keys) return ''
  const selected = Object.keys(keys).filter((key) => keys[key] && !key.startsWith('sec:'))
  return selected.length > 0 ? selected[0] : ''
}
