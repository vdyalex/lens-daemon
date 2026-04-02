// Package paths provides canonical file path construction for daemon runtime files.
package paths

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vdyalex/lens-daemon/src/utils/buildinfo"
)

// DaemonPath returns the canonical daemon file path: $TMPDIR/<binary>-<uid>.<extension>.
// The binary name prefix is set at compile time via buildinfo.BinaryName.
// Falls back to "unknown" if the current user UID cannot be determined.
// extension should not include a leading dot (e.g., "pid", "sock").
func DaemonPath(extension string) string {
	temporary := os.TempDir()
	uid := "unknown"
	if u, err := user.Current(); err == nil {
		uid = u.Uid
	}
	return filepath.Join(temporary, fmt.Sprintf("%s-%s.%s", buildinfo.BinaryName, uid, extension))
}
