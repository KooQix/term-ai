package fileprocessor

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// FileAttachment represents a processed file attachment
type FileAttachment struct {
	Path     string // Original file path
	Type     string // "image", "text", "pdf"
	Content  string // base64 for images, text for others
	MimeType string // MIME type for images
	Name     string // filename
}

// Supported file extensions
var (
	imageExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	textExtensions = map[string]bool{
		".txt":      true,
		".md":       true,
		".markdown": true,
	}

	codeExtensions = map[string]bool{
		".go":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".tsx":  true,
		".jsx":  true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".cc":   true,
		".h":    true,
		".hpp":  true,
		".rs":   true,
		".rb":   true,
		".php":  true,
		".sh":   true,
		".bash": true,
		".yaml": true,
		".yml":  true,
		".json": true,
		".xml":  true,
		".html": true,
		".css":  true,
		".sql":  true,
	}

	pdfExtensions = map[string]bool{
		".pdf": true,
	}
)

// ProcessFile processes a single file and returns a FileAttachment
func ProcessFile(path string) (*FileAttachment, error) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", path)
		}
		return nil, fmt.Errorf("error accessing file %s: %w", path, err)
	}

	// Check if it's a directory
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", path)
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(path))
	filename := filepath.Base(path)

	// Determine file type and process accordingly
	var attachment *FileAttachment

	if imageExtensions[ext] {
		attachment, err = processImage(path, filename)
	} else if pdfExtensions[ext] {
		attachment, err = processPDF(path, filename)
	} else if textExtensions[ext] || codeExtensions[ext] {
		attachment, err = processText(path, filename)
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	return attachment, nil
}

// ProcessFiles processes multiple files and returns a slice of FileAttachments
func ProcessFiles(paths []string) ([]*FileAttachment, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	attachments := make([]*FileAttachment, 0, len(paths))
	var errors []string

	for _, path := range paths {
		attachment, err := ProcessFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		attachments = append(attachments, attachment)
	}

	// If all files failed, return error
	if len(errors) > 0 && len(attachments) == 0 {
		return nil, fmt.Errorf("failed to process all files:\n%s", strings.Join(errors, "\n"))
	}

	// If some files failed, still return successful ones but note the errors
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: some files failed to process:\n%s\n", strings.Join(errors, "\n"))
	}

	return attachments, nil
}

// processImage reads an image file and encodes it as base64
func processImage(path, filename string) (*FileAttachment, error) {
	// Read the image file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// Detect MIME type
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback MIME types
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		default:
			mimeType = "application/octet-stream"
		}
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Create data URL format
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return &FileAttachment{
		Path:     path,
		Type:     "image",
		Content:  dataURL,
		MimeType: mimeType,
		Name:     filename,
	}, nil
}

// processPDF extracts text content from a PDF file
func processPDF(path, filename string) (*FileAttachment, error) {
	// Open the PDF file
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer f.Close()

	// Extract text from all pages
	var textBuilder strings.Builder
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		p := r.Page(pageNum)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			// Continue with other pages even if one fails
			fmt.Fprintf(os.Stderr, "Warning: failed to extract text from page %d of %s: %v\n", pageNum, filename, err)
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if extractedText == "" {
		return nil, fmt.Errorf("no text content could be extracted from PDF")
	}

	return &FileAttachment{
		Path:     path,
		Type:     "pdf",
		Content:  extractedText,
		MimeType: "application/pdf",
		Name:     filename,
	}, nil
}

// processText reads a text or code file
func processText(path, filename string) (*FileAttachment, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	content := string(data)

	// Determine if it's a code file or text file
	ext := strings.ToLower(filepath.Ext(path))
	fileType := "text"
	if codeExtensions[ext] {
		fileType = "code"
	}

	return &FileAttachment{
		Path:     path,
		Type:     fileType,
		Content:  content,
		MimeType: mime.TypeByExtension(ext),
		Name:     filename,
	}, nil
}

// GetSupportedExtensions returns all supported file extensions
func GetSupportedExtensions() []string {
	extensions := make([]string, 0)

	for ext := range imageExtensions {
		extensions = append(extensions, ext)
	}
	for ext := range pdfExtensions {
		extensions = append(extensions, ext)
	}
	for ext := range textExtensions {
		extensions = append(extensions, ext)
	}
	for ext := range codeExtensions {
		extensions = append(extensions, ext)
	}

	return extensions
}

// IsSupported checks if a file extension is supported
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return imageExtensions[ext] || pdfExtensions[ext] || textExtensions[ext] || codeExtensions[ext]
}
