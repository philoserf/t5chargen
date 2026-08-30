package render_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
)

// FuzzRenderMalformedRecord holds the two renderers to what a beta tester
// can do to them: hand them a record that has been edited, truncated, or
// written by a version that no longer exists.
//
// Neither may panic. render is the last thing standing between a damaged
// file and the terminal, and a crash there loses the character the user
// was trying to look at — which is the record the whole provenance
// contract exists to preserve.
//
// Both must also produce valid UTF-8, because the output is written to a
// file and read back by people; a renderer that echoes an invalid byte
// out of a damaged record spreads the damage.
//
// The seed corpus is deliberately small records rather than the golden
// fixtures. The goldens are the obvious choice and the wrong one: at
// ~220KB each they reduced the engine to eighteen executions in twenty
// seconds, because mutating and tracking coverage on an input that size
// costs more than the render it is testing. docs/character.minimal.json
// is 732 bytes and reaches the same code, so the fuzzer actually runs.
func FuzzRenderMalformedRecord(f *testing.F) {
	minimal, err := os.ReadFile(filepath.Join("..", "docs", "character.minimal.json"))
	if err != nil {
		f.Fatal(err)
	}

	f.Add(minimal)

	for _, seed := range []string{
		`{}`,
		`{"characteristics":{"str":-2147483648,"soc":2147483647}}`,
		`{"events":[{"seq":1,"kind":"consequence","consequence":{"cause":1,"kind":"skill_awarded","skill":"Admin"}}]}`,
		`{"events":[{"seq":1,"kind":"throw","throw":{"expr":"2D","total":7}}]}`,
		`{"events":[{"seq":1,"kind":"choice","choice":{"prompt":"p","options":["a"],"chosen":9}}]}`,
		`{"careers":[{"career":"Scout","medals":[{"code":"MCG","count":2147483647}]}]}`,
		`{"skills":[{"name":"\ud800","level":1}]}`,
		`{"name":"\u0000"}`,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		var c chargen.Character

		if json.Unmarshal(body, &c) != nil {
			return
		}

		for name, got := range map[string]string{
			"Sheet":   render.Sheet(c),
			"History": render.History(c),
		} {
			if !utf8.ValidString(got) {
				t.Fatalf("%s emitted invalid UTF-8 for a record that decoded", name)
			}
		}
	})
}
