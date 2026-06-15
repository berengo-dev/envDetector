// Package version parses and compares semantic-looking version strings.
package version

import (
	"fmt"
	"regexp"
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
