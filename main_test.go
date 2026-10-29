package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// setupDir creates the named empty files in a temp dir and makes it the
// working directory for the rest of the test.
func setupDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

// captureStdout runs fn and returns what it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()
	fn()
	w.Close()
	return <-done
}

// runMain feeds input to main as stdin and returns its output.
func runMain(t *testing.T, input string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()
	go func() {
		io.WriteString(w, input)
		w.Close()
	}()
	return captureStdout(t, main)
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name()
	}
	return out
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return names(entries)
}

func sortedEntries(t *testing.T, dir string) (jpg, raw []os.DirEntry) {
	t.Helper()
	jpg, raw = GetGoodEntries(dir)
	slices.SortFunc(jpg, better_sort)
	slices.SortFunc(raw, better_sort)
	return jpg, raw
}

func TestGetGoodEntries(t *testing.T) {
	dir := setupDir(t, "a1.jpg", "a2.dng", "a3.tif", "notes.txt", "a4.png", "a5.JPG")
	jpg, raw := GetGoodEntries(dir)

	if got, want := names(jpg), []string{"a1.jpg"}; !slices.Equal(got, want) {
		t.Errorf("jpg = %v, want %v", got, want)
	}
	if got, want := names(raw), []string{"a2.dng", "a3.tif"}; !slices.Equal(got, want) {
		t.Errorf("raw = %v, want %v", got, want)
	}
}

func TestGetGoodEntriesEmpty(t *testing.T) {
	dir := setupDir(t, "notes.txt")
	jpg, raw := GetGoodEntries(dir)
	if len(jpg) != 0 || len(raw) != 0 {
		t.Errorf("got jpg=%v raw=%v, want both empty", names(jpg), names(raw))
	}
}

func TestBetterSortNumeric(t *testing.T) {
	dir := setupDir(t, "IMG_10.jpg", "IMG_2.jpg", "IMG_1.jpg", "IMG_100.jpg")
	jpg, _ := sortedEntries(t, dir)

	want := []string{"IMG_1.jpg", "IMG_2.jpg", "IMG_10.jpg", "IMG_100.jpg"}
	if got := names(jpg); !slices.Equal(got, want) {
		t.Errorf("sorted = %v, want %v", got, want)
	}
}

func TestBetterSortPanicsWithoutDigits(t *testing.T) {
	dir := setupDir(t, "a.jpg", "b1.jpg")
	jpg, _ := GetGoodEntries(dir)

	defer func() {
		if recover() == nil {
			t.Error("better_sort did not panic on a name with no digits")
		}
	}()
	better_sort(jpg[0], jpg[1])
}

func TestDryDoesNotRename(t *testing.T) {
	dir := setupDir(t, "IMG_1.jpg", "IMG_2.jpg")
	jpg, _ := sortedEntries(t, dir)

	out := captureStdout(t, func() { dry(1, "trip", jpg) })

	want := "IMG_1.jpg -> trip-1.jpg\nIMG_2.jpg -> trip-2.jpg\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"IMG_1.jpg", "IMG_2.jpg"}) {
		t.Errorf("files changed by dry run: %v", got)
	}
}

func TestRealRenames(t *testing.T) {
	dir := setupDir(t, "IMG_3.dng", "IMG_10.dng")
	_, raw := sortedEntries(t, dir)

	var renamed []os.DirEntry
	captureStdout(t, func() { renamed = real(1, "trip", raw) })

	want := []string{"trip-1.dng", "trip-2.dng"}
	if got := names(renamed); !slices.Equal(got, want) {
		t.Errorf("returned = %v, want %v", got, want)
	}
	if got := dirNames(t, dir); !slices.Equal(got, want) {
		t.Errorf("dir = %v, want %v", got, want)
	}
}

func TestRealPadsToWidth(t *testing.T) {
	files := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		files = append(files, fmt.Sprintf("IMG_%d.jpg", i))
	}
	dir := setupDir(t, files...)
	jpg, _ := sortedEntries(t, dir)

	var renamed []os.DirEntry
	captureStdout(t, func() { renamed = real(2, "trip", jpg) })

	if got := renamed[0].Name(); got != "trip-01.jpg" {
		t.Errorf("first = %s, want trip-01.jpg", got)
	}
	if got := renamed[9].Name(); got != "trip-10.jpg" {
		t.Errorf("last = %s, want trip-10.jpg", got)
	}
}

