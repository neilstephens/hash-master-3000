package main

//TODO: check box to hide/unhide zero iteration entries

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type SavedSetting struct {
	Description      string `json:"description"`
	Algorithm        string `json:"algorithm"`
	CharRestrictions string `json:"char_restrictions"`
	Length           string `json:"length"`
	Iterations       string `json:"iterations"`
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
	settingsFile     fyne.URI
	filterEntry      *widget.Entry
	filteredKeys     []string
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
	}

	appLife := myApp.Lifecycle()
	appLife.SetOnStarted(generator.loadSettings)

	generator.setupUI()
	myWindow.SetContent(generator.createUI())
	myWindow.ShowAndRun()
}

func (hg *HashGenerator) setupUI() {
	// Description token entry
	hg.descriptionEntry = widget.NewEntry()
	hg.descriptionEntry.SetPlaceHolder("Enter description token...")

	// Master password entry
	hg.masterPassEntry = widget.NewPasswordEntry()
	hg.masterPassEntry.SetPlaceHolder("Enter master password token...")
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
	}, nil)
	hg.algorithmSelect.SetSelected("SHA-256")

	// Character restriction selection
	hg.charRestSelect = widget.NewSelect([]string{
		"All generated chars",
		"Alphanumeric (replace others with underscore)",
		"Alphanumeric (omit others)",
		"Alpha only",
		"Numeric only",
	}, nil)
	hg.charRestSelect.SetSelected("Alphanumeric (replace others with underscore)")

	// Length entry
	hg.lengthEntry = widget.NewEntry()
	hg.lengthEntry.SetPlaceHolder("Enter desired length (e.g. 12)")
	hg.lengthEntry.SetText("12")

	// Iterations entry
	hg.iterationsEntry = widget.NewEntry()
	hg.iterationsEntry.SetPlaceHolder("Enter number of iterations (e.g. 1)")
	hg.iterationsEntry.SetText("1")

	// Generate button
	hg.genButton = widget.NewButton("Generate", hg.generateHash)
	hg.genButton.Importance = widget.HighImportance

	// Output entry
	hg.outputEntry = widget.NewPasswordEntry()
	hg.outputEntry.SetPlaceHolder("Hash will appear here...")
	hg.outputEntry.TextStyle = fyne.TextStyle{Monospace: true}

	// Filter entry for settings
	hg.filterEntry = widget.NewEntry()
	hg.filterEntry.SetPlaceHolder("Filter settings...")
	hg.filterEntry.OnChanged = func(text string) {
		hg.filterSettings(text)
	}

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
}

func (hg *HashGenerator) createUI() fyne.CanvasObject {
	// Create main form elements with labels
	form := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Description:"), nil, hg.descriptionEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Master Pass:"), nil, hg.masterPassEntry),
		hg.algorithmSelect,
		hg.charRestSelect,
		container.NewBorder(nil, nil, widget.NewLabel("Length:"), nil, hg.lengthEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Iterations:"), nil, hg.iterationsEntry),
		widget.NewSeparator(),
		hg.genButton,
		hg.outputEntry,
	)

	// Create panel for saved settings
	settingsPanel := container.NewBorder(
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Saved Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			hg.filterEntry,
		),
		nil, nil, nil,
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
	length, _ := strconv.Atoi(hg.lengthEntry.Text)
	if len(processed) > length {
		processed = processed[:length]
	}

	// Set the output
	hg.outputEntry.SetText(processed)

	// Copy to clipboard
	hg.app.Clipboard().SetContent(processed)

	// Show success message
	//dialog.ShowInformation("Success", "Hash generated and copied to clipboard!", hg.window)
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

// Settings persistence functions
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
	hg.saveSettingsToFile()

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
				hg.saveSettingsToFile()
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

func (hg *HashGenerator) saveSettingsToFile() error {
	data, err := json.MarshalIndent(hg.savedSettings, "", "  ")
	if err != nil {
		return err
	}

	writeCloser, err := storage.Writer(hg.settingsFile)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error opening URI for writing"), hg.window)
		return err
	}
	defer writeCloser.Close()

	_, err = writeCloser.Write(data)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error writing to URI"), hg.window)
		return err
	}

	return err
}

func (hg *HashGenerator) loadSettings() {
	//Check if we stored the URI of the settings file
	// and if we can still access it, otherwise ask again
	savedURIstring := hg.app.Preferences().StringWithFallback("settingsFileURI", "")
	if savedURIstring == "" {
		hg.promptForURI()
		return
	}

	var err error
	hg.settingsFile, err = storage.ParseURI(savedURIstring)
	if err != nil {
		errDialog := dialog.NewError(fmt.Errorf("invalid settings file URI"), hg.window)
		errDialog.SetOnClosed(func() {
			hg.promptForURI()
		})
		errDialog.Show()
		return
	}
	reader, err := storage.Reader(hg.settingsFile)
	if err != nil {
		errDialog := dialog.NewError(fmt.Errorf("error opening URI"), hg.window)
		errDialog.SetOnClosed(func() {
			hg.promptForURI()
		})
		errDialog.Show()
		return
	}

	hg.readSettingsFromFile(reader)
}

func (hg *HashGenerator) promptForURI() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			errDialog := dialog.NewError(fmt.Errorf("file dialog error"), hg.window)
			errDialog.SetOnClosed(func() {
				hg.promptForURI()
			})
			errDialog.Show()
			return
		}
		// Save the URI for future use
		hg.app.Preferences().SetString("settingsFileURI", reader.URI().String())
		hg.settingsFile = reader.URI()
		hg.readSettingsFromFile(reader)
	}, hg.window)
}

func (hg *HashGenerator) readSettingsFromFile(reader fyne.URIReadCloser) {

	defer reader.Close()

	// Read the file's contents
	data, err := io.ReadAll(reader)
	if err != nil {
		errDialog := dialog.NewError(fmt.Errorf("error reading URI"), hg.window)
		errDialog.SetOnClosed(func() {
			hg.promptForURI()
		})
		errDialog.Show()
		return
	}

	err = json.Unmarshal(data, &hg.savedSettings)
	if err != nil {
		errDialog := dialog.NewError(fmt.Errorf("error parsing settings file"), hg.window)
		errDialog.SetOnClosed(func() {
			hg.promptForURI()
		})
		errDialog.Show()
		return
	}
	hg.filterSettings(hg.filterEntry.Text)
}
