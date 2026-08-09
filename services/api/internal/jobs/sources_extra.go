package jobs

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jobright/api/internal/models"
)

func (s *Service) syncMuse() SyncResult {
	res := SyncResult{Source: "themuse"}
	client := &http.Client{Timeout: 30 * time.Second}
	// Public API; optional key raises rate limits.
	for page := 0; page < 5; page++ {
		q := url.Values{}
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("category", "Software Engineering")
		q.Set("descending", "true")
		if key := strings.TrimSpace(s.config.MuseAPIKey); key != "" {
			q.Set("api_key", key)
		}
		endpoint := "https://www.themuse.com/api/public/jobs?" + q.Encode()
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			res.Error = err.Error()
			break
		}
		req.Header.Set("User-Agent", "jobright/1.0")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			res.Error = err.Error()
			break
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			res.Error = err.Error()
			break
		}
		if resp.StatusCode >= 300 {
			res.Error = fmt.Sprintf("http %d: %s", resp.StatusCode, trimBody(body))
			break
		}
		var payload struct {
			Results []struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Contents string `json:"contents"`
				Company  struct {
					Name string `json:"name"`
				} `json:"company"`
				Locations []struct {
					Name string `json:"name"`
				} `json:"locations"`
				Categories []struct {
					Name string `json:"name"`
				} `json:"categories"`
				Refs map[string]string `json:"refs"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			res.Error = err.Error()
			break
		}
		if len(payload.Results) == 0 {
			break
		}
		for _, j := range payload.Results {
			cats := make([]string, 0, len(j.Categories))
			for _, c := range j.Categories {
				cats = append(cats, c.Name)
			}
			if !isSoftwareRelated(j.Name, cats) {
				continue
			}
			loc := "Remote"
			if len(j.Locations) > 0 && strings.TrimSpace(j.Locations[0].Name) != "" {
				loc = j.Locations[0].Name
			}
			source := firstMapValue(j.Refs, "landing_page", "external_link", "internal_link")
			if source == "" && j.ID > 0 {
				source = fmt.Sprintf("https://www.themuse.com/jobs/%d", j.ID)
			}
			job := &models.Job{
				Title:       strings.TrimSpace(j.Name),
				Company:     nonEmpty(j.Company.Name, "The Muse"),
				Description: stripHTML(j.Contents),
				Location:    loc,
				SourceURL:   strings.TrimSpace(source),
			}
			if job.Title == "" || job.SourceURL == "" {
				continue
			}
			if err := s.UpsertBySourceURL(job); err == nil {
				res.Ingested++
			}
		}
	}
	return res
}

func (s *Service) syncJobspresso() SyncResult {
	res := SyncResult{Source: "jobspresso"}
	req, err := http.NewRequest(http.MethodGet, "https://jobspresso.co/?feed=job_feed", nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "jobright/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if resp.StatusCode >= 300 {
		res.Error = fmt.Sprintf("http %d", resp.StatusCode)
		return res
	}

	type rssItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
		Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	}
	var feed struct {
		Channel struct {
			Items []rssItem `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		res.Error = err.Error()
		return res
	}
	for _, item := range feed.Channel.Items {
		title := html.UnescapeString(strings.TrimSpace(item.Title))
		if !isSoftwareRelated(title, nil) {
			continue
		}
		company, loc := parseJobspressoCreator(item.Creator)
		desc := item.Content
		if strings.TrimSpace(desc) == "" {
			desc = item.Description
		}
		job := &models.Job{
			Title:       title,
			Company:     nonEmpty(company, "Jobspresso"),
			Description: stripHTML(html.UnescapeString(desc)),
			Location:    nonEmpty(loc, "Remote"),
			SourceURL:   strings.TrimSpace(item.Link),
		}
		if job.Title == "" || job.SourceURL == "" {
			continue
		}
		if err := s.UpsertBySourceURL(job); err == nil {
			res.Ingested++
		}
	}
	return res
}

