package machine

import "github.com/ANRCM0/TX-Node/internal/panel"

// machineControlPlane isolates machine transport operations from orchestration.
// The Xboard adapter preserves the existing wire format during migration.
type machineControlPlane interface {
 GetMachineNodes() (*panel.MachineNodesResponse, error)
 Handshake() (*panel.HandshakeResponse, error)
 ForNode(int) *panel.Client
 ReportMachineStatus(float64, [2]uint64, [2]uint64, [2]uint64, float64, float64, *panel.MachineRuntimeStatus) error
}

type xboardMachineControlPlane struct { client *panel.Client }

func newXboardMachineControlPlane(client *panel.Client) machineControlPlane {
 return &xboardMachineControlPlane{client: client}
}
func (x *xboardMachineControlPlane) GetMachineNodes() (*panel.MachineNodesResponse, error) { return x.client.GetMachineNodes() }
func (x *xboardMachineControlPlane) Handshake() (*panel.HandshakeResponse, error) { return x.client.Handshake() }
func (x *xboardMachineControlPlane) ForNode(id int) *panel.Client { return x.client.ForNode(id) }
func (x *xboardMachineControlPlane) ReportMachineStatus(cpu float64, mem, swap, disk [2]uint64, netIn, netOut float64, status *panel.MachineRuntimeStatus) error {
 return x.client.ReportMachineStatus(cpu, mem, swap, disk, netIn, netOut, status)
}
