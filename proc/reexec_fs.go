package proc

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func setRootFS(newRoot string, readOnly bool) error {
	putold := filepath.Join(newRoot, "/.temp_old")
	// Mount proc as we are in our own fs/pid namespace.
	os.Mkdir(path.Join(newRoot, "proc"), 0755)
	if err := syscall.Mount("proc", path.Join(newRoot, "proc"), "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %v", err)
	}

	// Do the pivot_root dance to switch over to our FS view.
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount failed: %v", err)
	}

	if err := os.MkdirAll(putold, 0700); err != nil {
		return fmt.Errorf("mkdir failed: %v", err)
	}

	if err := syscall.PivotRoot(newRoot, putold); err != nil {
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
