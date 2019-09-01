package core

import (
	"fmt"
	"path"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/laforge/core/formatter"
	"github.com/pkg/errors"
)

// ProvisioningStep is a build artifact type to denote a specific step inside of a provisioned host
//easyjson:json
type ProvisioningStep struct {
	formatter.Formatable
	ID                 string              `hcl:"id,label" json:"id,omitempty"`
	ProvisionerID      string              `hcl:"provisioner_id,attr" json:"provisioner_id,omitempty"`
	ProvisionerType    string              `hcl:"provisioner_type,attr" json:"provisioner_type,omitempty"`
	StepNumber         int                 `hcl:"step_number,attr" json:"step_number,omitempty"`
	Status             string              `hcl:"status,optional" json:"status,omitempty"`
	ProvisionedHost    *ProvisionedHost    `json:"-"`
	ProvisionedNetwork *ProvisionedNetwork `json:"-"`
	Host               *Host               `json:"-"`
	Network            *Network            `json:"-"`
	Team               *Team               `json:"-"`
	Build              *Build              `json:"-"`
	Environment        *Environment        `json:"-"`
	Competition        *Competition        `json:"-"`
	Provisioner        Provisioner         `json:"-"`
	Script             *Script             `json:"-"`
	Command            *Command            `json:"-"`
	RemoteFile         *RemoteFile         `json:"-"`
	DNSRecord          *DNSRecord          `json:"-"`
	OnConflict         *OnConflict         `json:"-"`
	Caller             Caller              `json:"-"`
	Dir                string              `json:"-"`
}

func (p ProvisioningStep) ToString() string {
	return fmt.Sprintf(`ProvisioningStep
┠ ID (string)              = %s
┠ ProvisionerID (string)   = %s
┠ ProvisionerType (string) = %s
┠ StepNumber (int)         = %d
┠ Status (string)          = %s
┗ Dir (string)             = %s
`,
		p.ID,
		p.ProvisionerID,
		p.ProvisionerType,
		p.StepNumber,
		p.Status,
		p.Dir)
}

// Given that the provisioning step will most likely include all of these other items
// that have already been listed, we are going to just leave them for now.
func (p ProvisioningStep) Iter() ([]formatter.Formatable, error) {
	return []formatter.Formatable{}, nil
}

// Hash implements the Hasher interface
func (p *ProvisioningStep) Hash() uint64 {
	return xxhash.Sum64String(
		fmt.Sprintf(
			"pid=%v ptype=%v phash=%v snum=%v",
			p.ProvisionerID,
			p.ProvisionerType,
			p.Provisioner.Hash(),
			p.StepNumber,
		),
	)
}

// Path implements the Pather interface
func (p *ProvisioningStep) Path() string {
	return p.ID
}

// Base implements the Pather interface
func (p *ProvisioningStep) Base() string {
	return path.Base(p.ID)
}

// ValidatePath implements the Pather interface
func (p *ProvisioningStep) ValidatePath() error {
	if err := ValidateGenericPath(p.Path()); err != nil {
		return err
	}
	return nil
}

// GetCaller implements the Mergeable interface
func (p *ProvisioningStep) GetCaller() Caller {
	return p.Caller
}

// LaforgeID implements the Mergeable interface
func (p *ProvisioningStep) LaforgeID() string {
	return p.ID
}

// ParentLaforgeID returns the Team's parent build ID
func (p *ProvisioningStep) ParentLaforgeID() string {
	return path.Dir(path.Dir(p.LaforgeID()))
}

// GetOnConflict implements the Mergeable interface
func (p *ProvisioningStep) GetOnConflict() OnConflict {
	if p.OnConflict == nil {
		return OnConflict{
			Do:     "default",
			Append: true,
		}
	}
	return *p.OnConflict
}

// SetCaller implements the Mergeable interface
func (p *ProvisioningStep) SetCaller(ca Caller) {
	p.Caller = ca
}

// SetOnConflict implements the Mergeable interface
func (p *ProvisioningStep) SetOnConflict(o OnConflict) {
	p.OnConflict = &o
}

// Swap implements the Mergeable interface
func (p *ProvisioningStep) Swap(m Mergeable) error {
	rawVal, ok := m.(*ProvisioningStep)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", p, m)
	}
	*p = *rawVal
	return nil
}

// SetID increments the revision and sets the team ID if needed
func (p *ProvisioningStep) SetID() string {
	if p.ID == "" {
		p.ID = path.Join(p.ProvisionedHost.Path(), "steps", fmt.Sprintf("%d-%s", p.StepNumber, p.Provisioner.Base()))
	}

	switch v := p.Provisioner.(type) {
	case *Command:
		p.Command = v
	case *DNSRecord:
		p.DNSRecord = v
	case *RemoteFile:
		p.RemoteFile = v
	case *Script:
		p.Script = v
	}

	return p.ID
}

// Gather implements the Dependency interface
func (p *ProvisioningStep) Gather(g *Snapshot) error {
	// switch v := p.Provisioner.(type) {
	// case *Command:
	// 	// err := g.Relate(p.Environment, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	g.AddNode(v)
	// 	// err := g.Relate(p.Host, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	// err = g.Relate(v, p)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// case *DNSRecord:
	// 	// err := g.Relate(p.Environment, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	g.AddNode(v)
	// 	// err := g.Relate(p.Host, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	// err = g.Relate(v, p)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// case *RemoteFile:
	// 	// err := g.Relate(p.Environment, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	g.AddNode(v)
	// 	// err := g.Relate(p.Host, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	// err = g.Relate(v, p)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// case *Script:
	// 	// err := g.Relate(p.Environment, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	g.AddNode(v)
	// 	// err := g.Relate(p.Host, v)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	// err = g.Relate(v, p)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// default:
	// 	return fmt.Errorf("invalid provisioner type for %s: %T", p.Path(), p.Provisioner)
	// }
	return nil
}
