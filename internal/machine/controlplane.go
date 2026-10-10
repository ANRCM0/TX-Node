package machine

import (
 "github.com/ANRCM0/TX-Node/internal/panel"
 "github.com/ANRCM0/TX-Node/internal/config"
 "github.com/ANRCM0/TX-Node/internal/controlplane"
)

// machineControlPlane isolates machine transport operations from orchestration.
// The Xboard adapter preserves the existing wire format during migration.
type machineControlPlane interface {
 GetMachineNodes() (*panel.MachineNodesResponse, error)
 Handshake() (*panel.HandshakeResponse, error)
 ForNode(int) machineNodeClient
 ReportMachineStatus(float64, [2]uint64, [2]uint64, [2]uint64, float64, float64, *panel.MachineRuntimeStatus) error
}

// machineNodeClient owns node REST snapshots and constructs the per-node
// control-plane adapter. The orchestrator never handles raw node REST clients.
type machineNodeClient interface {
 GetConfig() (*panel.NodeConfig, error)
 ResetConfigETag()
 ControlPlane(config.KernelConfig, controlplane.PushClient, func(chan<- controlplane.StatusChange) *controlplane.NodeMailbox) controlplane.ControlPlane
}

type xboardMachineNodeClient struct { client *panel.Client }
func (x *xboardMachineNodeClient) GetConfig() (*panel.NodeConfig, error) { return x.client.GetConfig() }
func (x *xboardMachineNodeClient) ResetConfigETag() { x.client.ResetConfigETag() }
func (x *xboardMachineNodeClient) ControlPlane(k config.KernelConfig, push controlplane.PushClient, register func(chan<- controlplane.StatusChange) *controlplane.NodeMailbox) controlplane.ControlPlane {
 return controlplane.NewMachineXboardControlPlane(x.client, k, push, register)
}

type xboardMachineControlPlane struct { client *panel.Client }

func newXboardMachineControlPlane(client *panel.Client) machineControlPlane {
 return &xboardMachineControlPlane{client: client}
}
func (x *xboardMachineControlPlane) GetMachineNodes() (*panel.MachineNodesResponse, error) { return x.client.GetMachineNodes() }
func (x *xboardMachineControlPlane) Handshake() (*panel.HandshakeResponse, error) { return x.client.Handshake() }
func (x *xboardMachineControlPlane) ForNode(id int) machineNodeClient { return &xboardMachineNodeClient{client: x.client.ForNode(id)} }
func (x *xboardMachineControlPlane) ReportMachineStatus(cpu float64, mem, swap, disk [2]uint64, netIn, netOut float64, status *panel.MachineRuntimeStatus) error {
 return x.client.ReportMachineStatus(cpu, mem, swap, disk, netIn, netOut, status)
}
