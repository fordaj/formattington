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
// used by "all" and by the exit-code aggregation rule. The order follows
// the Stack Overflow Developer Survey 2026 popularity ranking (see
// AGENTS.md, "Presentation order"); it is not alphabetical.
var registry = []Tool{
	{Name: "prettier", Extensions: prettierExtensions, Run: runPrettier},
	{Name: "sqlfluff", Extensions: sqlfluffExtensions, Run: runSqlfluff},
	{Name: "black", Extensions: blackExtensions, Run: runBlack},
	{Name: "shfmt", Extensions: shfmtExtensions, Run: runShfmt},
	{Name: "google-java-format", Extensions: googleJavaFormatExtensions, Run: runGoogleJavaFormat},
	{Name: "dotnet-format", Extensions: dotnetFormatExtensions, Run: runDotnetFormat},
	{Name: "powershell", Extensions: powershellExtensions, Run: runPowerShell},
	{Name: "clang-format", Extensions: clangFormatExtensions, Run: runClangFormat},
	{Name: "rustfmt", Extensions: rustfmtExtensions, Run: runRustfmt},
	{Name: "php-cs-fixer", Extensions: phpCsFixerExtensions, Run: runPhpCsFixer},
	{Name: "gofmt", Extensions: gofmtExtensions, Run: runGofmt},
	{Name: "goimports", Extensions: goimportsExtensions, Run: runGoimports},
	{Name: "ktfmt", Extensions: ktfmtExtensions, Run: runKtfmt},
	{Name: "stylua", Extensions: styluaExtensions, Run: runStylua},
	{Name: "swift-format", Extensions: swiftFormatExtensions, Run: runSwiftFormat},
	{Name: "rubocop", Extensions: rubocopExtensions, Run: runRubocop},
	{Name: "dart-format", Extensions: dartFormatExtensions, Run: runDartFormat},
	{Name: "styler", Extensions: stylerExtensions, Run: runStyler},
	{Name: "nixfmt", Extensions: nixfmtExtensions, Run: runNixfmt},
	{Name: "gdformat", Extensions: gdformatExtensions, Run: runGdformat},
	{Name: "npm-groovy-lint", Extensions: npmGroovyLintExtensions, Run: runNpmGroovyLint},
	{Name: "perltidy", Extensions: perltidyExtensions, Run: runPerltidy},
	{Name: "zig-fmt", Extensions: zigFmtExtensions, Run: runZigFmt},
	{Name: "mix-format", Extensions: mixFormatExtensions, Run: runMixFormat},
	{Name: "fourmolu", Extensions: fourmoluExtensions, Run: runFourmolu},
	{Name: "scalafmt", Extensions: scalafmtExtensions, Run: runScalafmt},
	{Name: "fprettify", Extensions: fprettifyExtensions, Run: runFprettify},
	{Name: "erlfmt", Extensions: erlfmtExtensions, Run: runErlfmt},
	{Name: "jlfmt", Extensions: jlfmtExtensions, Run: runJlfmt},
	{Name: "cljfmt", Extensions: cljfmtExtensions, Run: runCljfmt},
	{Name: "gleam-format", Extensions: gleamFormatExtensions, Run: runGleamFormat},
	{Name: "gnatformat", Extensions: gnatformatExtensions, Run: runGnatformat},
	{Name: "fantomas", Extensions: fantomasExtensions, Run: runFantomas},
	{Name: "ocamlformat", Extensions: ocamlformatExtensions, Run: runOcamlformat},
	{Name: "forge-fmt", Extensions: forgeFmtExtensions, Run: runForgeFmt},
	{Name: "mojo-format", Extensions: mojoFormatExtensions, Run: runMojoFormat},
	{Name: "odinfmt", Extensions: odinfmtExtensions, Run: runOdinfmt},
	{Name: "elm-format", Extensions: elmFormatExtensions, Run: runElmFormat},
	{Name: "markdownlint", Extensions: markdownlintExtensions, Run: runMarkdownlint},
	{Name: "taplo", Extensions: taploExtensions, Run: runTaplo},
}

