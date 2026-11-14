package formattington

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Tool describes one formatter integration.
type Tool struct {
	// Name is the CLI-visible formatter name.
	Name string
	// Extensions are the dotted suffixes the tool handles.
	Extensions []string
	// Run executes the tool.
	Run Formatter
}

// registry is the canonical formatter registry. The order is the order
// used by "all" and by the exit-code aggregation rule.
var registry = []Tool{
	{Name: "shfmt", Extensions: shfmtExtensions, Run: runShfmt},
	{Name: "clang-format", Extensions: clangFormatExtensions, Run: runClangFormat},
	{Name: "black", Extensions: blackExtensions, Run: runBlack},
	{Name: "prettier", Extensions: prettierExtensions, Run: runPrettier},
	{Name: "powershell", Extensions: powershellExtensions, Run: runPowerShell},
	{Name: "markdownlint", Extensions: markdownlintExtensions, Run: runMarkdownlint},
	{Name: "taplo", Extensions: taploExtensions, Run: runTaplo},
	{Name: "gofmt", Extensions: gofmtExtensions, Run: runGofmt},
	{Name: "goimports", Extensions: goimportsExtensions, Run: runGoimports},
	{Name: "stylua", Extensions: styluaExtensions, Run: runStylua},
	{Name: "nixfmt", Extensions: nixfmtExtensions, Run: runNixfmt},
	{Name: "php-cs-fixer", Extensions: phpCsFixerExtensions, Run: runPhpCsFixer},
	{Name: "perltidy", Extensions: perltidyExtensions, Run: runPerltidy},
	{Name: "fprettify", Extensions: fprettifyExtensions, Run: runFprettify},
	{Name: "erlfmt", Extensions: erlfmtExtensions, Run: runErlfmt},
	{Name: "sqlfluff", Extensions: sqlfluffExtensions, Run: runSqlfluff},
	{Name: "gdformat", Extensions: gdformatExtensions, Run: runGdformat},
	{Name: "google-java-format", Extensions: googleJavaFormatExtensions, Run: runGoogleJavaFormat},
	{Name: "ktfmt", Extensions: ktfmtExtensions, Run: runKtfmt},
	{Name: "scalafmt", Extensions: scalafmtExtensions, Run: runScalafmt},
	{Name: "npm-groovy-lint", Extensions: npmGroovyLintExtensions, Run: runNpmGroovyLint},
}

// languageAliases maps language or file-type names onto formatter names.
var languageAliases = map[string]string{
	"c":        "clang-format",
	"cpp":      "clang-format",
	"c++":      "clang-format",
	"bash":     "shfmt",
	"python":   "black",
	"py":       "black",
	"md":       "prettier",
	"markdown": "prettier",
	"html":     "prettier",
	"ps1":      "powershell",
	"toml":     "taplo",
	"go":       "gofmt",
	"lua":      "stylua",
	"nix":      "nixfmt",
	"php":      "php-cs-fixer",
	"perl":     "perltidy",
	"fortran":  "fprettify",
	"erlang":   "erlfmt",
	"sql":      "sqlfluff",
	"gdscript": "gdformat",
	"java":     "google-java-format",
	"kotlin":   "ktfmt",
	"scala":    "scalafmt",
	"groovy":   "npm-groovy-lint",
}

// maxFilesPerInvocation is the upper bound on discovered files per
// formatter invocation, to respect operating-system command-line limits on
// very large repositories.
var maxFilesPerInvocation = 256

// errHelpRequested signals that the help text was requested.
var errHelpRequested = errors.New("help requested")

// globalOptions holds Formattington's own command-line options.
type globalOptions struct {
	check     bool
	dryRun    bool
	series    bool
	verbose   bool
	config    string
	hasConfig bool
}

