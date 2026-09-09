package main

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/track"
)

// The merge rule behind the patrol position layer.
//
// It is tested as a pure function rather than through the handler because everything that can go
// wrong here is arithmetic and parsing, not HTTP: two time units meeting on one axis, and a VARCHAR
// column that does not always hold a coordinate. Both failures produce a plausible-looking marker in
// the wrong place or the wrong decade, which is the kind of thing nobody notices in a response body.

func trackAt(ts int64, lat, lng float64) *trackFix {
	return &trackFix{Lat: lat, Lng: lng, Ts: ts}
}

func scanAt(uts int64, lat, lng string) *scanFix {
	return &scanFix{Lat: lat, Lng: lng, Uts: uts}
}

func TestMergePosition(t *testing.T) {
	tests := []struct {
		name  string
		track *trackFix
		scan  *scanFix
		want  positionFix
		wantK bool
	}{
		{
			// 1_700_000_500_000 ms is 500 s after the scan's 1_700_000_000 s… only once the scan is
			// multiplied. Compared raw, 1_700_000_000 < 1_700_000_500_000 by a factor of a thousand
			// and telemetry would win every such case by accident rather than by being newer.
			name:  "track newer than scan",
			track: trackAt(1_700_000_500_000, 55.1, 12.1),
			scan:  scanAt(1_700_000_000, "55.9", "12.9"),
			want:  positionFix{Lat: 55.1, Lng: 12.1, Ts: 1_700_000_500_000, Source: "track"},
			wantK: true,
		},
		{
			name:  "scan newer than track",
			track: trackAt(1_700_000_000_000, 55.1, 12.1),
			scan:  scanAt(1_700_000_600, "55.9", "12.9"),
			want:  positionFix{Lat: 55.9, Lng: 12.9, Ts: 1_700_000_600_000, Source: "scan"},
			wantK: true,
		},
		{
			// The failure this guards is subtler than "scan wins": a scan whose seconds were never
			// multiplied still *is* a number, so the endpoint answers 200 with a marker dated 1970.
			name:  "scan seconds are converted to milliseconds",
			track: nil,
			scan:  scanAt(1_757_155_555, "55.5", "12.5"),
			want:  positionFix{Lat: 55.5, Lng: 12.5, Ts: 1_757_155_555_000, Source: "scan"},
			wantK: true,
		},
		{
			name:  "only track present",
			track: trackAt(1_700_000_000_000, 55.2, 12.2),
			want:  positionFix{Lat: 55.2, Lng: 12.2, Ts: 1_700_000_000_000, Source: "track"},
			wantK: true,
		},
		{
			name:  "only scan present",
			scan:  scanAt(1_700_000_000, "55.3", "12.3"),
			want:  positionFix{Lat: 55.3, Lng: 12.3, Ts: 1_700_000_000_000, Source: "scan"},
			wantK: true,
		},
		{
			// Omission, not a marker at 0,0 and not a row of nulls: the map layer must be able to
			// draw the response as it stands.
			name:  "neither source present is omitted",
			wantK: false,
		},
		{
			name: "empty-string scan coordinates are skipped",
			scan: scanAt(1_700_000_000, "", ""),
		},
		{
			name: "zero scan coordinates are skipped",
			scan: scanAt(1_700_000_000, "0", "0"),
		},
		{
			// Half a coordinate is not a coordinate. Accepting it would plot the team on the prime
			// meridian, which looks like a patrol that has walked to Greenwich.
			name: "half-blank scan coordinates are skipped",
			scan: scanAt(1_700_000_000, "55.4", ""),
		},
		{
			name: "non-numeric scan coordinates are skipped",
			scan: scanAt(1_700_000_000, "n/a", "n/a"),
		},
		{
			// An unusable newest scan must not shadow telemetry: the team is still on the map.
			name:  "unusable scan falls back to track",
			track: trackAt(1_700_000_000_000, 55.6, 12.6),
			scan:  scanAt(1_800_000_000, "0", "0"),
			want:  positionFix{Lat: 55.6, Lng: 12.6, Ts: 1_700_000_000_000, Source: "track"},
			wantK: true,
		},
		{
			// Equal evidence goes to the witnessed one.
			name:  "exact tie prefers the scan",
			track: trackAt(1_700_000_000_000, 55.1, 12.1),
			scan:  scanAt(1_700_000_000, "55.9", "12.9"),
			want:  positionFix{Lat: 55.9, Lng: 12.9, Ts: 1_700_000_000_000, Source: "scan"},
			wantK: true,
		},
		{
			name:  "negative-longitude scan is a real position",
			scan:  scanAt(1_700_000_000, "55.7", "-3.5"),
			want:  positionFix{Lat: 55.7, Lng: -3.5, Ts: 1_700_000_000_000, Source: "scan"},
			wantK: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := mergePosition(tc.track, tc.scan)
			if ok != tc.wantK {
				t.Fatalf("ok = %v, want %v (got %+v)", ok, tc.wantK, got)
			}
			if !tc.wantK {
				return
			}
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// The folds carry one rule each, and both are load-bearing: the newest row wins, and a scan without
// a fix never becomes the winner.
func TestFoldFixesKeepsNewestAndSkipsUnusableScans(t *testing.T) {
	tracks := foldTrackFixes([]track.TeamPoint{
		{TeamID: "team-1", Lat: 55.1, Lng: 12.1, Ts: 1_000},
		{TeamID: "team-1", Lat: 55.2, Lng: 12.2, Ts: 2_000},
		{TeamID: "team-2", Lat: 55.3, Lng: 12.3, Ts: 1_500},
	})
	if got := tracks["team-1"]; got == nil || got.Ts != 2_000 || got.Lat != 55.2 {
		t.Fatalf("team-1 track = %+v, want the newest member's point", got)
	}
	if got := tracks["team-2"]; got == nil || got.Ts != 1_500 {
		t.Fatalf("team-2 track = %+v", got)
	}

	scans := foldScanFixes([]scan.TeamPosition{
		{TeamID: "team-1", Uts: 1_000, Latitude: "55.1", Longitude: "12.1"},
		// Newer, but no fix — it must not shadow the row above, or the marker disappears rather
		// than going stale.
		{TeamID: "team-1", Uts: 2_000, Latitude: "0", Longitude: "0"},
		{TeamID: "team-3", Uts: 3_000, Latitude: "", Longitude: ""},
	})
	if got := scans["team-1"]; got == nil || got.Uts != 1_000 {
		t.Fatalf("team-1 scan = %+v, want the newest scan that has a position", got)
	}
	if _, ok := scans["team-3"]; ok {
		t.Fatalf("team-3 has no usable scan and must not appear")
	}
}

// The positions route is asserted in the composition root because its failure mode is startup, not a
// request: httprouter panics when a static segment sits beside a wildcard at the same level, so
// `/api/patrulje/positions` next to `/api/patrulje/:id` would stop the whole API booting. Under
// `/api/telemetry` every sibling at that depth is static, which is why this path was chosen — and
// "is expected to be safe" is not something to take on trust.
func TestTelemetryPositionsRouteDoesNotCollide(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registering the telemetry routes panicked: %v", r)
		}
	}()

	router := httprouter.New()
	h := func(w http.ResponseWriter, r *http.Request) {}

	router.HandlerFunc(http.MethodGet, "/api/telemetry/presence", h)
	router.HandlerFunc(http.MethodGet, "/api/telemetry/person/:personId/track", h)
	router.HandlerFunc(http.MethodGet, "/api/telemetry/patrulje/:teamId/track", h)
	router.HandlerFunc(http.MethodGet, "/api/telemetry/positions", h)

	handler, params, _ := router.Lookup(http.MethodGet, "/api/telemetry/positions")
	if handler == nil {
		t.Fatalf("/api/telemetry/positions does not route")
	}
	if len(params) != 0 {
		t.Fatalf("/api/telemetry/positions matched as a wildcard: %v", params)
	}
}
