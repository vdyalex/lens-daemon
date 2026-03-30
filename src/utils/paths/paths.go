// Package paths provides canonical file path construction for daemon runtime files.
package paths

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// DaemonPath returns the canonical daemon file path: $TMPDIR/lensd-<uid>.<extension>.
// Falls back to "unknown" if the current user UID cannot be determined.
// extension should not include a leading dot (e.g., "pid", "sock").
func DaemonPath(extension string) string {
	temporary := os.TempDir()
	uid := "unknown"
	if u, err := user.Current(); err == nil {
		uid = u.Uid
	}
	return filepath.Join(temporary, fmt.Sprintf("lensd-%s.%s", uid, extension))
}
