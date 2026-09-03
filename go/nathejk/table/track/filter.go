package track

// Filter selects points for one person, optionally bounded in time.
//
// Bounds are epoch milliseconds, matching the stored `ts` exactly. No `time.Time`, and no parsing
// of a formatted date: the endpoint takes the same integers the producer sends, so there is one
// representation of an instant from the phone to the map and no timezone can be misread on the way.
//
// Zero means "unbounded", which is why they are plain int64 rather than pointers — epoch 0 is 1970
// and no plausible point predates the producer's own 2020 floor, so the zero value is unambiguous
// here in a way it usually is not.
type Filter struct {
	PersonID string
	FromTs   int64
	ToTs     int64
}
