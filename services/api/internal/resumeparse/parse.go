package resumeparse

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type ProfileDraft struct {
	Name        string
	Phone       string
	LinkedIn    string
	GitHub      string
	Website     string
	Location    string
	Headline    string
	Experience  string
	CoverLetter string
	RawText     string
}

func ExtractText(filename string, content []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return extractPDF(content)
	case ".docx":
		return extractDOCX(content)
	default:
		return "", fmt.Errorf("unsupported file type")
	}
}

func extractPDF(content []byte) (string, error) {
	tmp, err := os.CreateTemp("", "jobright-resume-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	cmd := exec.Command("pdftotext", "-layout", "-nopgbrk", tmp.Name(), "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func extractDOCX(content []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", err
	}
	var xmlData []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				rc.Close()
				return "", err
			}
			rc.Close()
			xmlData = buf.Bytes()
			break
		}
	}
	if len(xmlData) == 0 {
		return "", fmt.Errorf("docx missing document.xml")
	}
	text := stripXML(string(xmlData))
	return strings.TrimSpace(text), nil
}

func stripXML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	return regexp.MustCompile(`\s+`).ReplaceAllString(b.String(), " ")
}

var (
	reEmail    = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	rePhone    = regexp.MustCompile(`(?:\+?\d{1,3}[\s\-.]*)?(?:\(?\d{2,4}\)?[\s\-.]*)?\d{3,4}[\s\-.]*\d{3,4}`)
	reLinkedIn = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?linkedin\.com/in/[a-z0-9\-_/]+`)
	reGitHub   = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?github\.com/[a-zA-Z0-9\-]+`)
	reURL      = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)
)

func ParseProfile(text string) ProfileDraft {
	draft := ProfileDraft{RawText: text}
	lines := nonEmptyLines(text)

	if m := reLinkedIn.FindString(text); m != "" {
		draft.LinkedIn = normalizeURL(m, "https://")
	}
	if m := reGitHub.FindString(text); m != "" {
		draft.GitHub = normalizeURL(m, "https://")
	}
	if m := rePhone.FindString(text); m != "" && looksLikePhone(m) {
		draft.Phone = strings.TrimSpace(m)
	}

	for _, m := range reURL.FindAllString(text, -1) {
		low := strings.ToLower(m)
		if strings.Contains(low, "linkedin.com") || strings.Contains(low, "github.com") {
			continue
		}
		if strings.Contains(low, "mailto:") {
			continue
		}
		draft.Website = strings.TrimRight(m, ".,);]")
		break
	}

	if len(lines) > 0 {
		draft.Name = cleanName(lines[0])
	}
	for _, line := range lines[1:] {
		if draft.Headline == "" && looksLikeHeadline(line) {
			draft.Headline = trimLen(line, 180)
		}
		if draft.Location == "" && looksLikeLocation(line) {
			draft.Location = trimLen(line, 120)
		}
		if draft.Headline != "" && draft.Location != "" {
			break
		}
	}

	draft.Experience = extractSection(text, []string{
		"experience", "work experience", "professional experience", "employment", "work history",
	})
	summary := extractSection(text, []string{"summary", "profile", "about", "objective"})
	if summary != "" {
		draft.CoverLetter = trimLen(summary, 1200)
	} else if draft.Experience != "" {
		draft.CoverLetter = trimLen(draft.Experience, 1200)
	}

	skills := extractSection(text, []string{"skills", "technical skills", "technologies"})
	if draft.Headline == "" && skills != "" {
		draft.Headline = trimLen(skills, 180)
	}

	return draft
}

func nonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	if reEmail.MatchString(s) || reURL.MatchString(s) || looksLikePhone(s) {
		return ""
	}
	if len(s) > 80 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) == 0 || len(words) > 5 {
		return ""
	}
	return s
}

func looksLikeHeadline(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "@") || strings.HasPrefix(low, "http") {
		return false
	}
	keys := []string{"engineer", "developer", "designer", "manager", "student", "intern", "analyst", "architect", "founder", "lead"}
	for _, k := range keys {
		if strings.Contains(low, k) {
			return true
		}
	}
	return len(s) <= 90 && !looksLikeLocation(s)
}

func looksLikeLocation(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "@") || strings.Contains(low, "http") {
		return false
	}
	keys := []string{"remote", "ethiopia", "addis", "usa", "uk", "canada", "germany", "berlin", "london", "new york", "san francisco", "city", ","}
	for _, k := range keys {
		if strings.Contains(low, k) {
			return len(s) <= 80
		}
	}
	return false
}

func looksLikePhone(s string) bool {
	digits := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	return digits >= 9 && digits <= 15
}

func extractSection(text string, headings []string) string {
	lines := nonEmptyLines(text)
	start := -1
	for i, line := range lines {
		norm := strings.ToLower(strings.Trim(line, ": "))
		for _, h := range headings {
			if norm == h || strings.HasPrefix(norm, h+" ") {
				start = i + 1
				break
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 || start >= len(lines) {
		return ""
	}
	var b strings.Builder
	for _, line := range lines[start:] {
		norm := strings.ToLower(strings.Trim(line, ": "))
		if isSectionHeading(norm) {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if b.Len() > 1500 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func isSectionHeading(norm string) bool {
	heads := []string{
		"experience", "education", "skills", "projects", "summary", "profile",
		"certifications", "awards", "languages", "interests", "objective",
	}
	for _, h := range heads {
		if norm == h {
			return true
		}
	}
	return false
}

func normalizeURL(v, prefix string) string {
	v = strings.TrimRight(strings.TrimSpace(v), ".,);]")
	if strings.HasPrefix(strings.ToLower(v), "http") {
		return v
	}
	return prefix + strings.TrimPrefix(v, "//")
}

func trimLen(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}
