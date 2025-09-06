package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"fyne.io/fyne/v2/dialog"
)

type SavedSetting struct {
	Description      string `json:"description"`
	Algorithm        string `json:"algorithm"`
	CharRestrictions string `json:"char_restrictions"`
	Length           string `json:"length"`
	Iterations       string `json:"iterations"`
}

type AppPreferences struct {
	LastDescription string `json:"last_description"`
	LastFilter      string `json:"last_filter"`
	LastAlgorithm   string `json:"last_algorithm"`
	LastCharRest    string `json:"last_char_rest"`
	LastLength      string `json:"last_length"`
	LastIter        string `json:"last_iterations"`
	HideZeroIter    bool   `json:"hide_zero_iter"`
	CopyToClipboard bool   `json:"copy_to_clipboard"`
}

// Settings persistence functions using Fyne preferences
func (hg *HashGenerator) saveSetting(description string) {
	if description == "" {
		return
	}

	setting := SavedSetting{
		Description:      description,
		Algorithm:        hg.algorithmSelect.Selected,
		CharRestrictions: hg.charRestSelect.Selected,
		Length:           hg.lengthEntry.Text,
		Iterations:       hg.iterationsEntry.Text,
	}

	hg.savedSettings[description] = setting
	hg.saveSettingsToPreferences()

	// Update filtered keys and refresh the list
	hg.updateFilteredKeys(hg.filterEntry.Text)
	hg.settingsList.Refresh()
}

func (hg *HashGenerator) loadSetting(key string) {
	setting, exists := hg.savedSettings[key]
	if !exists {
		return
	}

	hg.descriptionEntry.SetText(setting.Description)
	hg.algorithmSelect.SetSelected(setting.Algorithm)
	hg.charRestSelect.SetSelected(setting.CharRestrictions)
	hg.lengthEntry.SetText(setting.Length)
	hg.iterationsEntry.SetText(setting.Iterations)
}

func (hg *HashGenerator) deleteSetting(key string) {
	dialog.ShowConfirm("Delete Setting",
		fmt.Sprintf("Are you sure you want to delete the setting for '%s'?", key),
		func(confirmed bool) {
			if confirmed {
				delete(hg.savedSettings, key)
				hg.saveSettingsToPreferences()
				hg.updateFilteredKeys(hg.filterEntry.Text)
				hg.settingsList.Refresh()
			}
		}, hg.window)
}

func (hg *HashGenerator) getSettingsKeys() []string {
	keys := make([]string, 0, len(hg.savedSettings))
	for key := range hg.savedSettings {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Sort alphabetically
	return keys
}

// Save settings to Fyne preferences
func (hg *HashGenerator) saveSettingsToPreferences() {
	data, err := json.Marshal(hg.savedSettings)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error encoding settings: %v", err), hg.window)
		return
	}
	hg.app.Preferences().SetString("savedSettings", string(data))
}

// Save app preferences (filter, last used settings, etc.)
func (hg *HashGenerator) saveAppPreferences() {
	data, err := json.Marshal(hg.appPrefs)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error encoding app preferences: %v", err), hg.window)
		return
	}
	hg.app.Preferences().SetString("appPreferences", string(data))
}

// Load app preferences and restore last used settings
func (hg *HashGenerator) loadAppPreferences() {

	//load defaults first
	hg.appPrefs = hg.DefaultAppPrefs()

	prefsData := hg.app.Preferences().StringWithFallback("appPreferences", "")
	if prefsData == "" {
		// No saved preferences, stay with defaults
		return
	}

	err := json.Unmarshal([]byte(prefsData), &hg.appPrefs)
	if err != nil {
		// Error parsing preferences, reload defaults
		hg.appPrefs = hg.DefaultAppPrefs()
		return
	}
}

func (hg *HashGenerator) DefaultAppPrefs() AppPreferences {
	return AppPreferences{
		LastDescription: "",
		LastAlgorithm:   "SHA-256",
		LastCharRest:    "Alphanumeric (replace others with underscore)",
		LastLength:      "12",
		LastIter:        "1",
		HideZeroIter:    true,
		CopyToClipboard: true,
		LastFilter:      "",
	}
}

// Load settings from Fyne preferences
func (hg *HashGenerator) loadSettings() {

	// Load saved settings
	settingsData := hg.app.Preferences().StringWithFallback("savedSettings", "")
	if settingsData == "" {
		// No saved settings, start with empty map
		hg.savedSettings = make(map[string]SavedSetting)
		hg.filterSettings(hg.filterEntry.Text)
		return
	}

	err := json.Unmarshal([]byte(settingsData), &hg.savedSettings)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error parsing saved settings: %v", err), hg.window)
		hg.savedSettings = make(map[string]SavedSetting)
	}

	hg.filterSettings(hg.filterEntry.Text)
}
