// Copyright (c) 2025 Neil Stephens. All rights reserved.
// Use of this source code is governed by an MIT license that can be
// found in the LICENSE file.

package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"regexp"
	"strconv"

	"fyne.io/fyne/v2/dialog"
)

func (hg *HashGenerator) generateHash() {

	if hg.descriptionEntry.Validate() != nil {
		return
	}
	description := hg.descriptionEntry.Text
	hg.saveSetting(description)

	if hg.masterPassEntry.Validate() != nil || hg.iterationsEntry.Validate() != nil || hg.lengthEntry.Validate() != nil {
		return
	}
	masterPass := hg.masterPassEntry.Text

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
