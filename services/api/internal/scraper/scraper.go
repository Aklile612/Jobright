package scraper

import (
	"net/url"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/jobright/api/internal/jobs"
	"github.com/jobright/api/internal/models"
)

type Service struct {
	jobs *jobs.Service
}

func NewService(jobSvc *jobs.Service) *Service {
	return &Service{jobs: jobSvc}
}

type Result struct {
	Ingested int      `json:"ingested"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

func (s *Service) ScrapeURLs(urls []string) Result {
	var (
		mu     sync.Mutex
		result Result
		wg     sync.WaitGroup
	)
	sem := make(chan struct{}, 8)
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(target string) {
			defer wg.Done()
			defer func() { <-sem }()
			job, err := scrapePage(target)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, target+": "+err.Error())
				return
			}
			if err := s.jobs.UpsertBySourceURL(job); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, target+": "+err.Error())
				return
			}
			result.Ingested++
		}(raw)
	}
	wg.Wait()
	return result
}

func scrapePage(target string) (*models.Job, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.UserAgent("jobright-scraper/1.0"),
	)
	job := &models.Job{SourceURL: target}
	c.OnHTML("title", func(e *colly.HTMLElement) {
		if job.Title == "" {
			job.Title = strings.TrimSpace(e.Text)
		}
	})
	c.OnHTML("meta[property='og:title']", func(e *colly.HTMLElement) {
		if v := strings.TrimSpace(e.Attr("content")); v != "" {
			job.Title = v
		}
	})
	c.OnHTML("meta[property='og:description']", func(e *colly.HTMLElement) {
		if v := strings.TrimSpace(e.Attr("content")); v != "" {
			job.Description = v
		}
	})
	c.OnHTML("h1", func(e *colly.HTMLElement) {
		if job.Title == "" {
			job.Title = strings.TrimSpace(e.Text)
		}
	})
	c.OnHTML("body", func(e *colly.HTMLElement) {
		if job.Description == "" {
			text := strings.Join(strings.Fields(e.Text), " ")
			if len(text) > 4000 {
				text = text[:4000]
			}
			job.Description = text
		}
	})
	if err := c.Visit(target); err != nil {
		return nil, err
	}
	if job.Title == "" {
		job.Title = parsed.Host + " role"
	}
	if job.Description == "" {
		job.Description = "Imported from " + target
	}
	job.Company = parsed.Host
	return job, nil
}
