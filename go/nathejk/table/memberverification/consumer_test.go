package memberverification

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
)

// --- fakes ---

type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

type fakeMessage struct {
	subject stream.Subject
	body    any
	at      time.Time
}

func (m fakeMessage) Subject() stream.Subject { return m.subject }
func (m fakeMessage) Time() time.Time         { return m.at }
func (m fakeMessage) Sequence() uint64        { return 1 }
func (m fakeMessage) Body(dst any) error {
	b, err := json.Marshal(m.body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
func (m fakeMessage) Meta(any) error { return nil }
func (m fakeMessage) RawBody() any   { return m.body }
func (m fakeMessage) RawMeta() any   { return nil }

func msg(subj string, body any) stream.Message {
	return fakeMessage{subject: subject.FromStr(subj), body: body, at: time.Date(2026, 9, 26, 1, 20, 0, 0, time.UTC)}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

var verifiedAt = time.Date(2026, 9, 20, 18, 30, 0, 0, time.UTC)

// --- tests ---

func TestVerificationRecordsBothNumbersNormalized(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.verified", messages.NathejkMemberVerified{
		MemberID:     "m-1",
		Phone:        "12 34 56 78",
		PhoneContact: "+45 87-65-43-21",
		VerifiedAt:   verifiedAt,
	}))

	if len(w.stmts) != 1 {
		t.Fatalf("expected one statement, got %d: %v", len(w.stmts), w.stmts)
	}
	stmt := w.stmts[0]
	for _, want := range []string{
		`year="2026"`,
		`memberId="m-1"`,
		`phone="12345678"`,
		`phoneContact="4587654321"`,
	} {
		if !strings.Contains(stmt, want) {
			t.Errorf("statement missing %s: %s", want, stmt)
		}
	}
	// Both numbers were verified at the same moment, so both timestamps carry it.
	if !strings.Contains(stmt, "phoneUts=1789929000") || !strings.Contains(stmt, "phoneContactUts=1789929000") {
		t.Errorf("timestamps are not the event's verifiedAt: %s", stmt)
	}
}

// A login publishes `phone` alone. The contact number a member confirmed earlier must
// survive it — an absent value means "not verified now", never "no longer verified".
func TestALoginOnlyEventNeverErasesAVerifiedContactNumber(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.verified", messages.NathejkMemberVerified{
		MemberID:   "m-1",
		Phone:      "12345678",
		VerifiedAt: verifiedAt,
	}))

	stmt := w.stmts[0]
	if !strings.Contains(stmt, `phoneContact=""`) {
		t.Fatalf("expected the insert to carry an empty contact number: %s", stmt)
	}
	// The guard that protects the stored value: an empty incoming value keeps the column.
	if !strings.Contains(stmt, "phoneContact=IF(VALUES(phoneContact) <> ''") {
		t.Errorf("the upsert would clobber a verified contact number with '': %s", stmt)
	}
	// And its timestamp is not the event's, since this event verified no contact number.
	if !strings.Contains(stmt, "phoneContactUts=0") {
		t.Errorf("an unverified contact number must carry no timestamp: %s", stmt)
	}
}

// Replay order must not matter: an older verification delivered after a newer one must
// not move the number back.
func TestUpsertOnlyAcceptsANumberThatIsNotOlderThanTheStoredOne(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.verified", messages.NathejkMemberVerified{
		MemberID: "m-1", Phone: "12345678", VerifiedAt: verifiedAt,
	}))

	stmt := w.stmts[0]
	if !strings.Contains(stmt, "VALUES(phoneUts) >= phoneUts") {
		t.Errorf("the upsert is order-dependent: %s", stmt)
	}
	if !strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("not an upsert, so a replay would fail on the primary key: %s", stmt)
	}
}

// Nothing verified is nothing to record: the row would say "no numbers proven", which is
// what its absence already says.
func TestAVerificationCarryingNoNumberWritesNothing(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.verified", messages.NathejkMemberVerified{
		MemberID: "m-1", VerifiedAt: verifiedAt,
	}))

	if len(w.stmts) != 0 {
		t.Errorf("wrote %v for a verification proving nothing", w.stmts)
	}
}

// The body names the member, but a body that does not must still land: the subject does.
func TestTheMemberIsTakenFromTheSubjectWhenTheBodyOmitsIt(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-42.verified", messages.NathejkMemberVerified{
		Phone: "12345678", VerifiedAt: verifiedAt,
	}))

	if len(w.stmts) != 1 || !strings.Contains(w.stmts[0], `memberId="m-42"`) {
		t.Errorf("member not taken from the subject: %v", w.stmts)
	}
}

// The declared subject must be one the handler recognises. The ":" form normalises to the
// dotted one, and matching is case-insensitive, so this is a real check of the pair.
func TestConsumedSubjectIsHandled(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	subjects := c.Consumes()
	if len(subjects) != 1 {
		t.Fatalf("expected one consumed subject, got %d", len(subjects))
	}
	declared := subjects[0].Subject()
	if got, want := declared, "NATHEJK.*.spejder.*.verified"; got != want {
		t.Fatalf("consumed subject = %q, want %q", got, want)
	}
	concrete := strings.Replace(strings.Replace(declared, "*.spejder", "2026.spejder", 1), "spejder.*", "spejder.m-1", 1)
	handle(t, c, msg(concrete, messages.NathejkMemberVerified{MemberID: "m-1", Phone: "12345678", VerifiedAt: verifiedAt}))
	if len(w.stmts) != 1 {
		t.Errorf("the declared subject is not handled: %v", w.stmts)
	}
}
