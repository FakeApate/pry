package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Tideline palette -- raw hex constants. See brand-spec.md in the tideline
// skill. Dark values match dark-mode role assignments; light values are one
// step darker to preserve contrast against paler surfaces.
const (
	// Neutrals -- the tidal column.
	tlTide = "#324749"
	tlFog  = "#97ADB1"

	// Ocean -- primary accent, deep teal.
	tlOcean400 = "#2F9BA4"
	tlOcean500 = "#1A6971"

	// Kelp -- success.
	tlKelp400 = "#3E9A68"
	tlKelp500 = "#2B7F52"

	// Sun -- warning.
	tlSun400 = "#D29A2E"
	tlSun700 = "#7A5310"

	// Undertow -- danger. 200 for body-copy AA on dark, 500 on light.
	tlUndertow200 = "#E48D90"
	tlUndertow500 = "#B23840"
)

// Semantic role resolvers. Pass the terminal dark-mode flag; returns the
// token for that mode. Accents shift one stop darker in light mode.

func accent(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlOcean500), lipgloss.Color(tlOcean400))
}

func success(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlKelp500), lipgloss.Color(tlKelp400))
}

// errorC returns the danger colour for text. Uses undertow 200 on dark to
// meet WCAG AA for body copy; undertow 500 on light.
func errorC(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlUndertow500), lipgloss.Color(tlUndertow200))
}

func warning(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlSun700), lipgloss.Color(tlSun400))
}

func muted(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlTide), lipgloss.Color(tlFog))
}

func border(dark bool) color.Color {
	return lipgloss.LightDark(dark)(lipgloss.Color(tlFog), lipgloss.Color(tlTide))
}

// tab bar
func tabBarStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(border(dark))
}

func tabStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(muted(dark))
}

func activeTabStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(accent(dark)).
		Bold(true).
		Underline(true)
}

// content areas
var styleContent = lipgloss.NewStyle().Padding(1, 2)

func modalStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent(dark)).
		Padding(1, 2)
}

// status / help footer
func statusBarStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(muted(dark)).
		Padding(0, 1)
}

func keyNameStyle(dark bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(accent(dark)).
		Bold(true)
}

var styleKeyHint = lipgloss.NewStyle()
