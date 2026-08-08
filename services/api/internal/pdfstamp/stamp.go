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

	candidates := []string{pythonBin, "/app/.venv-pdf/bin/python", "/app/.venv-pdf/bin/python3", "python3", "python"}
	seen := map[string]struct{}{}
	var lastErr error
	for _, bin := range candidates {
		bin = strings.TrimSpace(bin)
		if bin == "" {
			continue
		}
		if _, ok := seen[bin]; ok {
			continue
		}
		seen[bin] = struct{}{}
		if bin != "python3" && bin != "python" {
			if st, err := os.Stat(bin); err != nil || st.IsDir() {
				continue
			}
		} else if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		cmd := exec.Command(
			bin,
			scriptPath,
			"--src", srcPDF,
			"--dst", dstPDF,
			"--title", title,
			"--keywords", strings.Join(clean, ", "),
		)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("stamp pdf (%s): %w (%s)", bin, err, strings.TrimSpace(string(out)))
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("stamp pdf: no python interpreter found (set PDF_PYTHON)")
}
