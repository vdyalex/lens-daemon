package paths_test

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/vdyalex/lens-daemon/src/utils/paths"
)

func TestDaemonPath_format(t *testing.T) {
	t.Parallel()

	extensions := []string{"pid", "sock", "log"}
	for _, extension := range extensions {
		t.Run(extension, func(t *testing.T) {
			t.Parallel()

			result := paths.DaemonPath(extension)

			u, err := user.Current()
			if err != nil {
				t.Fatalf("user.Current() failed: %v", err)
			}

			expected := filepath.Join(os.TempDir(), fmt.Sprintf("lensd-%s.%s", u.Uid, extension))
			if result != expected {
				t.Errorf("DaemonPath(%q) = %q, want %q", extension, result, expected)
			}
		})
	}
}

func TestDaemonPath_differentExtensions(t *testing.T) {
	t.Parallel()

	pid := paths.DaemonPath("pid")
	sock := paths.DaemonPath("sock")

	if pid == sock {
		t.Errorf("DaemonPath(pid) and DaemonPath(sock) should differ, both = %q", pid)
	}
}
