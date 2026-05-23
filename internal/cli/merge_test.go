package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/shuuuumai96/md-merge/internal/cli"
	"github.com/shuuuumai96/md-merge/internal/merge"
	"github.com/shuuuumai96/md-merge/internal/scan"
)

func TestParseArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no arguments", nil, "target"},
		{"more than one target", []string{"docs", "more"}, "only one"},
		{"unknown flag", []string{"docs", "--wat"}, "unknown flag"},
		{"short output without value", []string{"docs", "-o"}, "output"},
		{"long output without value", []string{"docs", "--output"}, "output"},
		{"exclude without value", []string{"docs", "--exclude"}, "exclude"},
		{"extensions without value", []string{"docs", "--extensions"}, "extensions"},
		{"sort without value", []string{"docs", "--sort"}, "sort"},
		{"max size without value", []string{"docs", "--max-size"}, "max-size"},
		{"max size non numeric", []string{"docs", "--max-size", "abc"}, "max-size"},
		{"max size negative", []string{"docs", "--max-size", "-1"}, "max-size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code, err := cli.ParseArgs(tt.args)
			if code != cli.ExitInvalidInput {
				t.Fatalf("exit code = %d, want %d", code, cli.ExitInvalidInput)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseArgsValidForms(t *testing.T) {
	t.Run("max size zero is valid", func(t *testing.T) {
		cfg, code, err := cli.ParseArgs([]string{"docs", "--max-size", "0"})
		if err != nil || code != cli.ExitSuccess {
			t.Fatalf("cli.ParseArgs returned code %d err %v", code, err)
		}
		assertEqual(t, "max size", cfg.MaxSizeBytes, int64(0))
	})

	t.Run("repeated exclude is preserved", func(t *testing.T) {
		cfg, code, err := cli.ParseArgs([]string{"docs", "--exclude", "archive", "--exclude", "**/drafts/**"})
		if err != nil || code != cli.ExitSuccess {
			t.Fatalf("cli.ParseArgs returned code %d err %v", code, err)
		}
		assertStringSlice(t, cfg.Excludes, []string{"archive", "**/drafts/**"})
	})

	t.Run("short and long output flags are equivalent", func(t *testing.T) {
		shortCfg, _, err := cli.ParseArgs([]string{"docs", "-o", "merged.md"})
		if err != nil {
			t.Fatal(err)
		}
		longCfg, _, err := cli.ParseArgs([]string{"docs", "--output", "merged.md"})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "short output path", shortCfg.OutputPath, longCfg.OutputPath)
	})

	t.Run("unsupported sort is rejected during normalization", func(t *testing.T) {
		root := t.TempDir()
		cfg, _, err := cli.ParseArgs([]string{root, "--sort", "size"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = cli.NormalizeConfig(cfg)
		if err == nil || !strings.Contains(err.Error(), "sort") {
			t.Fatalf("cli.NormalizeConfig error = %v, want sort error", err)
		}
	})
}

func TestScanRecursiveBehavior(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "a.md", "a")
	writeText(t, root, "nested/b.markdown", "b")
	writeText(t, root, "README.MD", "c")
	writeText(t, root, ".hidden.md", "hidden")
	writeText(t, root, "notes.txt", "no")
	mkdir(t, root, "empty")

	result, err := scan.MarkdownFiles(testConfig(t, root))
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, paths(result.Files), []string{".hidden.md", "README.MD", "a.md", "nested/b.markdown"})
}

func TestScanExcludesDefaultDirectoriesAtAnyDepth(t *testing.T) {
	root := t.TempDir()
	for _, segment := range merge.DefaultExcludes {
		writeText(t, root, filepath.ToSlash(filepath.Join("top", segment, "skip.md")), "no")
		writeText(t, root, filepath.ToSlash(filepath.Join(segment, "skip.md")), "no")
	}
	writeText(t, root, "docs/readme.md", "yes")

	result, err := scan.MarkdownFiles(testConfig(t, root))
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, paths(result.Files), []string{"docs/readme.md"})
}

func TestScanSkipsSymlinkedFilesAndDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows commonly requires elevated privileges")
	}
	root := t.TempDir()
	writeText(t, root, "real.md", "yes")
	writeText(t, root, "target/file.md", "no")
	if err := os.Symlink(filepath.Join(root, "target", "file.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target"), filepath.Join(root, "linked-dir")); err != nil {
		t.Fatal(err)
	}

	result, err := scan.MarkdownFiles(testConfig(t, root))
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, paths(result.Files), []string{"real.md", "target/file.md"})
}

