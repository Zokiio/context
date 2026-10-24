package factory

import "testing"

func TestProtectedMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern, file string
		want          bool
	}{
		{"tests/**", "tests/a.go", true},
		{"tests/**", "tests/deep/b.go", true},
		{"tests/**", "src/tests/a.go", false},
		{"**/*_test.go", "a_test.go", true},
		{"**/*_test.go", "internal/x/a_test.go", true},
		{"**/*_test.go", "internal/x/a.go", false},
		{"*_test.go", "internal/a_test.go", false},
		{"go.mod", "go.mod", true},
		{"go.mod", "sub/go.mod", false},
		{"testdata/**/golden.json", "testdata/golden.json", true},
		{"testdata/**/golden.json", "testdata/a/b/golden.json", true},
	}
	for _, tc := range cases {
		if got := protectedMatch(tc.pattern, tc.file); got != tc.want {
			t.Errorf("protectedMatch(%q, %q) = %v, want %v", tc.pattern, tc.file, got, tc.want)
		}
	}
}
