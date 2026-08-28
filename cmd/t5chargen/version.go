package main

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/philoserf/t5chargen/chargen"
)

// devel is what the toolchain records when a build carries no module
// version and no VCS stamp to derive one from — `go run`, or a build with
// -buildvcs=false. Reporting it plainly is more useful in a bug report
// than an empty field or a version invented at build time.
const devel = "(devel)"

// buildVersion reports the module version this binary was built from.
//
// Read from the build info the toolchain embeds rather than stamped with
// -ldflags: it needs no build plumbing and no dependency, and it cannot
// disagree with what the binary actually is. Three things it reports, all
// of them honest about the build in hand:
//
//   - `go install ...@v0.1.0-alpha.1` reports the tag;
//   - `go build` inside the git tree reports a VCS pseudo-version, with
//     "+dirty" appended when the tree has uncommitted changes — which is
//     exactly what a bug report from a working copy should say;
//   - `go run`, or a build with no VCS information, reports "(devel)".
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()

	return versionFrom(info, ok)
}

// versionFrom is buildVersion's reading of the build info, split out
// because the fallback is otherwise unreachable in a test: a test binary
// carries build info of its own, so a case asserting what happens without
// it passes without ever running it.
func versionFrom(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil || info.Main.Version == "" {
		return devel
	}

	return info.Main.Version
}

// writeVersion prints the build and everything a character record's
// provenance carries, in the layout the character sheet prints it
// (render.provenance). A bug report quoting this can be matched against
// any record the binary wrote: the three versions and the ruleset are the
// same strings the record stamps, and the seed is the only field of that
// line the build cannot know.
func writeVersion(w io.Writer) {
	fmt.Fprintf(w, "t5chargen %s\n\n", buildVersion())
	fmt.Fprintf(w, "schema %s · engine %s · policy %s\n\n",
		chargen.SchemaVersion, chargen.EngineVersion, chargen.PolicyVersion)
	fmt.Fprintf(w, "Ruleset: %s\n", chargen.Ruleset)
}
