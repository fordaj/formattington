package formattington

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	// configFilename is the auto-discovered configuration file name.
	configFilename = ".formattington.json"
	// configEnvVar points at a configuration file explicitly.
	configEnvVar = "FORMATTINGTON_CONFIG_PATH"
)

// resolveConfigPath resolves the config path by precedence.
//
// Precedence (highest first): explicit ("--config"), then
// FORMATTINGTON_CONFIG_PATH, then "./.formattington.json" in the working
// directory only (no parent search). It returns the path, the display
// string, and the label used in the could-not-find message. found is false
// when no config applies (the full-registry "all" behaviour).
func resolveConfigPath(explicit string, hasExplicit bool) (path, raw, source string, found bool) {
	if hasExplicit {
		return explicit, explicit, "--config", true
	}

	if envValue := os.Getenv(configEnvVar); envValue != "" {
		return envValue, envValue, configEnvVar, true
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", false
	}
	auto := filepath.Join(cwd, configFilename)
	if info, statErr := os.Stat(auto); statErr != nil || info.IsDir() {
		return "", "", "", false
	}
	return auto, auto, "auto-discovered .formattington.json", true
}

// readConfigData reads path and parses the JSON document.
//
// It returns a non-nil error whose message is the text after
// "formattington: " for the caller to print before exiting 2. raw and
// source preserve the could-not-find wording.
func readConfigData(path, raw, source string) (any, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Could not find config file '%s' (via %s)", raw, source)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read config file '%s': %v", path, err)
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("Invalid config file '%s': invalid JSON: %v", path, err)
	}
	return parsed, nil
}

// validateConfig validates the top-level shape and each command entry.
//
// It returns the validated commands (possibly empty) or an error with the
// same wording as the previous implementation.
func validateConfig(data any, path string) ([][]string, error) {
	object, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"Invalid config file '%s': top-level must be an object with a 'commands' array",
			path,
		)
	}

	var unknown []string
	for key := range object {
		if key != "commands" {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return nil, fmt.Errorf(
			"Invalid config file '%s': unknown key(s) %s; only 'commands' is supported",
			path, pyReprList(unknown),
		)
	}

	rawCommands, ok := object["commands"]
	if !ok {
		return nil, fmt.Errorf(
			"Invalid config file '%s': missing required 'commands' array", path,
		)
	}

	entries, ok := rawCommands.([]any)
	if !ok {
		return nil, fmt.Errorf("Invalid config file '%s': 'commands' must be a list", path)
	}

	commands := make([][]string, 0, len(entries))
	for index, entry := range entries {
		prefix := fmt.Sprintf("Invalid config file '%s': commands[%d]", path, index)

		switch typed := entry.(type) {
		case string:
			return nil, fmt.Errorf(
				"%s must be a list of strings, not a shell string: %s",
				prefix, pyRepr(entry),
			)
		case map[string]any:
			return nil, fmt.Errorf(
				"%s must be a list of strings, not an object: %s",
				prefix, pyRepr(entry),
			)
		case []any:
			if len(typed) == 0 {
				return nil, fmt.Errorf("%s must not be empty", prefix)
			}
			argv := make([]string, 0, len(typed))
			for _, item := range typed {
				value, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf(
						"%s must contain only strings: %s", prefix, pyRepr(entry),
					)
				}
				argv = append(argv, value)
			}
			tool := argv[0]
			switch {
			case isRegisteredTool(tool):
			case isRegisteredAlias(tool):
				if !isRegisteredTool(languageAliases[tool]) {
					return nil, fmt.Errorf("%s: '%s' is not yet supported", prefix, tool)
				}
			default:
				return nil, fmt.Errorf(
					"%s: unknown formatter or alias '%s'", prefix, tool,
				)
			}
			commands = append(commands, argv)
		default:
			return nil, fmt.Errorf(
				"%s must be a list of strings, got %s: %s",
				prefix, jsonTypeName(entry), pyRepr(entry),
			)
		}
	}

	return commands, nil
}

// loadConfig resolves, reads, and validates the Formattington config file.
//
// It returns the commands and true when a config is in effect, false and
// nil when no config applies (the full-registry "all" behaviour), or an
// error whose message is the text after "formattington: ".
func loadConfig(explicit string, hasExplicit bool) ([][]string, bool, error) {
	path, raw, source, found := resolveConfigPath(explicit, hasExplicit)
	if !found {
		return nil, false, nil
	}

	data, err := readConfigData(path, raw, source)
	if err != nil {
		return nil, false, err
	}

	commands, err := validateConfig(data, path)
	if err != nil {
		return nil, false, err
	}
	return commands, true, nil
}

// pyRepr renders a decoded JSON value the way Python's repr() would, so
// validation messages stay familiar.
func pyRepr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case string:
		return pyReprString(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'g', -1, 64)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, pyRepr(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, pyReprString(key)+": "+pyRepr(typed[key]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprintf("%v", value)
}

// pyReprList renders a list of strings the way Python's repr() would.
func pyReprList(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, pyReprString(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyReprString renders a string the way Python's repr() would.
func pyReprString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return "'" + escaped + "'"
}

// jsonTypeName names a decoded JSON value the way Python names its type.
func jsonTypeName(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case float64:
		if typed == float64(int64(typed)) {
			return "int"
		}
		return "float"
	case []any:
		return "list"
	case map[string]any:
		return "dict"
	}
	return "object"
}
