package main

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (hg *HashGenerator) makeUIcomponents() {
	// Description token entry
	hg.descriptionEntry = widget.NewEntry()
	hg.descriptionEntry.SetPlaceHolder("Enter description...")
	hg.descriptionEntry.SetText(hg.appPrefs.LastDescription)
	hg.descriptionEntry.OnChanged = func(text string) {
		// Save preference when changed
		hg.appPrefs.LastDescription = text
		hg.saveAppPreferences()
	}
	hg.descriptionEntry.Validator = func(text string) error {
		if text == "" {
			return fmt.Errorf("description cannot be empty")
		}
		return nil
	}
	// Force validation on startup if empty
	hg.descriptionEntry.FocusGained()
	hg.descriptionEntry.FocusLost()

	// Master password entry
	hg.masterPassEntry = widget.NewPasswordEntry()
	hg.masterPassEntry.SetPlaceHolder("Enter master pass...")
	hg.masterPassEntry.TextStyle = fyne.TextStyle{Monospace: true}
	// Pressing Return will trigger the same action as "Generate"
	hg.masterPassEntry.OnSubmitted = func(_ string) {
		hg.generateHash()
	}
	hg.masterPassEntry.Validator = func(text string) error {
		if text == "" {
			return fmt.Errorf("master password cannot be empty")
		}
		return nil
	}
	// Force validation on startup to make it clear it's required
	hg.masterPassEntry.FocusGained()
	hg.masterPassEntry.FocusLost()

	// Algorithm selection
	hg.algorithmSelect = widget.NewSelect([]string{
		"SHA-256",
		"SHA-512",
		"SHA-1",
		"MD5",
		"SHA-224",
		"SHA-384",
	}, func(selected string) {
		// Save preference when changed
		hg.appPrefs.LastAlgorithm = selected
		hg.saveAppPreferences()
	})
	hg.algorithmSelect.SetSelected(hg.appPrefs.LastAlgorithm)

	// Character restriction selection
	hg.charRestSelect = widget.NewSelect([]string{
		"All generated chars",
		"Alphanumeric (replace others with underscore)",
		"Alphanumeric (omit others)",
		"Alpha only",
		"Numeric only",
	}, func(selected string) {
		// Save preference when changed
		hg.appPrefs.LastCharRest = selected
		hg.saveAppPreferences()
	})
	hg.charRestSelect.SetSelected(hg.appPrefs.LastCharRest)

	// Length entry
	hg.lengthEntry = widget.NewEntry()
	hg.lengthEntry.SetPlaceHolder("Num char")
	hg.lengthEntry.SetText(hg.appPrefs.LastLength)
	hg.lengthEntry.OnChanged = func(text string) {
		// Save preference when changed
		hg.appPrefs.LastLength = text
		hg.saveAppPreferences()
	}
	hg.lengthEntry.Validator = func(text string) error {
		// non-negative integer or empty (means no length restriction)
		if text == "" {
			return nil
		}
		if num, err := strconv.Atoi(text); err != nil || num < 0 {
			return fmt.Errorf("length must be a non-negative integer")
		}
		return nil
	}
	// Force validation on startup
	hg.lengthEntry.FocusGained()
	hg.lengthEntry.FocusLost()

	// Iterations entry
	hg.iterationsEntry = widget.NewEntry()
	hg.iterationsEntry.SetPlaceHolder("Num hashes")
	hg.iterationsEntry.SetText(hg.appPrefs.LastIter)
	hg.iterationsEntry.OnChanged = func(text string) {
		// Save preference when changed
		hg.appPrefs.LastIter = text
		hg.saveAppPreferences()
	}
	hg.iterationsEntry.Validator = func(text string) error {
		// Must be a positive integer
		if num, err := strconv.Atoi(text); err != nil || num < 1 {
			return fmt.Errorf("iterations must be a positive integer")
		}
		return nil
	}
	// Force validation on startup
	hg.iterationsEntry.FocusGained()
	hg.iterationsEntry.FocusLost()

	// Generate button
	hg.genButton = widget.NewButton("Generate", hg.generateHash)
	hg.genButton.Importance = widget.HighImportance

	// Output entry
	hg.outputEntry = widget.NewPasswordEntry()
	hg.outputEntry.SetPlaceHolder("Hash will appear here...")
	hg.outputEntry.TextStyle = fyne.TextStyle{Monospace: true}

	// Filter entry for settings
	hg.filterEntry = widget.NewEntry()
	hg.filterEntry.SetText(hg.appPrefs.LastFilter)
	hg.filterEntry.SetPlaceHolder("Filter settings...")
	hg.filterEntry.OnChanged = func(text string) {
		hg.appPrefs.LastFilter = text
		hg.saveAppPreferences()
		hg.filterSettings(text)
	}

	// Backup and Restore buttons
	hg.backupButton = widget.NewButton("Backup", hg.backupSettings)
	hg.mergeButton = widget.NewButton("Merge", hg.mergeSettings)
	hg.restoreButton = widget.NewButton("Restore", hg.restoreSettings)

	// Initialize filtered keys
	hg.updateFilteredKeys("")

	// Settings list
	hg.settingsList = widget.NewList(
		func() int {
			return len(hg.filteredKeys)
		},
		func() fyne.CanvasObject {
			return NewClickableLabel("ListTemplateItemDummyText")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(hg.filteredKeys) {
				return
			}
			key := hg.filteredKeys[id]
			setting := hg.savedSettings[key]
			label := obj.(*ClickableLabel)

			label.SetText(setting.Description)
			label.OnTapped = func() {
				hg.loadSetting(key)
			}
			label.OnTappedSecondary = func(pos *fyne.PointEvent) {
				menu := fyne.NewMenu("",
					fyne.NewMenuItem("Delete", func() {
						hg.deleteSetting(key)
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, hg.window.Canvas(), pos.AbsolutePosition)
			}
		},
	)

	// Checkbox to hide/unhide zero iteration settings (a way to mark "inactive" settings)
	hg.hideZeroIterBox = widget.NewCheck("Hide Inactive", func(checked bool) {
		hg.appPrefs.HideZeroIter = checked
		hg.saveAppPreferences()
		hg.filterSettings(hg.filterEntry.Text)
	})
	hg.hideZeroIterBox.SetChecked(hg.appPrefs.HideZeroIter)

	// Checkbox to enable/disable auto copy to clipboard
	hg.copyToClipboard = widget.NewCheck("Auto Copy", func(checked bool) {
		hg.appPrefs.CopyToClipboard = checked
		hg.saveAppPreferences()
	})
	hg.copyToClipboard.SetChecked(hg.appPrefs.CopyToClipboard)
}

func (hg *HashGenerator) layoutUI() fyne.CanvasObject {
	// Create main form elements with labels
	form := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Description:"), nil, hg.descriptionEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Master Pass:"), nil, hg.masterPassEntry),
		container.NewGridWithColumns(2,
			hg.algorithmSelect,
			container.NewBorder(nil, nil, widget.NewLabel("Iterations:"), nil, hg.iterationsEntry),
		),
		container.NewGridWithColumns(2,
			hg.charRestSelect,
			container.NewBorder(nil, nil, widget.NewLabel("Length:"), nil, hg.lengthEntry),
		),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, nil, hg.copyToClipboard, hg.genButton),
		container.NewThemeOverride(hg.outputEntry, NewHashTheme(1.6)),
	)

	// Create backup/restore buttons container
	backupRestoreContainer := container.NewGridWithColumns(3,
		hg.backupButton,
		hg.mergeButton,
		hg.restoreButton,
	)

	// Create panel for saved settings
	settingsPanel := container.NewBorder(
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Saved Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			container.NewBorder(nil, nil, nil, hg.hideZeroIterBox,
				hg.filterEntry,
			),
		),
		backupRestoreContainer,
		nil, nil,
		container.NewScroll(hg.settingsList),
	)

	// Create main border
	content := container.NewBorder(form, nil, nil, nil, settingsPanel)

	return content
}
