// Copyright (c) 2025 Neil Stephens. All rights reserved.
// Use of this source code is governed by an MIT license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type MergeState struct {
	mergedSettings map[string]SavedSetting
	keys           []string
	index          int
	addedCount     int
	abortMerge     bool
}

// Backup settings to file
func (hg *HashGenerator) backupSettings() {
	if len(hg.savedSettings) == 0 {
		dialog.ShowInformation("No Settings", "No settings to backup.", hg.window)
		return
	}

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			if err != nil {
				dialog.ShowError(fmt.Errorf("backup failed: %v", err), hg.window)
			}
			return
		}
		defer writer.Close()

		data, err := json.MarshalIndent(hg.savedSettings, "", "  ")
		if err != nil {
			dialog.ShowError(fmt.Errorf("error encoding settings: %v", err), hg.window)
			return
		}

		_, err = writer.Write(data)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error writing backup file: %v", err), hg.window)
			return
		}

		dialog.ShowInformation("Backup Complete", "Settings backed up successfully!", hg.window)
	}, hg.window)
}

// Restore settings from file
func (hg *HashGenerator) restoreSettings() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			if err != nil {
				dialog.ShowError(fmt.Errorf("restore failed: %v", err), hg.window)
			}
			return
		}
		defer reader.Close()

		// Read the file's contents
		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error reading backup file: %v", err), hg.window)
			return
		}

		// Parse the settings
		var restoredSettings map[string]SavedSetting
		err = json.Unmarshal(data, &restoredSettings)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error parsing backup file: %v", err), hg.window)
			return
		}

		// Confirm restore operation
		dialog.ShowConfirm("Restore Settings",
			fmt.Sprintf("This will replace your current %d settings with %d settings from the backup file. Continue?",
				len(hg.savedSettings), len(restoredSettings)),
			func(confirmed bool) {
				if confirmed {
					hg.savedSettings = restoredSettings
					hg.saveSettingsToPreferences()
					hg.updateFilteredKeys(hg.filterEntry.Text)
					hg.settingsList.Refresh()
					dialog.ShowInformation("Restore Complete",
						fmt.Sprintf("Successfully restored %d settings!", len(restoredSettings)), hg.window)
				}
			}, hg.window)
	}, hg.window)
}

// Merge settings from file
func (hg *HashGenerator) mergeSettings() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			if err != nil {
				dialog.ShowError(fmt.Errorf("merge failed: %v", err), hg.window)
			}
			return
		}
		defer reader.Close()

		// Read the file's contents
		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error reading backup file: %v", err), hg.window)
			return
		}

		// Parse the settings
		var importedSettings map[string]SavedSetting
		err = json.Unmarshal(data, &importedSettings)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error parsing backup file: %v", err), hg.window)
			return
		}
		hg.recursiveMerge(importedSettings, nil)
	}, hg.window)
}

