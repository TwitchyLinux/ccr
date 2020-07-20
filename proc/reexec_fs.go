package proc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type fs interface {
	Close() error
	Root() string
}

type overlayFS struct {
	base    string
	overlay *exec.Cmd
}

func (fs *overlayFS) Root() string {
	return filepath.Join(fs.base, "top")
}

func (fs *overlayFS) Close() error {
	if err := fs.overlay.Process.Kill(); err != nil {
		return err
	}
	if err := syscall.Unmount(filepath.Join(fs.base, "l"), syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed unmounting lower bind: %v", err)
	}
	return nil
}

func setupWriteableFS(baseDir string) (fs, error) {
	l, u, work, top := filepath.Join(baseDir, "l"), filepath.Join(baseDir, "u"), filepath.Join(baseDir, "work"), filepath.Join(baseDir, "top")
	if err := os.Mkdir(l, 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(u, 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(u, "tmp"), 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(work, 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(top, 0755); err != nil {
		return nil, err
	}

	if err := syscall.Mount("/", l, "", syscall.MS_BIND|syscall.MS_REC|syscall.MS_SLAVE|syscall.MS_RDONLY, ""); err != nil {
		return nil, err
	}

	out := &overlayFS{base: baseDir}
	out.overlay = exec.Command("fuse-overlayfs", "-o", "upperdir="+u, "-o", "lowerdir="+l, "-o", "workdir="+work, top)
	if err := out.overlay.Start(); err != nil {
		syscall.Unmount(l, syscall.MNT_DETACH)
		return nil, fmt.Errorf("overlay: %v", err)
	}

	for i := 0; i < 100; i++ {
		time.Sleep(15 * time.Millisecond)
		d, err := ioutil.ReadFile("/proc/mounts")
		if err != nil {
			out.overlay.Process.Kill()
			syscall.Unmount(l, syscall.MNT_DETACH)
			return nil, err
		}
		for _, line := range strings.Split(string(d), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "fuse-overlayfs") && strings.Contains(line, top) {
				return out, nil // Its ready to go!
			}
		}
	}

	// Timeout!
	out.overlay.Process.Kill()
	syscall.Unmount(l, syscall.MNT_DETACH)
	return nil, errors.New("timeout while waiting for overlay to come up")
}

func setRootFS(newroot string, readOnly bool) error {
	base := filepath.Dir(newroot)
	ourBase, err := ioutil.TempDir(base, "")
	if err != nil {
		return fmt.Errorf("creating base dir %s: %v", base, err)
	}

	if err := syscall.Mount(newroot, ourBase, "", syscall.MS_BIND|syscall.MS_REC|syscall.MS_SLAVE, ""); err != nil {
		return fmt.Errorf("bind mount: %v", err)
	}

	putold := filepath.Join(ourBase, "/.temp_old")
	os.Mkdir(path.Join(ourBase, "proc"), 0755)
	if err := syscall.Mount("proc", path.Join(ourBase, "proc"), "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %v", err)
	}

	if err := syscall.Mount(ourBase, ourBase, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount failed: %v", err)
	}

	if err := os.MkdirAll(putold, 0700); err != nil {
		return fmt.Errorf("mkdir failed: %v", err)
	}
	if err := syscall.PivotRoot(ourBase, putold); err != nil {
		return fmt.Errorf("pivot root failed: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir failed: %v", err)
	}

	if err := syscall.Unmount("/.temp_old", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount failed: %v", err)
	}

	if readOnly {
		if err := syscall.Mount("/", "/", "", syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("bind ro-remount failed: %v", err)
		}
	}

	return nil
}
