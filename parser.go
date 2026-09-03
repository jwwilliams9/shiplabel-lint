package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Label is one shipping label parsed from a batch file: a block of
// "key: value" lines separated from the next label by a blank line.
type Label struct {
	Fields    map[string]Field
	StartLine int
}

// Field is a value together with the source position it came from, so
// a validation error can point at exactly what's wrong instead of
// just naming the field.
type Field struct {
	Value string
	Pos   Position
}

// knownKeys are the fields this tool understands and validates. Fields
// outside this set are passed through unvalidated unless --strict is
// set, since real batch files often carry extra bookkeeping fields
// (a carrier reference number, an internal note) that this tool has
// no opinion about.
var knownKeys = map[string]bool{
	"to":      true,
	"from":    true,
	"weight":  true,
	"dims":    true,
	"service": true,
}

var requiredKeys = []string{"to", "from", "weight", "dims"}

var (
	weightRe = regexp.MustCompile(`^(\d+(\.\d+)?)\s*(lb|oz|kg|g)$`)
	dimsRe   = regexp.MustCompile(`^(\d+(?:\.\d+)?)x(\d+(?:\.\d+)?)x(\d+(?:\.\d+)?)\s*(in|cm)$`)
	zipRe    = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

var validServices = map[string]bool{
	"ground":    true,
	"priority":  true,
	"express":   true,
	"overnight": true,
}

// serviceLimit is the largest weight and longest single side a carrier
// will take for a given service tier. These mirror the published limits
// for a typical ground/air split (an air tier tops out lighter and
// smaller than ground because it rides on a plane, not a truck).
type serviceLimit struct {
	maxWeightLb float64
	maxDimIn    float64
}

var serviceLimits = map[string]serviceLimit{
	"ground":    {maxWeightLb: 150, maxDimIn: 108},
	"priority":  {maxWeightLb: 70, maxDimIn: 108},
	"express":   {maxWeightLb: 150, maxDimIn: 108},
	"overnight": {maxWeightLb: 70, maxDimIn: 96},
}

// ParseFile parses a batch of labels and validates every field along
// the way. It never bails on the first problem: it collects every
// error it finds so one run can report everything wrong with a file,
// not just the first thing that broke.
//
// When strict is true, fields outside knownKeys are reported as
// errors instead of being passed through.
func ParseFile(filename string, data []byte, strict bool) ([]Label, []*LabelError) {
	lines := strings.Split(string(data), "\n")

	var labels []Label
	var errs []*LabelError

	current := Label{Fields: map[string]Field{}}
	open := false

	flush := func() {
		if !open {
			return
		}
		errs = append(errs, validateLabel(filename, current, lines)...)
		labels = append(labels, current)
		current = Label{Fields: map[string]Field{}}
		open = false
	}

	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		keyStart := 0
		for keyStart < len(line) && (line[keyStart] == ' ' || line[keyStart] == '\t') {
			keyStart++
		}

		colonIdx := strings.IndexByte(line, ':')
		if colonIdx == -1 {
			errs = append(errs, &LabelError{
				File:       filename,
				Pos:        Position{Line: lineNo, Col: len(line) + 1},
				SourceLine: line,
				Message:    `expected "key: value" but found no ':' on this line`,
			})
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[keyStart:colonIdx]))
		if key == "" {
			errs = append(errs, &LabelError{
				File:       filename,
				Pos:        Position{Line: lineNo, Col: keyStart + 1},
				SourceLine: line,
				Message:    "field name is empty before ':'",
			})
			continue
		}

		if !open {
			current.StartLine = lineNo
			open = true
		}

		if strict && !knownKeys[key] {
			errs = append(errs, &LabelError{
				File:       filename,
				Pos:        Position{Line: lineNo, Col: keyStart + 1},
				SourceLine: line,
				Message:    fmt.Sprintf("unknown field %q rejected by --strict (expected one of: to, from, weight, dims, service)", key),
			})
			continue
		}

		if _, dup := current.Fields[key]; dup {
			errs = append(errs, &LabelError{
				File:       filename,
				Pos:        Position{Line: lineNo, Col: keyStart + 1},
				SourceLine: line,
				Message:    fmt.Sprintf("field %q is already set for this label", key),
			})
			continue
		}

		valueStart := colonIdx + 1
		for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
			valueStart++
		}
		value := strings.TrimSpace(line[colonIdx+1:])

		col := valueStart + 1
		if value == "" {
			col = colonIdx + 2
		}

		current.Fields[key] = Field{Value: value, Pos: Position{Line: lineNo, Col: col}}
	}
	flush()

	return labels, errs
}

