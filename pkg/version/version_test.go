package version

import "testing"

func TestExtract(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"go version go1.21.5 linux/amd64", "1.21.5"},
		{"v20.10.0", "20.10.0"},
		{"Docker version 24.0.7, build afdd53b", "24.0.7"},
		{"1.21", "1.21"},
		{"no version here", ""},
	}

	for _, c := range cases {
		got := Extract(c.input)
		if got != c.want {
			t.Errorf("Extract(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		actual   string
		expected string
		want     bool
	}{
		{"1.21.5", "1.21.5", true},
		{"1.21.5", "1.21", true},
		{"1.21.0", "1.21", true},
		{"1.22.0", "1.21", false},
		{"24.0.7", "24.x", true},
		{"24.0.7", "24.*", true},
		{"23.0.7", "24.x", false},
		{"24.0.7", "24.0.x", true},
		{"24.0.7", "24.1.x", false},
		{"v20.10.0", "20.x", true},
		{"go version go1.21.5 linux/amd64", "1.21.x", true},
		{"1.21", "1.21.5", false},
	}

	for _, c := range cases {
		got, actualClean, err := Match(c.actual, c.expected)
		if err != nil {
			t.Fatalf("Match(%q, %q) returned error: %v", c.actual, c.expected, err)
		}
		if got != c.want {
			t.Errorf("Match(%q, %q) = %v (actual=%q), want %v", c.actual, c.expected, got, actualClean, c.want)
		}
	}
}

func TestMatchNoVersion(t *testing.T) {
	_, _, err := Match("not a version", "1.21")
	if err == nil {
		t.Error("expected an error when no version is present")
	}
}

func TestMatchLatest(t *testing.T) {
	got, _, err := Match("1.21.5", "latest")
	if err != nil {
		t.Fatalf("Match returned error: %v", err)
	}
	if !got {
		t.Error("expected latest to match any version")
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"8.x", "9.x", -1},
		{"9.x", "8.x", 1},
		{"1.21", "1.22", -1},
		{"1.22", "1.21", 1},
		{"1.21.5", "1.21.0", 1},
		{"1.21.0", "1.21.5", -1},
		{"20.x", "20.x", 0},
		{"latest", "9.x", -1},
		{"9.x", "latest", 1},
		{"alpha", "beta", -1},
	}

	for _, c := range cases {
		got := Compare(c.a, c.b)
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestConvertSemverToWildcard(t *testing.T) {
	cases := []struct {
		input  string
		want   string
		wantOk bool
	}{
		{"^16.2.7", "16.x", true},
		{"~16.2.7", "16.x", true},
		{">=20.0.0", "20.x", true},
		{"1.21.5", "1.21", true},
		{"*", "", false},
		{"latest", "", false},
		{"^0.2.3", "0.2", true},
		{">=1.0.0", "1.x", true},
		{"^5", "5.x", true},
		{"v18.0.0", "18.0", true},
		{"", "", false},
		{"not-a-version", "", false},
	}

	for _, c := range cases {
		got, ok := ConvertSemverToWildcard(c.input)
		if ok != c.wantOk || got != c.want {
			t.Errorf("ConvertSemverToWildcard(%q) = (%q, %v), want (%q, %v)", c.input, got, ok, c.want, c.wantOk)
		}
	}
}