func TestRealThenRevert(t *testing.T) {
	original := []string{"IMG_10.jpg", "IMG_3.jpg"}
	dir := setupDir(t, original...)
	jpg, _ := sortedEntries(t, dir)

	captureStdout(t, func() {
		recent := real(1, "trip", jpg)
		revert(recent, jpg)
	})

	if got := dirNames(t, dir); !slices.Equal(got, original) {
		t.Errorf("after revert dir = %v, want %v", got, original)
	}
}

func TestMainRun(t *testing.T) {
	dir := setupDir(t, "IMG_3.jpg", "IMG_10.jpg", "IMG_3.dng", "IMG_10.dng")
	pattern := filepath.Base(dir)

	runMain(t, "run\nquit\n")
	want := []string{pattern + "-1.dng", pattern + "-1.jpg", pattern + "-2.dng", pattern + "-2.jpg"}
	if got := dirNames(t, dir); !slices.Equal(got, want) {
		t.Fatalf("after run dir = %v, want %v", got, want)
	}
}

func TestMainRevertRestores(t *testing.T) {
	original := []string{"IMG_10.dng", "IMG_10.jpg", "IMG_3.dng", "IMG_3.jpg"}
	dir := setupDir(t, original...)

	runMain(t, "run\nrevert\nq\n")
	if got := dirNames(t, dir); !slices.Equal(got, original) {
		t.Errorf("after revert dir = %v, want %v", got, original)
	}
}

func TestMainSetPattern(t *testing.T) {
	dir := setupDir(t, "IMG_1.jpg")

	out := runMain(t, "set vacation\nrun\n")
	if !strings.Contains(out, "Pattern set to [vacation]") {
		t.Errorf("output missing pattern confirmation:\n%s", out)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"vacation-1.jpg"}) {
		t.Errorf("dir = %v, want [vacation-1.jpg]", got)
	}
}

func TestMainSetWithoutPattern(t *testing.T) {
	setupDir(t, "IMG_1.jpg")

	out := runMain(t, "set\nquit\n")
	if !strings.Contains(out, "No pattern provided") {
		t.Errorf("output missing error message:\n%s", out)
	}
}

func TestMainRevertWithoutRun(t *testing.T) {
	original := []string{"IMG_1.jpg"}
	dir := setupDir(t, original...)

	out := runMain(t, "revert\nquit\n")
	if !strings.Contains(out, "Nothing to revert") {
		t.Errorf("output missing message:\n%s", out)
	}
	if got := dirNames(t, dir); !slices.Equal(got, original) {
		t.Errorf("dir = %v, want %v", got, original)
	}
}

func TestMainDryRun(t *testing.T) {
	original := []string{"IMG_1.jpg", "IMG_2.jpg"}
	dir := setupDir(t, original...)

	out := runMain(t, "set x\ndry\nquit\n")
	if !strings.Contains(out, "IMG_1.jpg -> x-1.jpg") || !strings.Contains(out, "IMG_2.jpg -> x-2.jpg") {
		t.Errorf("dry output wrong:\n%s", out)
	}
	if got := dirNames(t, dir); !slices.Equal(got, original) {
		t.Errorf("dry run changed files: %v", got)
	}
}

func TestMainList(t *testing.T) {
	setupDir(t, "IMG_1.jpg", "IMG_1.dng")

	out := runMain(t, "set x\nrun\nls\nquit\n")
	if !strings.Contains(out, "x-1.dng\nx-1.jpg\n") {
		t.Errorf("list should show current names:\n%s", out)
	}
}

func TestMainNoFiles(t *testing.T) {
	setupDir(t, "notes.txt")

	out := runMain(t, "")
	if !strings.Contains(out, "No Files Found") {
		t.Errorf("output = %q, want No Files Found", out)
	}
}

func TestMainUnknownCommand(t *testing.T) {
	setupDir(t, "IMG_1.jpg")

	out := runMain(t, "bogus\nquit\n")
	if !strings.Contains(out, "No command found") {
		t.Errorf("output missing unknown-command message:\n%s", out)
	}
}
