package cli

import (
	"fmt"
	"io"

	"github.com/shuuuumai96/md-merge/internal/output"
	"github.com/shuuuumai96/md-merge/internal/render"
	"github.com/shuuuumai96/md-merge/internal/scan"
)

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, code, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return code
	}

	cfg, err = NormalizeConfig(cfg)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return ExitInvalidInput
	}

	scanResult, err := scan.MarkdownFiles(cfg)
	for _, warning := range scanResult.Warnings {
		fmt.Fprintf(stderr, "warning: %s: %s\n", warning.Path, warning.Message)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		if cfg.Strict {
			return ExitStrictFailure
		}
		return ExitGeneralError
	}
	if len(scanResult.Files) == 0 {
		fmt.Fprintln(stderr, "error: no Markdown files found")
		return ExitNoMarkdown
	}

	if cfg.DryRun {
		if _, err := stdout.Write(render.DryRun(scanResult.Files)); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return ExitGeneralError
		}
		return ExitSuccess
	}

	contents, readWarnings, err := scan.ReadFileContents(scanResult.Files, cfg)
	for _, warning := range readWarnings {
		fmt.Fprintf(stderr, "warning: %s: %s\n", warning.Path, warning.Message)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return ExitStrictFailure
	}
	if len(contents) == 0 {
		fmt.Fprintln(stderr, "error: no readable Markdown files found")
		if len(readWarnings) > 0 {
			return ExitPartialFailure
		}
		return ExitNoMarkdown
	}

	data, renderWarnings, err := render.Markdown(cfg, scanResult.Files, contents)
	for _, warning := range renderWarnings {
		fmt.Fprintf(stderr, "warning: %s: %s\n", warning.Path, warning.Message)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return ExitGeneralError
	}

	if err := output.Write(cfg, stdout, data); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return ExitGeneralError
	}

	if len(scanResult.Warnings)+len(readWarnings)+len(renderWarnings) > 0 {
		return ExitPartialFailure
	}
	return ExitSuccess
}
