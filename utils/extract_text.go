package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/otiai10/gosseract/v2"
)

func ExtractText(tempPath, ext string) (string, error) {

	switch ext {
	case ".pdf":
		isScanned, err := isScannedPDF(tempPath)
		fmt.Println(isScanned, "isScanned")
		if err != nil {
			fmt.Printf("Error checking if PDF is scanned: %v, attempting OCR anyway\n", err)
			// If we can't determine, try OCR as fallback
			return ocrPDF(tempPath)
		}
		if isScanned {
			return ocrPDF(tempPath)
		}
		f, r, err := pdf.Open(tempPath)
		if err != nil {
			fmt.Printf("Error opening PDF with pdf library: %v, attempting OCR fallback\n", err)
			// Fallback to OCR if PDF library fails
			return ocrPDF(tempPath)
		}
		defer f.Close()
		avgChars := avgCharPerLine(r)
		fmt.Println(avgChars, "avgChars")
		if avgChars < 200 {
			return ocrPDF(tempPath)
		}
		var buf strings.Builder
		text, err := r.GetPlainText()
		if err != nil {
			return "", err
		}
		_, err = io.Copy(&buf, text)
		if err != nil {
			return "", err
		}
		hasFilledFields := hasFilledFields(buf.String())
		fmt.Println(hasFilledFields, "hasFilledFields")
		if !hasFilledFields {
			return ocrPDF(tempPath)
		}
		// fmt.Println(buf.String(), "buf.String()", hasFilledFields)
		// alphabaticRatio := alphabaticRatio(buf.String())
		// fmt.Println(alphabaticRatio, "alphabaticRatio")
		return buf.String(), nil

	case ".csv":
		f, err := os.Open(tempPath)
		if err != nil {
			return "", err
		}
		defer f.Close()

		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		if err != nil {
			return "", err
		}

		var builder strings.Builder
		for _, row := range records {
			builder.WriteString(strings.Join(row, " "))
			builder.WriteString("\n")
		}
		return builder.String(), nil

	case ".png", ".jpg", ".jpeg":
		// client := gosseract.NewClient()
		// defer client.Close()
		// client.SetImage(tempPath)
		// text, err := client.Text()
		// if err != nil {
		// 	return "", err
		// }
		// return tesseractText(tempPath)
		return ocrImage(tempPath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func hasFilledFields(text string) bool {
	// fmt.Println(text, "text")
	lines := strings.Split(text, "\n")

	field := 0
	lable := 0

	for _, line := range lines {
		if strings.HasSuffix(line, ":") {
			lable++
		}
		if strings.Contains(line, ":") && len(strings.TrimSpace(strings.Split(line, ":")[1])) > 3 {
			field++
		}
	}
	fmt.Println(field, lable, "field and lable")
	return field > lable/3 && lable > 0
}

// func tesseractText(tempPath string) (string, error) {

// 	fmt.Println(tempPath, "tempPath from tesseractText")
// 	client := gosseract.NewClient()
// 	defer client.Close()
// 	client.SetImage(tempPath)
// 	text, err := client.Text()
// 	if err != nil {
// 		return "", err
// 	}
// 	fmt.Println(text, "text from tesseractText")
// 	return text, nil
// }

func ocrPDF(pdfPath string) (string, error) {
	// Check if pdftoppm is available
	pdftoppmPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", fmt.Errorf("pdftoppm not found in PATH: %v", err)
	}
	fmt.Printf("Using pdftoppm at: %s\n", pdftoppmPath)

	tempDir, err := os.MkdirTemp("", "ocr-pdf-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	fmt.Printf("Created temp directory for OCR: %s\n", tempDir)

	outputPath := filepath.Join(tempDir, "page")
	cmd := exec.Command(pdftoppmPath, "-r", "300", "-png", pdfPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running pdftoppm: %v, output: %s\n", err, string(output))
		return "", fmt.Errorf("pdftoppm failed: %v, output: %s", err, string(output))
	}
	fmt.Printf("pdftoppm output: %s\n", string(output))

	// List all files in temp directory to debug
	files, _ := os.ReadDir(tempDir)
	fmt.Printf("Files in tempDir %s after pdftoppm: ", tempDir)
	for _, f := range files {
		fmt.Printf("%s ", f.Name())
	}
	fmt.Println()

	image, err := filepath.Glob(filepath.Join(tempDir, "page-*.png"))
	if err != nil {
		return "", fmt.Errorf("glob error: %v", err)
	}
	if len(image) == 0 {
		// Try alternative patterns
		altImage, _ := filepath.Glob(filepath.Join(tempDir, "page*.png"))
		if len(altImage) > 0 {
			fmt.Printf("Found files with alternative pattern: %v\n", altImage)
			image = altImage
		} else {
			return "", fmt.Errorf("pdftoppm did not create any PNG files in %s", tempDir)
		}
	}
	fmt.Printf("Found %d PNG images: %v\n", len(image), image)

	var result strings.Builder
	var ocrErrors []error
	for i, img := range image {
		fmt.Printf("Processing OCR for image %d/%d: %s\n", i+1, len(image), img)
		text, err := ocrImage(img)
		if err != nil {
			fmt.Printf("OCR error for %s: %v\n", img, err)
			ocrErrors = append(ocrErrors, fmt.Errorf("OCR failed for %s: %v", img, err))
			continue
		}
		if text == "" {
			fmt.Printf("Warning: OCR returned empty text for %s\n", img)
		} else {
			fmt.Printf("OCR extracted %d characters from %s\n", len(text), img)
			result.WriteString(text)
			result.WriteString("\n")
		}
	}
	finalText := result.String()
	fmt.Printf("Total OCR result: %d characters\n", len(finalText))
	if finalText == "" {
		if len(ocrErrors) > 0 {
			return "", fmt.Errorf("OCR failed for all images: %v", ocrErrors)
		}
		return "", fmt.Errorf("OCR returned empty text for all images")
	}
	return finalText, nil
}

func ocrImage(imagePath string) (string, error) {
	// Verify image file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return "", fmt.Errorf("image file does not exist: %s", imagePath)
	}

	fmt.Printf("Initializing Tesseract OCR for: %s\n", imagePath)
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImage(imagePath); err != nil {
		return "", fmt.Errorf("failed to set image: %v", err)
	}

	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("OCR failed: %v", err)
	}

	if text == "" {
		return "", fmt.Errorf("OCR returned empty text")
	}

	fmt.Printf("OCR successful: extracted %d characters\n", len(text))
	return text, nil
}

func isScannedPDF(pdfPath string) (bool, error) {
	file, reader, err := pdf.Open(pdfPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	textObjects := 0

	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		content := page.Content()
		for _, txt := range content.Text {
			if strings.TrimSpace(txt.S) != "" {
				textObjects++
				if textObjects > 10 {
					// Enough real text → digital PDF
					return false, nil
				}
			}
		}
	}

	// No real text objects found → scanned PDF
	return true, nil
}

func avgCharPerLine(reader *pdf.Reader) float64 {
	totalChars := 0
	pages := reader.NumPage()

	for i := 1; i <= pages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		content := page.Content()
		for _, txt := range content.Text {
			if strings.TrimSpace(txt.S) != "" {
				totalChars += len(txt.S)
			}
		}
	}
	return float64(totalChars) / float64(pages)
}
