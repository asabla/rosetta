package authz

type Capability string

const (
	CreateSandbox Capability = "CreateSandbox"
	ConnectHost   Capability = "ConnectHost"
	UseSecret     Capability = "UseSecret"
	ReadPath      Capability = "ReadPath"
	WritePath     Capability = "WritePath"
	RunBinary     Capability = "RunBinary"
	UseModel      Capability = "UseModel"
)

type Request struct {
	Principal string            `json:"principal"`
	Action    Capability        `json:"action"`
	Resource  string            `json:"resource"`
	Context   map[string]string `json:"context,omitempty"`
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

type Authorizer interface{ IsAllowed(Request) Decision }
