package controlplane

import (
	"testing"

	"github.com/ANRCM0/TX-Node/internal/config"
)

func TestProviderForConfig(t *testing.T) {
	if got := ProviderForConfig(&config.Config{}); got != ProviderXboard {
		t.Fatalf("default provider = %q, want %q", got, ProviderXboard)
	}
	if got := ProviderForConfig(&config.Config{Panel: config.PanelConfig{Provider: "xboard"}}); got != ProviderXboard {
		t.Fatalf("explicit xboard provider = %q, want %q", got, ProviderXboard)
	}
	if got := ProviderForConfig(&config.Config{Panel: config.PanelConfig{Provider: "txboard"}}); got != ProviderTXBoard {
		t.Fatalf("txboard provider = %q, want %q", got, ProviderTXBoard)
	}
	if got := ProviderForConfig(&config.Config{Standalone: &config.StandaloneConfig{Enabled: true}}); got != ProviderLocal {
		t.Fatalf("standalone provider = %q, want %q", got, ProviderLocal)
	}
}

func TestNewForConfig(t *testing.T) {
	local := NewForConfig(&config.Config{Standalone: &config.StandaloneConfig{Enabled: true}})
	if _, ok := local.(*LocalControlPlane); !ok {
		t.Fatalf("standalone factory returned %T, want *LocalControlPlane", local)
	}

	remote := NewForConfig(&config.Config{})
	if _, ok := remote.(*XboardControlPlane); !ok {
		t.Fatalf("remote factory returned %T, want *XboardControlPlane", remote)
	}
}
