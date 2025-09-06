package main

import (
	"strconv"
	"strings"
)

func (hg *HashGenerator) filterSettings(filterText string) {
	hg.updateFilteredKeys(filterText)
	hg.settingsList.Refresh()
}

func (hg *HashGenerator) updateFilteredKeys(filterText string) {
	allKeys := hg.getSettingsKeys()
	hg.filteredKeys = []string{}

	filterLower := strings.ToLower(filterText)

	for _, key := range allKeys {
		setting := hg.savedSettings[key]

		// Check if we should hide hobbled settings
		if hg.hideZeroIterBox.Checked {
			iterations, err := strconv.Atoi(setting.Iterations)
			if err == nil && iterations <= 0 {
				continue // Skip this setting if it has 0 iterations and we're hiding them
			}
		}

		// Apply text filter
		if filterText == "" || strings.Contains(strings.ToLower(key), filterLower) {
			hg.filteredKeys = append(hg.filteredKeys, key)
		}
	}
}
