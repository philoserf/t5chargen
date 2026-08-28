package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestVersionReportsWhatARecordStamps verifies that the version output
// carries every provenance string a character record does, so a bug
// report quoting it can be matched against any record the binary wrote.
//
// Asserted against the constants rather than against literals: a version
// bump must not be able to move the record and leave this command
// reporting the old one.
func TestVersionReportsWhatARecordStamps(t *testing.T) {
	for _, invocation := range [][]string{{"version"}, {"--version"}, {"-version"}} {
		t.Run(strings.Join(invocation, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if code := run(invocation, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}

			for _, want := range []string{
				chargen.SchemaVersion,
				chargen.EngineVersion,
				chargen.PolicyVersion,
				chargen.Ruleset,
				buildVersion(),
			} {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("output omits %q:\n%s", want, stdout.String())
				}
			}

			if stderr.Len() != 0 {
				t.Errorf("wrote to stderr: %s", stderr.String())
			}
		})
	}
}

// TestVersionFromNamesEveryBuild verifies the fallback. A binary that
// cannot name its own build must say so rather than print a blank line:
// an empty version in a bug report is indistinguishable from one the
// reporter forgot to include.
//
// Driven through versionFrom rather than buildVersion, because a test
// binary carries build info of its own — asserting the fallback against
// the real reader passes without ever reaching it, which is what the
// first version of this test did.
func TestVersionFromNamesEveryBuild(t *testing.T) {
	for _, tt := range []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{name: "a tagged install", info: mainVersion("v0.1.0-alpha.1"), ok: true, want: "v0.1.0-alpha.1"},
		{name: "a build from the tree", info: mainVersion("v0.0.0-2026+dirty"), ok: true, want: "v0.0.0-2026+dirty"},
		{name: "build info unavailable", info: nil, ok: false, want: devel},
		{name: "build info without a version", info: mainVersion(""), ok: true, want: devel},
		{name: "ok but no info at all", info: nil, ok: true, want: devel},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFrom(tt.info, tt.ok); got != tt.want {
				t.Errorf("versionFrom = %q, want %q", got, tt.want)
			}
		})
	}
}

// mainVersion builds the only part of the build info this reads.
func mainVersion(version string) *debug.BuildInfo {
	return &debug.BuildInfo{Main: debug.Module{Version: version}}
}

// TestVersionIsListedInTheUsage verifies the usage string offers the
// subcommand. A command a caller cannot discover is one they will not
// use, and the whole point of this one is that a stranger filing a bug
// reaches it without being told.
func TestVersionIsListedInTheUsage(t *testing.T) {
	if !strings.Contains(usage, "version") {
		t.Errorf("usage does not mention version:\n%s", usage)
	}
}
