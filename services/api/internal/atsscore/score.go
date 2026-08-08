package atsscore

import (
	"regexp"
	"sort"
	"strings"
)

// Soft / generic JD fluff — never score these.
var fluff = map[string]bool{
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
	"communication": true, "collaborate": true, "collaboration": true, "documentation": true,
	"engineering": true, "engineer": true, "engineers": true, "technical": true, "solutions": true,
	"operations": true, "performance": true, "architecture": true, "architect": true,
	"production-ready": true, "production": true, "ready": true, "troubleshooting": true,
	"infrastructure": true, "system-design": true, "design": true, "systems": true,
	"software": true, "developer": true, "development": true, "develop": true,
	"company": true, "position": true, "remote": true, "united": true, "states": true,
	"professional": true, "accurate": true, "dependable": true, "execution": true,
	"focused": true, "seeking": true, "looking": true, "join": true, "build": true,
	"building": true, "help": true, "support": true, "ensure": true, "deliver": true,
	"delivery": true, "quality": true, "best": true, "practices": true, "understand": true,
	"understanding": true, "knowledge": true, "familiar": true, "preferred": true,
	"bonus": true, "nice": true, "skills": true,
	"tools": true, "technologies": true, "technology": true, "stack": true,
	"written": true, "verbal": true, "excellent": true, "proven": true, "track": true,
	"record": true, "passion": true, "passionate": true, "love": true, "like": true,
	"environment": true, "fast": true, "paced": true, "ownership": true, "impact": true,
	"product": true, "products": true, "customers": true, "users": true, "business": true,
	"cross": true, "functional": true, "stakeholders": true, "manage": true, "management": true,
	"lead": true, "leader": true, "leadership": true, "mentor": true, "mentoring": true,
	"write": true, "writing": true, "maintain": true, "maintaining": true, "improve": true,
	"improving": true, "create": true, "creating": true, "implement": true, "implementing": true,
}

// Concrete tech / role terms we score on when they appear in the JD.
var tech = map[string]bool{
	"python": true, "java": true, "golang": true, "go": true, "rust": true, "kotlin": true,
	"swift": true, "typescript": true, "javascript": true, "react": true, "angular": true,
	"vue": true, "django": true, "flask": true, "fastapi": true, "spring": true, "rails": true,
	"docker": true, "kubernetes": true, "k8s": true, "aws": true, "azure": true, "gcp": true,
	"postgres": true, "postgresql": true, "mysql": true, "mongodb": true, "redis": true,
	"kafka": true, "rabbitmq": true, "graphql": true, "grpc": true, "linux": true, "git": true,
	"terraform": true, "ansible": true, "jenkins": true, "pytest": true, "junit": true,
	"nextjs": true, "next.js": true, "nodejs": true, "node.js": true, "express": true, "laravel": true,
	"android": true, "ios": true, "figma": true, "selenium": true, "cypress": true,
	"spark": true, "hadoop": true, "airflow": true, "snowflake": true, "databricks": true,
	"microservices": true, "api": true, "apis": true, "sql": true, "nosql": true,
	"devops": true, "sre": true, "agile": true, "scrum": true, "kanban": true,
	"ci": true, "cd": true, "ci/cd": true, "oop": true, "tdd": true, "rest": true, "http": true,
	"security": true, "testing": true, "backend": true, "frontend": true, "fullstack": true,
	"observability": true, "prometheus": true, "grafana": true, "datadog": true, "elk": true,
	"elasticsearch": true, "nginx": true, "helm": true, "argo": true, "argocd": true,
	"github": true, "gitlab": true, "bitbucket": true, "bash": true, "shell": true,
	"c++": true, "c#": true, ".net": true, "dotnet": true, "php": true, "ruby": true,
	"scala": true, "haskell": true, "elixir": true, "clojure": true, "dart": true, "flutter": true,
	"svelte": true, "redux": true, "webpack": true, "vite": true, "tailwind": true,
	"prisma": true, "gorm": true, "hibernate": true, "jpa": true, "celery": true,
	"pandas": true, "numpy": true, "pytorch": true, "tensorflow": true, "sklearn": true,
	"langchain": true, "openai": true, "llm": true, "mlops": true, "etl": true,
	"bigquery": true, "redshift": true, "dynamodb": true, "s3": true, "ec2": true, "lambda": true,
	"ecs": true, "eks": true, "cloudformation": true, "pulumi": true, "vault": true,
	"oauth": true, "oidc": true, "jwt": true, "saml": true, "rbac": true,
	"websocket": true, "websockets": true, "protobuf": true, "openapi": true, "swagger": true,
	"mockito": true, "jest": true, "mocha": true, "playwright": true,
	"flink": true, "pulsar": true, "cassandra": true, "neo4j": true,
	"sqlite": true, "oracle": true, "mssql": true, "mariadb": true,
	"html": true, "css": true, "sass": true, "less": true, "wasm": true,
	"swiftui": true, "compose": true,
	"unix": true, "networking": true, "tcp": true, "udp": true, "dns": true,
	"monitoring": true, "logging": true, "alerting": true, "tracing": true,
	"incident": true, "oncall": true, "sla": true, "slo": true, "error-budget": true,
	"platform": true, "reliability": true, "scalability": true, "latency": true,
	"concurrency": true, "multithreading": true, "async": true, "asyncio": true,
	"gin": true, "fiber": true, "echo": true, "chi": true,
	"springboot": true, "quarkus": true, "micronaut": true,
}

