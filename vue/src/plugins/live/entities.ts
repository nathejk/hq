/**
 * Dev-only check: does a declared dependency name an entity the stream can emit?
 *
 * `dependsOn` is a list of bare strings, and a wrong one fails in the worst
 * possible way — the page looks live, never errors, and simply never updates.
 * Task 037 got two of six tokens wrong (`scan` for what is really `qr`,
 * `personnel` for what is really `gøgler`/`friend`/`bandit`) and both were caught
 * only by reading Go source. The API now advertises the tokens its wired consumers
 * can produce, so the mistake can be reported instead.
 *
 * Three rules this deliberately follows:
 *
 *  - **Warn, never throw.** A stale or wrong allow-list must not be able to break a
 *    page. The check is an aid, not a gate.
 *  - **Dev only.** In production it does nothing at all, so it cannot cost anything
 *    or confuse an operator.
 *  - **Never claim more than the server does.** The set is not always exhaustive:
 *    some consumers subscribe with a wildcard in the entity position, so a token
 *    outside the set may still be legitimate. The wording says so.
 *
 * See roadmap/tasks/doing/040-validate-dependson-tokens.md.
 */
import type { LiveDependency } from './signals';

/** What the server advertises on connect, as an `entities` frame. */
export type EntitySet = {
  entities: string[];
  /**
   * False when some consumer subscribes with a wildcard entity, in which case a
   * token outside `entities` may still arrive and a warning may be a false
   * positive.
   */
  exhaustive: boolean;
};

let known: EntitySet | undefined;

/** Everything validated so far, so a late-arriving set can re-check it. */
const registered = new Map<string, LiveDependency[]>();

/** Tokens already warned about, so a noisy page warns once per token, not per render. */
const warned = new Set<string>();

/**
 * Dev only — see the module comment. Each body is wrapped in a plain
 * `if (import.meta.env.DEV)` block rather than an early return or a shared helper,
 * which looks like duplication and is deliberate: Vite substitutes the literal
 * `false` in a production build, so the block becomes `if (false) {…}`, `check`
 * becomes unreferenced, and the checking code and its message strings are dropped
 * from the bundle rather than merely never running. Written as an early return or
 * routed through a constant, the dead code survives minification — verified by
 * grepping `dist` for the warning text each way.
 *
 * Beware when checking that yourself: `docker compose run ui` sets
 * `NODE_ENV=development`, and Vite derives its production mode from `NODE_ENV`
 * *before* the build mode, so `npm run build-only` there emits a **dev-mode** bundle
 * in which this code is (correctly) retained. The real image is unaffected —
 * `docker/Dockerfile`'s `ui-builder` stage sets no `NODE_ENV`, and `node:20-alpine`
 * leaves it unset — so run the build with `NODE_ENV=production` to reproduce what
 * ships.
 */

/**
 * Record the set advertised by the server, and re-check everything already
 * declared.
 *
 * Called on every (re)connect, which is deliberate: a client that reconnects to a
 * newly deployed build must validate against that build's set, not the previous
 * one.
 */
export function setKnownEntities(set: EntitySet | undefined): void {
  known = set;
  if (import.meta.env.DEV) {
    if (!set) return;
    // Re-check from scratch: a set that changed may exonerate a token warned about
    // earlier, or condemn one that passed.
    warned.clear();
    for (const [key, deps] of registered) check(key, deps, set);
  }
}

/** The set currently held, for diagnostics and tests. */
export function knownEntities(): EntitySet | undefined {
  return known;
}

/**
 * Validate a resource's declared dependencies, now or when the set arrives.
 *
 * Registration happens even in production, but costs one Map entry and no work:
 * keeping the call site unconditional is what stops the check from silently
 * applying to only some resources.
 */
export function validateDependencies(key: string, dependsOn: LiveDependency[]): void {
  registered.set(key, [...dependsOn]);
  if (import.meta.env.DEV) {
    if (known) check(key, dependsOn, known);
  }
}

/** Discard state. For tests. */
export function resetKnownEntities(): void {
  known = undefined;
  registered.clear();
  warned.clear();
}

/**
 * The entity part of a dependency.
 *
 * Dependencies are either a type (`'patrulje'`) or an instance
 * (`'patrulje:abc-123'`), and only the type part is a token. Split on the *first*
 * colon: ids are opaque and could contain one.
 */
function entityOf(dependency: LiveDependency): string {
  const colon = dependency.indexOf(':');
  return colon === -1 ? dependency : dependency.slice(0, colon);
}

function check(key: string, dependsOn: LiveDependency[], set: EntitySet): void {
  for (const dep of dependsOn) {
    const entity = entityOf(dep);
    if (!entity || set.entities.includes(entity)) continue;
    if (warned.has(entity)) continue;
    warned.add(entity);

    const caveat = set.exhaustive
      ? 'No wired consumer can emit signals for it, so this resource will never update from it.'
      : 'No wired consumer names it. Some consumers subscribe to every entity, so this may be a ' +
        'false positive — but check it against the projection that owns the event.';

    // console.warn rather than an error: this is advisory, and an uncaught error
    // here would be worse than the bug it reports.
    console.warn(
      `[live] "${key}" depends on unknown entity "${entity}". ${caveat}\n` +
        `Known entities: ${set.entities.join(', ')}\n` +
        `Note the token is the event *subject's* entity, not the projection's name ` +
        `(scans are "qr", not "scan"; personnel are "gøgler"/"friend"/"bandit").`,
    );
  }
}
