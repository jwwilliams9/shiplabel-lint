package main

import "testing"

func TestFixSource(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		out   string
		fixes int
	}{
		{"weight spacing", "weight: 3.2lb\n", "weight: 3.2 lb\n", 1},
		{"weight plural", "weight: 1.5lbs\n", "weight: 1.5 lb\n", 1},
		{"weight case", "Weight: 4 KG\n", "Weight: 4 kg\n", 1},
		{"dims spacing", "dims: 12x8x6in\n", "dims: 12x8x6 in\n", 1},
		{"dims upper x", "dims: 12X8X6 in\n", "dims: 12x8x6 in\n", 1},
		{"dims spaced x", "dims: 12 x 8 x 6 inches\n", "dims: 12x8x6 in\n", 1},
		{"already valid", "weight: 3 lb\ndims: 1x2x3 cm\n", "weight: 3 lb\ndims: 1x2x3 cm\n", 0},
		{"missing unit untouched", "weight: 3\n", "weight: 3\n", 0},
		{"other fields untouched", "to: 12 Main St 62704\nnotes: 5lbs\n", "to: 12 Main St 62704\nnotes: 5lbs\n", 0},
		{"comment untouched", "# weight: 3lb\n", "# weight: 3lb\n", 0},
		{"crlf preserved", "weight: 3lb\r\ndims: 1x2x3in\r\n", "weight: 3 lb\r\ndims: 1x2x3 in\r\n", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, fixes := FixSource([]byte(tc.in))
			if string(got) != tc.out {
				t.Errorf("output = %q, want %q", got, tc.out)
			}
			if len(fixes) != tc.fixes {
				t.Errorf("got %d fixes, want %d", len(fixes), tc.fixes)
			}
		})
	}
}

func TestFixSourcePosition(t *testing.T) {
	_, fixes := FixSource([]byte("to: x 62704\nweight:   1.5lbs\n"))
	if len(fixes) != 1 {
		t.Fatalf("got %d fixes, want 1", len(fixes))
	}
	f := fixes[0]
	if f.Line != 2 || f.Col != 10 || f.Old != "1.5lbs" || f.New != "1.5 lb" {
		t.Errorf("unexpected fix: %+v", f)
	}
}