func TestCustomExclusions(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "archive/a.md", "no")
	writeText(t, root, "nested/archive/b.md", "no")
	writeText(t, root, "x/drafts/c.md", "no")
	writeText(t, root, "archive-not/d.md", "yes")
	writeText(t, root, "keep/e.md", "yes")

	cfg := testConfig(t, root)
	cfg.Excludes = append(cfg.Excludes, "archive", "**/drafts/**")
	cfg, err := cli.NormalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := scan.MarkdownFiles(cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, paths(result.Files), []string{"archive-not/d.md", "keep/e.md"})
}

func TestCustomExclusionsApplyToFilesAndAreSlashIndependent(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/draft.md", "no")
	writeText(t, root, "docs/final.md", "yes")

	cfg := testConfig(t, root)
	cfg.Excludes = append(cfg.Excludes, filepath.Join("docs", "draft.md"))
	cfg, err := cli.NormalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := scan.MarkdownFiles(cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, paths(result.Files), []string{"docs/final.md"})
}

func TestExtensionFiltering(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "a.md", "a")
	writeText(t, root, "b.markdown", "b")
	writeText(t, root, "c.MDX", "c")
	writeText(t, root, "d.txt", "d")

	tests := []struct {
		name string
		list string
		want []string
	}{
		{"default includes md and markdown", "md,markdown", []string{"a.md", "b.markdown"}},
		{"md only", "md", []string{"a.md"}},
		{"markdown only", "markdown", []string{"b.markdown"}},
		{"multiple custom extensions", "md,mdx", []string{"a.md", "c.MDX"}},
		{"leading dots and whitespace", " .MD , .markdown ", []string{"a.md", "b.markdown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(t, root)
			cfg.Extensions = scan.ParseExtensions(tt.list)
			result, err := scan.MarkdownFiles(cfg)
			if err != nil {
				t.Fatal(err)
			}
			assertStringSlice(t, paths(result.Files), tt.want)
		})
	}

	t.Run("empty entries are ignored but all-empty list is invalid", func(t *testing.T) {
		cfg, code, err := cli.ParseArgs([]string{root, "--extensions", " , "})
		if err == nil || code != cli.ExitInvalidInput {
			t.Fatalf("cli.ParseArgs returned cfg %+v code %d err %v, want invalid input", cfg, code, err)
		}
	})
}

func TestSortingModesAndTieBreakers(t *testing.T) {
	base := time.Date(2026, 5, 23, 1, 2, 3, 0, time.UTC)
	files := []merge.FileRecord{
		{Name: "same.md", DisplayPath: "b/same.md", ModifiedAt: base.Add(2 * time.Hour)},
		{Name: "a.md", DisplayPath: "z/a.md", ModifiedAt: base.Add(time.Hour)},
		{Name: "same.md", DisplayPath: "a/same.md", ModifiedAt: base.Add(2 * time.Hour)},
	}
	scan.SortFiles(files, merge.SortByPath)
	assertStringSlice(t, paths(files), []string{"a/same.md", "b/same.md", "z/a.md"})

	scan.SortFiles(files, merge.SortByName)
	assertStringSlice(t, paths(files), []string{"z/a.md", "a/same.md", "b/same.md"})

	scan.SortFiles(files, merge.SortByModified)
	assertStringSlice(t, paths(files), []string{"z/a.md", "a/same.md", "b/same.md"})
}

