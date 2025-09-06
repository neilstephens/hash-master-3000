package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// A custom theme that only overrides the text size, to make the output easier to read
type hashTheme struct {
	fyne.Theme
	scale float32
}

func NewHashTheme(scale float32) fyne.Theme {
	return &hashTheme{
		Theme: theme.DefaultTheme(),
		scale: scale,
	}
}

func (t *hashTheme) Size(name fyne.ThemeSizeName) float32 {
	baseSize := t.Theme.Size(name)

	// Only multiply text-related sizes
	switch name {
	case theme.SizeNameText:
		return baseSize * t.scale
	case theme.SizeNameHeadingText:
		return baseSize * t.scale
	case theme.SizeNameSubHeadingText:
		return baseSize * t.scale
	case theme.SizeNameCaptionText:
		return baseSize * t.scale
	default:
		// Return original size for non-text elements
		return baseSize
	}
}

func (t *hashTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.Theme.Font(style)
}

func (t *hashTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.Theme.Color(name, variant)
}

func (t *hashTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.Theme.Icon(name)
}
