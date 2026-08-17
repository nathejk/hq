import { describe, it, expect } from 'vitest';
import {
  activityLabel,
  activityIcon,
  isMemberSummaryType,
  parseMemberSummary,
  memberStatusPhrase,
} from './sos';

// The member lifecycle summaries (PRD 006) are the only timeline entries whose value is
// structured rather than a bare string, so they are the only ones with a parse step that
// can fail on data from the wire.

describe('parseMemberSummary', () => {
  it('reads a status-change summary', () => {
    const summary = parseMemberSummary(
      JSON.stringify({
        sosId: 'sos-1',
        teamId: 't-1',
        teamName: 'Ulvene',
        members: [{ memberId: 'm-1', name: 'Ida', from: 'racing', to: 'waiting' }],
        teamStrength: 2,
      }),
    );
    expect(summary?.teamName).toBe('Ulvene');
    expect(summary?.members).toHaveLength(1);
    expect(summary?.members?.[0].from).toBe('racing');
    expect(summary?.teamStrength).toBe(2);
  });

  it('reads several members from one operation', () => {
    // The property the whole N-events-plus-one-summary design exists for: collecting a
    // patrol of three is one entry naming three people, not three entries.
    const summary = parseMemberSummary(
      JSON.stringify({
        members: [
          { memberId: 'm-1', name: 'Ida', from: 'racing', to: 'waiting' },
          { memberId: 'm-2', name: 'Bo', from: 'racing', to: 'waiting' },
          { memberId: 'm-3', name: 'Sol', from: 'racing', to: 'waiting' },
        ],
        teamStrength: 0,
      }),
    );
    expect(summary?.members).toHaveLength(3);
    expect(summary?.teamStrength).toBe(0);
  });

  it('keeps per-member destinations for a move', () => {
    // Two survivors may go to two different patrols, so the destination is per member.
    const summary = parseMemberSummary(
      JSON.stringify({
        fromTeamId: 't-1',
        fromTeamName: 'Ulvene',
        members: [
          { memberId: 'm-1', name: 'Ida', toTeamId: 't-2', toTeamName: 'Bjørnene' },
          { memberId: 'm-2', name: 'Bo', toTeamId: 't-3', toTeamName: 'Ravnene' },
        ],
        fromTeamStrength: 0,
      }),
    );
    expect(summary?.members?.map((m) => m.toTeamName)).toEqual(['Bjørnene', 'Ravnene']);
  });

  it('returns null rather than throwing on unparseable data', () => {
    // A malformed entry must degrade to the raw value, not blank the line: an entry an
    // operator cannot fully read still belongs on a handover record.
    expect(parseMemberSummary('not json at all')).toBeNull();
    expect(parseMemberSummary('')).toBeNull();
    expect(parseMemberSummary(undefined)).toBeNull();
  });

  it('returns null for JSON that is not an object', () => {
    expect(parseMemberSummary('42')).toBeNull();
    expect(parseMemberSummary('null')).toBeNull();
  });
});

describe('isMemberSummaryType', () => {
  it('covers exactly the three summarising types', () => {
    expect(isMemberSummaryType('member.status.changed')).toBe(true);
    expect(isMemberSummaryType('member.moved')).toBe(true);
    expect(isMemberSummaryType('team.collected')).toBe(true);
  });

  it('leaves the PRD 001 types alone', () => {
    // These carry bare strings, so treating one as JSON would blank a working line.
    for (const type of ['created', 'commented', 'severity.specified', 'team.associated']) {
      expect(isMemberSummaryType(type)).toBe(false);
    }
  });
});

describe('labels and icons', () => {
  it('labels the new entry types in Danish', () => {
    expect(activityLabel('member.status.changed')).toBe('Deltagerstatus ændret');
    expect(activityLabel('member.moved')).toBe('Deltagere flyttet');
    expect(activityLabel('team.collected')).toBe('Hele patruljen hentes');
  });

  it('falls back to the raw type for an unknown entry', () => {
    // PRD 001 requires the timeline to tolerate types it does not know, because the car
    // and shelter interfaces will add more.
    expect(activityLabel('member.teleported')).toBe('member.teleported');
    expect(activityIcon('member.teleported')).toBe('pi pi-circle');
  });

  it('gives each new type its own icon', () => {
    const icons = ['member.status.changed', 'member.moved', 'team.collected'].map(activityIcon);
    expect(new Set(icons).size).toBe(3);
    expect(icons).not.toContain('pi pi-circle');
  });
});

describe('memberStatusPhrase', () => {
  it('renders mid-sentence phrases, not tag labels', () => {
    expect(memberStatusPhrase('racing')).toBe('i løbet');
    expect(memberStatusPhrase('waiting')).toBe('venter på at blive hentet');
  });

  it('names the empty status rather than rendering a gap', () => {
    expect(memberStatusPhrase('')).toBe('ikke startet');
  });

  it('falls back to the raw slug for something it does not know', () => {
    expect(memberStatusPhrase('hibernating')).toBe('hibernating');
  });
});