func TestModifiedSortUsesFileTimesAndDryRunMatchesMergeOrder(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "new.md", "new")
	writeText(t, root, "old.md", "old")
	mustChtimes(t, filepath.Join(root, "new.md"), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	mustChtimes(t, filepath.Join(root, "old.md"), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	dry := runInProcess(t, root, "--sort", "modified", "--dry-run")
	assertEqual(t, "dry-run code", dry.code, cli.ExitSuccess)
	assertEqual(t, "dry-run stdout", dry.stdout, "old.md\nnew.md\n")

	merged := runInProcess(t, root, "--sort", "modified")
	assertEqual(t, "merge code", merged.code, cli.ExitSuccess)
	assertOrder(t, merged.stdout, "<!-- BEGIN FILE: old.md -->", "<!-- BEGIN FILE: new.md -->")
}

func TestRenderingFormat(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "nested/file name [x].md", "# Heading\n\n```go\nfmt.Println(1)\n```\n")
	writeText(t, root, "empty.md", "")
	writeText(t, root, "unicode.md", "こんにちは\n")
	writeText(t, root, "no-newline.md", "tail")

	result := runInProcess(t, root)
	assertEqual(t, "exit code", result.code, cli.ExitSuccess)
	out := result.stdout
	assertContains(t, out, "# Merged Markdown\n\n")
	assertContains(t, out, fmt.Sprintf("Source directory: `%s`", filepath.ToSlash(filepath.Clean(root))))
	assertContains(t, out, "Generated by: `md-merge`")
	assertContains(t, out, "<!-- BEGIN FILE: nested/file name [x].md -->")
	assertContains(t, out, "## File: `nested/file name [x].md`")
	assertContains(t, out, "# Heading\n\n```go\nfmt.Println(1)\n```\n")
	assertContains(t, out, "<!-- END FILE: nested/file name [x].md -->")
	assertContains(t, out, "## File: `unicode.md`\n\nこんにちは\n")
	assertContains(t, out, "## File: `no-newline.md`\n\ntail\n\n<!-- END FILE: no-newline.md -->")
	assertContains(t, out, "## File: `empty.md`\n\n\n\n<!-- END FILE: empty.md -->")
	assertNotContains(t, out, "```md\n# Heading")
	assertNotContains(t, out, "## Files")
}

func TestWithTOCBehavior(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "b.md", "# B")
	writeText(t, root, "a.md", "# A")
	output := filepath.Join(t.TempDir(), "merged.md")

	result := runInProcess(t, root, "--with-toc", "-o", output)
	assertEqual(t, "exit code", result.code, cli.ExitSuccess)
	assertEqual(t, "stdout", result.stdout, "")
	out := readFile(t, output)
	assertContains(t, out, "## Files\n\n- `a.md`\n- `b.md`\n\n---")
	assertNotContains(t, out, "- `# A`")
}

func TestDryRunBehavior(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "a.md", "# A")
	writeText(t, root, "skip.txt", "no")
	writeText(t, root, "archive/b.md", "# B")
	writeText(t, root, "c.markdown", "# C")
	output := filepath.Join(root, "merged.md")

	result := runInProcess(t, root, "--dry-run", "-o", output, "--exclude", "archive", "--extensions", "md")
	assertEqual(t, "exit code", result.code, cli.ExitSuccess)
	assertEqual(t, "stdout", result.stdout, "a.md\n")
	assertEqual(t, "stderr", result.stderr, "")
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create output file, stat err = %v", err)
	}
	assertNotContains(t, result.stdout, "# Merged Markdown")

	empty := runInProcess(t, t.TempDir(), "--dry-run")
	assertEqual(t, "empty dry-run code", empty.code, cli.ExitNoMarkdown)
}

func TestOutputFileBehavior(t *testing.T) {
	t.Run("short and long flags write equivalent output and stdout is empty", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		shortOut := filepath.Join(t.TempDir(), "short.md")
		longOut := filepath.Join(t.TempDir(), "long.md")

		short := runInProcess(t, root, "-o", shortOut)
		long := runInProcess(t, root, "--output", longOut)
		assertEqual(t, "short code", short.code, cli.ExitSuccess)
		assertEqual(t, "long code", long.code, cli.ExitSuccess)
		assertEqual(t, "short stdout", short.stdout, "")
		assertEqual(t, "long stdout", long.stdout, "")
		assertEqual(t, "output contents", readFile(t, shortOut), readFile(t, longOut))
	})

	t.Run("existing output is overwritten and excluded from scan", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		writeText(t, root, "merged.md", "# Old")

		result := runInProcess(t, root, "-o", filepath.Join(root, "merged.md"))
		assertEqual(t, "exit code", result.code, cli.ExitSuccess)
		out := readFile(t, filepath.Join(root, "merged.md"))
		assertContains(t, out, "<!-- BEGIN FILE: a.md -->")
		assertNotContains(t, out, "<!-- BEGIN FILE: merged.md -->")
		assertNotContains(t, out, "# Old")
	})

	t.Run("absolute and relative self exclusion both work", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		writeText(t, root, "out.md", "# Old")
		abs := runInProcess(t, root, "-o", filepath.Join(root, "out.md"))
		assertEqual(t, "absolute code", abs.code, cli.ExitSuccess)
		assertNotContains(t, readFile(t, filepath.Join(root, "out.md")), "<!-- BEGIN FILE: out.md -->")

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(cwd) })
		rel := runInProcess(t, ".", "-o", "out.md")
		assertEqual(t, "relative code", rel.code, cli.ExitSuccess)
		assertNotContains(t, readFile(t, filepath.Join(root, "out.md")), "<!-- BEGIN FILE: out.md -->")
	})

	t.Run("missing output parent errors", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		result := runInProcess(t, root, "-o", filepath.Join(root, "missing", "out.md"))
		assertEqual(t, "exit code", result.code, cli.ExitGeneralError)
		assertContains(t, result.stderr, "error:")
	})
}

