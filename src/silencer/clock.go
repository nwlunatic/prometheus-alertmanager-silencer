package silencer

import "time"

type clock interface {
	Now() time.Time
}

type Clock struct{}

func (c Clock) Now() time.Time {
	return time.Now()
}

type ClockMock struct {
	T time.Time
}

func (c ClockMock) Now() time.Time {
	return c.T
}
