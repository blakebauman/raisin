package ippool

import "testing"

func TestNextWarmupDay(t *testing.T) {
	plan := []int{50, 100, 200, 400}
	day, cap, active := NextWarmupDay(0, plan)
	if day != 1 || cap != 100 || active {
		t.Fatalf("got day=%d cap=%d active=%v", day, cap, active)
	}
	day, cap, active = NextWarmupDay(3, plan)
	if day != 4 || cap != 400 || !active {
		t.Fatalf("end of plan: day=%d cap=%d active=%v", day, cap, active)
	}
	day, cap, active = NextWarmupDay(0, nil)
	if day != 1 || cap != 50 || !active {
		t.Fatalf("empty plan: day=%d cap=%d active=%v", day, cap, active)
	}
}
