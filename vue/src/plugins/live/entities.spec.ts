import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  setKnownEntities,
  knownEntities,
  validateDependencies,
  resetKnownEntities,
} from './entities';

// The check is dev-only, and vitest sets DEV=true, so it is active here. That is
// the interesting configuration: production behaviour is "do nothing".
describe('dependency validation', () => {
  let warn: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    resetKnownEntities();
    warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    warn.mockRestore();
    resetKnownEntities();
  });

  const set = (entities: string[], exhaustive = true) => ({ entities, exhaustive });

  it('says nothing about a token the server advertises', () => {
    setKnownEntities(set(['patrulje', 'qr']));
    validateDependencies('patrulje:list', ['patrulje']);
    expect(warn).not.toHaveBeenCalled();
  });

  it('warns about a token nothing can emit', () => {
    setKnownEntities(set(['qr']));
    validateDependencies('post:list', ['scan']);

    expect(warn).toHaveBeenCalledTimes(1);
    const message = String(warn.mock.calls[0][0]);
    expect(message).toContain('post:list');
    expect(message).toContain('"scan"');
  });

  // The two mistakes this exists to catch, named explicitly so a refactor that
  // breaks the check fails here rather than in production silence.
  it('catches the real historical mistakes: scan and personnel', () => {
    setKnownEntities(set(['qr', 'gøgler', 'friend', 'bandit']));
    validateDependencies('a', ['scan']);
    validateDependencies('b', ['personnel']);
    expect(warn).toHaveBeenCalledTimes(2);
  });

  it('accepts the non-ASCII token as-is', () => {
    setKnownEntities(set(['gøgler']));
    validateDependencies('badut:list', ['gøgler']);
    expect(warn).not.toHaveBeenCalled();
  });

  it('validates the type part of an instance dependency', () => {
    setKnownEntities(set(['patrulje']));
    validateDependencies('detail', ['patrulje:abc-123']);
    expect(warn).not.toHaveBeenCalled();

    validateDependencies('detail2', ['patrouille:abc-123']);
    expect(warn).toHaveBeenCalledTimes(1);
    expect(String(warn.mock.calls[0][0])).toContain('"patrouille"');
  });

  it('splits an instance dependency on the first colon, since ids are opaque', () => {
    setKnownEntities(set(['patrulje']));
    validateDependencies('detail', ['patrulje:a:b:c']);
    expect(warn).not.toHaveBeenCalled();
  });

  it('warns once per token however many resources declare it', () => {
    setKnownEntities(set(['qr']));
    validateDependencies('one', ['scan']);
    validateDependencies('two', ['scan']);
    validateDependencies('three', ['scan']);
    expect(warn).toHaveBeenCalledTimes(1);
  });

  // Registration must not depend on connection order: pages mount before the
  // stream connects, and that is the normal case rather than an edge case.
  it('checks dependencies declared before the set arrived', () => {
    validateDependencies('early', ['scan']);
    expect(warn).not.toHaveBeenCalled();

    setKnownEntities(set(['qr']));
    expect(warn).toHaveBeenCalledTimes(1);
    expect(String(warn.mock.calls[0][0])).toContain('early');
  });

  // A reconnect to a newly deployed build must re-evaluate: the set is the
  // server's, and the server may have changed.
  it('re-checks everything when a new set arrives, exonerating a fixed token', () => {
    validateDependencies('r', ['sos']);
    setKnownEntities(set(['patrulje']));
    expect(warn).toHaveBeenCalledTimes(1);

    warn.mockClear();
    setKnownEntities(set(['patrulje', 'sos']));
    expect(warn).not.toHaveBeenCalled();
  });

  it('softens the wording when the set is not exhaustive', () => {
    setKnownEntities(set(['klan'], false));
    validateDependencies('x', ['mystery']);
    expect(String(warn.mock.calls[0][0])).toContain('false positive');

    warn.mockClear();
    resetKnownEntities();
    setKnownEntities(set(['klan'], true));
    validateDependencies('y', ['mystery']);
    expect(String(warn.mock.calls[0][0])).toContain('never update');
  });

  it('does nothing until a set is known', () => {
    validateDependencies('x', ['whatever']);
    expect(warn).not.toHaveBeenCalled();
    expect(knownEntities()).toBeUndefined();
  });

  it('ignores an empty dependency list', () => {
    setKnownEntities(set(['klan']));
    validateDependencies('x', []);
    expect(warn).not.toHaveBeenCalled();
  });

  it('exposes the set it holds', () => {
    setKnownEntities(set(['klan', 'qr'], false));
    expect(knownEntities()).toEqual({ entities: ['klan', 'qr'], exhaustive: false });
  });
});