// Main runs the Formattington command line and returns the process exit
// code. argv excludes the program name.
func Main(argv []string) int {
	// Formattington owns global options before the formatter name.
	// "--config" values are skipped during the scan so a path that
	// happens to equal a formatter name is not mistaken for the selector.
	formatterIndex := findFormatterIndex(argv)

	// Only Formattington's arguments count here.
	formattingtonArgs := argv
	if formatterIndex >= 0 {
		formattingtonArgs = argv[:formatterIndex]
	}

	if contains(formattingtonArgs, "--version") {
		fmt.Println(Version())
		return ExitOK
	}

	// Global flags: one shared parser for every path.
	opts, err := parseGlobalArgs(formattingtonArgs)
	if err != nil {
		if errors.Is(err, errHelpRequested) {
			printHelp(os.Stdout)
			return ExitOK
		}
		usageError(err)
		return ExitUsage
	}

	if formatterIndex < 0 {
		return ExitOK
	}

	formatterName := argv[formatterIndex]
	formatterArgs := argv[formatterIndex+1:]

	var names []string

	if formatterName == "all" {
		// "formattington all --series" is the documented spelling:
		// --series may appear after "all" (and, for consistency with
		// --check/--dry-run, also before "all" via the global parser
		// above). "--config PATH" is likewise honored after "all". Any
		// other trailing argument is still an error.
		seriesAfterAll, afterConfig, hasAfterConfig, err := parseAllTrailing(formatterArgs)
		if err != nil {
			usageError(err)
			return ExitUsage
		}
		if seriesAfterAll {
			opts.series = true
		}
		if opts.hasConfig && hasAfterConfig {
			usageError(errors.New("--config may only be given once"))
			return ExitUsage
		}
		if !opts.hasConfig {
			opts.config = afterConfig
			opts.hasConfig = hasAfterConfig
		}
		formatterArgs = nil

		commands, found, err := loadConfig(opts.config, opts.hasConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", err)
			return ExitUsage
		}
		if found {
			runOpts := Options{
				Check:   opts.check,
				DryRun:  opts.dryRun,
				Verbose: opts.verbose,
			}
			if opts.series {
				return runConfigSeries(commands, runOpts)
			}
			return runConfigParallel(commands, runOpts)
		}
		for _, tool := range registry {
			names = append(names, tool.Name)
		}
	} else {
		// Config only changes what "all" means: an explicit --config
		// before a single formatter is a usage error, while the env
		// variable and ./.formattington.json are ignored here.
		if opts.hasConfig {
			usageError(errors.New("--config is only supported with 'all'"))
			return ExitUsage
		}
		target := formatterName
		if alias, ok := languageAliases[formatterName]; ok {
			target = alias
		}
		if !isRegisteredTool(target) {
			// A documented alias whose backend has not landed yet must
			// fail loudly, never silently succeed.
			usageError(fmt.Errorf("'%s' is not yet supported", formatterName))
			return ExitUsage
		}
		names = []string{target}
	}

	if formatterName != "all" || len(names) == 1 {
		// Single-formatter runs have no concurrency to order; run
		// directly (equivalently: --series is a no-op here).
		result := runFormatter(names[0], Options{
			Check:   opts.check,
			DryRun:  opts.dryRun,
			Args:    formatterArgs,
			Verbose: opts.verbose,
		})
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", result.err)
		}
		return result.code
	}

	runOpts := Options{
		Check:   opts.check,
		DryRun:  opts.dryRun,
		Args:    formatterArgs,
		Verbose: opts.verbose,
	}
	if opts.series {
		return runSeries(names, runOpts)
	}
	return runParallel(names, runOpts)
}

// findFormatterIndex returns the index of the first formatter, alias, or
// "all" token, skipping "--config" values. It returns -1 when there is
// none.
func findFormatterIndex(argv []string) int {
	selectors := map[string]bool{"all": true}
	for _, tool := range registry {
		selectors[tool.Name] = true
	}
	for alias := range languageAliases {
		selectors[alias] = true
	}

	index := 0
	for index < len(argv) {
		token := argv[index]
		switch {
		case token == "--config":
			index += 2
			continue
		case strings.HasPrefix(token, "--config="):
			index += 1
			continue
		case selectors[token]:
			return index
		}
		index++
	}
	return -1
}

