package game

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePath_FindsFileWhateverTheCaseOfEachComponent(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "Sub", "File.AIR"))
	got, err := ResolvePath(base, "sub/file.air")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "Sub", "File.AIR"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolvePath_AcceptsBackslashSeparator(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "sub", "File.air"))
	got, err := ResolvePath(base, `sub\file.air`)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "sub", "File.air"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolvePath_RejectsSymlinkedFileAndFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	base := t.TempDir()
	outside := t.TempDir()
	touch(t, filepath.Join(outside, "secret.txt"))
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(base, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "dir")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolvePath(base, "link.txt"); err == nil {
		t.Error("symlinked file was resolved")
	}
	if _, err := ResolvePath(base, "DIR/secret.txt"); err == nil {
		t.Error("file under a symlinked folder was resolved")
	}
}

func TestResolvePath_ErrorsOnMissingFile(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "a.txt"))
	if _, err := ResolvePath(base, "b.txt"); err == nil {
		t.Fatal("missing file resolved")
	}
}

func TestResolvePath_RejectsPathsEscapingTheBase(t *testing.T) {
	parent := t.TempDir()
	base := filepath.Join(parent, "base")
	touch(t, filepath.Join(base, "in.txt"))
	touch(t, filepath.Join(parent, "out.txt"))
	if err := os.MkdirAll(filepath.Join(base, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../out.txt", `..\out.txt`, "sub/../../out.txt", filepath.Join(parent, "out.txt"), "/out.txt"} {
		if got, err := ResolvePath(base, rel); err == nil {
			t.Errorf("%q resolved to %q", rel, got)
		}
	}
}

func TestResolvePath_AllowsDotDotThatStaysInsideTheBase(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "a", "x.txt"))
	if err := os.MkdirAll(filepath.Join(base, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolvePath(base, "b/../A/X.TXT")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "a", "x.txt"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentRoot_IsTheProgramFolder(t *testing.T) {
	exe := filepath.Join("games", "kakutou", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("games", "kakutou"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentRoot_IsTheFolderHoldingTheAppBundleOnMac(t *testing.T) {
	exe := filepath.Join("Applications", "Fight", "Kakutou.APP", "Contents", "MacOS", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("Applications", "Fight"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentRoot_IgnoresMacOSFolderOutsideAnAppBundle(t *testing.T) {
	exe := filepath.Join("x", "Contents", "MacOS", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("x", "Contents", "MacOS"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentFolders_NamesTheMissingFolder(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "chars"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err := ContentFolders(root)
	if err == nil {
		t.Fatal("missing stages folder not reported")
	}
	if want := filepath.Join(root, "stages"); !contains(err.Error(), want) {
		t.Fatalf("message %q does not name %q", err, want)
	}
	if contains(err.Error(), filepath.Join(root, "chars")) {
		t.Fatalf("message %q names the folder that exists", err)
	}
}

func TestContentFolders_NamesBothWhenBothAreMissing(t *testing.T) {
	root := t.TempDir()
	_, _, err := ContentFolders(root)
	if err == nil {
		t.Fatal("missing folders not reported")
	}
	for _, name := range []string{"chars", "stages"} {
		if want := filepath.Join(root, name); !contains(err.Error(), want) {
			t.Errorf("message %q does not name %q", err, want)
		}
	}
}

func TestContentFolders_ReportsAFileStandingWhereAFolderShouldBe(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "chars"))
	if err := os.MkdirAll(filepath.Join(root, "stages"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err := ContentFolders(root)
	if err == nil || !contains(err.Error(), filepath.Join(root, "chars")) {
		t.Fatalf("file named chars not reported: %v", err)
	}
	if contains(err.Error(), filepath.Join(root, "stages")) {
		t.Fatalf("message %q names the folder that exists", err)
	}
}

func TestContentFolders_ReturnsTheOnDiskFolders(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"Chars", "STAGES"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	chars, stages, err := ContentFolders(root)
	if err != nil {
		t.Fatal(err)
	}
	if chars != filepath.Join(root, "Chars") || stages != filepath.Join(root, "STAGES") {
		t.Fatalf("got %q, %q", chars, stages)
	}
}

func TestResolvePath_PrefersTheExactSpellingOverACaseVariant(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "a.txt"))
	if err := os.WriteFile(filepath.Join(base, "A.TXT"), []byte("y"), 0o644); err != nil {
		t.Skip("case-insensitive file system")
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 2 {
		t.Skip("case-insensitive file system")
	}
	for _, name := range []string{"a.txt", "A.TXT"} {
		got, err := ResolvePath(base, name)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(base, name); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestResolvePath_ReturnsTheOnDiskSpellingNotTheCallers(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "Dir", "File.Txt"))
	got, err := ResolvePath(base, "dir/FILE.TXT")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "Dir", "File.Txt"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestJoinResolved_NeverReturnsAnEscapingOrSymlinkedPathBehindAMissingFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	parent := t.TempDir()
	base := filepath.Join(parent, "base")
	touch(t, filepath.Join(parent, "out.txt"))
	touch(t, filepath.Join(parent, "target.txt"))
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(parent, "target.txt"), filepath.Join(base, "link.txt")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"missing/../../out.txt", `missing\..\..\out.txt`, "missing/../link.txt"} {
		if got := joinResolved([]string{base}, rel); got != "" {
			t.Errorf("%q gave openable path %q", rel, got)
		}
	}
}

func TestJoinResolved_MissingFileStillNamesThePathInsideTheBase(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "sub", "gone.sff")
	if got := joinResolved([]string{base}, `missing\..\sub\gone.sff`); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
