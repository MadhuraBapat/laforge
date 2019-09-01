package core

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/laforge/core/formatter"
	"github.com/pkg/errors"
)

// RemoteFile is a configurable type that defines a static file that will be placed on a configured target host.
//easyjson:json
type RemoteFile struct {
	formatter.Formatable
	ID          string            `hcl:"id,label" json:"id,omitempty"`
	SourceType  string            `hcl:"source_type,attr" json:"source_type,omitempty"`
	Source      string            `hcl:"source,attr" json:"source,omitempty"`
	Destination string            `hcl:"destination,attr" json:"destination,omitempty"`
	Vars        map[string]string `hcl:"vars,optional" json:"vars,omitempty"`
	Tags        map[string]string `hcl:"tags,optional" json:"tags,omitempty"`
	Template    bool              `hcl:"template,optional" json:"template,omitempty"`
	Perms       string            `hcl:"perms,optional" json:"perms,omitempty"`
	Disabled    bool              `hcl:"disabled,optional" json:"disabled,omitempty"`
	OnConflict  *OnConflict       `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
	MD5         string            `hcl:"md5,optional" json:"md5,omitempty"`
	Caller      Caller            `json:"-"`
	AbsPath     string            `json:"-"`
	Ext         string            `json:"-"`
}

func (r RemoteFile) ToString() string {
	return fmt.Sprintf(`RemoteFile
┠ ID (string)         = %s
┠ SourceType (string) = %s
┠ Source (string)     = %s
┠ Destination(string) = %s
┠ Vars (map)
%s
┠ Tags (map)
%s
┠ Template(bool)      = %t
┠ Perms(string)       = %s
┠ Disabled(bool)      = %t
┠ MD5(string)         = %s
┠ AbsPath(string)     = %s
┗ Ext (string)        = %s
`,
		r.ID,
		r.SourceType,
		r.Source,
		r.Destination,
		formatter.FormatStringMap(r.Vars),
		formatter.FormatStringMap(r.Tags),
		r.Template,
		r.Perms,
		r.Disabled,
		r.MD5,
		r.AbsPath,
		r.Ext)
}

// We have no children on a RemoteFile, so nothing to iterate on, we'll just return
func (r RemoteFile) Iter() ([]formatter.Formatable, error) {
	return []formatter.Formatable{}, nil
}

// Hash implements the Hasher interface
func (r *RemoteFile) Hash() uint64 {
	return xxhash.Sum64String(
		fmt.Sprintf(
			"sourcetype=%v destination=%v vars=%v template=%v perms=%v disabled=%v source=%v",
			r.SourceType,
			r.Destination,
			r.Vars,
			r.Template,
			r.Perms,
			r.Disabled,
			r.ResourceHash(),
		),
	)
}

// Path implements the Pather interface
func (r *RemoteFile) Path() string {
	return r.ID
}

// Base implements the Pather interface
func (r *RemoteFile) Base() string {
	return path.Base(r.ID)
}

// ValidatePath implements the Pather interface
func (r *RemoteFile) ValidatePath() error {
	if err := ValidateGenericPath(r.Path()); err != nil {
		return err
	}
	if topdir := strings.Split(r.Path(), `/`); topdir[1] != "files" {
		return fmt.Errorf("path %s is not rooted in /%s", r.Path(), topdir[1])
	}
	return nil
}

// ResourceHash implements the ResourceHasher interface
func (r *RemoteFile) ResourceHash() uint64 {
	dep, err := ioutil.ReadFile(r.AbsPath)
	if err != nil {
		fmt.Printf("dependency error for %s: %s could not be read: %v", r.Path(), r.AbsPath, err)
		return 666
	}
	return xxhash.Sum64(dep)
}

// GetCaller implements the Mergeable interface
func (r *RemoteFile) GetCaller() Caller {
	return r.Caller
}

// LaforgeID implements the Mergeable interface
func (r *RemoteFile) LaforgeID() string {
	return r.ID
}

// Fullpath implements the Pather interface
func (r *RemoteFile) Fullpath() string {
	return r.LaforgeID()
}

// ParentLaforgeID implements the Dependency interface
func (r *RemoteFile) ParentLaforgeID() string {
	return r.Path()
}

// Gather implements the Dependency interface
func (r *RemoteFile) Gather(g *Snapshot) error {
	return nil
}

// GetOnConflict implements the Mergeable interface
func (r *RemoteFile) GetOnConflict() OnConflict {
	if r.OnConflict == nil {
		return OnConflict{
			Do: "default",
		}
	}
	return *r.OnConflict
}

// SetCaller implements the Mergeable interface
func (r *RemoteFile) SetCaller(c Caller) {
	r.Caller = c
}

// SetOnConflict implements the Mergeable interface
func (r *RemoteFile) SetOnConflict(o OnConflict) {
	r.OnConflict = &o
}

// Kind implements the Provisioner interface
func (r *RemoteFile) Kind() string {
	return "remote_file"
}

// Swap implements the Mergeable interface
func (r *RemoteFile) Swap(m Mergeable) error {
	rawVal, ok := m.(*RemoteFile)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", r, m)
	}
	*r = *rawVal
	return nil
}

// ResolveSource attempts to locate the referenced source file with a laforge base configuration
func (r *RemoteFile) ResolveSource(base *Laforge, pr *PathResolver, caller CallFile) error {
	if r.Source == "" {
		return nil
	}
	if r.SourceType != "" && r.SourceType != "local" {
		return nil
	}
	cwd, _ := os.Getwd()
	testSrc := r.Source
	if !filepath.IsAbs(r.Source) {
		testSrc = filepath.Join(caller.CallerDir, r.Source)
	}
	if !PathExists(testSrc) {
		pr.Unresolved[r.Source] = true
		return errors.Wrapf(ErrAbsPathDeclNotExist, "caller=%s path=%s", caller.CallerFile, r.Source)
	}
	rel, _ := filepath.Rel(cwd, testSrc)
	rel2, _ := filepath.Rel(caller.CallerDir, testSrc)
	lfr := &LocalFileRef{
		Base:          filepath.Base(testSrc),
		AbsPath:       testSrc,
		RelPath:       rel,
		Cwd:           cwd,
		DeclaredPath:  r.Source,
		RelToCallFile: rel2,
	}
	r.AbsPath = testSrc
	pr.Mapping[r.Source] = lfr
	return nil
}

// MD5Sum returns the MD5 checksum of a local file
func (r *RemoteFile) MD5Sum() (string, error) {
	f, err := os.Open(r.AbsPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CopyTo copies the local file to another location on the local machine
func (r *RemoteFile) CopyTo(dst string) error {
	in, err := os.Open(r.AbsPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// AssetName returns the asset's name calculated as intended
func (r *RemoteFile) AssetName() (string, error) {
	if r.AbsPath == "" {
		return "", errors.New("no absolute path determined")
	}

	if r.MD5 == "" {
		cs, err := r.MD5Sum()
		if err != nil {
			return "", err
		}

		r.MD5 = cs
	}

	if r.Ext == "" {
		r.Ext = filepath.Ext(r.AbsPath)
	}

	return fmt.Sprintf("%s%s", r.MD5, r.Ext), nil

}
