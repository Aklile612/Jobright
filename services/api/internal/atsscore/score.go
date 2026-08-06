package atsscore

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var stop = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "of": true, "to": true,
	"in": true, "on": true, "for": true, "with": true, "as": true, "by": true, "at": true,
	"from": true, "is": true, "are": true, "be": true, "this": true, "that": true, "it": true,
	"you": true, "we": true, "our": true, "your": true, "will": true, "can": true, "have": true,
	"has": true, "was": true, "were": true, "not": true, "but": true, "if": true, "than": true,
	"into": true, "also": true, "etc": true, "using": true, "use": true, "used": true,
	"role": true, "job": true, "team": true, "work": true, "working": true, "experience": true,
	"years": true, "year": true, "including": true, "include": true, "ability": true,
	"strong": true, "good": true, "well": true, "must": true, "should": true, "required": true,
	"requirements": true, "responsibilities": true, "about": true, "who": true, "what": true,
	"their": true, "they": true, "them": true, "all": true, "any": true, "more": true,
	"other": true, "such": true, "via": true, "per": true, "within": true, "across": true,
	"new": true, "based": true, "able": true, "plus": true, "etcetera": true,
}

var multiWord = []string{
	"machine learning", "deep learning", "computer science", "software engineer",
	"full stack", "fullstack", "front end", "frontend", "back end", "backend",
	"react native", "node.js", "ci/cd", "rest api", "graphql", "unit testing",
	"system design", "data structures", "distributed systems", "cloud computing",
	"amazon web services", "google cloud", "continuous integration",
	"object oriented", "test driven", "agile scrum", "version control",
}

type Result struct {
	MatchScore      float64
	Present         []string
	MissingKeywords []string
	MissingSkills   []string
	Covered         int
	Total           int
}

// Score compares resume text to job title + description using keyword coverage.
func Score(resumeText, title, description string) Result {
	resume := strings.ToLower(resumeText)
	keywords := ExtractKeywords(title + "\n" + description)
	if len(keywords) == 0 {
		return Result{MatchScore: 0}
	}

	var present, missing []string
	for _, kw := range keywords {
		if strings.Contains(resume, strings.ToLower(kw)) {
			present = append(present, kw)
		} else {
			missing = append(missing, kw)
		}
	}

	score := 100.0 * float64(len(present)) / float64(len(keywords))
	// Soft floor/ceiling for readability
	if score > 0 && score < 5 {
		score = 5
	}
	if score > 98 && len(missing) > 0 {
		score = 98
	}

	// Split missing into skills-ish (shorter tech tokens) vs keywords
	var skills, keys []string
	for _, m := range missing {
		if len(strings.Fields(m)) <= 2 && len(m) <= 24 {
			skills = append(skills, m)
		} else {
			keys = append(keys, m)
		}
	}
	if len(skills) > 8 {
		skills = skills[:8]
	}
	if len(keys) > 8 {
		keys = keys[:8]
	}
	if len(present) > 12 {
		present = present[:12]
	}

	return Result{
		MatchScore:      round1(score),
		Present:         present,
		MissingKeywords: keys,
		MissingSkills:   skills,
		Covered:         len(present),
		Total:           len(keywords),
	}
}

func ExtractKeywords(text string) []string {
	lower := strings.ToLower(text)
	seen := map[string]bool{}
	var out []string

	add := func(s string) {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.Trim(s, ".,;:()[]{}/\\\"'")
		if s == "" || stop[s] || seen[s] {
			return
		}
		if len(s) < 2 || len(s) > 40 {
			return
		}
		seen[s] = true
		out = append(out, s)
	}

	for _, phrase := range multiWord {
		if strings.Contains(lower, phrase) {
			add(phrase)
		}
	}

	re := regexp.MustCompile(`[A-Za-z][A-Za-z0-9.+#\-]{1,30}`)
	for _, tok := range re.FindAllString(lower, -1) {
		if stop[tok] {
			continue
		}
		// Prefer tokens with digits/symbols or capitalized-looking tech, or length>=3
		if !isUsefulToken(tok) {
			continue
		}
		add(tok)
	}

	// Cap to most distinctive: prefer longer / tech-looking
	sort.SliceStable(out, func(i, j int) bool {
		return keywordWeight(out[i]) > keywordWeight(out[j])
	})
	if len(out) > 28 {
		out = out[:28]
	}
	return out
}

func isUsefulToken(tok string) bool {
	if stop[tok] {
		return false
	}
	if len(tok) < 3 {
		return false
	}
	for _, r := range tok {
		if unicode.IsDigit(r) || r == '.' || r == '+' || r == '#' {
			return true
		}
	}
	techish := map[string]bool{
		"python": true, "java": true, "golang": true, "rust": true, "kotlin": true,
		"swift": true, "typescript": true, "javascript": true, "react": true, "angular": true,
		"vue": true, "django": true, "flask": true, "fastapi": true, "spring": true,
		"docker": true, "kubernetes": true, "aws": true, "azure": true, "gcp": true,
		"postgres": true, "postgresql": true, "mysql": true, "mongodb": true, "redis": true,
		"kafka": true, "graphql": true, "grpc": true, "linux": true, "git": true,
		"terraform": true, "ansible": true, "jenkins": true, "pytest": true, "junit": true,
		"nextjs": true, "nodejs": true, "express": true, "laravel": true, "rails": true,
		"android": true, "ios": true, "figma": true, "selenium": true, "cypress": true,
		"spark": true, "hadoop": true, "airflow": true, "snowflake": true, "databricks": true,
		"microservices": true, "api": true, "apis": true, "sql": true, "nosql": true,
		"devops": true, "sre": true, "agile": true, "scrum": true, "kanban": true,
		"ci": true, "cd": true, "oop": true, "tdd": true, "rest": true, "http": true,
		"security": true, "testing": true, "backend": true, "frontend": true, "fullstack": true,
	}
	return techish[tok] || len(tok) >= 5
}

func keywordWeight(s string) int {
	w := len(s)
	if strings.ContainsAny(s, "0123456789.+#/") {
		w += 8
	}
	if strings.Contains(s, " ") {
		w += 6
	}
	return w
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
