package version

import (
	"fmt"
	"strconv"
	"strings"
)

func IsDevelop(v string) bool {
	lower := strings.ToLower(strings.TrimSpace(v))
	return lower == "develop"
}

func Parse(v string) (major, minor int, err error) {
	s := strings.TrimPrefix(v, "v")
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid version format %q: expected vX.Y or X.Y", v)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major in version %q: %w", v, err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor in version %q: %w", v, err)
	}
	return major, minor, nil
}

// Compare compares two version strings a and b.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Returns an error if either string cannot be parsed.
func Compare(a, b string) (int, error) {
	aMaj, aMin, err := Parse(a)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", a, err)
	}
	bMaj, bMin, err := Parse(b)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", b, err)
	}

	if aMaj != bMaj {
		if aMaj < bMaj {
			return -1, nil
		}
		return 1, nil
	}
	if aMin != bMin {
		if aMin < bMin {
			return -1, nil
		}
		return 1, nil
	}
	return 0, nil
}