// parseGlobalArgs parses Formattington's own options.
func parseGlobalArgs(args []string) (globalOptions, error) {
	opts := globalOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--check":
			if opts.dryRun {
				return opts, errors.New("argument --check: not allowed with argument --dry-run")
			}
			opts.check = true
		case arg == "--dry-run":
			if opts.check {
				return opts, errors.New("argument --dry-run: not allowed with argument --check")
			}
			opts.dryRun = true
		case arg == "--series":
			opts.series = true
		case arg == "--verbose":
			opts.verbose = true
		case arg == "--config":
			if index+1 >= len(args) {
				return opts, errors.New("argument --config: expected one argument")
			}
			index++
			if opts.hasConfig {
				return opts, errors.New("--config may only be given once")
			}
			opts.config = args[index]
			opts.hasConfig = true
		case strings.HasPrefix(arg, "--config="):
			if opts.hasConfig {
				return opts, errors.New("--config may only be given once")
			}
			opts.config = strings.TrimPrefix(arg, "--config=")
			opts.hasConfig = true
		case arg == "-h" || arg == "--help":
			return opts, errHelpRequested
		default:
			return opts, fmt.Errorf("unrecognized arguments: %s", arg)
		}
	}

	return opts, nil
}

// parseAllTrailing parses the options allowed after "all".
//
// "--series" may appear after "all" ("formattington all --series") and
// "--config PATH" is honored after "all" (including the "--config=PATH"
// spelling). Any other trailing argument is a usage error.
func parseAllTrailing(args []string) (seriesSeen bool, afterConfig string, hasAfterConfig bool, err error) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--series":
			seriesSeen = true
		case arg == "--config":
			if index+1 >= len(args) {
				return false, "", false, errors.New("--config requires a path")
			}
			if hasAfterConfig {
				return false, "", false, errors.New("--config may only be given once")
			}
			afterConfig = args[index+1]
			hasAfterConfig = true
			index++
		case strings.HasPrefix(arg, "--config="):
			if hasAfterConfig {
				return false, "", false, errors.New("--config may only be given once")
			}
			afterConfig = strings.TrimPrefix(arg, "--config=")
			hasAfterConfig = true
		default:
			return false, "", false, errors.New("extra arguments are not supported with 'all'")
		}
	}
	return seriesSeen, afterConfig, hasAfterConfig, nil
}

// formatterResult carries the outcome of one formatter run.
type formatterResult struct {
	name string
	code int
	err  error
}

// runFormatter runs one formatter, translating missing-tool errors to 127.
//
// When the user passes no explicit file arguments (and the args are not
// informational help/version passthroughs), matching files are discovered
// recursively and appended to the invocation, in batches of at most
// maxFilesPerInvocation files.
func runFormatter(name string, opts Options) formatterResult {
	tool := lookupTool(name)
	extensions := tool.Extensions

	userArgs := append([]string{}, opts.Args...)
	var batches [][]string
	if IsInformational(userArgs) || hasUserFileArgs(userArgs, extensions) {
		batches = [][]string{userArgs}
	} else {
		discovered := DiscoverFiles(extensions, "")
		for _, chunk := range iterChunks(discovered, maxFilesPerInvocation) {
			batches = append(batches, withArgs(userArgs, chunk))
		}
		if len(batches) == 0 {
			batches = [][]string{userArgs}
		}
	}

	exitCode := ExitOK
	for _, batch := range batches {
		code, err := tool.Run(Options{
			Check:   opts.Check,
			DryRun:  opts.DryRun,
			Args:    batch,
			Verbose: opts.Verbose,
		})
		if err != nil {
			return formatterResult{name: name, code: ExitNotFound, err: err}
		}
		if code != ExitOK {
			exitCode = code
		}
	}
	return formatterResult{name: name, code: exitCode}
}

// hasUserFileArgs reports whether the user supplied explicit file
// arguments.
//
// A non-flag argument counts as an explicit file when it exists on disk or
// ends with one of the formatter's extensions.
func hasUserFileArgs(args []string, extensions []string) bool {
	wanted := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		wanted[strings.ToLower(ext)] = true
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if _, err := os.Stat(arg); err == nil {
			return true
		}
		if wanted[pathSuffix(arg)] {
			return true
		}
	}
	return false
}

