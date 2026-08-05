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
	Email       string
	LinkedIn    string
	GitHub      string
	Website     string
	Location    string
	Headline    string
	Skills      []string
	Education   string
	Experience  string
	Projects    string
	CoverLetter string
	RawText     string
}

func ExtractDocument(filename string, content []byte) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		text, err := extractPDFText(content)
		links := extractPDFLinks(content)
		return text, uniqueLinks(append(links, linksFromText(text)...)), err
	case ".docx":
		text, links, err := extractDOCX(content)
		return text, uniqueLinks(append(links, linksFromText(text)...)), err
	default:
		return "", nil, fmt.Errorf("unsupported file type")
	}
}

func ExtractText(filename string, content []byte) (string, error) {
	text, _, err := ExtractDocument(filename, content)
	return text, err
}

func extractPDFText(content []byte) (string, error) {
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
	text := strings.TrimSpace(string(out))

	// Also pull visible + annotated links via pdftohtml xml when available.
	xmlCmd := exec.Command("pdftohtml", "-xml", "-stdout", "-i", "-hidden", tmp.Name())
	if xmlOut, xmlErr := xmlCmd.Output(); xmlErr == nil {
		extra := stripXML(string(xmlOut))
		if len(extra) > len(text) {
			text = strings.TrimSpace(extra)
		}
	}
	return text, nil
}

func extractPDFLinksFromXML(content []byte) []string {
	tmp, err := os.CreateTemp("", "jobright-resume-*.pdf")
	if err != nil {
		return nil
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return nil
	}
	tmp.Close()
	xmlCmd := exec.Command("pdftohtml", "-xml", "-stdout", "-i", "-hidden", tmp.Name())
	xmlOut, err := xmlCmd.Output()
	if err != nil {
		return nil
	}
	var links []string
	reHref := regexp.MustCompile(`(?i)(?:href|uri)="([^"]+)"`)
	for _, m := range reHref.FindAllSubmatch(xmlOut, -1) {
		links = append(links, cleanLink(string(m[1])))
	}
	return links
}

var (
	rePDFURI     = regexp.MustCompile(`(?i)/URI\s*\(([^)]+)\)`)
	rePDFURIHex  = regexp.MustCompile(`(?i)/URI\s*<([0-9a-fA-F]+)>`)
	reRawURL     = regexp.MustCompile(`(?i)https?://[^\s<>"'\\\x00-\x1f]+`)
	reWWW        = regexp.MustCompile(`(?i)\b(?:www\.)?(linkedin\.com/in/[a-z0-9\-_/]+|github\.com/[a-zA-Z0-9\-]+)`)
	reEmail      = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	rePhone      = regexp.MustCompile(`(?:\+?\d{1,3}[\s\-.]*)?(?:\(?\d{2,4}\)?[\s\-.]*)?\d{3,4}[\s\-.]*\d{3,4}`)
	reLinkedIn   = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?linkedin\.com/in/[a-z0-9\-_/]+`)
	reGitHub     = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?github\.com/[a-zA-Z0-9\-]+/?`)
	reURL        = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)
	reSkillSplit = regexp.MustCompile(`[,|/•·\n\t]| {2,}`)
)

func extractPDFLinks(content []byte) []string {
	var links []string
	links = append(links, extractPDFParenURIs(content)...)
	for _, m := range rePDFURI.FindAllSubmatch(content, -1) {
		links = append(links, cleanLink(unescapePDFString(string(m[1]))))
	}
	for _, m := range rePDFURIHex.FindAllSubmatch(content, -1) {
		if decoded, err := decodePDFHex(string(m[1])); err == nil {
			links = append(links, cleanLink(decoded))
		}
	}
	for _, m := range reRawURL.FindAll(content, -1) {
		links = append(links, cleanLink(string(m)))
	}
	for _, m := range reWWW.FindAll(content, -1) {
		links = append(links, cleanLink(string(m)))
	}
	links = append(links, extractPDFLinksFromXML(content)...)
	return links
}

// extractPDFParenURIs walks PDF bytes for /URI ( ... ) with escape-aware parens.
func extractPDFParenURIs(content []byte) []string {
	var links []string
	lower := bytes.ToLower(content)
	needle := []byte("/uri")
	for i := 0; i < len(lower); {
		idx := bytes.Index(lower[i:], needle)
		if idx < 0 {
			break
		}
		matchStart := i + idx
		pos := matchStart + len(needle)
		for pos < len(content) && (content[pos] == ' ' || content[pos] == '\n' || content[pos] == '\r' || content[pos] == '\t') {
			pos++
		}
		if pos >= len(content) {
			break
		}
		switch content[pos] {
		case '(':
			raw, next := readPDFLiteral(content, pos)
			if raw != "" {
				links = append(links, cleanLink(unescapePDFString(raw)))
			}
			i = next
		case '<':
			end := bytes.IndexByte(content[pos:], '>')
			if end > 0 {
				hex := string(content[pos+1 : pos+end])
				if decoded, err := decodePDFHex(hex); err == nil {
					links = append(links, cleanLink(decoded))
				}
				i = pos + end + 1
			} else {
				i = matchStart + 1
			}
		default:
			// e.g. "/S /URI /URI (...)" — keep scanning past this /URI token
			i = matchStart + 1
		}
	}
	return links
}

