package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"image/color"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

type MergeState struct {
	mergedSettings map[string]SavedSetting
	keys           []string
	index          int
	addedCount     int
	abortMerge     bool
}

func main() {
	myApp := app.NewWithID("com.hashmaster3000.app")
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

	generator.setupUI()
	myWindow.SetContent(generator.createUI())
	myWindow.Canvas().Focus(generator.filterEntry)
	myWindow.ShowAndRun()
}

func (hg *HashGenerator) setupUI() {
	// Description token entry
	hg.descriptionEntry = widget.NewEntry()
	hg.descriptionEntry.SetPlaceHolder("Enter description...")
	hg.descriptionEntry.SetText(hg.appPrefs.LastDescription)
	hg.descriptionEntry.OnChanged = func(text string) {
		// Save preference when changed
		hg.appPrefs.LastDescription = text
		hg.saveAppPreferences()
	}

	// Master password entry
	hg.masterPassEntry = widget.NewPasswordEntry()
	hg.masterPassEntry.SetPlaceHolder("Enter master pass...")
	hg.masterPassEntry.TextStyle = fyne.TextStyle{Monospace: true}
	// Pressing Return will trigger the same action as "Generate"
	hg.masterPassEntry.OnSubmitted = func(_ string) {
		hg.generateHash()
	}

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

	// Iterations entry
	hg.iterationsEntry = widget.NewEntry()
	hg.iterationsEntry.SetPlaceHolder("Num hashes")
	hg.iterationsEntry.SetText(hg.appPrefs.LastIter)
	hg.iterationsEntry.OnChanged = func(text string) {
		// Save preference when changed
		hg.appPrefs.LastIter = text
		hg.saveAppPreferences()
	}

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
			return container.NewBorder(nil, nil, nil,
				widget.NewButton("Del", nil),
				widget.NewLabel("Setting"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(hg.filteredKeys) {
				return
			}
			key := hg.filteredKeys[id]
			setting := hg.savedSettings[key]

			border := obj.(*fyne.Container)
			label := border.Objects[0].(*widget.Label)
			deleteBtn := border.Objects[1].(*widget.Button)

			label.SetText(setting.Description)

			deleteBtn.OnTapped = func() {
				hg.deleteSetting(key)
			}
		},
	)

	// Load setting when item is selected
	hg.settingsList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(hg.filteredKeys) {
			key := hg.filteredKeys[id]
			hg.loadSetting(key)
		}
	}

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

func (hg *HashGenerator) createUI() fyne.CanvasObject {
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

		// Check if we should hide zero iteration settings
		if hg.hideZeroIterBox.Checked {
			iterations, err := strconv.Atoi(setting.Iterations)
			if err == nil && iterations == 0 {
				continue // Skip this setting if it has 0 iterations and we're hiding them
			}
		}

		// Apply text filter
		if filterText == "" || strings.Contains(strings.ToLower(key), filterLower) {
			hg.filteredKeys = append(hg.filteredKeys, key)
		}
	}
}

func (hg *HashGenerator) generateHash() {
	description := hg.descriptionEntry.Text
	masterPass := hg.masterPassEntry.Text

	if description == "" || masterPass == "" {
		dialog.ShowError(fmt.Errorf("both description and master password tokens are required"), hg.window)
		return
	}

	// Save current settings (but not master password or output)
	hg.saveSetting(description)

	// Combine the tokens
	combined := description + masterPass

	// Get the hash using system commands with iterations
	hash, err := hg.getHashWithIterations(combined, hg.algorithmSelect.Selected, hg.iterationsEntry.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("hashing failed: %v", err), hg.window)
		return
	}

	// Apply character restrictions
	processed := hg.applyCharacterRestrictions(hash, hg.charRestSelect.Selected)

	// Apply length restriction
	length, err := strconv.Atoi(hg.lengthEntry.Text)
	if err == nil && length > 0 && len(processed) > length {
		processed = processed[:length]
	}

	// Set the output
	hg.outputEntry.SetText(processed)

	// Copy to clipboard
	if hg.copyToClipboard.Checked {
		hg.app.Clipboard().SetContent(processed)
	}
}

func (hg *HashGenerator) getHashWithIterations(input string, algorithm string, iterations string) (string, error) {
	iterCount, err := strconv.Atoi(iterations)
	if err != nil || iterCount < 1 {
		return "", fmt.Errorf("invalid iteration count: %s", iterations)
	}

	result := []byte(input)
	for i := 0; i < iterCount; i++ {
		result, err = hg.getHash(result, algorithm)
		if err != nil {
			return "", fmt.Errorf("iteration %d failed: %v", i+1, err)
		}
	}

	return base64.StdEncoding.EncodeToString(result), nil
}

func (hg *HashGenerator) getHash(input []byte, algorithm string) ([]byte, error) {
	var h hash.Hash

	switch algorithm {
	case "SHA-256":
		h = sha256.New()
	case "SHA-512":
		h = sha512.New()
	case "SHA-1":
		h = sha1.New()
	case "MD5":
		h = md5.New()
	case "SHA-224":
		h = sha256.New224()
	case "SHA-384":
		h = sha512.New384()
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	_, err := h.Write(input)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (hg *HashGenerator) applyCharacterRestrictions(hash, restriction string) string {
	switch restriction {
	case "All generated chars":
		return hash
	case "Alphanumeric (replace others with underscore)":
		re := regexp.MustCompile(`[^a-zA-Z0-9]`)
		return re.ReplaceAllString(hash, "_")
	case "Alphanumeric (omit others)":
		re := regexp.MustCompile(`[^a-zA-Z0-9]`)
		return re.ReplaceAllString(hash, "")
	case "Alpha only":
		re := regexp.MustCompile(`[^a-zA-Z]`)
		return re.ReplaceAllString(hash, "")
	case "Numeric only":
		re := regexp.MustCompile(`[^0-9]`)
		result := re.ReplaceAllString(hash, "")
		return result
	default:
		return hash
	}
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