// aggregateCodes returns the last non-zero code in order; 0 only if all
// succeed.
//
// Callers pass codes already ordered by registration/file order.
func aggregateCodes(codes []int) int {
	exitCode := ExitOK
	for _, code := range codes {
		if code != ExitOK {
			exitCode = code
		}
	}
	return exitCode
}

// runSeries runs formatters one at a time in registration order.
func runSeries(names []string, opts Options) int {
	codes := make([]int, 0, len(names))
	for _, name := range names {
		result := runFormatter(name, opts)
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", result.err)
		}
		codes = append(codes, result.code)
	}
	return aggregateCodes(codes)
}

// runParallel runs formatters concurrently.
//
// Results are aggregated in registration order (not completion order) so
// the exit code is deterministic; missing-tool messages are printed in
// registration order after all formatters have run. Formatters that write
// to stdout/stderr may interleave in this mode -- use --series for ordered
// output.
func runParallel(names []string, opts Options) int {
	type outcome struct {
		code int
		err  error
	}
	results := make([]outcome, len(names))

	var wg sync.WaitGroup
	for index, name := range names {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()
			result := runFormatter(name, opts)
			results[index] = outcome{code: result.code, err: result.err}
		}(index, name)
	}
	wg.Wait()

	codes := make([]int, len(names))
	for index := range names {
		if results[index].err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", results[index].err)
		}
		codes[index] = results[index].code
	}
	return aggregateCodes(codes)
}

// runConfigSeries runs config commands sequentially in file order.
func runConfigSeries(commands [][]string, opts Options) int {
	codes := make([]int, 0, len(commands))
	for _, entry := range commands {
		target := entry[0]
		if alias, ok := languageAliases[target]; ok {
			target = alias
		}
		result := runFormatter(target, Options{
			Check:   opts.Check,
			DryRun:  opts.DryRun,
			Args:    entry[1:],
			Verbose: opts.Verbose,
		})
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", result.err)
		}
		codes = append(codes, result.code)
	}
	return aggregateCodes(codes)
}

// runConfigParallel runs config commands concurrently (the default for
// "all").
//
// Results and missing-tool messages are aggregated in file order regardless
// of completion order. Duplicate tool names are supported because results
// are keyed by command index rather than tool name. An empty "commands"
// runs nothing and exits 0.
func runConfigParallel(commands [][]string, opts Options) int {
	if len(commands) == 0 {
		return ExitOK
	}

	type outcome struct {
		code int
		err  error
	}
	results := make([]outcome, len(commands))

	var wg sync.WaitGroup
	for index, entry := range commands {
		wg.Add(1)
		go func(index int, entry []string) {
			defer wg.Done()
			target := entry[0]
			if alias, ok := languageAliases[target]; ok {
				target = alias
			}
			result := runFormatter(target, Options{
				Check:   opts.Check,
				DryRun:  opts.DryRun,
				Args:    entry[1:],
				Verbose: opts.Verbose,
			})
			results[index] = outcome{code: result.code, err: result.err}
		}(index, entry)
	}
	wg.Wait()

	codes := make([]int, len(commands))
	for index := range commands {
		if results[index].err != nil {
			fmt.Fprintf(os.Stderr, "formattington: %s\n", results[index].err)
		}
		codes[index] = results[index].code
	}
	return aggregateCodes(codes)
}

// lookupTool returns the registry entry for name.
func lookupTool(name string) Tool {
	for _, tool := range registry {
		if tool.Name == name {
			return tool
		}
	}
	return Tool{}
}

// isRegisteredTool reports whether name is a registered formatter.
func isRegisteredTool(name string) bool {
	for _, tool := range registry {
		if tool.Name == name {
			return true
		}
	}
	return false
}

// isRegisteredAlias reports whether name is a registered language alias.
func isRegisteredAlias(name string) bool {
	_, ok := languageAliases[name]
	return ok
}

// usageError prints a usage error in argparse style.
func usageError(err error) {
	fmt.Fprintln(os.Stderr, usageLine)
	fmt.Fprintf(os.Stderr, "formattington: error: %s\n", err)
}