func readPDFLiteral(content []byte, openParen int) (string, int) {
	if openParen >= len(content) || content[openParen] != '(' {
		return "", openParen + 1
	}
	depth := 0
	var b strings.Builder
	for i := openParen; i < len(content); i++ {
		c := content[i]
		if c == '\\' && i+1 < len(content) {
			b.WriteByte(c)
			b.WriteByte(content[i+1])
			i++
			continue
		}
		if c == '(' {
			depth++
			if depth > 1 {
				b.WriteByte(c)
			}
			continue
		}
		if c == ')' {
			depth--
			if depth == 0 {
				return b.String(), i + 1
			}
			b.WriteByte(c)
			continue
		}
		if depth > 0 {
			b.WriteByte(c)
		}
	}
	return b.String(), len(content)
}

func unescapePDFString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case '(', ')', '\\':
			b.WriteByte(s[i])
		default:
			if s[i] >= '0' && s[i] <= '7' {
				val := int(s[i] - '0')
				n := 1
				for n < 3 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7' {
					i++
					val = val*8 + int(s[i]-'0')
					n++
				}
				b.WriteByte(byte(val))
			} else {
				b.WriteByte(s[i])
			}
		}
	}
	return b.String()
}

func decodePDFHex(hexStr string) (string, error) {
	if len(hexStr)%2 == 1 {
		hexStr = "0" + hexStr
	}
	out := make([]byte, 0, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		var b byte
		_, err := fmt.Sscanf(hexStr[i:i+2], "%02x", &b)
		if err != nil {
			return "", err
		}
		out = append(out, b)
	}
	return string(out), nil
}

func extractDOCX(content []byte) (string, []string, error) {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", nil, err
	}

	rels := map[string]string{}
	var docXML string
	for _, f := range r.File {
		switch f.Name {
		case "word/_rels/document.xml.rels":
			rc, err := f.Open()
			if err != nil {
				return "", nil, err
			}
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			reRel := regexp.MustCompile(`Id="(rId\d+)"[^>]*Target="([^"]+)"`)
			for _, m := range reRel.FindAllStringSubmatch(buf.String(), -1) {
				rels[m[1]] = m[2]
			}
		case "word/document.xml":
			rc, err := f.Open()
			if err != nil {
				return "", nil, err
			}
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			docXML = buf.String()
		}
	}
	if docXML == "" {
		return "", nil, fmt.Errorf("docx missing document.xml")
	}

	var links []string
	reHyper := regexp.MustCompile(`r:id="(rId\d+)"`)
	for _, m := range reHyper.FindAllStringSubmatch(docXML, -1) {
		if target, ok := rels[m[1]]; ok && (strings.HasPrefix(target, "http") || strings.Contains(target, "linkedin") || strings.Contains(target, "github")) {
			links = append(links, cleanLink(target))
		}
	}
	reInline := regexp.MustCompile(`Target="(https?://[^"]+)"`)
	for _, m := range reInline.FindAllStringSubmatch(docXML, -1) {
		links = append(links, cleanLink(m[1]))
	}

	text := stripXML(docXML)
	return strings.TrimSpace(text), links, nil
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

func linksFromText(text string) []string {
	var links []string
	for _, m := range reURL.FindAllString(text, -1) {
		links = append(links, cleanLink(m))
	}
	for _, m := range reWWW.FindAllString(text, -1) {
		links = append(links, cleanLink(m))
	}
	return links
}

