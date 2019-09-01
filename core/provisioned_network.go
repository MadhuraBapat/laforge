package core

import (
	"fmt"
	"path"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/laforge/core/formatter"
	"github.com/pkg/errors"
)

// ProvisionedNetwork is a build artifact type to denote a network inside a team's provisioend infrastructure.
//easyjson:json
type ProvisionedNetwork struct {
	formatter.Formatable
	ID               string                      `hcl:"id,label" json:"id,omitempty"`
	Name             string                      `hcl:"name,attr" json:"name,omitempty"`
	CIDR             string                      `hcl:"cidr,attr" json:"cidr,omitempty"`
	NetworkID        string                      `hcl:"network_id,attr" json:"network_id,omitempty"`
	ProvisionedHosts map[string]*ProvisionedHost `json:"provisioned_hosts"`
	Status           Status                      `hcl:"status,optional" json:"status"`
	Network          *Network                    `json:"-"`
	Team             *Team                       `json:"-"`
	Build            *Build                      `json:"-"`
	Environment      *Environment                `json:"-"`
	Competition      *Competition                `json:"-"`
	OnConflict       *OnConflict                 `json:"-"`
	Caller           Caller                      `json:"-"`
	Dir              string                      `json:"-"`
}

func (p ProvisionedNetwork) ToString() string {
	return fmt.Sprintf(`ProvisionedNetwork
┠ ID (string)        = %s
┠ Name (string)      = %s
┠ CIDR (string)      = %s
┠ NetworkID (string) = %s
┗ Dir (string)       = %s
`,
		p.ID,
		p.Name,
		p.CIDR,
		p.NetworkID,
		p.Dir)
}

// We have no children on a DNSRecord, so nothing to iterate on, we'll just return
func (p ProvisionedNetwork) Iter() ([]formatter.Formatable, error) {
	tmp := []formatter.Formatable{
		p.Network,
		p.Team,
		p.Build,
		p.Environment,
		p.Competition,
	}

	for _, v := range p.ProvisionedHosts {
		tmp = append(tmp, v)
	}

	return tmp, nil
}

// Hash implements the Hasher interface
func (p *ProvisionedNetwork) Hash() uint64 {
	return xxhash.Sum64String(
		fmt.Sprintf(
			"name=%v cidr=%v net=%v team=%v status=%v",
			p.Name,
			p.CIDR,
			p.Network.Hash(),
			p.Team.Hash(),
			p.Status.Hash(),
		),
	)
}

// Path implements the Pather interface
func (p *ProvisionedNetwork) Path() string {
	return p.ID
}

// Base implements the Pather interface
func (p *ProvisionedNetwork) Base() string {
	return path.Base(p.ID)
}

// ValidatePath implements the Pather interface
func (p *ProvisionedNetwork) ValidatePath() error {
	if err := ValidateGenericPath(p.Path()); err != nil {
		return err
	}
	return nil
}

// GetCaller implements the Mergeable interface
func (p *ProvisionedNetwork) GetCaller() Caller {
	return p.Caller
}

// LaforgeID implements the Mergeable interface
func (p *ProvisionedNetwork) LaforgeID() string {
	return p.ID
}

// ParentLaforgeID returns the Team's parent build ID
func (p *ProvisionedNetwork) ParentLaforgeID() string {
	return path.Dir(path.Dir(p.LaforgeID()))
}

// GetOnConflict implements the Mergeable interface
func (p *ProvisionedNetwork) GetOnConflict() OnConflict {
	if p.OnConflict == nil {
		return OnConflict{
			Do:     "default",
			Append: true,
		}
	}
	return *p.OnConflict
}

// SetCaller implements the Mergeable interface
func (p *ProvisionedNetwork) SetCaller(ca Caller) {
	p.Caller = ca
}

// SetOnConflict implements the Mergeable interface
func (p *ProvisionedNetwork) SetOnConflict(o OnConflict) {
	p.OnConflict = &o
}

// Swap implements the Mergeable interface
func (p *ProvisionedNetwork) Swap(m Mergeable) error {
	rawVal, ok := m.(*ProvisionedNetwork)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", p, m)
	}
	*p = *rawVal
	return nil
}

// SetID increments the revision and sets the team ID if needed
func (p *ProvisionedNetwork) SetID() string {
	if p.ID == "" {
		p.ID = path.Join(p.Team.Path(), "networks", p.Network.Base())
	}
	if p.NetworkID == "" {
		p.NetworkID = p.Network.Path()
	}
	return p.ID
}

// CreateProvisionedHost creates the actual provisioned host object and assigns the parental objects accordingly.
func (p *ProvisionedNetwork) CreateProvisionedHost(host *Host) *ProvisionedHost {
	ph := &ProvisionedHost{
		Host:               host,
		SubnetIP:           host.CalcIP(p.CIDR),
		ProvisioningSteps:  map[string]*ProvisioningStep{},
		StepsByOffset:      []*ProvisioningStep{},
		ProvisionedNetwork: p,
		Team:               p.Team,
		Build:              p.Build,
		Environment:        p.Environment,
		Competition:        p.Competition,
	}
	p.ProvisionedHosts[ph.SetID()] = ph
	ph.Conn = ph.CreateConnection()
	ph.Conn.SetID()
	return ph
}

// CreateProvisionedHosts enumerates the parent environment's host by network and creates provisioned host objects in this tree.
func (p *ProvisionedNetwork) CreateProvisionedHosts() error {
	for _, h := range p.Team.Environment.HostByNetwork[p.Network.Path()] {
		ph := p.CreateProvisionedHost(h)
		err := ph.CreateProvisioningSteps()
		if err != nil {
			return err
		}
	}
	return nil
}

// Gather implements the Dependency interface
func (p *ProvisionedNetwork) Gather(g *Snapshot) error {
	// var err error
	// for _, h := range p.ProvisionedHosts {

	// 	// err = g.Relate(p, h)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	g.AddNode(h)
	// 	err = h.Gather(g)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// err = g.Relate(p, p.Network)
	// if err != nil {
	// 	return err
	// }
	// err = g.Relate(p.Network, p)
	// if err != nil {
	// 	return err
	// }
	return nil
}
