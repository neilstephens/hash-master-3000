// Copyright (c) 2025 Neil Stephens. All rights reserved.
// Use of this source code is governed by an MIT license that can be
// found in the LICENSE file.

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

type HashGenerator struct {
	descriptionEntry *widget.Entry
	masterPassEntry  *widget.Entry
	algorithmSelect  *widget.Select
	charRestSelect   *widget.Select
	lengthEntry      *widget.Entry
	iterationsEntry  *widget.Entry
	genButton        *widget.Button
	outputEntry      *widget.Entry
	app              fyne.App
	window           fyne.Window
	savedSettings    map[string]SavedSetting
	settingsList     *widget.List
	filterEntry      *widget.Entry
	filteredKeys     []string
	backupButton     *widget.Button
	mergeButton      *widget.Button
	restoreButton    *widget.Button
	hideZeroIterBox  *widget.Check
	copyToClipboard  *widget.Check
	appPrefs         AppPreferences
}

func main() {
	myApp := app.NewWithID("link.multifarious.hm3k")
	myApp.SetIcon(nil)
	myWindow := myApp.NewWindow("Hash Master 3000")
	myWindow.Resize(fyne.NewSize(400, 700))
	myWindow.SetCloseIntercept(func() {
		myApp.Quit()
	})

	generator := &HashGenerator{
		app:           myApp,
		window:        myWindow,
		savedSettings: make(map[string]SavedSetting),
		filteredKeys:  []string{},
		appPrefs:      AppPreferences{}, // Initialize preferences
	}
	generator.loadAppPreferences()

	appLife := myApp.Lifecycle()
	appLife.SetOnStarted(generator.loadSettings)

	generator.makeUIcomponents()
	myWindow.SetContent(generator.layoutUI())
	myWindow.Canvas().Focus(generator.filterEntry)
	myWindow.ShowAndRun()
}
