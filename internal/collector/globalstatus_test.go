package collector

import (
	"testing"
)

func TestGlobalStatus_Signature(t *testing.T) {
	// GlobalStatus requires a real DB connection — tested via integration tests (WO-23).
	// This test verifies the function exists and compiles.
	_ = GlobalStatus
}
