package svcmap

import "testing"

func TestComputeGrade(t *testing.T) {
	tests := []struct {
		name string
		cpu  float64
		mem  float64
		disk float64
		want string
	}{
		// Grade A: peak < 30%
		{name: "all zeros", cpu: 0, mem: 0, disk: 0, want: "A"},
		{name: "low usage", cpu: 10, mem: 15, disk: 20, want: "A"},
		{name: "just under A boundary", cpu: 29, mem: 29, disk: 29, want: "A"},

		// Grade B: 30% <= peak < 60%
		{name: "at B boundary", cpu: 30, mem: 0, disk: 0, want: "B"},
		{name: "mid B range", cpu: 45, mem: 30, disk: 20, want: "B"},
		{name: "just under C boundary", cpu: 59, mem: 59, disk: 59, want: "B"},

		// Grade C: 60% <= peak < 80%
		{name: "at C boundary", cpu: 60, mem: 0, disk: 0, want: "C"},
		{name: "mid C range", cpu: 70, mem: 65, disk: 50, want: "C"},
		{name: "just under D boundary", cpu: 79, mem: 79, disk: 79, want: "C"},

		// Grade D: 80% <= peak < 90%
		{name: "at D boundary", cpu: 80, mem: 0, disk: 0, want: "D"},
		{name: "mid D range", cpu: 85, mem: 70, disk: 60, want: "D"},
		{name: "just under F boundary", cpu: 89, mem: 89, disk: 89, want: "D"},

		// Grade F: peak >= 90%
		{name: "at F boundary", cpu: 90, mem: 0, disk: 0, want: "F"},
		{name: "over F boundary", cpu: 91, mem: 50, disk: 50, want: "F"},
		{name: "full utilization", cpu: 100, mem: 100, disk: 100, want: "F"},

		// Peak selection: highest metric determines grade
		{name: "cpu is peak", cpu: 85, mem: 20, disk: 30, want: "D"},
		{name: "mem is peak", cpu: 20, mem: 85, disk: 30, want: "D"},
		{name: "disk is peak", cpu: 20, mem: 30, disk: 85, want: "D"},

		// Edge cases
		{name: "one metric at boundary rest zero", cpu: 0, mem: 0, disk: 30, want: "B"},
		{name: "fractional values", cpu: 29.9, mem: 29.9, disk: 29.9, want: "A"},
		{name: "just over boundary fractional", cpu: 30.1, mem: 0, disk: 0, want: "B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeGrade(tt.cpu, tt.mem, tt.disk)
			if got != tt.want {
				t.Errorf("ComputeGrade(%v, %v, %v) = %q, want %q",
					tt.cpu, tt.mem, tt.disk, got, tt.want)
			}
		})
	}
}

func TestComputeFleetSummary_Empty(t *testing.T) {
	fleet := ComputeFleetSummary(nil)
	if fleet.TotalDevices != 0 {
		t.Errorf("expected 0 devices, got %d", fleet.TotalDevices)
	}
	if fleet.TotalServices != 0 {
		t.Errorf("expected 0 services, got %d", fleet.TotalServices)
	}
	if len(fleet.ByGrade) != 0 {
		t.Errorf("expected empty grade map, got %v", fleet.ByGrade)
	}
}