func TestMaxSizeBehavior(t *testing.T) {
	t.Run("zero means unlimited and exact limit is included", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "12345")
		assertEqual(t, "unlimited code", runInProcess(t, root, "--max-size", "0").code, cli.ExitSuccess)
		exact := runInProcess(t, root, "--max-size", "5")
		assertEqual(t, "exact code", exact.code, cli.ExitSuccess)
		assertContains(t, exact.stdout, "12345")
	})

	t.Run("one byte above limit is partial in non strict", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "ok.md", "12")
		writeText(t, root, "large.md", "123")
		result := runInProcess(t, root, "--max-size", "2")
		assertEqual(t, "exit code", result.code, cli.ExitPartialFailure)
		assertContains(t, result.stderr, "large.md")
		assertContains(t, result.stderr, "max size")
		assertContains(t, result.stdout, "<!-- BEGIN FILE: ok.md -->")
		assertNotContains(t, result.stdout, "<!-- BEGIN FILE: large.md -->")
	})

	t.Run("strict size skip fails without writing output", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "large.md", "123")
		output := filepath.Join(root, "out.md")
		result := runInProcess(t, root, "--strict", "--max-size", "2", "-o", output)
		assertEqual(t, "exit code", result.code, cli.ExitStrictFailure)
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("strict max-size failure should not write output, stat err = %v", err)
		}
	})
}

func TestInvalidUTF8Behavior(t *testing.T) {
	t.Run("non strict skips invalid file and writes readable files", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "ok.md", "héllo")
		writeBytes(t, root, "bad.md", []byte{0xff, 0xfe})
		result := runInProcess(t, root)
		assertEqual(t, "exit code", result.code, cli.ExitPartialFailure)
		assertContains(t, result.stderr, "invalid UTF-8")
		assertContains(t, result.stdout, "héllo")
		assertNotContains(t, result.stdout, "<!-- BEGIN FILE: bad.md -->")
	})

	t.Run("strict invalid utf8 fails without writing output", func(t *testing.T) {
		root := t.TempDir()
		writeBytes(t, root, "bad.md", []byte{0xff})
		output := filepath.Join(root, "out.md")
		result := runInProcess(t, root, "--strict", "-o", output)
		assertEqual(t, "exit code", result.code, cli.ExitStrictFailure)
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("strict UTF-8 failure should not write output, stat err = %v", err)
		}
	})
}

func TestExitCodeIntegration(t *testing.T) {
	t.Run("success stdout merge", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		result := runCLI(t, root)
		assertEqual(t, "exit code", result.code, cli.ExitSuccess)
		assertContains(t, result.stdout, "# Merged Markdown")
	})

	t.Run("success output merge", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "a.md", "# A")
		result := runCLI(t, root, "-o", filepath.Join(t.TempDir(), "out.md"))
		assertEqual(t, "exit code", result.code, cli.ExitSuccess)
	})

	t.Run("invalid target", func(t *testing.T) {
		result := runCLI(t, filepath.Join(t.TempDir(), "missing"))
		assertEqual(t, "exit code", result.code, cli.ExitInvalidInput)
	})

	t.Run("empty target", func(t *testing.T) {
		result := runCLI(t, t.TempDir())
		assertEqual(t, "exit code", result.code, cli.ExitNoMarkdown)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		result := runCLI(t, "--bad")
		assertEqual(t, "exit code", result.code, cli.ExitInvalidInput)
	})

	t.Run("non strict partial read failure", func(t *testing.T) {
		root := t.TempDir()
		writeText(t, root, "ok.md", "ok")
		writeBytes(t, root, "bad.md", []byte{0xff})
		result := runCLI(t, root)
		assertEqual(t, "exit code", result.code, cli.ExitPartialFailure)
	})

	t.Run("strict read failure", func(t *testing.T) {
		root := t.TempDir()
		writeBytes(t, root, "bad.md", []byte{0xff})
		result := runCLI(t, root, "--strict")
		assertEqual(t, "exit code", result.code, cli.ExitStrictFailure)
	})
}

func TestPermissionReadFailureWhenSupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based unreadable file behavior is not reliable on Windows")
	}
	root := t.TempDir()
	writeText(t, root, "ok.md", "ok")
	writeText(t, root, "locked.md", "no")
	locked := filepath.Join(root, "locked.md")
	if err := os.Chmod(locked, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0644) })

	result := runInProcess(t, root)
	assertEqual(t, "non-strict code", result.code, cli.ExitPartialFailure)
	assertContains(t, result.stdout, "ok")

	strict := runInProcess(t, root, "--strict")
	assertEqual(t, "strict code", strict.code, cli.ExitStrictFailure)
}

func TestNormalizeAndExcludePureHelpers(t *testing.T) {
	assertEqual(t, "display path", scan.DisplayPath(filepath.FromSlash("/tmp/root"), filepath.FromSlash("/tmp/root/a/b.md")), "a/b.md")
	if !scan.ShouldExclude("a/drafts/b.md", "b.md", []string{"**/drafts/**"}) {
		t.Fatalf("expected glob-like drafts exclusion to match")
	}
	if scan.ShouldExclude("a/archive-not/b.md", "b.md", []string{"archive"}) {
		t.Fatalf("segment exclusion matched unrelated path")
	}
}

func FuzzParseExtensions(f *testing.F) {
	for _, seed := range []string{"md,markdown", ".md, .MDX", " , ", "go,txt"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		got := scan.ParseExtensions(input)
		for ext := range got {
			if ext == "" {
				t.Fatalf("empty extension returned for %q", input)
			}
			if strings.Contains(ext, ".") && strings.HasPrefix(ext, ".") {
				t.Fatalf("extension %q still has leading dot", ext)
			}
			if ext != strings.ToLower(strings.TrimSpace(ext)) {
				t.Fatalf("extension %q is not normalized", ext)
			}
		}
	})
}

func FuzzShouldExclude(f *testing.F) {
	seeds := []struct {
		path    string
		name    string
		pattern string
	}{
		{"archive/a.md", "a.md", "archive"},
		{"x/drafts/a.md", "a.md", "**/drafts/**"},
		{"docs/final.md", "final.md", "archive"},
	}
	for _, seed := range seeds {
		f.Add(seed.path, seed.name, seed.pattern)
	}
	f.Fuzz(func(t *testing.T, display, name, pattern string) {
		_ = scan.ShouldExclude(filepath.ToSlash(display), filepath.Base(name), []string{pattern})
	})
}

type cliResult struct {
	code   int
	stdout string
	stderr string
}

func runInProcess(t *testing.T, args ...string) cliResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cli.Main(args, &stdout, &stderr)
	return cliResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	cmdArgs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "MD_MERGE_HELPER_PROCESS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := cli.ExitSuccess
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running helper process: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return cliResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("MD_MERGE_HELPER_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Exit(cli.Main(os.Args[i+1:], os.Stdout, os.Stderr))
		}
	}
	os.Exit(cli.ExitInvalidInput)
}

func testConfig(t *testing.T, root string) merge.Config {
	t.Helper()
	cfg, err := cli.NormalizeConfig(merge.Config{RootDir: root, Extensions: scan.ParseExtensions("md,markdown"), SortMode: "path"})
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func writeText(t *testing.T, root, rel, content string) {
	t.Helper()
	writeBytes(t, root, rel, []byte(content))
}

func writeBytes(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mustChtimes(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func paths(files []merge.FileRecord) []string {
	out := make([]string, len(files))
	for i, file := range files {
		out[i] = file.DisplayPath
	}
	return out
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q\nactual:\n%s", want, got)
	}
}

func assertNotContains(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("expected output not to contain %q\nactual:\n%s", unwanted, got)
	}
}

func assertEqual[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func assertOrder(t *testing.T, got string, first string, second string) {
	t.Helper()
	firstAt := strings.Index(got, first)
	secondAt := strings.Index(got, second)
	if firstAt < 0 || secondAt < 0 || firstAt >= secondAt {
		t.Fatalf("expected %q before %q\nactual:\n%s", first, second, got)
	}
}