func (s *Service) syncAdzuna() SyncResult {
	res := SyncResult{Source: "adzuna"}
	appID := strings.TrimSpace(s.config.AdzunaAppID)
	appKey := strings.TrimSpace(s.config.AdzunaAppKey)
	if appID == "" || appKey == "" {
		res.Error = "skipped — set ADZUNA_APP_ID and ADZUNA_APP_KEY"
		return res
	}
	countries := s.config.AdzunaCountries
	if len(countries) == 0 {
		countries = []string{"us", "gb"}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	// Fresh software roles only (today + yesterday ≈ max_days_old=2).
	queries := []string{"software engineer", "software developer", "software jobs"}
	for _, country := range countries {
		country = strings.ToLower(strings.TrimSpace(country))
		if country == "" {
			continue
		}
		for _, what := range queries {
			for page := 1; page <= 2; page++ {
				q := url.Values{}
				q.Set("app_id", appID)
				q.Set("app_key", appKey)
				q.Set("results_per_page", "50")
				q.Set("what", what)
				q.Set("category", "it-jobs")
				q.Set("max_days_old", "2")
				q.Set("sort_by", "date")
				q.Set("content-type", "application/json")
				endpoint := fmt.Sprintf("https://api.adzuna.com/v1/api/jobs/%s/search/%d?%s", country, page, q.Encode())
				req, err := http.NewRequest(http.MethodGet, endpoint, nil)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				req.Header.Set("User-Agent", "jobright/1.0")
				req.Header.Set("Accept", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
				resp.Body.Close()
				if err != nil {
					res.Error = err.Error()
					return res
				}
				if resp.StatusCode >= 300 {
					res.Error = fmt.Sprintf("http %d: %s", resp.StatusCode, trimBody(body))
					return res
				}
				var payload struct {
					Results []struct {
						Title       string  `json:"title"`
						Description string  `json:"description"`
						RedirectURL string  `json:"redirect_url"`
						SalaryMin   float64 `json:"salary_min"`
						SalaryMax   float64 `json:"salary_max"`
						Company     struct {
							DisplayName string `json:"display_name"`
						} `json:"company"`
						Location struct {
							DisplayName string `json:"display_name"`
						} `json:"location"`
					} `json:"results"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					res.Error = err.Error()
					return res
				}
				if len(payload.Results) == 0 {
					break
				}
				for _, j := range payload.Results {
					if !isSoftwareRelated(j.Title, nil) {
						continue
					}
					salary := formatSalaryRange(j.SalaryMin, j.SalaryMax)
					job := &models.Job{
						Title:       strings.TrimSpace(j.Title),
						Company:     nonEmpty(j.Company.DisplayName, "Adzuna"),
						Description: stripHTML(j.Description),
						Location:    nonEmpty(j.Location.DisplayName, strings.ToUpper(country)),
						SourceURL:   strings.TrimSpace(j.RedirectURL),
						SalaryRange: salary,
					}
					if job.Title == "" || job.SourceURL == "" {
						continue
					}
					if err := s.UpsertBySourceURL(job); err == nil {
						res.Ingested++
					}
				}
			}
		}
	}
	return res
}

func (s *Service) syncJSearch() SyncResult {
	res := SyncResult{Source: "jsearch"}
	key := strings.TrimSpace(s.config.JSearchAPIKey)
	if key == "" {
		res.Error = "skipped — set JSEARCH_API_KEY or RAPIDAPI_KEY"
		return res
	}
	client := &http.Client{Timeout: 35 * time.Second}
	// JSearch date_posted options: today | 3days | week | month.
	// "3days" is the closest filter that covers today + yesterday.
	queries := []string{
		"software engineer",
		"software developer",
		"software jobs",
	}
	for _, query := range queries {
		for page := 1; page <= 1; page++ {
			q := url.Values{}
			q.Set("query", query)
			q.Set("page", fmt.Sprintf("%d", page))
			q.Set("num_pages", "1")
			q.Set("date_posted", "3days")
			q.Set("country", "us")
			endpoint := "https://jsearch.p.rapidapi.com/search-v2?" + q.Encode()
			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				res.Error = err.Error()
				return res
			}
			req.Header.Set("User-Agent", "jobright/1.0")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-RapidAPI-Key", key)
			req.Header.Set("X-RapidAPI-Host", "jsearch.p.rapidapi.com")
			resp, err := client.Do(req)
			if err != nil {
				res.Error = err.Error()
				return res
			}
			body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
			resp.Body.Close()
			if err != nil {
				res.Error = err.Error()
				return res
			}
			if resp.StatusCode >= 300 {
				// Fall back to classic /search if search-v2 is unavailable.
				endpoint = "https://jsearch.p.rapidapi.com/search?" + q.Encode()
				req2, err2 := http.NewRequest(http.MethodGet, endpoint, nil)
				if err2 != nil {
					res.Error = err2.Error()
					return res
				}
				req2.Header = req.Header.Clone()
				resp2, err2 := client.Do(req2)
				if err2 != nil {
					res.Error = err2.Error()
					return res
				}
				body, err = io.ReadAll(io.LimitReader(resp2.Body, 8<<20))
				resp2.Body.Close()
				if err != nil {
					res.Error = err.Error()
					return res
				}
				if resp2.StatusCode >= 300 {
					res.Error = fmt.Sprintf("http %d: %s", resp2.StatusCode, trimBody(body))
					return res
				}
			}
			rows := extractJSearchJobs(body)
			if len(rows) == 0 {
				break
			}
			for _, j := range rows {
				title := strings.TrimSpace(j.Title)
				if !isSoftwareRelated(title, j.Tags) {
					continue
				}
				job := &models.Job{
					Title:       title,
					Company:     nonEmpty(j.Company, "JSearch"),
					Description: stripHTML(j.Description),
					Location:    nonEmpty(j.Location, "Remote"),
					SourceURL:   strings.TrimSpace(j.ApplyURL),
					SalaryRange: j.Salary,
				}
				if job.Title == "" || job.SourceURL == "" {
					continue
				}
				if err := s.UpsertBySourceURL(job); err == nil {
					res.Ingested++
				}
			}
		}
	}
	return res
}

type jsearchRow struct {
	Title       string
	Company     string
	Description string
	Location    string
	ApplyURL    string
	Salary      string
	Tags        []string
}

func extractJSearchJobs(body []byte) []jsearchRow {
	// Classic shape: { "data": [ ... ] }
	var classic struct {
		Data []map[string]any `json:"data"`
	}
	if json.Unmarshal(body, &classic) == nil && len(classic.Data) > 0 {
		return mapJSearchRows(classic.Data)
	}
	// v2/v5 shapes may use "jobs"
	var v2 struct {
		Jobs []map[string]any `json:"jobs"`
		Data struct {
			Jobs []map[string]any `json:"jobs"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &v2) == nil {
		if len(v2.Jobs) > 0 {
			return mapJSearchRows(v2.Jobs)
		}
		if len(v2.Data.Jobs) > 0 {
			return mapJSearchRows(v2.Data.Jobs)
		}
	}
	return nil
}

func mapJSearchRows(rows []map[string]any) []jsearchRow {
	out := make([]jsearchRow, 0, len(rows))
	for _, row := range rows {
		title := asString(row["job_title"])
		if title == "" {
			title = asString(row["title"])
		}
		company := asString(row["employer_name"])
		if company == "" {
			company = asString(row["company"])
		}
		desc := asString(row["job_description"])
		if desc == "" {
			desc = asString(row["description"])
		}
		locParts := []string{
			asString(row["job_city"]),
			asString(row["job_state"]),
			asString(row["job_country"]),
		}
		loc := strings.Join(filterEmpty(locParts), ", ")
		if loc == "" {
			loc = asString(row["job_location"])
		}
		apply := asString(row["job_apply_link"])
		if apply == "" {
			apply = asString(row["apply_link"])
		}
		if apply == "" {
			apply = asString(row["job_google_link"])
		}
		var tags []string
		if raw, ok := row["job_required_skills"].([]any); ok {
			for _, t := range raw {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		salary := formatSalaryRange(asFloat(row["job_min_salary"]), asFloat(row["job_max_salary"]))
		out = append(out, jsearchRow{
			Title:       title,
			Company:     company,
			Description: desc,
			Location:    loc,
			ApplyURL:    apply,
			Salary:      salary,
			Tags:        tags,
		})
	}
	return out
}

func parseJobspressoCreator(raw string) (company, location string) {
	raw = html.UnescapeString(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "<br>", "\n")
	raw = strings.ReplaceAll(raw, "<br/>", "\n")
	raw = strings.ReplaceAll(raw, "<br />", "\n")
	raw = stripHTML(raw)
	raw = strings.ReplaceAll(raw, "⚲", "\n")
	parts := strings.SplitN(raw, "\n", 2)
	company = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		location = strings.TrimSpace(parts[1])
	}
	return company, location
}

func formatSalaryRange(min, max float64) string {
	switch {
	case min > 0 && max > 0:
		return fmt.Sprintf("%.0f–%.0f", min, max)
	case min > 0:
		return fmt.Sprintf("%.0f+", min)
	case max > 0:
		return fmt.Sprintf("up to %.0f", max)
	default:
		return ""
	}
}

func firstMapValue(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	return ""
}

func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 180 {
		return s[:180] + "…"
	}
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%.0f", t)
		}
		return fmt.Sprintf("%v", t)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(strings.TrimSpace(t), "%f", &f)
		return f
	default:
		return 0
	}
}

func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
