package formattington

// markdownlintExtensions are the file extensions handled by markdownlint.
var markdownlintExtensions = []string{".md", ".markdown"}

// runMarkdownlint lints Markdown files with markdownlint-cli.
//
// markdownlint never rewrites files without an explicit "--fix", so every
// mode behaves the same: findings exit 1, no findings exit 0. Check and
// DryRun are accepted for CLI uniformity; they do not change the lint
// invocation. Arguments are forwarded verbatim (so "--fix", "--config",
// or a glob pattern all work natively); with no file arguments, matching
// Markdown files are discovered recursively by the caller.
func runMarkdownlint(opts Options) (int, error) {
	executable, err := FindTool("markdownlint", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	arguments := append([]string{}, opts.Args...)
	return runTool(executable, arguments, false)
}