// languageAliases maps language or file-type names onto formatter names.
var languageAliases = map[string]string{
	"c":           "clang-format",
	"cpp":         "clang-format",
	"c++":         "clang-format",
	"bash":        "shfmt",
	"python":      "black",
	"py":          "black",
	"md":          "prettier",
	"markdown":    "prettier",
	"html":        "prettier",
	"ps1":         "powershell",
	"toml":        "taplo",
	"go":          "gofmt",
	"lua":         "stylua",
	"nix":         "nixfmt",
	"php":         "php-cs-fixer",
	"perl":        "perltidy",
	"fortran":     "fprettify",
	"erlang":      "erlfmt",
	"sql":         "sqlfluff",
	"gdscript":    "gdformat",
	"java":        "google-java-format",
	"kotlin":      "ktfmt",
	"scala":       "scalafmt",
	"groovy":      "npm-groovy-lint",
	"micropython": "black",
	"clojure":     "cljfmt",
	"fsharp":      "fantomas",
	"ruby":        "rubocop",
	"r":           "styler",
	"julia":       "jlfmt",
	"elm":         "elm-format",
	"haskell":     "fourmolu",
	"ocaml":       "ocamlformat",
	"rust":        "rustfmt",
	"swift":       "swift-format",
	"dart":        "dart-format",
	"csharp":      "dotnet-format",
	"vbnet":       "dotnet-format",
	"elixir":      "mix-format",
	"zig":         "zig-fmt",
	"gleam":       "gleam-format",
	"mojo":        "mojo-format",
	"odin":        "odinfmt",
	"solidity":    "forge-fmt",
	"ada":         "gnatformat",
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
		code, handled := runAllCommand(&opts, formatterArgs)
		if handled {
			return code
		}
		formatterArgs = nil
		for _, tool := range registry {
			names = append(names, tool.Name)
		}
	} else if formatterName == "list" {
		// "list" is informational: it takes no arguments and no mode
		// flags, and --config is only meaningful with 'all'. Anything
		// trailing (including flags) counts as extra arguments.
		if err := unsupportedMix("list", opts, len(formatterArgs) > 0); err != nil {
			usageError(err)
			return ExitUsage
		}
		printList(os.Stdout)
		return ExitOK
	} else if formatterName == "init" {
		// "init" examines the current directory and writes
		// .formattington.json. It takes no arguments and no mode
		// flags; --config only changes what 'all' means.
		if err := unsupportedMix("init", opts, len(formatterArgs) > 0); err != nil {
			usageError(err)
			return ExitUsage
		}
		return runInit(opts.verbose)
	} else {
		target, err := resolveSingleTool(formatterName, opts)
		if err != nil {
			usageError(err)
			return ExitUsage
		}
		names = []string{target}
	}

	runOpts := Options{
		Check:   opts.check,
		DryRun:  opts.dryRun,
		Args:    formatterArgs,
		Verbose: opts.verbose,
	}
	return runTargets(targetsFromNames(names, formatterArgs), runOpts, opts.series)
}

// runAllCommand implements the "all" command. It reports handled=false
// when no configuration file applies, in which case the caller runs the
// full registry; otherwise it has already run (or rejected) the command and
// handled=true with the exit code to return.
func runAllCommand(opts *globalOptions, formatterArgs []string) (int, bool) {
	// "formattington all --series" is the documented spelling: --series may
	// appear after "all" (and, for consistency with --check/--dry-run, also
	// before "all" via the global parser). "--config PATH" is likewise
	// honored after "all". Any other trailing argument is an error.
	seriesAfterAll, afterConfig, hasAfterConfig, err := parseAllTrailing(formatterArgs)
	if err != nil {
		usageError(err)
		return ExitUsage, true
	}
	if seriesAfterAll {
		opts.series = true
	}
	if opts.hasConfig && hasAfterConfig {
		usageError(errors.New("--config may only be given once"))
		return ExitUsage, true
	}
	if !opts.hasConfig {
		opts.config = afterConfig
		opts.hasConfig = hasAfterConfig
	}

	commands, found, source, raw, err := loadConfig(opts.config, opts.hasConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formattington: %s\n", err)
		return ExitUsage, true
	}
	if !found {
		return 0, false
	}
	// An auto-discovered ./.formattington.json runs repo-declared argv on a
	// bare 'formattington all' in an untrusted clone, so name the file in
	// effect (not for --config or the environment variable, which are
	// explicit).
	if source == "auto-discovered .formattington.json" {
		fmt.Fprintf(os.Stderr, "formattington: using auto-discovered configuration %s\n", raw)
	}
	return runTargets(targetsFromCommands(commands), Options{
		Check:   opts.check,
		DryRun:  opts.dryRun,
		Verbose: opts.verbose,
	}, opts.series), true
}

// resolveSingleTool resolves a tool name or language alias to a registered
// tool. Config only changes what "all" means: an explicit --config before a
// single formatter is a usage error, while the env variable and
// ./.formattington.json are ignored here.
func resolveSingleTool(formatterName string, opts globalOptions) (string, error) {
	if opts.hasConfig {
		return "", errors.New("--config is only supported with 'all'")
	}
	target := resolveTarget(formatterName)
	if !isRegisteredTool(target) {
		// A documented alias whose backend has not landed yet must fail
		// loudly, never silently succeed.
		return "", fmt.Errorf("'%s' is not yet supported", formatterName)
	}
	return target, nil
}

