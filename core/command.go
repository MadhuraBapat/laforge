package core

import (
	"fmt"
	"path"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/laforge/core/formatter"

	"github.com/pkg/errors"
)

// Command represents an executable command that can be defined as part of a host configuration step
//easyjson:json
type Command struct {
	formatter.Formatable
	ID           string            `hcl:"id,label" json:"id,omitempty"`
	Name         string            `hcl:"name,attr" json:"name,omitempty"`
	Description  string            `hcl:"description,attr" json:"description,omitempty"`
	Program      string            `hcl:"program,attr" json:"program,omitempty"`
	Args         []string          `hcl:"args,attr" json:"args,omitempty"`
	IgnoreErrors bool              `hcl:"ignore_errors,attr" json:"ignore_errors,omitempty"`
	Cooldown     int               `hcl:"cooldown,attr" json:"cooldown,omitempty"`
	IO           *IO               `hcl:"io,block" json:"io,omitempty"`
	Disabled     bool              `hcl:"disabled,attr" json:"disabled,omitempty"`
	Vars         map[string]string `hcl:"vars,attr" json:"vars,omitempty"`
	Tags         map[string]string `hcl:"tags,attr" json:"tags,omitempty"`
	OnConflict   *OnConflict       `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
	Maintainer   *User             `hcl:"maintainer,block" json:"maintainer,omitempty"`
	Caller       Caller            `json:"-"`
}

func (c Command) ToString() []string {
	return fmt.Sprintf(`Command
┠ ID (string)          = %s
┠ Name (string)        = %d
┠ Description (string) = %d
┠ Program (string)     = %d
┠ Args (array)
%s
┠ Vars (map)
%s
┠ Tags (map)
%s
┠ Cooldown (int)       = %d
┠ IgnoreErrors (bool)  = %s
┗ Disabled (bool)      = %s
`,
		c.ID,
		c.Name,
		c.Description,
		c.Program,
		formatter.FormatStringSlice(c.Args),
		formatter.FormatStringMap(c.Vars),
		formatter.FormatStringMap(c.Tags),
		c.Cooldown,
		c.IgnoreErrors,
		c.Disabled)
}

// We have no children on a Command, so nothing to iterate on, we'll just return
func (c Command) Iter() ([]formatter.Formatable, error) {
	return []formatter.Formatable{}, nil
}

// Hash implements the Hasher interface
func (c *Command) Hash() uint64 {
	iostr := "n/a"
	if c.IO != nil {
		iostr = c.IO.Stderr + c.IO.Stdin + c.IO.Stdout
	}

	return xxhash.Sum64String(
		fmt.Sprintf(
			"program=%v args=%v ignoreerrors=%v cooldown=%v io=%v disabled=%v vars=%v",
			c.Program,
			strings.Join(c.Args, ","),
			c.IgnoreErrors,
			c.Cooldown,
			iostr,
			c.Disabled,
			c.Vars,
		),
	)
}

// Path implements the Pather interface
func (c *Command) Path() string {
	return c.ID
}

// Base implements the Pather interface
func (c *Command) Base() string {
	return path.Base(c.ID)
}

// ValidatePath implements the Pather interface
func (c *Command) ValidatePath() error {
	if err := ValidateGenericPath(c.Path()); err != nil {
		return err
	}
	if topdir := strings.Split(c.Path(), `/`); topdir[1] != "commands" {
		return fmt.Errorf("path %s is not rooted in /%s", c.Path(), topdir[1])
	}
	return nil
}

// GetCaller implements the Mergeable interface
func (c *Command) GetCaller() Caller {
	return c.Caller
}

// LaforgeID implements the Mergeable interface
func (c *Command) LaforgeID() string {
	return c.ID
}

// Fullpath implements the Pather interface
func (c *Command) Fullpath() string {
	return c.LaforgeID()
}

// ParentLaforgeID implements the Dependency interface
func (c *Command) ParentLaforgeID() string {
	return c.Path()
}

// Gather implements the Dependency interface
func (c *Command) Gather(g *Snapshot) error {
	return nil
}

// GetOnConflict implements the Mergeable interface
func (c *Command) GetOnConflict() OnConflict {
	if c.OnConflict == nil {
		return OnConflict{
			Do: "default",
		}
	}
	return *c.OnConflict
}

// SetCaller implements the Mergeable interface
func (c *Command) SetCaller(ca Caller) {
	c.Caller = ca
}

// SetOnConflict implements the Mergeable interface
func (c *Command) SetOnConflict(o OnConflict) {
	c.OnConflict = &o
}

// Kind implements the Provisioner interface
func (c *Command) Kind() string {
	return "command"
}

// CommandString is a template helper function to embed commands into the output
func (c *Command) CommandString() string {
	cmd := []string{c.Program}
	for _, x := range c.Args {
		cmd = append(cmd, x)
	}
	return strings.Join(cmd, " ")
}

// Swap implements the Mergeable interface
func (c *Command) Swap(m Mergeable) error {
	rawVal, ok := m.(*Command)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", c, m)
	}
	*c = *rawVal
	return nil
}
