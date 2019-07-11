package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gen0cide/laforge/provisioner"

	"github.com/gen0cide/laforge/core/cli"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/juju/utils/filepath"
	"golang.org/x/crypto/ssh"

	"github.com/cespare/xxhash"
	"github.com/gen0cide/winrmcp/winrmcp"
	"github.com/shiena/ansicolor"

	"github.com/pkg/errors"
)

// Connection defines an access method provisioned host within a team's environment
//easyjson:json
type Connection struct {
	ID                 string              `hcl:"id,label" json:"id,omitempty"`
	Active             bool                `hcl:"active,attr" json:"active,omitempty"`
	LocalAddr          string              `hcl:"local_addr,attr" json:"local_addr,omitempty"`
	RemoteAddr         string              `hcl:"remote_addr,attr" json:"remote_addr,omitempty"`
	ResourceName       string              `hcl:"resource_name,attr" json:"resource_name,omitempty"`
	SSHAuthConfig      *SSHAuthConfig      `hcl:"ssh,block" json:"ssh,omitempty"`
	WinRMAuthConfig    *WinRMAuthConfig    `hcl:"winrm,block" json:"winrm,omitempty"`
	Revision           int64               `hcl:"revision,optional" json:"revision,omitempty"`
	OnConflict         *OnConflict         `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
	Competition        *Competition        `json:"-"`
	Environment        *Environment        `json:"-"`
	Build              *Build              `json:"-"`
	Team               *Team               `json:"-"`
	Network            *Network            `json:"-"`
	Host               *Host               `json:"-"`
	ProvisionedHost    *ProvisionedHost    `json:"-"`
	ProvisionedNetwork *ProvisionedNetwork `json:"-"`
	Caller             Caller              `json:"-"`
}

// Hash implements the Hasher interface
func (c *Connection) Hash() uint64 {
	sshc := uint64(666)
	wrmc := uint64(666)
	if c.IsWinRM() {
		wrmc = c.WinRMAuthConfig.Hash()
	}
	if c.IsSSH() {
		sshc = c.SSHAuthConfig.Hash()
	}
	return xxhash.Sum64String(
		fmt.Sprintf(
			"id=%v localaddr=%v rmaddr=%v rname=%v sshc=%v wrmc=%v",
			c.ID,
			c.LocalAddr,
			c.RemoteAddr,
			c.ResourceName,
			sshc,
			wrmc,
		),
	)
}

// Path implements the Pather interface
func (c *Connection) Path() string {
	return c.ID
}

// Base implements the Pather interface
func (c *Connection) Base() string {
	return path.Base(c.ID)
}

// ValidatePath implements the Pather interface
func (c *Connection) ValidatePath() error {
	if err := ValidateGenericPath(c.Path()); err != nil {
		return err
	}
	return nil
}

// IsSSH is a convenience method for checking if the provisioned host is setup for remote SSH
func (c *Connection) IsSSH() bool {
	return c.SSHAuthConfig != nil
}

// ParentLaforgeID returns connections parent provisioned host
func (c *Connection) ParentLaforgeID() string {
	return path.Dir(c.Path())
}

// IsWinRM is a convenience method for checking if the provisioned host is setup for remote WinRM
func (c *Connection) IsWinRM() bool {
	return c.WinRMAuthConfig != nil
}

// GetCaller implements the Mergeable interface
func (c *Connection) GetCaller() Caller {
	return c.Caller
}

// LaforgeID implements the Mergeable interface
func (c *Connection) LaforgeID() string {
	return c.ID
}

// GetOnConflict implements the Mergeable interface
func (c *Connection) GetOnConflict() OnConflict {
	if c.OnConflict == nil {
		return OnConflict{
			Do: "default",
		}
	}
	return *c.OnConflict
}

// SetCaller implements the Mergeable interface
func (c *Connection) SetCaller(ca Caller) {
	c.Caller = ca
}

// SetOnConflict implements the Mergeable interface
func (c *Connection) SetOnConflict(o OnConflict) {
	c.OnConflict = &o
}

// Swap implements the Mergeable interface
func (c *Connection) Swap(m Mergeable) error {
	rawVal, ok := m.(*Connection)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", c, m)
	}
	*c = *rawVal
	return nil
}

// SetID increments the revision and sets the ID if needed
func (c *Connection) SetID() string {
	if c.ID == "" {
		c.ID = path.Join(c.ProvisionedHost.LaforgeID(), "conn")
	}
	return c.ID
}

// RemoteShell connects your local console to a remote provisioned host
func (c *Connection) RemoteShell() error {
	if c.IsWinRM() {
		return c.InteractiveWinRM()
	}
	return c.InteractiveSSH()
}

// Upload uploads a src file/dir to a dst file/dir on the provisioned host
func (c *Connection) Upload(src, dst string) error {
	if c.IsWinRM() {
		return c.UploadWinRM(src, dst)
	}
	return c.UploadSFTP(src, dst)
}

// ExecuteCommand is the generic interface for a connection to execute a command on the remote system
func (c *Connection) ExecuteCommand(cmd *RemoteCommand) error {
	if c.IsWinRM() {
		return c.ExecuteCommandWinRM(cmd)
	}
	return c.ExecuteCommandSSH(cmd)
}

func (c *Connection) UploadExecuteAndDeleteV2(j Doer, scriptsrc string, tmpname string, logdir string) error {
	if _, err := os.Stat(scriptsrc); err != nil {
		return fmt.Errorf("problem locating file %s: %v", scriptsrc, err)
	}
	if _, err := os.Stat(logdir); err != nil {
		return fmt.Errorf("problem locating logdir %s: %v", logdir, err)
	}

	winfp, err := filepath.NewRenderer("windows")
	if err != nil {
		return err
	}

	currfp, err := filepath.NewRenderer("")
	if err != nil {
		return err
	}

	filename := currfp.Base(scriptsrc)
	if tmpname != "" {
		filename = tmpname
	}

	logfilename := strings.Replace(filename, currfp.Ext(filename), ``, -1)
	logprefix := currfp.Join(logdir, logfilename)
	stdoutfile := fmt.Sprintf("%s.stdout.log", logprefix)
	stderrfile := fmt.Sprintf("%s.stderr.log", logprefix)

	stdoutdone := make(chan struct{})
	stderrdone := make(chan struct{})

	debugstdoutpr, debugstdoutpw := io.Pipe()
	debugstderrpr, debugstderrpw := io.Pipe()

	wg := new(sync.WaitGroup)
	stdoutScanner := bufio.NewScanner(debugstdoutpr)
	stderrScanner := bufio.NewScanner(debugstderrpr)
	wg.Add(2)

	go func() {
		defer wg.Done()
		for stdoutScanner.Scan() {
			text := stdoutScanner.Text()
			j.StandardOutput(text)
		}
		stdoutdone <- struct{}{}
	}()

	go func() {
		defer wg.Done()
		for stderrScanner.Scan() {
			text := stderrScanner.Text()
			j.StandardError(text)
		}
		stderrdone <- struct{}{}
	}()

	defer func() {
		<-stdoutdone
		<-stderrdone
		wg.Wait()
		err := stdoutScanner.Err()
		if err != nil {
			cli.Logger.Errorf("Debug STDOUT Scanner Error for %s: %v", j.GetTargetID(), err)
		}
		err = stderrScanner.Err()
		if err != nil {
			cli.Logger.Errorf("Debug STDERR Scanner Error for %s: %v", j.GetTargetID(), err)
		}
	}()

	commcfg := &communicator.Config{}
	var svccfg interface{}

	if c.IsWinRM() {
		commcfg.WinRMHost = c.RemoteAddr
		commcfg.WinRMUser = c.WinRMAuthConfig.User
		commcfg.WinRMPassword = c.WinRMAuthConfig.Password
		commcfg.WinRMPort = c.WinRMAuthConfig.Port
		commcfg.WinRMTimeout = time.Duration(12 * time.Minute)
		commcfg.WinRMUseSSL = c.WinRMAuthConfig.HTTPS
		commcfg.WinRMInsecure = c.WinRMAuthConfig.SkipVerify
		if winfp.Ext(scriptsrc) == `.ps1` {
			commcfg.Type = "powershell"
			svccfg = provisioner.WindowsPowershellConfig(scriptsrc, logfilename, 60)
		} else {
			commcfg.Type = "windows-shell"
			svccfg = provisioner.WindowsShellConfig(scriptsrc, logfilename)
		}
	} else {
		commcfg.SSHHost = c.RemoteAddr
		commcfg.SSHUsername = c.SSHAuthConfig.User
		commcfg.SSHPort = c.SSHAuthConfig.Port
		realKeyPath := ""
		if _, err := os.Stat(c.SSHAuthConfig.IdentityFile); err != nil && os.IsNotExist(err) {
			if c.SSHAuthConfig.IdentityFileRef == nil {
				return errors.New("could not locate SSH private key for authentication")
			}
			if c.SSHAuthConfig.IdentityFileRef != nil {
				if _, err := os.Stat(c.SSHAuthConfig.IdentityFileRef.AbsPath); err == nil {
					realKeyPath = c.SSHAuthConfig.IdentityFileRef.AbsPath
				}
			}
		} else {
			realKeyPath = c.SSHAuthConfig.IdentityFile
		}
		commcfg.SSHPrivateKeyFile = realKeyPath
		commcfg.Type = "ssh"
		svccfg = provisioner.LinuxShellConfig(scriptsrc, logfilename)
	}

	label := fmt.Sprintf("%s://%s/%s", commcfg.Type, c.ProvisionedHost.Base(), logfilename)
	stderrfh, err := os.OpenFile(stderrfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer stderrfh.Close()
	cli.Logger.Infof("Logging STDERR to %s", stderrfile)
	stdoutfh, err := os.OpenFile(stdoutfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer stdoutfh.Close()
	cli.Logger.Infof("Logging STDOUT to %s", stdoutfile)
	newstdout := io.MultiWriter(debugstdoutpw, stdoutfh)
	newstderr := io.MultiWriter(debugstderrpw, stderrfh)
	defer debugstdoutpw.Close()
	defer debugstderrpw.Close()

	prov, err := provisioner.New(label, commcfg, svccfg, nil, newstdout, newstderr)
	if err != nil {
		return err
	}

	return prov.Provision()
}

// UploadExecuteAndDelete is a helper function to chain together a common pattern of execution
func (c *Connection) UploadExecuteAndDelete(j Doer, scriptsrc string, tmpname string, logdir string) error {
	if _, err := os.Stat(scriptsrc); err != nil {
		return fmt.Errorf("problem locating file %s: %v", scriptsrc, err)
	}
	if _, err := os.Stat(logdir); err != nil {
		return fmt.Errorf("problem locating logdir %s: %v", logdir, err)
	}

	winfp, err := filepath.NewRenderer("windows")
	if err != nil {
		return err
	}
	nixfp, err := filepath.NewRenderer("linux")
	if err != nil {
		return err
	}
	currfp, err := filepath.NewRenderer("")
	if err != nil {
		return err
	}

	filename := currfp.Base(scriptsrc)
	if tmpname != "" {
		filename = tmpname
	}

	logfilename := strings.Replace(filename, currfp.Ext(filename), ``, -1)
	logprefix := currfp.Join(logdir, logfilename)
	stdoutfile := fmt.Sprintf("%s.stdout.log", logprefix)
	stderrfile := fmt.Sprintf("%s.stderr.log", logprefix)

	stdoutdone := make(chan struct{})
	stderrdone := make(chan struct{})

	debugstdoutpr, debugstdoutpw := io.Pipe()
	debugstderrpr, debugstderrpw := io.Pipe()

	wg := new(sync.WaitGroup)
	stdoutScanner := bufio.NewScanner(debugstdoutpr)
	stderrScanner := bufio.NewScanner(debugstderrpr)
	wg.Add(2)

	go func() {
		defer wg.Done()
		for stdoutScanner.Scan() {
			text := stdoutScanner.Text()
			j.StandardOutput(text)
		}
		stdoutdone <- struct{}{}
	}()

	go func() {
		defer wg.Done()
		for stderrScanner.Scan() {
			text := stderrScanner.Text()
			j.StandardError(text)
		}
		stderrdone <- struct{}{}
	}()

	defer func() {
		<-stdoutdone
		<-stderrdone
		wg.Wait()
		err := stdoutScanner.Err()
		if err != nil {
			cli.Logger.Errorf("Debug STDOUT Scanner Error for %s: %v", j.GetTargetID(), err)
		}
		err = stderrScanner.Err()
		if err != nil {
			cli.Logger.Errorf("Debug STDERR Scanner Error for %s: %v", j.GetTargetID(), err)
		}
	}()

	// cli.Logger.Infof("We got here?")

	if c.IsWinRM() {
		finalpath := winfp.Join(`C:`, filename)
		err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
			err = c.UploadWinRM(scriptsrc, finalpath)
			if err != nil {
				cli.Logger.Errorf("%s Upload Connection Issue: %v", c.Path(), err)
				e <- NewTimeoutExtension(err)
				return
			}
			e <- nil
			return
		})
		if err != nil {
			cli.Logger.Errorf("%s Final Upload Issue: %v", c.Path(), err)
			return err
		}
		cli.Logger.Infof("WinRM Upload Complete: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
		err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
			rc := NewRemoteCommand()
			stderrfh, err := os.OpenFile(stderrfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				e <- err
				return
			}
			defer stderrfh.Close()
			cli.Logger.Infof("Logging STDERR to %s", stderrfile)
			stdoutfh, err := os.OpenFile(stdoutfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				e <- err
				return
			}
			defer stdoutfh.Close()
			cli.Logger.Infof("Logging STDOUT to %s", stdoutfile)
			rc.Stdout = io.MultiWriter(debugstdoutpw, stdoutfh)
			rc.Stderr = io.MultiWriter(debugstderrpw, stderrfh)
			defer debugstdoutpw.Close()
			defer debugstderrpw.Close()
			rc.Command = finalpath
			err = c.ExecuteCommandWinRM(rc)
			if err != nil {
				cli.Logger.Errorf("%s Execute Connection Issue: %v", c.Path(), err)
				e <- NewTimeoutExtension(err)
				return
			}
			e <- nil
			return
		})
		if err != nil {
			cli.Logger.Errorf("%s Final Execute Issue: %v", c.Path(), err)
			return err
		}
		cli.Logger.Infof("WinRM Execution Complete: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
		time.Sleep(4 * time.Second)
		err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
			delrc := NewRemoteCommand()
			stderrfh2, err := os.OpenFile(stderrfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				e <- err
				return
			}
			defer stderrfh2.Close()
			stdoutfh2, err := os.OpenFile(stdoutfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				e <- err
				return
			}
			defer stdoutfh2.Close()
			delrc.Stdout = io.MultiWriter(debugstdoutpw, stdoutfh2)
			delrc.Stderr = io.MultiWriter(debugstderrpw, stderrfh2)
			delrc.Command = fmt.Sprintf("del %s", finalpath)
			err = c.ExecuteCommandWinRM(delrc)
			if err != nil {
				cli.Logger.Errorf("%s Delete Script Connection Issue: %v", c.Path(), err)
				e <- NewTimeoutExtension(err)
				return
			}
			e <- nil
			return
		})
		if err != nil {
			cli.Logger.Errorf("%s Final Delete Issue: %v", c.Path(), err)
			return err
		}
		cli.Logger.Infof("WinRM Script Deleted: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
		return nil
	}
	finalpath := nixfp.Join(`/root`, filename)
	err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
		err = c.UploadScriptSFTP(scriptsrc, finalpath)
		if err != nil {
			cli.Logger.Errorf("%s Upload Script Connection Issue: %v", c.Path(), err)
			e <- NewTimeoutExtension(err)
			return
		}
		e <- nil
		return
	})
	if err != nil {
		wmerr, ok := err.(*ssh.ExitError)
		if !ok {
			return err
		}
		if wmerr.Waitmsg.Signal() != "" || wmerr.Waitmsg.Msg() != "" || wmerr.Waitmsg.ExitStatus() != 1 {
			return err
		}
	}
	if c.ProvisionedHost == nil {
		return errors.New("provisioned host was nil")
	}
	if c.ProvisionedHost.Host == nil {
		return errors.New("provisioned host's host was nil")
	}
	cli.Logger.Infof("SFTP Upload Complete: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
	err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
		rc := NewRemoteCommand()
		stdoutfile := fmt.Sprintf("%s.stdout.log", logprefix)
		stderrfile := fmt.Sprintf("%s.stderr.log", logprefix)
		stderrfh, err := os.OpenFile(stderrfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			e <- err
			return
		}
		defer stderrfh.Close()
		cli.Logger.Infof("Logging script STDERR to %s", stderrfile)
		stdoutfh, err := os.OpenFile(stdoutfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			e <- err
			return
		}
		defer stdoutfh.Close()
		cli.Logger.Infof("Logging script STDOUT to %s", stdoutfile)
		rc.Stdout = io.MultiWriter(debugstdoutpw, stdoutfh)
		rc.Stderr = io.MultiWriter(debugstderrpw, stderrfh)
		defer debugstdoutpw.Close()
		defer debugstderrpw.Close()
		rc.Command = finalpath
		err = c.ExecuteCommandSSH(rc)
		if err != nil {
			cli.Logger.Errorf("%s Execute Script Connection Issue: %v", c.Path(), err)
			e <- NewTimeoutExtension(err)
			return
		}
		e <- nil
		return
	})
	if err != nil {
		cli.Logger.Errorf("%s Final Execute Issue: %v", c.Path(), err)
		return err
	}
	cli.Logger.Infof("SSH Execution Complete: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
	err = PerformInTimeout(j.GetTimeout(), func(e chan error) {
		err = c.DeleteScriptSFTP(finalpath)
		if err != nil {
			cli.Logger.Errorf("%s Delete Script Connection Issue: %v", c.Path(), err)
			e <- NewTimeoutExtension(err)
			return
		}
		e <- nil
		return
	})
	if err != nil {
		cli.Logger.Errorf("%s Final Delete Issue: %v", c.Path(), err)
		return err
	}
	cli.Logger.Infof("SFTP Deletion Successful: %s (%s) -> %s", c.ProvisionedHost.Host.Base(), c.RemoteAddr, finalpath)
	return nil
}

// ExecuteCommandWinRM executes a remote command over WinRM
func (c *Connection) ExecuteCommandWinRM(cmd *RemoteCommand) error {
	client := &WinRMClient{}
	err := client.SetConfig(c.WinRMAuthConfig)
	if err != nil {
		return err
	}
	client.SetIO(
		cmd.Stdout,
		cmd.Stderr,
		cmd.Stdin,
	)

	err = client.ExecuteNonInteractive(cmd)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

// ExecuteCommandSSH executes a remote command over SSH
func (c *Connection) ExecuteCommandSSH(cmd *RemoteCommand) error {
	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	err = client.Start(cmd)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

// UploadWinRM uses WinRM to upload src to dst on the provisioned host
func (c *Connection) UploadWinRM(src, dst string) error {
	addr, config := c.WinRMAuthConfig.ToUploadConfig()
	client, err := winrmcp.New(addr, &config)
	if err != nil {
		return err
	}
	return client.Copy(src, dst)
}

// Gather implements the dependency interface
func (c *Connection) Gather(s *Snapshot) error {
	return nil
}

// InteractiveWinRM launches an interactive shell over WinRM
func (c *Connection) InteractiveWinRM() error {
	client := &WinRMClient{}
	err := client.SetConfig(c.WinRMAuthConfig)
	if err != nil {
		return err
	}
	client.SetIO(
		ansicolor.NewAnsiColorWriter(os.Stdout),
		ansicolor.NewAnsiColorWriter(os.Stderr),
		os.Stdin,
	)

	err = client.LaunchInteractiveShell()
	if err != nil {
		return err
	}
	return nil
}

// InteractiveSSH launches an interactive shell over SSH
func (c *Connection) InteractiveSSH() error {
	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	err = client.LaunchInteractiveShell()
	if err != nil {
		return err
	}
	return nil
}

// UploadScriptSFTP uses the really nice golang SFTP client to upload remote files
func (c *Connection) UploadScriptSFTP(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return errors.New("script source cannot be a directory")
	}

	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	err = client.UploadScriptV2(src, dst)
	if err != nil {
		return err
	}
	return nil
}

// UploadSFTP uses the really nice golang SFTP client to upload remote files
func (c *Connection) UploadSFTP(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return errors.New("source file cannot be a directory")
	}

	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	err = client.UploadFileV2(src, dst)
	if err != nil {
		return err
	}
	return nil
}

// DeleteScriptSFTP uses the really nice golang SFTP client to zero and delete remote files
func (c *Connection) DeleteScriptSFTP(remotefile string) error {
	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	err = client.DeleteScriptV2(remotefile)
	if err != nil {
		return err
	}
	return nil
}

// UploadSCP uses scp to upload src to dst on the provisioned host
func (c *Connection) UploadSCP(src, dst string) error {
	isDir := false
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		isDir = true
	}

	client, err := NewSSHClient(c.SSHAuthConfig, "")
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}
	defer client.Disconnect()

	if isDir {
		err = client.UploadDir(dst, src)
		if err != nil {
			return err
		}
		return nil
	}

	fileInput, err := os.Open(src)
	if err != nil {
		return err
	}
	err = client.Upload(dst, fileInput)
	if err != nil {
		return err
	}
	return nil
}
