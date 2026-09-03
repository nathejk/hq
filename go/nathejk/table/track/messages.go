package track

// Reported is the body of `TELEMETRY.{year}.track.{personId}.reported`.
//
// A local copy of the producer's struct rather than an import: hej-app's `track` package is early
// work and has not been lifted to shared-go yet. When it is, this type should go and the import
// take its place — the field names and JSON tags here are deliberately identical so that swap is a
// deletion rather than a translation.
//
// The person and year are in the body even though the subject already carries them. That is the
// producer's decision and a good one: a consumer should not have to parse a routing address to
// read a message. This package reads them from the body for exactly that reason.
type Reported struct {
	PersonID string `json:"personId"`

	// UserType is the app role the person held when the batch was reported — "gøgler", "spejder",
	// "crewmember", … — stamped at publish time rather than looked up at read time.
	//
	// Stored as sent (see table.sql). The producer's reasoning applies with full force to a
	// consumer: this stream is kept indefinitely while roles change, so resolving the role at read
	// time against today's directory would silently reinterpret last year's history.
	UserType string `json:"userType"`

	Year string `json:"year"`

	// Points is a batch. One message carries many positions — the client accumulates offline and
	// flushes — so a handler must loop, and a single message can insert hundreds of history rows
	// while advancing the latest-position row at most once.
	Points []Point `json:"points"`
}
