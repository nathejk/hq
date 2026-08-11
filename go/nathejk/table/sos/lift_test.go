package sos

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// This package is written to shared-go's guidelines so that lifting it to
// shared-go/tables/sos (roadmap task 055) is a file move rather than a rewrite.
// Exactly one thing decides which of those it will be: whether anything here
// imports nathejk.dk/... — and nothing in the build complains if it does, so the
// discipline rots silently. Hence a test.
//
// The local precedent runs the other way: table/year/commands.go,
// table/checkgroup/commands.go and table/checkpoint/command.go all reach into
// nathejk.dk/internal/requestctx for the acting user, which is precisely what
// makes those packages awkward to move. Here the actor is passed in by the
// handler instead (cmd/api/actor.go).
//
// If this test ever fails, the fix is almost never "add the import to an
// allowlist" — it is to take what is needed as an argument or as a port declared
// in this package (see shared-go/tables/interfaces.go for that pattern).
const forbiddenPrefix = "nathejk.dk/"

func TestPackageDoesNotImportApplicationCode(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		// Test files are not lifted, so they may import whatever they need. Only
		// the shipped files have to stay clean.
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		checked++

		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(path, forbiddenPrefix) {
				t.Errorf("%s imports %q: this package must not depend on application code, "+
					"or lifting it to shared-go/tables/sos stops being a file move. "+
					"Take what you need as an argument or declare a port in this package.",
					name, path)
			}
		}
	}

	// Guard against the guard passing because it found nothing: a rename or a move
	// that left this test looking at an empty directory would otherwise be reported
	// as success.
	if checked == 0 {
		t.Fatal("no non-test .go files found — this check is not looking at the package")
	}
}
