package formattington

// rubocopExtensions are the file extensions handled by RuboCop.
var rubocopExtensions = []string{".rb", ".rake", ".gemspec", ".ru"}

// runRubocop lints Ruby files using RuboCop (github.com/rubocop/rubocop).
//
// RuboCop is a LINTER with ~100 layout cops, not a gofmt-style formatter;
// it is integrated like markdownlint/powershell, not like taplo. Every
// mode (default, --check, --dry-run) runs the same non-modifying lint
// invocation: findings exit 1, no findings exit 0, config errors exit 2.
// Formattington never injects fix flags — "-a", "-A", and "-x"
// (--fix-layout) are forwarded verbatim when the user passes them, so
// "formattington rubocop -x" is the format-like invocation and plain
// "formattington rubocop" is the lint invocation. "-x" applies
// layout-only corrections and never changes semantics.
//
// PATH-only everywhere (RuboCop is gem-only; bundling was considered and
// rejected); projects pin their own version via a Gemfile. ".rubocop.yml"
// is hierarchical ("inherit_from" supported), discovered natively.
func runRubocop(opts Options) (int, error) {
	executable, err := FindTool("rubocop", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	args := append([]string{}, opts.Args...)
	return runTool(executable, args, false)
}
