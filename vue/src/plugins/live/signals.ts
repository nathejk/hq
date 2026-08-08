/**
 * Live-update signal vocabulary.
 *
 * The stream carries *invalidation signals, not data*: the client is told that
 * something changed and refetches it over REST. That keeps one serialization
 * path, prevents the pushed and fetched shapes from drifting apart, and means a
 * signal leaks nothing an unauthorised client could not already ask for.
 *
 * See roadmap/prd/004-live-updates-spa.md.
 */

/** Signal names. Dispatch on this so new kinds are additive, not a format change. */
export const SIGNAL_ENTITY_CHANGED = 'entity.changed' as const;
export const SIGNAL_RESYNC = 'resync' as const;

/**
 * One entity instance changed.
 *
 * `entity` is the subject token from the event stream (`patrulje`, `payment`,
 * `sos`, …) — not a UI name. `id` is absent for subjects that carry no id, such
 * as year-level or collection-level events, in which case the signal means
 * "something of this type changed".
 */
export type EntityChangedSignal = {
  type: typeof SIGNAL_ENTITY_CHANGED;
  entity: string;
  id?: string;
  year: string;
  /**
   * The originating event name, advisory only.
   *
   * Signals are coalesced per (entity, id), so when several events collapse the
   * surviving name is arbitrary. Never branch on this to decide *what* to
   * refetch — it is for debugging and display.
   */
  event?: string;
};

/**
 * "You have missed something; revalidate everything you hold."
 *
 * Emitted on (re)connect, when a server-side buffer overflows rather than
 * dropping invalidations, and on every tick of the polling transport — which is
 * why the cache needs no polling-specific code path.
 */
export type ResyncSignal = {
  type: typeof SIGNAL_RESYNC;
  /** Why the resync happened. Advisory; useful in logs. */
  reason?: 'connect' | 'reconnect' | 'poll' | 'overflow';
};

export type LiveSignal = EntityChangedSignal | ResyncSignal;

/** Narrowing helpers, so consumers do not compare string literals by hand. */
export const isEntityChanged = (s: LiveSignal): s is EntityChangedSignal =>
  s.type === SIGNAL_ENTITY_CHANGED;

export const isResync = (s: LiveSignal): s is ResyncSignal => s.type === SIGNAL_RESYNC;

/**
 * A dependency a resource can declare: either a whole entity type (`'sos'`) or a
 * single instance (`'sos:123'`).
 *
 * Type-level dependencies are what make new rows appear: a freshly created
 * entity has an id the client has never seen, so an instance-keyed cache alone
 * would never learn about it. They are also the only way derived aggregates —
 * counts, strengths — can be invalidated at all, since they have no id.
 */
export type LiveDependency = string;

/** Does `signal` invalidate something declaring `dependency`? */
export function signalMatches(signal: EntityChangedSignal, dependency: LiveDependency): boolean {
  if (dependency === signal.entity) return true;
  return signal.id !== undefined && dependency === `${signal.entity}:${signal.id}`;
}
