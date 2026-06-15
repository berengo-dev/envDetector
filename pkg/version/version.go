// Package version parses and compares semantic-looking version strings.
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// versionRegex matches the first semantic version-like substring in arbitrary text.
// Examples: "1.21.5", "20.10.0", "24.0".
var versionRegex = regexp.MustCompile(`\d+(?:\.\d+)?(?:\.\d+)?`)

// Extract returns the first version-like string found in raw, or an empty string
// if none is found. The leading "v" prefix, if present, is stripped.
func Extract(raw string) string {
	m := versionRegex.FindString(raw)
	if m == "" {
		return ""
	}
	return strings.TrimPrefix(m, "v")
}

// semverConstraintRegex matches common semver constraint prefixes and captures
// the operator and the version core.
var semverConstraintRegex = regexp.MustCompile(`^(?:(\^|~|>=|<=|>|<=|=)\s*)?v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

// ConvertSemverToWildcard converts common semver constraints to a wildcard
// pattern usable by Match. It returns the wildcard and true on success, or an
// empty string and false when the constraint cannot be converted.
//
// Examples:
//
//	"^16.2.7" -> "16.x", true
//	"~16.2.7" -> "16.x", true
//	">=20.0.0" -> "20.x", true
//	"1.21.5"  -> "1.21", true
//	"*"       -> "", false
//	"latest"  -> "", false
//	"^0.2.3"  -> "0.2", true
//	">=1.0.0" -> "1.x", true
func ConvertSemverToWildcard(constraint string) (string, bool) {
	c := strings.TrimSpace(strings.ToLower(constraint))
	if c == "" || c == "*" || c == "x" || c == "latest" {
		return "", false
	}

	m := semverConstraintRegex.FindStringSubmatch(c)
	if m == nil {
		return "", false
	}

	operator := m[1]
	major := m[2]
	minor := m[3]

	switch operator {
	case "^":
		if major == "0" && minor != "" {
			return major + "." + minor, true
		}
		return major + ".x", true
	case "~":
		return major + ".x", true
	}

	// Comparisons and exact versions.
	if minor != "" && operator == "" {
		return major + "." + minor, true
	}
	return major + ".x", true
}

// Compare returns -1, 0, or 1 depending on whether a is less than, equal to,
// or greater than b. It parses the leading numeric version core from each
// string (e.g. "9.x" -> 9, "1.21.5" -> 1.21.5). When only one value parses,
// the parsed value is considered greater. When neither parses, strings are
// compared lexicographically.
func Compare(a, b string) int {
	parse := func(s string) ([]int, bool) {
		core := Extract(s)
		if core == "" {
			return nil, false
		}
		parts := strings.Split(core, ".")
		out := make([]int, 0, len(parts))
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				return nil, false
			}
			out = append(out, n)
		}
		return out, len(out) > 0
	}

	pa, okA := parse(a)
	pb, okB := parse(b)

	switch {
	case okA && okB:
		for i := 0; i < len(pa) || i < len(pb); i++ {
			var av, bv int
			if i < len(pa) {
				av = pa[i]
			}
			if i < len(pb) {
				bv = pb[i]
			}
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
		}
		return 0
	case okA:
		return 1
	case okB:
		return -1
	default:
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

// Match compares an actual version string (free-form tool output) against an
// expected version pattern. Supported patterns are:
//   - exact:   "1.21.5" matches "1.21.5"
//   - prefix:  "1.21"   matches "1.21.5" (and "1.21.0")
//   - wildcard:"24.x"   matches "24.0.1" (x or * may replace any suffix)
//   - latest:  "latest" matches any version
//
// It returns whether the versions match, the cleaned actual version, and any
// parse error.
func Match(actual, expected string) (bool, string, error) {
	actual = Extract(actual)
	if strings.ToLower(expected) == "latest" {
		return true, actual, nil
	}
	if actual == "" {
		return false, "", fmt.Errorf("no version found in output")
	}

	actParts := strings.Split(actual, ".")
	expParts := strings.Split(expected, ".")

	for i, exp := range expParts {
		exp = strings.ToLower(exp)
		if exp == "x" || exp == "*" {
			// Wildcard accepts any value at this position and beyond, but the
			// actual version must at least reach this position.
			if len(actParts) < i+1 {
				return false, actual, nil
			}
			return true, actual, nil
		}

		if len(actParts) <= i {
			return false, actual, nil
		}

		if actParts[i] != exp {
			return false, actual, nil
		}
	}

	return true, actual, nil
}
