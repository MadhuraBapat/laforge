package core

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/dag"

	"github.com/gen0cide/laforge/core/formatter"
	"github.com/gen0cide/laforge/core/graph"
	"github.com/pkg/errors"
)

// Pather is an interface to define hosts which need to conform to valid pathing schemes
type Pather interface {
	Path() string
	Base() string
	ValidatePath() error
}

// Dependency is an interface to define a laforge object that can be represented on the graph
type Dependency interface {
	Pather
	graph.Hasher
	ParentLaforgeID() string
	Gather(g *Snapshot) error
}

// ResourceHasher is an interface to define types who have file dependencies to checkum them
type ResourceHasher interface {
	ResourceHash() uint64
}

var (
	genericPathRegexp      = regexp.MustCompile(`^\/[a-z0-9\-\/]{3,}[a-z0-9]$`)
	consecutiveSlashRegexp = regexp.MustCompile(`\/\/`)

	// ErrPathEndsInSlash is thrown when a path ends in a trailing slash
	ErrPathEndsInSlash = errors.New("path ends in a trailing slash")

	// ErrPathContainsInvalidChars is thrown when a path contains invalid characters
	ErrPathContainsInvalidChars = errors.New("path contains invalid characters")

	// ErrPathContainsDuplicateSlash is thrown when a path contains two consecutive slashes
	ErrPathContainsDuplicateSlash = errors.New("path contains consecutive slash characters")
)

// ValidateGenericPath covers basic rules validating a path generically for invalid schema
func ValidateGenericPath(p string) error {
	if !genericPathRegexp.MatchString(p) {
		return ErrPathContainsInvalidChars
	}
	if consecutiveSlashRegexp.MatchString(p) {
		return ErrPathContainsDuplicateSlash
	}
	return nil
}

