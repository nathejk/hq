package searchperson

import (
	"strings"
	"time"

	"github.com/nathejk/shared-go/types"
)

// quote renders s as a single-quoted SQL string literal, escaping what MySQL and
// MariaDB treat specially.
//
// Deliberately not fmt.Sprintf("%q", s), which is what shared-go's older entities use.
// %q is *Go* quoting: double quotes and Go escape sequences, which MySQL happens to
// accept in its default mode but which are not the same language. A backslash-terminated
// value, or ANSI_QUOTES mode where "…" is an identifier rather than a string, turns the
// statement into either a syntax error or a different statement than intended. Names and
// free-text phone numbers are user input, so this matters here more than most places.
func quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('\'')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case 0:
			// A NUL would truncate the statement at the driver or the server.
			b.WriteString(`\0`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case 26:
			b.WriteString(`\Z`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// normalizePhone reduces a number to the digits an operator would type, in national form.
//
// Two steps, and the second one is load-bearing:
//
//  1. types.PhoneNumber.Normalize() — the same function the SMS gateway uses, so we agree
//     with it about what a digit is. Reimplementing its regexp here would let the two
//     drift, and the failure would be a number that receives messages but cannot be
//     looked up.
//
//  2. Reduce to national form. Step 1 alone does **not** unify the formats actually in the
//     source tables: it keeps every digit, so `+45 12 34 56 78` comes out as
//     `4512345678` while `12 34 56 78` comes out as `12345678`, and the two would never
//     match each other. (shared-go is not much help here — its own IsValid() calls an
//     8-digit number valid and therefore calls the `+45` form invalid, and
//     InternationalNumber() concatenates an empty country code onto it.)
//
// The country code is stripped only from a 10-digit result, never from 8 digits. `45` is a
// real Danish prefix, so `45123456` is somebody's actual number and must survive intact —
// getting this backwards would silently make every 45-prefixed subscriber unfindable.
func normalizePhone(s string) string {
	digits := types.PhoneNumber(s).Normalize()

	// International access prefix, as dialled: 0045 12345678.
	digits = strings.TrimPrefix(digits, "00")

	if len(digits) == 10 && strings.HasPrefix(digits, "45") {
		digits = digits[2:]
	}
	return digits
}

// datetime renders t as a SQL literal, or NULL for the zero time.
//
// NULL rather than a sentinel, because MariaDB in strict mode rejects 0000-00-00
// outright, and because "we do not know when this row was last touched" is a real state
// for a row rebuilt from an event with no timestamp.
func datetime(t time.Time) string {
	if t.IsZero() {
		return "NULL"
	}
	return quote(t.UTC().Format(time.DateTime))
}

// truncate clips s to n bytes.
//
// Not cosmetic: MariaDB in strict mode *errors* on a value too long for its column, and
// an error here dead-letters the event, so one absurd name in the register would cost us
// that person's searchability entirely. Clipping keeps them findable by their first n
// bytes, which is what a search index is for.
//
// Bytes rather than runes, matching the column's declaration. A clip that lands mid-rune
// can leave a broken UTF-8 tail — acceptable for an index whose purpose is substring
// matching, and cheaper than carrying a rune-aware limit that still has to agree with the
// column width.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
