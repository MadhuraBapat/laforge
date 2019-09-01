package core

import (
	"fmt"
	"path"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/laforge/core/formatter"
	"github.com/pkg/errors"
)

// DNSRecord is a configurable type for defining DNS entries related to this host in the core DNS infrastructure (if enabled)
//easyjson:json
type DNSRecord struct {
	formatter.Formatable
	ID         string            `hcl:"id,label" json:"id,omitempty"`
	Name       string            `hcl:"name,attr" json:"name,omitempty"`
	Values     []string          `hcl:"values,optional" json:"values,omitempty"`
	Type       string            `hcl:"type,attr" json:"type,omitempty"`
	Zone       string            `hcl:"zone,attr" json:"zone,omitempty"`
	Vars       map[string]string `hcl:"vars,optional" json:"vars,omitempty"`
	Tags       map[string]string `hcl:"tags,optional" json:"tags,omitempty"`
	Disabled   bool              `hcl:"disabled,optional" json:"disabled,omitempty"`
	OnConflict *OnConflict       `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
	Caller     Caller            `json:"-"`
}

// ToString returns a string based representation of this DNSRecord
func (r DNSRecord) ToString() string {
	return fmt.Sprintf(`DNSRecord
┠ ID (string)     = %s
┠ Name (string)   = %s
┠ Disabled (bool) = %t
┠ Values (array)
%s
┠ Vars (map)
%s
┠ Tags (map)
%s
┠ Type (string)   = %s
┗ Zone (string)   = %s
`,
		r.ID,
		r.Name,
		r.Disabled,
		formatter.FormatStringSlice(r.Values),
		formatter.FormatStringMap(r.Vars),
		formatter.FormatStringMap(r.Tags),
		r.Type,
		r.Zone)
}

// We have no children on a DNSRecord, so nothing to iterate on, we'll just return
func (r DNSRecord) Iter() ([]formatter.Formatable, error) {
	return []formatter.Formatable{}, nil
}

// Hash implements the Hasher interface
func (r *DNSRecord) Hash() uint64 {
	return xxhash.Sum64String(
		fmt.Sprintf(
			"name=%v values=%v type=%v zone=%v vars=%v disabled=%v",
			r.Name,
			r.Values,
			r.Type,
			r.Zone,
			r.Vars,
			r.Disabled,
		),
	)
}

// Path implements the Pather interface
func (r *DNSRecord) Path() string {
	return r.ID
}

// Base implements the Pather interface
func (r *DNSRecord) Base() string {
	return path.Base(r.ID)
}

// ValidatePath implements the Pather interface
func (r *DNSRecord) ValidatePath() error {
	if err := ValidateGenericPath(r.Path()); err != nil {
		return err
	}
	if topdir := strings.Split(r.Path(), `/`); topdir[1] != "dns-records" {
		return fmt.Errorf("path %s is not rooted in /%s", r.Path(), topdir[1])
	}
	return nil
}

// GetCaller implements the Mergeable interface
func (r *DNSRecord) GetCaller() Caller {
	return r.Caller
}

// LaforgeID implements the Mergeable interface
func (r *DNSRecord) LaforgeID() string {
	return r.ID
}

// GetOnConflict implements the Mergeable interface
func (r *DNSRecord) GetOnConflict() OnConflict {
	if r.OnConflict == nil {
		return OnConflict{
			Do: "default",
		}
	}
	return *r.OnConflict
}

// SetCaller implements the Mergeable interface
func (r *DNSRecord) SetCaller(c Caller) {
	r.Caller = c
}

// SetOnConflict implements the Mergeable interface
func (r *DNSRecord) SetOnConflict(o OnConflict) {
	r.OnConflict = &o
}

// Kind implements the Provisioner interface
func (r *DNSRecord) Kind() string {
	return "dns_record"
}

// Fullpath implements the Pather interface
func (r *DNSRecord) Fullpath() string {
	return r.LaforgeID()
}

// ParentLaforgeID implements the Dependency interface
func (r *DNSRecord) ParentLaforgeID() string {
	return r.Path()
}

// Gather implements the Dependency interface
func (r *DNSRecord) Gather(g *Snapshot) error {
	return nil
}

// Swap implements the Mergeable interface
func (r *DNSRecord) Swap(m Mergeable) error {
	rawVal, ok := m.(*DNSRecord)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", r, m)
	}
	*r = *rawVal
	return nil
}

// Inherited is a boolean condition that is triggered when a DNS record is not statically defined
func (r *DNSRecord) Inherited() bool {
	return len(r.Values) == 0
}

// SetValue is an override which allows you to set the value of a DNS record during a template run
func (r *DNSRecord) SetValue(val string) {
	r.Values = append(r.Values, val)
}
