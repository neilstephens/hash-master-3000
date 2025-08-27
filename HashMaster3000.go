package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
	lengthSelect     *widget.Select
	iterationsSelect *widget.Select
	outputEntry      *widget.Entry
	window           fyne.Window
	savedSettings    map[string]SavedSetting
	settingsList     *widget.List
	settingsFile     string
}

func main() {
	myApp := app.New()
	myApp.SetIcon(nil)
	myWindow := myApp.NewWindow("Hash Generator")
	myWindow.Resize(fyne.NewSize(800, 600))

	// Get settings file path
	homeDir, _ := os.UserHomeDir()
	settingsFile := filepath.Join(homeDir, ".hashmaster_settings.json")

	generator := &HashGenerator{
		window:        myWindow,
		savedSettings: make(map[string]SavedSetting),
		settingsFile:  settingsFile,
	}

	generator.loadSettings()
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

	// Length selection
	lengthOptions := make([]string, 0)
	for i := 8; i <= 128; i += 4 {
		lengthOptions = append(lengthOptions, strconv.Itoa(i))
	}
	hg.lengthSelect = widget.NewSelect(lengthOptions, nil)
	hg.lengthSelect.SetSelected("12")

	// Hash iterations selection
	iterationOptions := []string{"1", "2", "3", "5", "10", "100", "1000", "10000"}
	hg.iterationsSelect = widget.NewSelect(iterationOptions, nil)
	hg.iterationsSelect.SetSelected("1")

	// Output entry
	hg.outputEntry = widget.NewPasswordEntry()
	hg.outputEntry.SetPlaceHolder("Generated hash will appear here...")

	// Settings list
	hg.settingsList = widget.NewList(
		func() int {
			return len(hg.savedSettings)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, nil,
				container.NewHBox(
					widget.NewButton("Load", nil),
					widget.NewButton("Delete", nil),
				),
				widget.NewLabel("Setting"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			keys := hg.getSettingsKeys()
			if id >= len(keys) {
				return
			}
			key := keys[id]
			setting := hg.savedSettings[key]

			border := obj.(*fyne.Container)
			label := border.Objects[0].(*widget.Label)
			buttons := border.Objects[1].(*fyne.Container)
			loadBtn := buttons.Objects[0].(*widget.Button)
			deleteBtn := buttons.Objects[1].(*widget.Button)

			label.SetText(setting.Description)

			loadBtn.OnTapped = func() {
				hg.loadSetting(key)
			}

			deleteBtn.OnTapped = func() {
				hg.deleteSetting(key)
			}
		},
	)
}

func (hg *HashGenerator) createUI() fyne.CanvasObject {
	// Create main form elements with labels
	form := container.NewVBox(
		widget.NewLabel("Description Token:"),
		hg.descriptionEntry,
		widget.NewLabel("Master Password Token:"),
		hg.masterPassEntry,
		widget.NewLabel("Hashing Algorithm:"),
		hg.algorithmSelect,
		widget.NewLabel("Character Restrictions:"),
		hg.charRestSelect,
		widget.NewLabel("Length Truncation:"),
		hg.lengthSelect,
		widget.NewLabel("Hash Iterations:"),
		hg.iterationsSelect,
		widget.NewSeparator(),
		widget.NewButton("Generate", hg.generateHash),
		widget.NewSeparator(),
		widget.NewLabel("Generated Output:"),
		hg.outputEntry,
	)

	// Create side panel for saved settings
	sidePanel := container.NewVBox(
		widget.NewLabel("Saved Settings:"),
		widget.NewSeparator(),
		container.NewScroll(hg.settingsList),
	)
	sidePanel.Resize(fyne.NewSize(250, 0))

	// Create horizontal split with main form and settings panel
	content := container.NewHSplit(form, sidePanel)
	content.SetOffset(0.7) // Make form take up 70% of width

	return content
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
	hash, err := hg.getHashWithIterations(combined, hg.algorithmSelect.Selected, hg.iterationsSelect.Selected)
	if err != nil {
		dialog.ShowError(fmt.Errorf("hashing failed: %v", err), hg.window)
		return
	}

	// Apply character restrictions
	processed := hg.applyCharacterRestrictions(hash, hg.charRestSelect.Selected)

	// Apply length restriction
	length, _ := strconv.Atoi(hg.lengthSelect.Selected)
	if len(processed) > length {
		processed = processed[:length]
	}

	// Set the output
	hg.outputEntry.SetText(processed)

	// Copy to clipboard
	hg.window.Clipboard().SetContent(processed)

	// Show success message
	//dialog.ShowInformation("Success", "Hash generated and copied to clipboard!", hg.window)
}

func (hg *HashGenerator) getHashWithIterations(input, algorithm, iterations string) (string, error) {
	iterCount, err := strconv.Atoi(iterations)
	if err != nil {
		return "", fmt.Errorf("invalid iteration count: %s", iterations)
	}

	result := input
	for i := 0; i < iterCount; i++ {
		result, err = hg.getHash(result, algorithm)
		if err != nil {
			return "", fmt.Errorf("iteration %d failed: %v", i+1, err)
		}
	}
	return result, nil
}

func (hg *HashGenerator) getHash(input, algorithm string) (string, error) {
	var cmd *exec.Cmd

	switch algorithm {
	case "SHA-256":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | sha256sum | cut -d' ' -f1", input))
	case "SHA-512":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | sha512sum | cut -d' ' -f1", input))
	case "SHA-1":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | sha1sum | cut -d' ' -f1", input))
	case "MD5":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | md5sum | cut -d' ' -f1", input))
	case "SHA-224":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | sha224sum | cut -d' ' -f1", input))
	case "SHA-384":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo -n '%s' | sha384sum | cut -d' ' -f1", input))
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
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
		// If no numbers found, generate some from the hash
		if result == "" {
			return hg.hashToNumbers(hash)
		}
		return result
	default:
		return hash
	}
}

func (hg *HashGenerator) hashToNumbers(hash string) string {
	result := ""
	for _, char := range hash {
		if char >= '0' && char <= '9' {
			result += string(char)
		} else if char >= 'a' && char <= 'f' {
			// Convert hex digits to numbers
			result += strconv.Itoa(int(char-'a') + 10)
		} else if char >= 'A' && char <= 'F' {
			result += strconv.Itoa(int(char-'A') + 10)
		}
	}
	if result == "" {
		result = "123456" // Fallback
	}
	return result
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
		Length:           hg.lengthSelect.Selected,
		Iterations:       hg.iterationsSelect.Selected,
	}

	hg.savedSettings[description] = setting
	hg.saveSettingsToFile()
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
	hg.lengthSelect.SetSelected(setting.Length)
	hg.iterationsSelect.SetSelected(setting.Iterations)
}

func (hg *HashGenerator) deleteSetting(key string) {
	dialog.ShowConfirm("Delete Setting",
		fmt.Sprintf("Are you sure you want to delete the setting for '%s'?", key),
		func(confirmed bool) {
			if confirmed {
				delete(hg.savedSettings, key)
				hg.saveSettingsToFile()
				hg.settingsList.Refresh()
			}
		}, hg.window)
}

func (hg *HashGenerator) getSettingsKeys() []string {
	keys := make([]string, 0, len(hg.savedSettings))
	for key := range hg.savedSettings {
		keys = append(keys, key)
	}
	return keys
}

func (hg *HashGenerator) saveSettingsToFile() error {
	data, err := json.MarshalIndent(hg.savedSettings, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(hg.settingsFile, data, 0644)
}

func (hg *HashGenerator) loadSettings() error {
	if _, err := os.Stat(hg.settingsFile); os.IsNotExist(err) {
		return nil // File doesn't exist yet, that's OK
	}

	data, err := ioutil.ReadFile(hg.settingsFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &hg.savedSettings)
}