func ParseProfile(text string, links []string) ProfileDraft {
	combined := text + "\n" + strings.Join(links, "\n")
	draft := ProfileDraft{RawText: text}
	lines := nonEmptyLines(text)

	if m := reEmail.FindString(combined); m != "" {
		draft.Email = strings.ToLower(m)
	}
	if m := reLinkedIn.FindString(combined); m != "" {
		draft.LinkedIn = normalizeURL(m, "https://")
	}
	if m := reGitHub.FindString(combined); m != "" {
		draft.GitHub = strings.TrimRight(normalizeURL(m, "https://"), "/")
	}
	if m := rePhone.FindString(text); m != "" && looksLikePhone(m) {
		draft.Phone = strings.TrimSpace(m)
	}

	for _, link := range append(links, reURL.FindAllString(combined, -1)...) {
		low := strings.ToLower(link)
		if strings.Contains(low, "linkedin.com") || strings.Contains(low, "github.com") || strings.Contains(low, "mailto:") {
			continue
		}
		if strings.HasPrefix(low, "http") {
			draft.Website = cleanLink(link)
			break
		}
	}

	for _, line := range lines {
		if draft.Name == "" {
			if n := cleanName(line); n != "" {
				draft.Name = n
				continue
			}
		}
		if draft.Headline == "" && looksLikeHeadline(line) {
			draft.Headline = trimLen(line, 180)
		}
		if draft.Location == "" && looksLikeLocation(line) {
			draft.Location = trimLen(line, 120)
		}
	}

	draft.Experience = extractSection(text, []string{
		"experience", "work experience", "professional experience", "employment", "work history",
	})
	draft.Education = extractSection(text, []string{"education", "academic background"})
	draft.Projects = extractSection(text, []string{"projects", "personal projects", "selected projects"})
	skillsText := extractSection(text, []string{
		"skills", "technical skills", "technologies", "tech stack", "core skills", "key skills",
	})
	draft.Skills = parseSkills(skillsText)

	summary := extractSection(text, []string{"summary", "profile", "about", "objective", "professional summary"})
	switch {
	case summary != "":
		draft.CoverLetter = trimLen(summary, 2000)
	case draft.Experience != "":
		draft.CoverLetter = trimLen(draft.Experience, 2000)
	}

	if draft.Headline == "" && len(draft.Skills) > 0 {
		draft.Headline = trimLen(strings.Join(draft.Skills[:min(5, len(draft.Skills))], " · "), 180)
	}

	return draft
}

func parseSkills(section string) []string {
	if section == "" {
		return nil
	}
	parts := reSkillSplit.Split(section, -1)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(strings.Trim(p, "-*•"))
		if p == "" || len(p) > 48 {
			continue
		}
		if strings.Count(p, " ") > 5 {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
		if len(out) >= 40 {
			break
		}
	}
	return out
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
	low := strings.ToLower(s)
	if isSectionHeading(low) {
		return ""
	}
	if len(s) > 80 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) == 0 || len(words) > 6 {
		return ""
	}
	return s
}

func looksLikeHeadline(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "@") || strings.HasPrefix(low, "http") || isSectionHeading(low) {
		return false
	}
	keys := []string{
		"engineer", "developer", "designer", "manager", "student", "intern", "analyst",
		"architect", "founder", "lead", "scientist", "consultant", "specialist",
	}
	for _, k := range keys {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

func looksLikeLocation(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "@") || strings.Contains(low, "http") {
		return false
	}
	keys := []string{
		"remote", "ethiopia", "addis", "usa", "united states", "uk", "canada", "germany",
		"berlin", "london", "new york", "san francisco", "city", ",", "based in",
	}
	for _, k := range keys {
		if strings.Contains(low, k) {
			return len(s) <= 90
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
	inline := ""
	for i, line := range lines {
		norm := normalizeHeading(line)
		for _, h := range headings {
			if norm == h || strings.HasPrefix(norm, h+" ") || strings.HasPrefix(norm, h+":") {
				start = i + 1
				// "Skills: Go, React" on the same line
				if idx := strings.Index(line, ":"); idx >= 0 {
					rest := strings.TrimSpace(line[idx+1:])
					if rest != "" {
						inline = rest
					}
				}
				break
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return ""
	}
	var b strings.Builder
	if inline != "" {
		b.WriteString(inline)
		b.WriteByte('\n')
	}
	if start < len(lines) {
		for _, line := range lines[start:] {
			norm := normalizeHeading(line)
			if isSectionHeading(norm) {
				break
			}
			b.WriteString(line)
			b.WriteByte('\n')
			if b.Len() > 2500 {
				break
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func normalizeHeading(line string) string {
	return strings.ToLower(strings.Trim(line, ": |-•*_"))
}

func isSectionHeading(norm string) bool {
	heads := []string{
		"experience", "work experience", "professional experience", "education", "skills",
		"technical skills", "technologies", "projects", "summary", "profile", "about",
		"certifications", "awards", "languages", "interests", "objective", "employment",
		"work history", "publications", "volunteer",
	}
	for _, h := range heads {
		if norm == h {
			return true
		}
	}
	return false
}

func normalizeURL(v, prefix string) string {
	v = cleanLink(v)
	if strings.HasPrefix(strings.ToLower(v), "http") {
		return v
	}
	return prefix + strings.TrimPrefix(v, "//")
}

func cleanLink(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, `\`, "")
	for {
		if v == "" {
			return v
		}
		last := v[len(v)-1]
		switch last {
		case '.', ',', ';', '"', '>', ']':
			v = v[:len(v)-1]
			continue
		case ')':
			if strings.Count(v, "(") < strings.Count(v, ")") {
				v = v[:len(v)-1]
				continue
			}
		}
		break
	}
	return v
}

func uniqueLinks(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, l := range in {
		l = cleanLink(l)
		if l == "" {
			continue
		}
		key := strings.ToLower(l)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, l)
	}
	return out
}

func trimLen(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