func (hg *HashGenerator) recursiveMerge(importedSettings map[string]SavedSetting, m *MergeState) {
	if m == nil {
		// Initialize merge state
		m = &MergeState{
			mergedSettings: make(map[string]SavedSetting),
			keys:           make([]string, 0, len(importedSettings)),
			index:          0,
			addedCount:     0,
			abortMerge:     false,
		}
		// Copy existing settings
		for k, v := range hg.savedSettings {
			m.mergedSettings[k] = v
		}
		// Prepare keys for iteration
		for k := range importedSettings {
			m.keys = append(m.keys, k)
		}
		sort.Strings(m.keys)
	} else if m.index >= len(m.keys) {
		if m.abortMerge {
			dialog.ShowInformation("Merge Aborted",
				"Merge operation was aborted by the user.", hg.window)
			return
		}
		hg.savedSettings = m.mergedSettings
		hg.saveSettingsToPreferences()
		hg.updateFilteredKeys(hg.filterEntry.Text)
		hg.settingsList.Refresh()
		dialog.ShowInformation("Merge Complete",
			fmt.Sprintf("Successfully merged %d new/changed settings!", m.addedCount), hg.window)
		return
	}

	key := m.keys[m.index]
	newSetting := importedSettings[key]
	existingSetting, exists := m.mergedSettings[key]

	// Merge settings
	// - ignore duplicates (all fields must match)
	// - add new unique settings
	// - prompt if any conflicts (same description, different fields)
	//   - show side-by-side comparison (highlight different fields), options "skip", "overwrite", "cancel merge"

	if !exists {
		// New unique setting, add it
		m.mergedSettings[key] = newSetting
		m.addedCount++
		m.index++
		hg.recursiveMerge(importedSettings, m)
		return
	}

	// Check if all fields match
	if existingSetting == newSetting {
		// Duplicate, ignore
		m.index++
		hg.recursiveMerge(importedSettings, m)
		return
	}

	// Conflict detected, show comparison dialog
	// TODO: figure out how to add a third "Cancel Merge" option that aborts the entire merge process (set m.abortMerge = true and return)
	var overwrite bool
	var keepCheck *widget.Check
	overwriteCheck := widget.NewCheck("", func(checked bool) {
		overwrite = checked
		keepCheck.SetChecked(!checked)
	})
	keepCheck = widget.NewCheck("", func(checked bool) {
		overwrite = !checked
		overwriteCheck.SetChecked(!checked)
	})
	keepCheck.SetChecked(true) // default to keep existing

	dialogContent := container.NewVBox(
		hg.conflictContent(existingSetting, newSetting),
		container.NewGridWithColumns(3,
			widget.NewLabelWithStyle("Choose:", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			keepCheck,
			overwriteCheck,
		),
	)

	dialog.ShowCustomConfirm(key, "Continue", "Abort Merge", dialogContent,
		func(confirmed bool) {
			if !confirmed {
				m.abortMerge = true
				return
			}
			// Apply user's choice
			if overwrite {
				// Overwrite existing setting
				m.mergedSettings[key] = newSetting
				m.addedCount++
			} // else skip
			m.index++
			hg.recursiveMerge(importedSettings, m)
		}, hg.window)
}

func (hg *HashGenerator) conflictContent(existing, newSetting SavedSetting) fyne.CanvasObject {

	//a container for the matching fields (for context)
	contextList := container.NewVBox()

	// and a grid for the differing fields
	diffList := container.NewVBox(container.NewGridWithColumns(3,
		// Grid Headers
		widget.NewLabelWithStyle("Conflict", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Existing", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Imported", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	))

	// A helper to compare and add to the appropriate container
	addRow := func(label, existingVal, newVal string) {
		if existingVal == newVal {
			line := widget.NewLabel(fmt.Sprintf("%s%s", label, existingVal))
			line.Wrapping = fyne.TextWrapBreak
			contextList.Add(line)
		} else {
			comparisonGrid := container.NewGridWithColumns(3)
			comparisonGrid.Add(widget.NewLabel(label))
			existingWrap := widget.NewLabelWithStyle(existingVal, fyne.TextAlignLeading, fyne.TextStyle{})
			existingWrap.Wrapping = fyne.TextWrapBreak
			newWrap := widget.NewLabelWithStyle(newVal, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			newWrap.Wrapping = fyne.TextWrapBreak
			comparisonGrid.Add(existingWrap)
			comparisonGrid.Add(newWrap)
			diffList.Add(comparisonGrid)
		}
	}

	// Add rows for each field. Algorithm and CharRestrictions don't need labels
	addRow("", existing.Algorithm, newSetting.Algorithm)
	addRow("", existing.CharRestrictions, newSetting.CharRestrictions)
	addRow("Length: ", existing.Length, newSetting.Length)
	addRow("Iterations: ", existing.Iterations, newSetting.Iterations)

	return container.NewVBox(
		contextList,
		widget.NewSeparator(),
		diffList,
	)
}
