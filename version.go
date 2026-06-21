package runj

import (
	_ "embed" // use to embed VERSION and REV_OVERRIDE files
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	//go:embed VERSION
	version string
	//go:embed REV_OVERRIDE
	revOverride string

	rendered string
)

func init() {
	rendered = render()
}

// Version returns a version string for runj and its dependencies
func Version() string {
	return rendered
}

func render() string {
	version = strings.TrimSpace(version)
	revOverride = strings.TrimSpace(revOverride)
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	revision := ""
	modified := false
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				modified = true
			}
		}
	}
	if modified {
		revision = revision + "*"
	}
	if revOverride != "" {
		revision = revOverride
	}
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "%s (%s)\n", version, revision)
	sb.WriteString("go: " + bi.GoVersion)
	for _, dep := range bi.Deps {
		fmt.Fprintf(&sb, "\n%s: %s", dep.Path, dep.Version)
	}
	return sb.String()
}