var multiWord = []string{
	"machine learning", "deep learning", "computer science", "software engineer",
	"full stack", "fullstack", "front end", "frontend", "back end", "backend",
	"react native", "node.js", "next.js", "ci/cd", "rest api", "graphql",
	"unit testing", "system design", "data structures", "distributed systems",
	"cloud computing", "amazon web services", "google cloud", "continuous integration",
	"continuous delivery", "object oriented", "test driven", "version control",
	"github actions", "gitlab ci", "message queue", "event driven",
	"infrastructure as code", "site reliability", "observability stack",
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
	resume := normalize(resumeText)
	keywords := ExtractKeywords(title + "\n" + description)
	if len(keywords) == 0 {
		return Result{MatchScore: 0}
	}

	var present, missing []string
	for _, kw := range keywords {
		if strings.Contains(resume, normalize(kw)) {
			present = append(present, kw)
		} else {
			missing = append(missing, kw)
		}
	}

	score := 100.0 * float64(len(present)) / float64(len(keywords))
	if score > 0 && score < 5 {
		score = 5
	}
	if score > 98 && len(missing) > 0 {
		score = 98
	}

	// Missing keywords = full missing list (what users expect to see).
	// Missing skills = tech/tool subset of those.
	keys := append([]string(nil), missing...)
	var skills []string
	for _, m := range missing {
		if isTechSkill(m) {
			skills = append(skills, m)
		}
	}
	if len(keys) > 16 {
		keys = keys[:16]
	}
	if len(skills) > 12 {
		skills = skills[:12]
	}
	if len(present) > 16 {
		present = present[:16]
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
	lower := normalize(text)
	seen := map[string]bool{}
	var out []string

	add := func(s string) {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.Trim(s, ".,;:()[]{}\"'")
		s = strings.ReplaceAll(s, "—", "-")
		if s == "" || fluff[s] || seen[s] {
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

	// Only keep concrete tech tokens from the JD (plus symbol/digit tokens).
	re := regexp.MustCompile(`[A-Za-z][A-Za-z0-9.+#/\-]{1,30}`)
	for _, tok := range re.FindAllString(lower, -1) {
		tok = strings.Trim(tok, "-/")
		if fluff[tok] {
			continue
		}
		if !isScorableToken(tok) {
			continue
		}
		add(tok)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return keywordWeight(out[i]) > keywordWeight(out[j])
	})
	if len(out) > 28 {
		out = out[:28]
	}
	return out
}

func isScorableToken(tok string) bool {
	if tech[tok] {
		return true
	}
	// c++, node.js, ci/cd, etc.
	if strings.ContainsAny(tok, "0123456789.+#/") {
		return len(tok) >= 2 && !fluff[tok]
	}
	return false
}

func isTechSkill(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if tech[s] {
		return true
	}
	for _, p := range multiWord {
		if s == p {
			return true
		}
	}
	return strings.ContainsAny(s, "0123456789.+#/")
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "system-design", "system design")
	s = strings.ReplaceAll(s, "full-stack", "full stack")
	s = strings.ReplaceAll(s, "front-end", "front end")
	s = strings.ReplaceAll(s, "back-end", "back end")
	s = strings.ReplaceAll(s, "ci / cd", "ci/cd")
	return s
}

func keywordWeight(s string) int {
	w := len(s)
	if tech[s] {
		w += 20
	}
	if strings.ContainsAny(s, "0123456789.+#/") {
		w += 10
	}
	if strings.Contains(s, " ") {
		w += 8
	}
	return w
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