// findFormatterIndex returns the index of the first formatter, alias, "all",
// "list", or "init" token, skipping "--config" values. It returns -1 when
// there is none.
func findFormatterIndex(argv []string) int {
	selectors := map[string]bool{"all": true, "list": true, "init": true}
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
	tool, _ := lookupTool(name) // an unregistered name yields a zero Tool; no extensions match
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

// runTarget is one resolved formatter invocation: the tool to run and the
// arguments to forward to it.
type runTarget struct {
	name string
	args []string
}

// resolveTarget maps a language alias to its backend tool; a registered
// tool name is returned unchanged.
func resolveTarget(name string) string {
	if alias, ok := languageAliases[name]; ok {
		return alias
	}
	return name
}

// targetsFromNames builds one target per name, all forwarding the same
// arguments. Names are expected to be resolved tool names.
func targetsFromNames(names []string, args []string) []runTarget {
	targets := make([]runTarget, 0, len(names))
	for _, name := range names {
		targets = append(targets, runTarget{name: name, args: args})
	}
	return targets
}

// targetsFromCommands builds one target per config command (element 0 is a
// tool name or language alias; the rest is forwarded verbatim).
func targetsFromCommands(commands [][]string) []runTarget {
	targets := make([]runTarget, 0, len(commands))
	for _, command := range commands {
		targets = append(targets, runTarget{name: resolveTarget(command[0]), args: command[1:]})
	}
	return targets
}

// runTargets runs targets and returns the aggregated exit code. Series
// execution preserves order exactly; concurrent execution aggregates in
// target order (not completion order) and prints missing-tool messages in
// target order after all formatters have run, so the exit code is
// deterministic. Formatters that write to stdout/stderr may interleave in
// the concurrent mode -- use --series for ordered output.
//
// An empty slice runs nothing and exits 0.
func runTargets(targets []runTarget, opts Options, series bool) int {
	if series {
		return runTargetsSeries(targets, opts)
	}
	return runTargetsParallel(targets, opts)
}

// runTargetsSeries runs targets one at a time, in order.
func runTargetsSeries(targets []runTarget, opts Options) int {
	codes := make([]int, 0, len(targets))
	for _, target := range targets {
		codes = append(codes, runOneTarget(target, opts))
	}
	return aggregateCodes(codes)
}

// runTargetsParallel runs targets concurrently and aggregates in target
// order.
func runTargetsParallel(targets []runTarget, opts Options) int {
	if len(targets) == 0 {
		return ExitOK
	}

	type outcome struct {
		code int
		err  error
	}
	results := make([]outcome, len(targets))

	var wg sync.WaitGroup
	for index, target := range targets {
		wg.Add(1)
		go func(index int, target runTarget) {
			defer wg.Done()
			result := runFormatter(target.name, targetOptions(target, opts))
			results[index] = outcome{code: result.code, err: result.err}
		}(index, target)
	}
	wg.Wait()

	codes := make([]int, len(targets))
	for index := range targets {
		printFormatterError(results[index].err)
		codes[index] = results[index].code
	}
	return aggregateCodes(codes)
}

// runOneTarget runs one target, reporting a missing tool on stderr, and
// returns its exit code.
func runOneTarget(target runTarget, opts Options) int {
	result := runFormatter(target.name, targetOptions(target, opts))
	printFormatterError(result.err)
	return result.code
}

// targetOptions carries opts through to one target, substituting the
// target's forwarded arguments.
func targetOptions(target runTarget, opts Options) Options {
	return Options{
		Check:   opts.Check,
		DryRun:  opts.DryRun,
		Args:    target.args,
		Verbose: opts.Verbose,
	}
}

// printFormatterError reports err on stderr when it is non-nil.
func printFormatterError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "formattington: %s\n", err)
	}
}

// lookupTool returns the registry entry for name.
func lookupTool(name string) (Tool, bool) {
	for _, tool := range registry {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}

// isRegisteredTool reports whether name is a registered formatter.
func isRegisteredTool(name string) bool {
	_, ok := lookupTool(name)
	return ok
}

// isRegisteredAlias reports whether name is a registered language alias.
func isRegisteredAlias(name string) bool {
	_, ok := languageAliases[name]
	return ok
}

// unsupportedMix reports why a bare informational command ('list' or
// 'init') cannot be combined with the global options or trailing
// arguments: these commands take no arguments and no mode flags, and
// --config is only meaningful with 'all'. It returns nil when the
// combination is allowed.
func unsupportedMix(command string, opts globalOptions, hasExtraArgs bool) error {
	switch {
	case hasExtraArgs:
		return fmt.Errorf("extra arguments are not supported with '%s'", command)
	case opts.hasConfig:
		return errors.New("--config is only supported with 'all'")
	case opts.check:
		return fmt.Errorf("--check is not supported with '%s'", command)
	case opts.dryRun:
		return fmt.Errorf("--dry-run is not supported with '%s'", command)
	case opts.series:
		return fmt.Errorf("--series is not supported with '%s'", command)
	}
	return nil
}

// usageError prints a usage error in argparse style.
func usageError(err error) {
	fmt.Fprintln(os.Stderr, usageLine)
	fmt.Fprintf(os.Stderr, "formattington: error: %s\n", err)
}
