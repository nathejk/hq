package types

import "time"

type UnixtimeString string

func (uts UnixtimeString) Time() *time.Time {
	tm, err := time.Parse("1136239445", string(uts))
	if err != nil || tm.IsZero() {
		return nil
	}
	return &tm
}
