package tui

import (
	"testing"
)

func TestPagentTheme(t *testing.T) {
	theme := PagentTheme()
	if theme == nil {
		t.Error("PagentTheme() returned nil")
	}
}

func TestHeaderStyle(t *testing.T) {
	style := HeaderStyle()
	// Style should be non-empty (has properties set)
	rendered := style.Render("test")
	if rendered == "" {
		t.Error("HeaderStyle() rendered empty string")
	}
}

func TestTitleStyle(t *testing.T) {
	style := TitleStyle()
	rendered := style.Render("test")
	if rendered == "" {
		t.Error("TitleStyle() rendered empty string")
	}
}

func TestSuccessStyle(t *testing.T) {
	style := SuccessStyle()
	rendered := style.Render("test")
	if rendered == "" {
		t.Error("SuccessStyle() rendered empty string")
	}
}

func TestMutedStyle(t *testing.T) {
	style := MutedStyle()
	rendered := style.Render("test")
	if rendered == "" {
		t.Error("MutedStyle() rendered empty string")
	}
}

func TestBanner(t *testing.T) {
	banner := Banner()
	if banner == "" {
		t.Error("Banner() returned empty string")
	}
	// Should contain PAGENT in ASCII art
	if len(banner) < 100 {
		t.Error("Banner() seems too short for ASCII art")
	}
}
