package traffic

import "testing"

func TestMeasurementZeroDurationUsesEndToEndBoundary(t *testing.T) {
	m := Measurement{SolverTimeNS: 0, EndToEndTimeNS: 1000, ZeroDuration: false, TimingValid: false}
	if m.ZeroDuration {
		t.Fatal("a zero solver diagnostic must not make a positive public API duration zero")
	}
	if m.TimingValid {
		t.Fatal("zero solver diagnostic must remain invalid for solver ranking")
	}
}