func validateLabel(filename string, label Label, lines []string) []*LabelError {
	var errs []*LabelError

	for _, key := range requiredKeys {
		if _, ok := label.Fields[key]; !ok {
			errs = append(errs, &LabelError{
				File:       filename,
				Pos:        Position{Line: label.StartLine, Col: 1},
				SourceLine: lineAt(lines, label.StartLine),
				Message:    fmt.Sprintf("label is missing required field %q", key),
			})
		}
	}

	if f, ok := label.Fields["to"]; ok {
		if err := validateAddress(f.Value); err != nil {
			errs = append(errs, fieldError(filename, f, lines, err))
		}
	}
	if f, ok := label.Fields["from"]; ok {
		if err := validateAddress(f.Value); err != nil {
			errs = append(errs, fieldError(filename, f, lines, err))
		}
	}

	// Service defaults to ground (the loosest limits) when the field is
	// left out. If it's present but doesn't name a known service, that's
	// already reported below, and there's no sensible limit to check
	// weight and dims against, so the limit checks are skipped.
	service := "ground"
	serviceKnown := true
	if f, ok := label.Fields["service"]; ok {
		if err := validateService(f.Value); err != nil {
			errs = append(errs, fieldError(filename, f, lines, err))
			serviceKnown = false
		} else {
			service = strings.ToLower(f.Value)
		}
	}

	if f, ok := label.Fields["weight"]; ok {
		if err := validateWeight(f.Value); err != nil {
			errs = append(errs, fieldError(filename, f, lines, err))
		} else if serviceKnown {
			if err := validateWeightLimit(f.Value, service); err != nil {
				errs = append(errs, fieldError(filename, f, lines, err))
			}
		}
	}
	if f, ok := label.Fields["dims"]; ok {
		if err := validateDims(f.Value); err != nil {
			errs = append(errs, fieldError(filename, f, lines, err))
		} else if serviceKnown {
			if err := validateDimsLimit(f.Value, service); err != nil {
				errs = append(errs, fieldError(filename, f, lines, err))
			}
		}
	}

	return errs
}

func fieldError(filename string, f Field, lines []string, err error) *LabelError {
	return &LabelError{
		File:       filename,
		Pos:        f.Pos,
		SourceLine: lineAt(lines, f.Pos.Line),
		Message:    err.Error(),
	}
}

func validateAddress(v string) error {
	fields := strings.Fields(v)
	if len(fields) == 0 {
		return fmt.Errorf("address is empty")
	}
	last := fields[len(fields)-1]
	if !zipRe.MatchString(last) {
		return fmt.Errorf("address %q does not end with a valid US zip code (expected 5 digits or 5+4, like \"62704\" or \"62704-1234\")", v)
	}
	return nil
}

func validateWeight(v string) error {
	m := weightRe.FindStringSubmatch(v)
	if m == nil {
		return fmt.Errorf("invalid weight %q: expected a number followed by a unit (lb, oz, kg, or g), like \"4.5 lb\"", v)
	}
	amount, err := strconv.ParseFloat(m[1], 64)
	if err != nil || amount <= 0 {
		return fmt.Errorf("invalid weight %q: amount must be greater than zero", v)
	}
	return nil
}

func validateDims(v string) error {
	if !dimsRe.MatchString(v) {
		return fmt.Errorf("invalid dimensions %q: expected LENGTHxWIDTHxHEIGHT followed by a unit (in or cm), like \"12x8x6 in\"", v)
	}
	return nil
}

// validateWeightLimit checks a weight that has already passed
// validateWeight against the given service's limit. It's called
// separately so a malformed weight produces one format error instead
// of a format error and a nonsensical limit error together.
func validateWeightLimit(v, service string) error {
	limit := serviceLimits[service]
	m := weightRe.FindStringSubmatch(v)
	amount, _ := strconv.ParseFloat(m[1], 64)
	lbs := weightToPounds(amount, m[3])
	if lbs > limit.maxWeightLb {
		return fmt.Errorf("weight %q (%.2f lb) exceeds the %s limit of %g lb", v, lbs, service, limit.maxWeightLb)
	}
	return nil
}

// validateDimsLimit checks dims that have already passed validateDims
// against the given service's longest-side limit.
func validateDimsLimit(v, service string) error {
	limit := serviceLimits[service]
	m := dimsRe.FindStringSubmatch(v)
	l, _ := strconv.ParseFloat(m[1], 64)
	w, _ := strconv.ParseFloat(m[2], 64)
	h, _ := strconv.ParseFloat(m[3], 64)
	longestIn := dimToInches(max(l, w, h), m[4])
	if longestIn > limit.maxDimIn {
		return fmt.Errorf("dimensions %q (longest side %.1f in) exceed the %s limit of %g in", v, longestIn, service, limit.maxDimIn)
	}
	return nil
}

func weightToPounds(amount float64, unit string) float64 {
	switch unit {
	case "oz":
		return amount / 16
	case "kg":
		return amount * 2.2046226218
	case "g":
		return amount * 2.2046226218 / 1000
	default: // lb
		return amount
	}
}

func dimToInches(amount float64, unit string) float64 {
	if unit == "cm" {
		return amount / 2.54
	}
	return amount
}

func validateService(v string) error {
	if !validServices[strings.ToLower(v)] {
		return fmt.Errorf("unknown service %q: must be one of ground, priority, express, overnight", v)
	}
	return nil
}

func lineAt(lines []string, n int) string {
	if n < 1 || n > len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[n-1], "\r")
}
