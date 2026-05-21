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

func TestLookupCommandFlag(t *testing.T) {
	flag, ok := LookupCommandFlag("generate")
	if !ok {
		t.Fatalf("expected generate command flag to be registered")
	}
	if flag != Generate {
		t.Fatalf("expected generate flag, got %#v", flag)
	}

	if _, ok := LookupCommandFlag("missing"); ok {
		t.Fatalf("did not expect missing command flag to resolve")
	}
}

func TestDisabledCommandError(t *testing.T) {
	err := DisabledCommandError(Generate)
	if got, want := err.Error(), "\"generate\" is an experimental command and is currently disabled; enable it with GALAXIO_FEATURE_GENERATE=true"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
