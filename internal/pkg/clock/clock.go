// Package clock abstracts time so loops can be tested/deterministic, and so
// "now" is resolved from a single source.
package clock

import "time"

// Clock returns the current time. The default implementation is the system
// clock; tests may supply a fake.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// System returns the real system clock.
func System() Clock { return systemClock{} }
