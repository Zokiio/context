package recordread

import (
	"regexp"
	"time"
)

var offsetTimestamp = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](?:\.[0-9]+)?(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])$`)

// ValidOffsetTimestamp checks RFC 3339 syntax and an explicit UTC offset.
// Go's parser alone also accepts some out-of-range offset components.
func ValidOffsetTimestamp(value string) bool {
	if !offsetTimestamp.MatchString(value) {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
