// Package utils contains small helpers that do not depend on internal
// application packages.
package utils

import "time"

const jakartaTimezone = "Asia/Jakarta"

var jakartaFallback = time.FixedZone("WIB", 7*60*60)

// Jakarta returns the business timezone. The fixed UTC+7 fallback keeps the
// service usable on a minimal container that does not carry tzdata.
func Jakarta() *time.Location {
	location, err := time.LoadLocation(jakartaTimezone)
	if err != nil {
		return jakartaFallback
	}

	return location
}

func NowJakarta() time.Time {
	return time.Now().In(Jakarta())
}
