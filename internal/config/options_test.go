package config

import (
	"testing"
)

func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions(nil)

	if opts.Persona != PersonaBalanced {
		t.Errorf("expected persona %q, got %q", PersonaBalanced, opts.Persona)
	}

	if opts.OutputDir != "./outputs" {
		t.Errorf("expected output dir %q, got %q", "./outputs", opts.OutputDir)
	}

	if opts.ResumeMode != ResumeModeNormal {
		t.Errorf("expected resume mode %q, got %q", ResumeModeNormal, opts.ResumeMode)
	}

	if opts.Verbosity != VerbosityNormal {
		t.Errorf("expected verbosity %q, got %q", VerbosityNormal, opts.Verbosity)
	}
}

func TestDefaultRunOptionsWithConfig(t *testing.T) {
	cfg := Default()
	cfg.Persona = PersonaProduction
	cfg.OutputDir = "/custom/output"
	cfg.Timeout = 60

	opts := DefaultRunOptions(cfg)

	if opts.Persona != PersonaProduction {
		t.Errorf("expected persona %q, got %q", PersonaProduction, opts.Persona)
	}

	if opts.OutputDir != "/custom/output" {
		t.Errorf("expected output dir %q, got %q", "/custom/output", opts.OutputDir)
	}

	if opts.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", opts.Timeout)
	}
}

func TestRunOptionsIsVerbose(t *testing.T) {
	opts := RunOptions{Verbosity: VerbosityVerbose}
	if !opts.IsVerbose() {
		t.Error("expected IsVerbose() to return true")
	}

	opts.Verbosity = VerbosityNormal
	if opts.IsVerbose() {
		t.Error("expected IsVerbose() to return false")
	}
}

func TestRunOptionsIsQuiet(t *testing.T) {
	opts := RunOptions{Verbosity: VerbosityQuiet}
	if !opts.IsQuiet() {
		t.Error("expected IsQuiet() to return true")
	}

	opts.Verbosity = VerbosityNormal
	if opts.IsQuiet() {
		t.Error("expected IsQuiet() to return false")
	}
}

func TestPersonaOptions(t *testing.T) {
	if len(PersonaOptions) != 3 {
		t.Errorf("expected 3 persona options, got %d", len(PersonaOptions))
	}

	// Verify values match constants
	if PersonaOptions[0].Value != PersonaMinimal {
		t.Errorf("expected first persona value to be %q", PersonaMinimal)
	}
	if PersonaOptions[1].Value != PersonaBalanced {
		t.Errorf("expected second persona value to be %q", PersonaBalanced)
	}
	if PersonaOptions[2].Value != PersonaProduction {
		t.Errorf("expected third persona value to be %q", PersonaProduction)
	}
}

func TestVerbosityOptions(t *testing.T) {
	if len(VerbosityOptions) != 3 {
		t.Errorf("expected 3 verbosity options, got %d", len(VerbosityOptions))
	}
}

func TestExecutionOptions(t *testing.T) {
	if len(ExecutionOptions) != 2 {
		t.Errorf("expected 2 execution options, got %d", len(ExecutionOptions))
	}
}

func TestResumeModeOptions(t *testing.T) {
	if len(ResumeModeOptions) != 3 {
		t.Errorf("expected 3 resume mode options, got %d", len(ResumeModeOptions))
	}
}

func TestArchitectureOptions(t *testing.T) {
	if len(ArchitectureOptions) != 3 {
		t.Errorf("expected 3 architecture options, got %d", len(ArchitectureOptions))
	}
}
