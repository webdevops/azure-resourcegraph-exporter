package kusto

import (
	"fmt"
	"time"
)

var (
	timeFormats = []string{
		// prefered format
		time.RFC3339,

		// human format
		"2006-01-02 15:04:05 +07:00",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",

		// allowed formats
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339Nano,
	}
)

func convertStringToUnixtime(val string) (ret string) {
	for _, timeFormat := range timeFormats {
		if parseVal, parseErr := time.Parse(timeFormat, val); parseErr == nil && parseVal.Unix() > 0 {
			ret = fmt.Sprintf("%v", parseVal.Unix())
			break
		}
	}

	return
}

func toFloat64Ptr(val float64) *float64 {
	return &val
}
