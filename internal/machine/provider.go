package machine
import (
 "fmt"
 "github.com/ANRCM0/TX-Node/internal/config"
 "github.com/ANRCM0/TX-Node/internal/panel"
)
func newMachineControlPlane(provider string, cfg config.PanelConfig) (machineControlPlane, error) {
 switch provider {
 case "", "xboard": return newXboardMachineControlPlane(panel.NewClient(cfg)), nil
 default: return nil, fmt.Errorf("unsupported machine control-plane provider %q", provider)
 }
}
func mustMachineControlPlane(provider string, cfg config.PanelConfig) machineControlPlane {
 cp, err := newMachineControlPlane(provider, cfg)
 if err != nil { panic(err) }
 return cp
}