// Metadata stores metadata about different structs within the environment
//easyjson:json
type Metadata struct {
	formatter.Formatable
	Dependency Dependency     `json:"-"`
	ID         string         `json:"id"`
	ObjectType LFType         `json:"object_type"`
	Created    bool           `json:"provisioned,omitempty"`
	Tainted    bool           `json:"tainted,omitempty"`
	Addition   bool           `json:"addition,omitempty"`
	Checksum   uint64         `json:"checksum,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
	ModifiedAt time.Time      `json:"modified_at,omitempty"`
	Resources  []MetaResource `json:"resources,omitempty"`
}

func (m Metadata) ToString() string {
	return fmt.Sprintf(`Metadata
┠ ID (string)       = %s
┠ Created (bool)    = %t
┠ Tainted (bool)    = %t
┠ Addition (bool)   = %t
┠ Checksum (string) = %s
┠ CreatedAt (time)  = %s
┗ ModifiedAt (time) = %s
`,
		m.ID,
		m.Created,
		m.Tainted,
		m.Addition,
		m.Checksum,
		m.CreatedAt,
		m.ModifiedAt)
}

// We have no children on a DNSRecord, so nothing to iterate on, we'll just return
func (m Metadata) Iter() ([]formatter.Formatable, error) {
	return []formatter.Formatable{}, nil
}

// LFType describes a string representation of elements in Laforge
type LFType string

const (
	// LFTypeCompetition is a constant to define object type when serialized
	LFTypeCompetition LFType = `competition`

	// LFTypeNetwork is a constant to define object type when serialized
	LFTypeNetwork LFType = `network`

	// LFTypeHost is a constant to define object type when serialized
	LFTypeHost LFType = `host`

	// LFTypeRemoteFile is a constant to define object type when serialized
	LFTypeRemoteFile LFType = `remote_file`

	// LFTypeCommand is a constant to define object type when serialized
	LFTypeCommand LFType = `command`

	// LFTypeDNSRecord is a constant to define object type when serialized
	LFTypeDNSRecord LFType = `dns_record`

	// LFTypeScript is a constant to define object type when serialized
	LFTypeScript LFType = `script`

	// LFTypeEnvironment is a constant to define object type when serialized
	LFTypeEnvironment LFType = `environment`

	// LFTypeBuild is a constant to define object type when serialized
	LFTypeBuild LFType = `build`

	// LFTypeTeam is a constant to define object type when serialized
	LFTypeTeam LFType = `team`

	// LFTypeProvisionedNetwork is a constant to define object type when serialized
	LFTypeProvisionedNetwork LFType = `provisioned_network`

	// LFTypeProvisionedHost is a constant to define object type when serialized
	LFTypeProvisionedHost LFType = `provisioned_host`

	// LFTypeConnection is a constant to define object type when serialized
	LFTypeConnection LFType = `connection`

	// LFTypeProvisioningStep is a constant to define object type when serialized
	LFTypeProvisioningStep LFType = `provisioning_step`

	// LFTypeUnknown is totally a fucker
	LFTypeUnknown LFType = "unknown"
)

// TypeByPath is a helper function specifically for metadata to call TypeByPath easily
func (m *Metadata) TypeByPath() LFType {
	return TypeByPath(m.ID)
}

// IsGlobalType is a helper function specifically for metadata to call IsGlobalType easily
func (m *Metadata) IsGlobalType() bool {
	return IsGlobalType(m.ID)
}

// IsGlobalType is a helper function that attempts to determine if the specified path is of a global type
func IsGlobalType(p string) bool {
	switch TypeByPath(p) {
	case LFTypeCompetition:
		return true
	case LFTypeCommand:
		return true
	case LFTypeNetwork:
		return true
	case LFTypeHost:
		return true
	case LFTypeDNSRecord:
		return true
	case LFTypeRemoteFile:
		return true
	case LFTypeScript:
		return true
	case LFTypeEnvironment:
		return false
	case LFTypeTeam:
		return false
	case LFTypeProvisionedNetwork:
		return false
	case LFTypeProvisionedHost:
		return false
	case LFTypeProvisioningStep:
		return false
	case LFTypeUnknown:
		return false
	default:
		return false
	}
}

// TypeByPath attempts to resolve what type the object is based on it's ID schema
func TypeByPath(p string) LFType {
	if !path.IsAbs(p) {
		return LFTypeCompetition
	}
	pelms := strings.Split(p, `/`)

	switch pelms[1] {
	case "scripts":
		return LFTypeScript
	case "networks":
		return LFTypeNetwork
	case "hosts":
		return LFTypeHost
	case "commands":
		return LFTypeCommand
	case "dns-records":
		return LFTypeDNSRecord
	case "files":
		return LFTypeRemoteFile
	}

	if path.Base(path.Dir(p)) == "envs" {
		return LFTypeEnvironment
	}

	if path.Base(path.Dir(path.Dir(p))) == "envs" {
		return LFTypeBuild
	}

	if path.Base(path.Dir(p)) == "teams" {
		return LFTypeTeam
	}

	if path.Base(path.Dir(p)) == "networks" && pelms[1] == "envs" {
		return LFTypeProvisionedNetwork
	}

	if path.Base(path.Dir(p)) == "hosts" && pelms[1] == "envs" {
		return LFTypeProvisionedHost
	}

	if path.Base(p) == "conn" && path.Base(path.Dir(path.Dir(p))) == "hosts" {
		return LFTypeConnection
	}

	dir := path.Dir(p)
	_ = dir
	if path.Base(path.Dir(p)) == "steps" && pelms[1] == "envs" {
		return LFTypeProvisioningStep
	}

	return LFTypeUnknown
}

// GetTeamIDFromPath attempts to resolve the team's unique ID from the provided ID
func GetTeamIDFromPath(p string) (string, error) {
	if !path.IsAbs(p) {
		return "", errors.New("not a valid absolute path ID schema")
	}

	pelms := strings.Split(p, `/`)
	if len(pelms) < 6 {
		return "", errors.New("path does not meet minimum expected structure for team identification")
	}

	if pelms[1] != "envs" {
		return "", errors.New("path is not rooted inside an environment")
	}

	return strings.Join(pelms[0:6], `/`), nil
}

// GetID implements the DotNode interface
func (m *Metadata) GetID() string {
	return m.ID
}

// // GetGID implements the DotNode interface
// func (m *Metadata) GetGID() int {
// 	return m.GID
// }

// // GetGCost implements the DotNode interface
// func (m *Metadata) GetGCost() int64 {
// 	return m.GCost
// }

// GetChecksum implements the DotNode interface
func (m *Metadata) GetChecksum() uint64 {
	return m.Checksum
}

// DotNode implements the DotNodder interface
func (m *Metadata) DotNode(s string, d *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: s,
		Attrs: map[string]string{
			"checksum": fmt.Sprintf("%d", m.Checksum),
		},
	}
}

// Name implements the DotNode interface
// func (m *Metadata) Name() string {
// 	return m.ID
// }

// Label implements the DotNode interface
func (m *Metadata) Label() string {
	if m.Dependency == nil {
		panic(fmt.Errorf("could not find dependency for %s", m.ID))
	}
	return fmt.Sprintf("<table border=\"0\" cellborder=\"0\" cellspacing=\"1\">"+
		"<tr><td align=\"center\"><b>%s</b></td></tr>"+
		"<tr><td align=\"left\"><font face=\"Courier\" point-size=\"11\">type     = %s</font></td></tr>"+
		"<tr><td align=\"left\"><font face=\"Courier\" point-size=\"11\">checksum = %x</font></td></tr></table>",
		m.ID,
		m.ObjectType,
		m.Checksum,
	)
	// return fmt.Sprintf("%s"+`++++`+"|type = %s"+`++++`+"checksum = %x"+`++++`+"parent = %s"+`++++`,
	// 	m.ID,
	// 	m.ObjectType,
	// 	m.Checksum,
	// 	m.Dependency.ParentLaforgeID(),
	// )
}

// Shape implements the DotNode interface
func (m *Metadata) Shape() string {
	if m.IsGlobalType() {
		return "Mrecord"
	}
	return "record"
	// switch m.TypeByPath() {
	// case LFTypeEnvironment:
	// 	return "tripleoctagon"
	// case LFTypeBuild:
	// 	return "doubleoctagon"
	// case LFTypeTeam:
	// 	return "star"
	// case LFTypeProvisionedNetwork:
	// 	return "component"
	// case LFTypeProvisionedHost:
	// 	return "box3d"
	// case LFTypeProvisioningStep:
	// 	return "note"
	// case LFTypeConnection:
	// 	return "rarrow"
	// default:
	// 	return "ellipse"
	// }
}

// Style implements the DotNode interface
func (m *Metadata) Style() string {
	if m.IsGlobalType() {
		return "filled,rounded,dotted"
	}
	switch m.TypeByPath() {
	case LFTypeEnvironment:
		return "filled"
	case LFTypeBuild:
		return "filled"
	case LFTypeTeam:
		return "filled,bold"
	case LFTypeProvisionedNetwork:
		return "filled,bold"
	case LFTypeProvisionedHost:
		return "filled,bold"
	case LFTypeProvisioningStep:
		return "filled,bold"
	case LFTypeConnection:
		return "filled,bold"
	default:
		return "filled"
	}
}

// FillColor implements the DotNode interface
func (m *Metadata) FillColor() string {
	switch m.TypeByPath() {
	case LFTypeCompetition:
		return "gold"
	case LFTypeCommand:
		return "aquamarine"
	case LFTypeNetwork:
		return "lightblue1"
	case LFTypeHost:
		return "palegreen1"
	case LFTypeDNSRecord:
		return "mistyrose2"
	case LFTypeRemoteFile:
		return "navajowhite"
	case LFTypeScript:
		return "lightgoldenrod1"
	case LFTypeEnvironment:
		return "chartreuse"
	case LFTypeBuild:
		return "coral"
	case LFTypeTeam:
		return "green"
	case LFTypeProvisionedNetwork:
		return "steelblue1"
	case LFTypeProvisionedHost:
		return "springgreen1"
	case LFTypeProvisioningStep:
		return "yellow2"
	case LFTypeConnection:
		return "snow3"
	case LFTypeUnknown:
		return "khaki"
	default:
		return "cornsilk"
	}
}

// Hash implements the hasher interface
func (m *Metadata) Hash() uint64 {
	if m.Checksum == 0 {
		m.CalculateChecksum()
	}
	return m.Checksum
}

// Hashcode implements the Hashable interface
func (m *Metadata) Hashcode() interface{} {
	return m.Checksum
}

// String implements the stringer interface
func (m *Metadata) String() string {
	return m.ID
}

// CalculateChecksum assigns the metadata object's checksum field with the dependency's hash
func (m *Metadata) CalculateChecksum() {
	m.Checksum = m.Dependency.Hash()
}

// ToRevision generates a revision object for m
func (m *Metadata) ToRevision() *Revision {
	return &Revision{
		ID:        m.ID,
		Type:      TypeByPath(m.ID),
		Status:    RevStatusPlanned,
		Checksum:  m.Checksum,
		Timestamp: time.Now(),
	}
}

// MetaResource stores information about a local file dependency. This can be a directory.
// If the resource is a directory, it will be recursively gzip'd and that will be checksum'd.
// If the resource is a directory, size will be the size of the final gzip file.
// Note creation and modification date refer to meta resource validation, not the actual file.
//easyjson:json
type MetaResource struct {
	ID           string    `json:"id,omitempty"`
	PathFromBase string    `json:"path_from_base,omitempty"`
	Basename     string    `json:"basename,omitempty"`
	ParentIDs    []string  `json:"parent_ids,omitempty"`
	IsDir        bool      `json:"is_dir,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	ModifiedAt   time.Time `json:"modified_at,omitempty"`
	Checksum     uint64    `json:"checksum,omitempty"`
	Size         int       `json:"size,omitempty"`
}
