package proc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const bindROFlags = syscall.MS_BIND | syscall.MS_REC | syscall.MS_SLAVE | syscall.MS_RDONLY

type fs interface {
	Close() error
	Root() string
	EnsurePatched(cmd procCommand) procResp
}

// overlayLayout encapsulates the configuration of directories and bind mounts which
// overlayFS will use.
type overlayLayout struct {
	base  string
	binds []string
}

func (l overlayLayout) Close() error {
	for i := len(l.binds) - 1; i >= 0; i-- {
		p := l.binds[i]
		if err := syscall.Unmount(p, syscall.MNT_DETACH); err != nil {
			return fmt.Errorf("umount %q: %v", p, err)
		}
	}
	if err := syscall.Unmount(l.LowerBindPath(), syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount lower bind: %v", err)
	}
	return nil
}

// LowerBindPath returns the absolute path to the read-only bind mount,
// which maps to the system root ('/') path.
func (l overlayLayout) LowerBindPath() string {
	return filepath.Join(l.base, "l")
}

// OverlayMountPath returns the absolute path to where overlayfs is mounted.
// This path fuses access to LowerBindPath() and OverlayUpperPath().
func (l overlayLayout) OverlayMountPath() string {
	return filepath.Join(l.base, "top")
}

// OverlayUpperPath returns the absolute path to where the overlay commits
// new/modified files.
func (l overlayLayout) OverlayUpperPath() string {
	return filepath.Join(l.base, "u")
}

// OverlayWorkingPath returns the absolute path to where the overlay stages
// in-progress writes.
func (l overlayLayout) OverlayWorkingPath() string {
	return filepath.Join(l.base, "work")
}

// DevPath returns the absolute path to where the /dev special files are setup.
func (l overlayLayout) DevPath() string {
	return filepath.Join(l.base, "dev")
}

// RootPath returns the absolute path to the root fs tree for child processes.
func (l overlayLayout) RootPath() string {
	return filepath.Join(l.base, "root")
}

