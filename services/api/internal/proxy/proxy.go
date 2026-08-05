package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobright/api/pkg/response"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

func Handler(c *gin.Context) {
	raw := c.Query("url")
	if raw == "" {
		response.BadRequest(c, "url is required")
		return
	}
	target, err := url.Parse(raw)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") {
		response.BadRequest(c, "invalid url")
		return
	}
	host := strings.ToLower(target.Hostname())
	if host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".local") {
		response.BadRequest(c, "url not allowed")
		return
	}

	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; JobRightApply/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		response.Internal(c, "failed to fetch job page")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		response.Internal(c, "failed to read job page")
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/html; charset=utf-8"
	}

	if strings.Contains(strings.ToLower(contentType), "text/html") {
		html := string(body)
		base := target.Scheme + "://" + target.Host + "/"
		if !strings.Contains(strings.ToLower(html), "<base ") {
			html = strings.Replace(html, "<head>", "<head><base href=\""+base+"\">", 1)
			html = strings.Replace(html, "<HEAD>", "<HEAD><base href=\""+base+"\">", 1)
		}
		html = injectAutofillBridge(html)
		body = []byte(html)
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Frame-Options", "ALLOWALL")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(body)
}

func injectAutofillBridge(html string) string {
	script := `<script>
(function(){
  window.addEventListener('message', function(event){
    if (!event.data || event.data.type !== 'jobright-autofill') return;
    var data = event.data.payload || {};
    var map = [
      ['input[name*=email i],input[type=email],input[autocomplete=email]', data.email],
      ['input[name*=name i],input[autocomplete=name],input[name*=full_name i]', data.name],
      ['input[name*=phone i],input[type=tel],input[autocomplete=tel]', data.phone],
      ['input[name*=linkedin i]', data.linkedin],
      ['input[name*=github i]', data.github],
      ['input[name*=portfolio i],input[name*=website i]', data.website],
      ['textarea[name*=cover i],textarea[id*=cover i]', data.coverLetter]
    ];
    map.forEach(function(pair){
      if (!pair[1]) return;
      document.querySelectorAll(pair[0]).forEach(function(el){
        el.focus();
        el.value = pair[1];
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
    });
  });
})();
</script>`
	if strings.Contains(strings.ToLower(html), "</body>") {
		return strings.Replace(html, "</body>", script+"</body>", 1)
	}
	return html + script
}
