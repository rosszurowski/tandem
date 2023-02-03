// Package ansi provides ANSI escape codes for color and formatting.
package ansi

import (
	"fmt"
	"os"
)

// NoColor disables ANSI color output. By default it is set to true if the
// NO_COLOR environment variable is set.
var NoColor = os.Getenv("NO_COLOR") != ""

// Red returns a string wrapped in ANSI escape codes to make it red.
func Red(s string) string {
	if NoColor {
		return s
	}
	return "\033[0;31m" + s + "\033[0m"
}

// Gray returns a string wrapped in ANSI escape codes to make it gray.
func Gray(s string) string {
	if NoColor {
		return s
	}
	return "\033[0;38;5;8m" + s + "\033[0m"
}

// Dim returns a string wrapped in ANSI escape codes to make it dim.
func Dim(s string) string {
	if NoColor {
		return s
	}
	return "\033[0;2m" + s + "\033[0m"
}

// Bold returns a string wrapped in ANSI escape codes to make it bold.
func Bold(s string) string {
	if NoColor {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func ColorStart(i int) string {
	if NoColor {
		return ""
	}
	return fmt.Sprintf("\033[0;38;5;%vm", i)
}

func ColorEnd() string {
	if NoColor {
		return ""
	}
	return "\033[0m"
}
