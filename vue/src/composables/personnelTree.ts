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
 * The tree: a branch per section, people as leaves.
 *
 * Sections are `selectable: false` — a section cannot staff a post, and offering one as a
 * choice would produce a confusing no-op (or worse, a saved row with a section's slug as its
 * userId). The count in the branch label is what makes a collapsed branch worth collapsing:
 * "Hønsegård (8)" says whether it is worth opening.
 */
export function buildPersonnelTree(groups: PersonnelGroup[]): PersonnelNode[] {
  return groups.map((group) => ({
    key: group.key,
    label: `${group.label} (${group.items.length})`,
    selectable: false,
    children: group.items.map((person) => ({
      key: person.id,
      label: person.name,
      leaf: true,
    })),
  }))
}

/**
 * Which branches start open: the prioritised ones, and nothing else.
 *
 * The postmandskab normally staff the posts, so their names should be there to click without a
 * second gesture, while everyone else stays one click rather than one scroll away. Derived from
 * the data rather than from a hardcoded slug, so the day a section is renamed the expansion
 * follows the priority instead of quietly opening nothing.
 */
export function expandedPriorityKeys(groups: PersonnelGroup[]): Record<string, boolean> {
  const keys: Record<string, boolean> = {}
  for (const group of groups) {
    if (group.priority) keys[group.key] = true
  }
  return keys
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
