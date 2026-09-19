package recordread

import "testing"

func TestValidOffsetTimestamp(t *testing.T) {
	for _, value := range []string{"2026-09-19T12:00:00Z", "2026-09-19T12:00:00.125+02:00", "2026-09-19T12:00:00-23:59"} {
		if !ValidOffsetTimestamp(value) {
			t.Errorf("valid timestamp rejected: %s", value)
		}
	}
	for _, value := range []string{"2026-09-19T12:00:00+00:60", "2026-09-19T12:00:00-23:60", "2026-09-19T12:00:00+24:00", "2026-09-19T12:00:00", "2026-09-19T1:00:00Z", "2026-09-19T12:00:00,5Z", "2026-02-30T12:00:00Z"} {
		if ValidOffsetTimestamp(value) {
			t.Errorf("invalid timestamp accepted: %s", value)
		}
	}
}
