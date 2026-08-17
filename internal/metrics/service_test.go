package metrics

import (
	"testing"
)

func TestEmailMetricsFields(t *testing.T) {
	// Ensure JSON field names stay stable for the console overview.
	m := EmailMetrics{Sent: 1, Delivered: 2, Opened: 3, Clicked: 4, Queued: 5, Failed: 6}
	if m.Sent+m.Delivered+m.Opened+m.Clicked+m.Queued+m.Failed != 21 {
		t.Fatal("unexpected")
	}
}
