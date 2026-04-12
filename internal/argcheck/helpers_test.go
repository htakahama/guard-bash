package argcheck

import "testing"

func TestHasShortFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		flag rune
		want bool
	}{
		{"single flag", []string{"-r"}, 'r', true},
		{"merged flags", []string{"-rf"}, 'r', true},
		{"merged flags second", []string{"-rf"}, 'f', true},
		{"not present", []string{"-v"}, 'r', false},
		{"after double dash", []string{"--", "-r"}, 'r', false},
		{"long flag ignored", []string{"--recursive"}, 'r', false},
		{"empty args", nil, 'r', false},
		{"non-flag arg", []string{"file.txt"}, 'r', false},
		{"uppercase R", []string{"-R"}, 'R', true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasShortFlag(tc.args, tc.flag)
			if got != tc.want {
				t.Errorf("hasShortFlag(%v, %c) = %v, want %v", tc.args, tc.flag, got, tc.want)
			}
		})
	}
}

func TestHasLongFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{"present", []string{"--force"}, "--force", true},
		{"not present", []string{"--verbose"}, "--force", false},
		{"after double dash", []string{"--", "--force"}, "--force", false},
		{"force-with-lease", []string{"--force-with-lease"}, "--force-with-lease", true},
		{"empty", nil, "--force", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasLongFlag(tc.args, tc.flag)
			if got != tc.want {
				t.Errorf("hasLongFlag(%v, %q) = %v, want %v", tc.args, tc.flag, got, tc.want)
			}
		})
	}
}

func TestIsBroadPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"~", true},
		{".", true},
		{"..", true},
		{"/home", true},
		{"/etc", true},
		{"/tmp", true},
		{"/usr/local", false},
		{"./subdir", false},
		{"foo", false},
		{"/home/user", false},
		{"/opt", true},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := isBroadPath(tc.path)
			if got != tc.want {
				t.Errorf("isBroadPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestNonFlagArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"no flags", []string{"a", "b"}, []string{"a", "b"}},
		{"with flags", []string{"-r", "a", "-f", "b"}, []string{"a", "b"}},
		{"after double dash", []string{"-r", "--", "-a"}, []string{"-a"}},
		{"empty", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nonFlagArgs(tc.args)
			if len(got) != len(tc.want) {
				t.Errorf("nonFlagArgs(%v) = %v, want %v", tc.args, got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("nonFlagArgs(%v)[%d] = %q, want %q", tc.args, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// EOF
