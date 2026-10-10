package machine
import (
 "testing"
 "github.com/ANRCM0/TX-Node/internal/config"
)
func TestMachineProviderFactory(t *testing.T) {
 if cp, err := newMachineControlPlane("", config.PanelConfig{}); err != nil || cp == nil { t.Fatalf("default provider: %v", err) }
 if cp, err := newMachineControlPlane("xboard", config.PanelConfig{}); err != nil || cp == nil { t.Fatalf("xboard provider: %v", err) }
 if cp, err := newMachineControlPlane("txboard", config.PanelConfig{}); err == nil || cp != nil { t.Fatal("unimplemented txboard provider must fail closed") }
}
