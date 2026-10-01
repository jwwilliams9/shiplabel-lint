package main

import (
	"regexp"
	"strings"
)

// Fix records one value that FixSource rewrote.
type Fix struct {
	Line int
	Col  int
	Key  string
	Old  string
	New  string
}

// Looser patterns than the ones in parser.go. They accept the spellings
// people actually type ("1.5lbs", "12 X 8 X 6in") so they can be rewritten
// into the canonical form the validators expect.
var (
	looseWeightRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(lbs?|ozs?|kgs?|gs?)$`)
	looseDimsRe   = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*x\s*(\d+(?:\.\d+)?)\s*x\s*(\d+(?:\.\d+)?)\s*(ins?|inch|inches|cm)$`)
)

// FixSource applies mechanical fixes to weight and dims values: spacing
// before the unit, unit case, plural units, and case or spacing around the
// "x" in dimensions. A value is only rewritten if it is currently invalid
// and the rewritten form passes validation, so a value that is wrong in a
// way that needs a human (a missing unit, say) is left for the linter to
// report. Line endings and everything outside the value are preserved.
func FixSource(data []byte) ([]byte, []Fix) {
	lines := strings.Split(string(data), "\n")
	var fixes []Fix

	for i, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		ending := raw[len(line):]

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])

		var fixed string
		switch key {
		case "weight":
			if validateWeight(value) == nil {
				continue
			}
			fixed = fixWeight(value)
			if fixed == "" || validateWeight(fixed) != nil {
				continue
			}
		case "dims":
			if validateDims(value) == nil {
				continue
			}
			fixed = fixDims(value)
			if fixed == "" || validateDims(fixed) != nil {
				continue
			}
		default:
			continue
		}

		valueStart := colonIdx + 1
		for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
			valueStart++
		}
		lines[i] = line[:valueStart] + fixed + ending
		fixes = append(fixes, Fix{Line: i + 1, Col: valueStart + 1, Key: key, Old: value, New: fixed})
	}

	if len(fixes) == 0 {
		return data, nil
	}
	return []byte(strings.Join(lines, "\n")), fixes
}

// fixWeight returns the canonical form of a loosely written weight, or ""
// if the value isn't recognizable.
func fixWeight(v string) string {
	m := looseWeightRe.FindStringSubmatch(strings.ToLower(v))
	if m == nil {
		return ""
	}
	// None of the canonical units (lb, oz, kg, g) ends in "s", so any
	// trailing "s" is a plural.
	return m[1] + " " + strings.TrimSuffix(m[2], "s")
}

// fixDims returns the canonical form of loosely written dimensions, or ""
// if the value isn't recognizable.
func fixDims(v string) string {
	m := looseDimsRe.FindStringSubmatch(strings.ToLower(v))
	if m == nil {
		return ""
	}
	unit := m[4]
	if unit != "cm" {
		unit = "in"
	}
	return m[1] + "x" + m[2] + "x" + m[3] + " " + unit
}
