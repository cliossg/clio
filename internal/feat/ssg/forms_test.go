package ssg

import (
	"strings"
	"testing"
)

func TestProcessForms_Disabled(t *testing.T) {
	input := `<pre><code class="language-form">type: contact</code></pre>`
	result := processForms(input, "site-123", "", false)
	if result != input {
		t.Errorf("expected unchanged input when forms disabled, got %s", result)
	}
}

func TestProcessForms_ContactForm(t *testing.T) {
	input := `<p>Hello</p><pre><code class="language-form">type: contact</code></pre><p>Bye</p>`
	result := processForms(input, "site-123", "http://localhost:8081", true)

	if !strings.Contains(result, `<form class="clio-form"`) {
		t.Error("expected form element in output")
	}
	if !strings.Contains(result, `action="http://localhost:8081/api/v1/forms/submit"`) {
		t.Error("expected correct action URL")
	}
	if !strings.Contains(result, `value="site-123"`) {
		t.Error("expected site ID in hidden field")
	}
	if !strings.Contains(result, `name="_honeypot"`) {
		t.Error("expected honeypot field")
	}
	if !strings.Contains(result, `name="name"`) {
		t.Error("expected name field")
	}
	if !strings.Contains(result, `name="email"`) {
		t.Error("expected email field")
	}
	if !strings.Contains(result, `name="message"`) {
		t.Error("expected message field")
	}
	if !strings.Contains(result, `<p>Hello</p>`) {
		t.Error("expected surrounding HTML to be preserved")
	}
	if !strings.Contains(result, `<p>Bye</p>`) {
		t.Error("expected surrounding HTML to be preserved")
	}
}

func TestProcessForms_RelativePath(t *testing.T) {
	input := `<pre><code class="language-form">type: contact</code></pre>`
	result := processForms(input, "site-123", "", true)
	if !strings.Contains(result, `action="/api/v1/forms/submit"`) {
		t.Errorf("expected relative action URL when endpoint is empty, got %s", result)
	}
}

func TestProcessForms_UnknownType(t *testing.T) {
	input := `<pre><code class="language-form">type: newsletter</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081", true)
	if result != input {
		t.Errorf("expected unchanged input for unknown type, got %s", result)
	}
}

func TestProcessForms_InvalidYAML(t *testing.T) {
	input := `<pre><code class="language-form">: invalid: yaml: [</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081", true)
	if result != input {
		t.Errorf("expected unchanged input for invalid YAML, got %s", result)
	}
}

func TestProcessForms_TrailingSlashEndpoint(t *testing.T) {
	input := `<pre><code class="language-form">type: contact</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081/", true)
	if !strings.Contains(result, `action="http://localhost:8081/api/v1/forms/submit"`) {
		t.Error("expected trailing slash to be trimmed from endpoint")
	}
}

func TestProcessForms_MultipleBlocks(t *testing.T) {
	input := `<pre><code class="language-form">type: contact</code></pre><p>sep</p><pre><code class="language-form">type: contact</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081", true)
	count := strings.Count(result, `<form class="clio-form"`)
	if count != 2 {
		t.Errorf("expected 2 forms, got %d", count)
	}
}

func TestProcessForms_HTMLEscapedContent(t *testing.T) {
	input := `<pre><code class="language-form">type: contact
</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081", true)
	if !strings.Contains(result, `<form class="clio-form"`) {
		t.Error("expected form to be generated from HTML-escaped content")
	}
}

func TestProcessForms_NoFormBlocks(t *testing.T) {
	input := `<p>Hello world</p><pre><code class="language-go">func main() {}</code></pre>`
	result := processForms(input, "site-123", "http://localhost:8081", true)
	if result != input {
		t.Errorf("expected unchanged input when no form blocks, got %s", result)
	}
}
