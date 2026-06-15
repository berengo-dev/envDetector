// Package ui renders the check results to the terminal with color and layout.
package ui

import (
	"fmt"
	"strings"

	"env-doctor/internal/checker"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)

	passStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	failStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("250"))

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

const (
	nameWidth     = 22
	statusWidth   = 12
	expectedWidth = 18
	actualWidth   = 18
	messageWidth  = 40
)

// Render prints the results table and final summary to stdout.
func Render(results []checker.Result) {
	fmt.Println(headerStyle.Render("env-doctor environment health check"))
	fmt.Println()

	fmt.Println(rowStyle.Render(formatRow(
		"Check",
		"Status",
		"Expected",
		"Actual",
		"Message",
		true,
	)))
	fmt.Println(rowStyle.Render(strings.Repeat("─", nameWidth+statusWidth+expectedWidth+actualWidth+messageWidth+12)))

	passes, fails := 0, 0
	for _, r := range results {
		if r.Status == checker.StatusPass {
			passes++
		} else {
			fails++
		}
		fmt.Println(formatResult(r))
	}

	fmt.Println()
	var summary string
	if fails == 0 {
		summary = fmt.Sprintf("%d checks passed, %d checks failed, 0 warnings", passes, fails)
		fmt.Println(passStyle.Render(summary))
	} else {
		summary = fmt.Sprintf("%d checks passed, %d checks failed, 0 warnings", passes, fails)
		fmt.Println(failStyle.Render(summary))
	}
}

func formatResult(r checker.Result) string {
	status := string(r.Status)
	if r.Status == checker.StatusPass {
		status = passStyle.Render("✅ " + status)
	} else {
		status = failStyle.Render("❌ " + status)
	}

	return formatRow(
		r.Name,
		status,
		r.Expected,
		r.Actual,
		r.Message,
		false,
	)
}

func formatRow(name, status, expected, actual, message string, header bool) string {
	if header {
		name = labelStyle.Render(name)
		status = labelStyle.Render(status)
		expected = labelStyle.Render(expected)
		actual = labelStyle.Render(actual)
		message = labelStyle.Render(message)
	}

	return fmt.Sprintf("  %s  %s  %s  %s  %s",
		pad(name, nameWidth),
		pad(status, statusWidth),
		pad(expected, expectedWidth),
		pad(actual, actualWidth),
		pad(message, messageWidth),
	)
}

func pad(s string, width int) string {
	if lipgloss.Width(s) > width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for _, r := range runes {
		if lipgloss.Width(b.String()+string(r)) >= width-1 {
			b.WriteString("…")
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}
