package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type HashGenerator struct {
	descriptionEntry *widget.Entry
	masterPassEntry  *widget.Entry
	algorithmSelect  *widget.Select
	charRestSelect   *widget.Select
	lengthSelect     *widget.Select
	outputEntry      *widget.Entry
	maskCheck        *widget.Check
	window           fyne.Window
}

func main() {
	myApp := app.New()
	myApp.SetIcon(nil)
	myWindow := myApp.NewWindow("Hash Generator")
	myWindow.Resize(fyne.NewSize(600, 500))

	generator := &HashGenerator{window: myWindow}
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

	// Output entry
	hg.outputEntry = widget.NewPasswordEntry()
	hg.outputEntry.SetPlaceHolder("Generated hash will appear here...")
}

func (hg *HashGenerator) createUI() fyne.CanvasObject {
	// Create form elements with labels
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
		widget.NewSeparator(),
		widget.NewButton("Generate", hg.generateHash),
		widget.NewSeparator(),
		widget.NewLabel("Generated Output:"),
		hg.outputEntry,
	)

	return form
}

func (hg *HashGenerator) generateHash() {
	description := hg.descriptionEntry.Text
	masterPass := hg.masterPassEntry.Text

	if description == "" || masterPass == "" {
		dialog.ShowError(fmt.Errorf("both description and master password tokens are required"), hg.window)
		return
	}

	// Combine the tokens
	combined := description + masterPass

	// Get the hash using system commands
	hash, err := hg.getHash(combined, hg.algorithmSelect.Selected)
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

func (hg *HashGenerator) hashToUppercase(hash string) string {
	result := ""
	for _, char := range hash {
		if char >= 'a' && char <= 'z' {
			result += strings.ToUpper(string(char))
		} else if char >= 'A' && char <= 'Z' {
			result += string(char)
		} else {
			// Convert other characters to letters
			result += string('A' + (char % 26))
		}
	}
	return result
}

func (hg *HashGenerator) hashToLowercase(hash string) string {
	result := ""
	for _, char := range hash {
		if char >= 'a' && char <= 'z' {
			result += string(char)
		} else if char >= 'A' && char <= 'Z' {
			result += strings.ToLower(string(char))
		} else {
			// Convert other characters to letters
			result += string('a' + (char % 26))
		}
	}
	return result
}

func (hg *HashGenerator) toggleMask(bool) {
	if hg.maskCheck.Checked {
		// Mask the output with dots
		text := hg.outputEntry.Text
		if text != "" {
			masked := strings.Repeat("•", len(text))
			hg.outputEntry.SetText(masked)
		}
	} else {
		// If we need to unmask, we need to regenerate since we've lost the original
		// This is a limitation - in a real app you'd store the original separately
		if hg.outputEntry.Text != "" && strings.Contains(hg.outputEntry.Text, "•") {
			dialog.ShowInformation("Info", "Please regenerate to see unmasked output", hg.window)
		}
	}
}
