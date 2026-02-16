package recon

import "testing"

func TestInferOSFromTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
		want string
	}{
		{"zero", 0, ""},
		{"cisco_switch_direct", 255, "network_equipment"},
		{"cisco_switch_1hop", 254, "network_equipment"},
		{"cisco_switch_5hops", 250, "network_equipment"},
		{"network_boundary", 225, "network_equipment"},
		{"windows_direct", 128, "windows"},
		{"windows_1hop", 127, "windows"},
		{"windows_5hops", 123, "windows"},
		{"windows_boundary", 110, "windows"},
		{"linux_direct", 64, "linux"},
		{"linux_1hop", 63, "linux"},
		{"linux_5hops", 59, "linux"},
		{"linux_boundary", 35, "linux"},
		{"ambiguous_below_linux", 34, ""},
		{"ambiguous_between_linux_windows", 100, ""},
		{"ambiguous_between_windows_network", 200, ""},
		{"negative", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferOSFromTTL(tt.ttl)
			if got != tt.want {
				t.Errorf("InferOSFromTTL(%d) = %q, want %q", tt.ttl, got, tt.want)
			}
		})
	}
}
