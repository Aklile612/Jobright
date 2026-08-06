package pdfstamp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Stamp copies srcPDF to dstPDF and stamps a compact keyword footer on page 1.
// pythonBin should point at a venv python that has pypdf + reportlab.
func Stamp(pythonBin, scriptPath, srcPDF, dstPDF, title string, keywords []string) error {
	if pythonBin == "" {
		pythonBin = "python3"
	}
	if err := os.MkdirAll(filepath.Dir(dstPDF), 0o755); err != nil {
		return err
	}
	clean := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.TrimSpace(k)
		if k != "" {
			clean = append(clean, k)
		}
		if len(clean) >= 12 {
			break
		}
	}
	cmd := exec.Command(
		pythonBin,
		scriptPath,
		"--src", srcPDF,
		"--dst", dstPDF,
		"--title", title,
		"--keywords", strings.Join(clean, ", "),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stamp pdf: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
