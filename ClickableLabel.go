package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// custom label that supports click events
type ClickableLabel struct {
	*widget.Label
	OnTapped          func()
	OnTappedSecondary func(*fyne.PointEvent)
}

func NewClickableLabel(text string) *ClickableLabel {
	label := &ClickableLabel{
		widget.NewLabel(text),
		func() {},                 // default no-op
		func(*fyne.PointEvent) {}, // default no-op
	}
	label.ExtendBaseWidget(label)
	return label
}
func (l *ClickableLabel) Tapped(_ *fyne.PointEvent) {
	l.OnTapped()
}
func (l *ClickableLabel) TappedSecondary(pos *fyne.PointEvent) {
	l.OnTappedSecondary(pos)
}