func (l overlayLayout) makeOpaque(p string) error {
	if err := ioutil.WriteFile(filepath.Join(l.OverlayUpperPath(), p, ".wh..wh..opq"), nil, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	return unix.Setxattr(filepath.Join(l.OverlayUpperPath(), p), "user.fuseoverlayfs.opaque", []byte{'y'}, unix.XATTR_CREATE)
	// unix.Setxattr(filepath.Join(l.OverlayUpperPath(), p), "user.overlay.opaque", []byte{'y'}, unix.XATTR_CREATE)
}

func (l overlayLayout) setupDevLayout() error {
	if err := os.Mkdir(l.DevPath(), 0755); err != nil {
		return err
	}
	if err := syscall.Mount("", l.DevPath(), "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_STRICTATIME, "mode=755,size=65536k"); err != nil {
		return err
	}
	l.binds = append(l.binds, l.DevPath())

	if err := ioutil.WriteFile(filepath.Join(l.DevPath(), "null"), nil, 0666); err != nil {
		return err
	}
	if err := syscall.Mount("/dev/null", filepath.Join(l.DevPath(), "null"), "", syscall.MS_BIND|syscall.MS_REC|syscall.MS_SLAVE, ""); err != nil {
		return fmt.Errorf("mounting /dev/null: %v", err)
	}
	l.binds = append(l.binds, filepath.Join(l.DevPath(), "null"))
	return nil
}

func (l overlayLayout) Setup() error {
	if err := os.Mkdir(l.LowerBindPath(), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(l.OverlayUpperPath(), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(l.OverlayUpperPath(), "tmp"), 0755); err != nil {
		return err
	}
	l.makeOpaque("/tmp") // best effort

	if err := os.Mkdir(l.OverlayWorkingPath(), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(l.OverlayMountPath(), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(l.RootPath(), 0755); err != nil {
		return err
	}
	if err := syscall.Mount("", l.RootPath(), "tmpfs", 0, "mode=777,size=65536k"); err != nil {
		return err
	}
	l.binds = append(l.binds, l.RootPath())

	if err := l.setupDevLayout(); err != nil {
		return fmt.Errorf("dev: %v", err)
	}

	if err := syscall.Mount("/", l.LowerBindPath(), "", bindROFlags, ""); err != nil {
		return err
	}
	return nil
}

// SetupRootBinds wires the top-level folder hierarchy to the correct place
// on the system. This is usually to the top of the fuse-overlayfs mount,
// a specially-created and adjacent directory, or to the host system.
func (l overlayLayout) SetupRootBinds() (err error) {
	var rootFiles []os.FileInfo
	if rootFiles, err = ioutil.ReadDir(l.OverlayMountPath()); err != nil {
		return err
	}
	for _, f := range rootFiles {
		// We can ignore non-directories unless they are symlinks.
		if !f.IsDir() {
			var target string
			if target, err = os.Readlink("/" + f.Name()); err == nil {
				if err = os.Symlink(target, filepath.Join(l.RootPath(), f.Name())); err != nil {
					return err
				}
			}
			continue
		}

		n := filepath.Base(f.Name())
		src, dest := "", filepath.Join(l.RootPath(), n)
		switch n {
		case "dev":
			src = l.DevPath()
		case "proc", "boot", "lost+found":
			continue
		default:
			src = filepath.Join(l.OverlayMountPath(), n)
		}

		if err := os.Mkdir(dest, 0777); err != nil && !os.IsExist(err) {
			return fmt.Errorf("mkdir %q: %v", n, err)
		}
		if err = syscall.Mount(src, dest, "", syscall.MS_BIND|syscall.MS_REC|syscall.MS_SLAVE, ""); err != nil {
			return fmt.Errorf("mounting %q: %v", n, err)
		}
		defer func(dest string) {
			if err != nil {
				syscall.Unmount(dest, syscall.MNT_DETACH)
			}
		}(dest)
		l.binds = append(l.binds, dest)
	}

	return nil
}

type overlayFS struct {
	layout overlayLayout
	proc   *exec.Cmd
}

// EnsurePatched makes sure a top-level directory or file is patched
// through into the isolated environment.
func (fs *overlayFS) EnsurePatched(cmd procCommand) procResp {
	out := procResp{Code: cmd.Code}
	src, dest := filepath.Join(fs.layout.OverlayUpperPath(), cmd.Dir), filepath.Join(fs.layout.RootPath(), cmd.Dir)
	s, err := os.Stat(filepath.Join(fs.layout.OverlayUpperPath(), cmd.Dir))
	if err != nil {
		out.Error = err.Error()
		return out
	}

	if s.IsDir() {
		if err := os.Mkdir(dest, s.Mode()); err != nil {
			out.Error = err.Error()
			return out
		}
	} else {
		if err := ioutil.WriteFile(dest, nil, s.Mode()); err != nil {
			out.Error = err.Error()
			return out
		}
	}

	if err := syscall.Mount(src, dest, "", syscall.MS_BIND|syscall.MS_REC|syscall.MS_SLAVE, ""); err != nil {
		out.Error = err.Error()
		return out
	}
	fs.layout.binds = append(fs.layout.binds, dest)
	return out
}

// Root returns the path that isolated processes should use as their
// filesystem root.
func (fs *overlayFS) Root() string {
	return fs.layout.RootPath()
}

func (fs *overlayFS) Close() error {
	if err := fs.proc.Process.Kill(); err != nil {
		return err
	}
	return fs.layout.Close()
}

func setupEnvFS(baseDir string) (outFS fs, err error) {
	l := overlayLayout{base: baseDir}
	if err := l.Setup(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			l.Close()
		}
	}()

	out := &overlayFS{
		layout: l,
		proc: exec.Command("fuse-overlayfs",
			"-o", "upperdir="+l.OverlayUpperPath(),
			"-o", "lowerdir="+l.LowerBindPath(),
			"-o", "workdir="+l.OverlayWorkingPath(),
			l.OverlayMountPath()),
	}

	if err = out.proc.Start(); err != nil {
		return nil, fmt.Errorf("overlay: %v", err)
	}
	defer func() {
		if err != nil && out.proc.Process != nil {
			out.proc.Process.Kill()
		}
	}()

	timeout, checkTick := time.NewTimer(2500*time.Millisecond), time.NewTicker(20*time.Millisecond)
	defer checkTick.Stop()
startupSpinLoop:
	for {
		select {
		case <-timeout.C:
			return nil, errors.New("timeout while waiting for overlay to come up")
		case <-checkTick.C:
			var d []byte
			if d, err = ioutil.ReadFile("/proc/mounts"); err != nil {
				return nil, err
			}
			for _, line := range strings.Split(string(d), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "fuse-overlayfs") && strings.Contains(line, l.OverlayMountPath()) {
					break startupSpinLoop
				}
			}
		}
	}

	if err := l.SetupRootBinds(); err != nil {
		return nil, err
	}
	return out, nil
}
