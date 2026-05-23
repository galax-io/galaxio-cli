package featureflags

import "testing"

func TestEnabled(t *testing.T) {
	flag := Flag{Command: "example", EnvVar: "GALAXIO_FEATURE_EXAMPLE"}

	t.Setenv(flag.EnvVar, "true")
	if !Enabled(flag) {
		t.Fatalf("expected flag to be enabled")
	}

	t.Setenv(flag.EnvVar, "false")
	if Enabled(flag) {
		t.Fatalf("expected flag to be disabled")
	}

	t.Setenv(flag.EnvVar, "invalid")
	if Enabled(flag) {
		t.Fatalf("expected invalid bool value to be treated as disabled")
	}
}
