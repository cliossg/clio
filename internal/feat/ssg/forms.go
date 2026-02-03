package ssg

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormConfig represents the YAML configuration inside a ```form code block.
type FormConfig struct {
	Type string `yaml:"type"`
}

var formCodeBlockRegex = regexp.MustCompile(`<pre><code class="language-form">([\s\S]*?)</code></pre>`)

// processForms replaces ```form code blocks with HTML form elements.
// If endpointURL is empty, uses a relative path (/api/v1/forms/submit).
// If formsEnabled is false, form blocks are left untouched.
func processForms(htmlContent string, siteID string, endpointURL string, formsEnabled bool) string {
	if !formsEnabled {
		return htmlContent
	}

	return formCodeBlockRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := formCodeBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		content := strings.TrimSpace(submatches[1])
		content = unescapeHTML(content)

		var config FormConfig
		if err := yaml.Unmarshal([]byte(content), &config); err != nil {
			return match
		}

		if config.Type != "contact" {
			return match
		}

		return generateContactForm(siteID, endpointURL)
	})
}

// generateContactForm returns the HTML for a contact form.
// If endpointURL is empty, uses a relative path.
func generateContactForm(siteID string, endpointURL string) string {
	var action string
	if endpointURL != "" {
		action = html.EscapeString(strings.TrimRight(endpointURL, "/") + "/api/v1/forms/submit")
	} else {
		action = "/api/v1/forms/submit"
	}
	escapedSiteID := html.EscapeString(siteID)

	return fmt.Sprintf(`<form class="clio-form" action="%s" method="POST">
  <input type="hidden" name="_site" value="%s">
  <input type="hidden" name="_form" value="contact">
  <input type="text" name="_honeypot" style="display:none" tabindex="-1" autocomplete="off">
  <div class="form-field">
    <label for="cf-name">Name</label>
    <input type="text" id="cf-name" name="name" required>
  </div>
  <div class="form-field">
    <label for="cf-email">Email</label>
    <input type="email" id="cf-email" name="email" required>
  </div>
  <div class="form-field">
    <label for="cf-message">Message</label>
    <textarea id="cf-message" name="message" rows="5" required></textarea>
  </div>
  <button type="submit">Send</button>
  <div class="form-status"></div>
</form>
<script>
(function(){
  var form = document.querySelector('.clio-form');
  if (!form) return;
  form.addEventListener('submit', function(e) {
    e.preventDefault();
    var btn = form.querySelector('button[type="submit"]');
    var status = form.querySelector('.form-status');
    btn.disabled = true;
    btn.textContent = 'Sending...';
    status.className = 'form-status';
    status.style.display = 'none';
    fetch(form.action, {
      method: 'POST',
      headers: {'Accept': 'application/json'},
      body: new FormData(form)
    }).then(function(r) {
      if (r.ok) {
        status.className = 'form-status success';
        status.textContent = 'Message sent. Thank you!';
        status.style.display = 'block';
        form.reset();
      } else {
        return r.json().then(function(d) { throw new Error(d.error || 'Failed'); });
      }
    }).catch(function(err) {
      status.className = 'form-status error';
      status.textContent = err.message || 'Something went wrong. Please try again.';
      status.style.display = 'block';
    }).finally(function() {
      btn.disabled = false;
      btn.textContent = 'Send';
    });
  });
})();
</script>`, action, escapedSiteID)
}
