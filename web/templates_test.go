package web

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// openModal/closeModal live in /static/js/admin.js. A page that renders them —
// directly or through any partial it includes — but forgets the <script> tag
// still swaps HTMX content into a modal that never becomes visible: the button
// looks dead with no console-visible failure. Guarding the whole template tree
// catches it once for every page instead of per-page review.

var (
	reDefine       = regexp.MustCompile(`{{\s*define\s+"([^"]+)"\s*}}`)
	reTemplateCall = regexp.MustCompile(`{{\s*template\s+"([^"]+)"`)
	reAdminScript  = regexp.MustCompile(`src="/static/js/admin\.js"`)
	reModalHelper  = regexp.MustCompile(`\b(?:open|close)Modal\(`)
)

func TestPagesRenderingModalHelpersLoadAdminJS(t *testing.T) {
	files := map[string]string{} // path -> content
	defined := map[string]string{}
	err := filepath.WalkDir("templates", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[path] = string(b)
		for _, m := range reDefine.FindAllStringSubmatch(string(b), -1) {
			defined[m[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no templates found — is the test running from web/?")
	}

	for path, content := range files {
		if !strings.Contains(content, "<!DOCTYPE html>") {
			continue // partial: it inherits the page's <script> tags
		}
		if needsModalHelpers(path, files, defined, map[string]bool{}) && !reAdminScript.MatchString(content) {
			t.Errorf("%s renders openModal/closeModal but does not load /static/js/admin.js — its modals will never open", path)
		}
	}
}

// needsModalHelpers reports whether path, or anything it transitively includes,
// calls openModal/closeModal.
func needsModalHelpers(path string, files, defined map[string]string, seen map[string]bool) bool {
	if seen[path] {
		return false
	}
	seen[path] = true

	content := files[path]
	if reModalHelper.MatchString(content) {
		return true
	}
	for _, m := range reTemplateCall.FindAllStringSubmatch(content, -1) {
		if target, ok := defined[m[1]]; ok && needsModalHelpers(target, files, defined, seen) {
			return true
		}
	}
	return false
}
