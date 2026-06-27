package compiler

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

type Grant struct {
	Kind, Host, Secret, Path, Mode, Binary, Model string
	Port                                          int
}

type Policy struct {
	Version    int                      `yaml:"version"`
	Filesystem *Filesystem              `yaml:"filesystem_policy,omitempty"`
	Landlock   map[string]string        `yaml:"landlock,omitempty"`
	Process    map[string]string        `yaml:"process,omitempty"`
	Network    map[string]NetworkPolicy `yaml:"network_policies,omitempty"`
}
type Filesystem struct {
	IncludeWorkdir bool     `yaml:"include_workdir"`
	ReadOnly       []string `yaml:"read_only,omitempty"`
	ReadWrite      []string `yaml:"read_write,omitempty"`
}
type NetworkPolicy struct {
	Name      string     `yaml:"name,omitempty"`
	Endpoints []Endpoint `yaml:"endpoints"`
	Binaries  []Binary   `yaml:"binaries"`
}
type Endpoint struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Protocol    string `yaml:"protocol,omitempty"`
	Enforcement string `yaml:"enforcement,omitempty"`
	Access      string `yaml:"access,omitempty"`
}
type Binary struct {
	Path string `yaml:"path"`
}

func Compile(grants []Grant) ([]byte, string, error) {
	p := Policy{Version: 1, Filesystem: &Filesystem{IncludeWorkdir: true}, Landlock: map[string]string{"compatibility": "hard_requirement"}, Process: map[string]string{"run_as_user": "sandbox", "run_as_group": "sandbox"}, Network: map[string]NetworkPolicy{}}
	sort.Slice(grants, func(i, j int) bool { return fmt.Sprintf("%+v", grants[i]) < fmt.Sprintf("%+v", grants[j]) })
	for i, g := range grants {
		switch g.Kind {
		case "ReadPath":
			p.Filesystem.ReadOnly = append(p.Filesystem.ReadOnly, g.Path)
		case "WritePath":
			p.Filesystem.ReadWrite = append(p.Filesystem.ReadWrite, g.Path)
		case "ConnectHost":
			bin := g.Binary
			if bin == "" {
				bin = "/usr/bin/curl"
			}
			port := g.Port
			if port == 0 {
				port = 443
			}
			p.Network[fmt.Sprintf("connect_%03d", i)] = NetworkPolicy{Name: "cedar-granted-host", Endpoints: []Endpoint{{Host: g.Host, Port: port, Protocol: "rest", Enforcement: "enforce", Access: "full"}}, Binaries: []Binary{{Path: bin}}}
		}
	}
	out, err := yaml.Marshal(p)
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(out)
	return out, fmt.Sprintf("sha256:%x", h), nil
}
