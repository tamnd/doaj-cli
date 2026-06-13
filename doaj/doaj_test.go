package doaj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	return NewClient(cfg)
}

const articleSearchPayload = `{
	"total": 2,
	"pageSize": 10,
	"page": 1,
	"results": [
		{
			"id": "art001",
			"bibjson": {
				"title": "Machine Learning in Science",
				"author": [{"name": "Doe, John"}, {"name": "Smith, Jane"}, {"name": "Kumar, Raj"}, {"name": "Chen, Wei"}],
				"journal": {"title": "Journal of AI", "issns": ["1234-5678"]},
				"year": "2023",
				"month": "05",
				"abstract": "An abstract.",
				"identifier": [{"type": "doi", "id": "10.1234/jai.2023.001"}, {"type": "purl", "id": "xyz"}],
				"link": [{"type": "fulltext", "url": "https://example.com/art001"}],
				"subject": [{"term": "Computer Science"}]
			}
		},
		{
			"id": "art002",
			"bibjson": {
				"title": "Deep Learning Methods",
				"author": [{"name": "Lee, Anna"}],
				"journal": {"title": "Neural Computing", "issns": ["8765-4321"]},
				"year": "2024",
				"identifier": [],
				"link": [],
				"subject": []
			}
		}
	]
}`

const journalSearchPayload = `{
	"total": 1,
	"pageSize": 10,
	"page": 1,
	"results": [
		{
			"id": "jrn001",
			"bibjson": {
				"title": "Open Biology Journal",
				"publisher": {"name": "Open Science Publishers"},
				"subject": [{"term": "Biology", "code": "Q"}],
				"language": ["EN", "FR"],
				"country": {"code": "GB", "name": "United Kingdom"},
				"eissn": "2345-6789",
				"pissn": "1234-5678",
				"apc": {"has_apc": false},
				"license": [{"type": "CC BY", "url": "https://creativecommons.org/licenses/by/4.0/"}],
				"ref": {"homepage": "https://openbiology.example.com/"}
			},
			"created_date": "2020-01-01T00:00:00Z"
		}
	]
}`

const singleJournalPayload = `{
	"id": "jrn002",
	"bibjson": {
		"title": "PLOS ONE",
		"publisher": {"name": "Public Library of Science"},
		"language": ["EN"],
		"country": {"code": "US", "name": "United States"},
		"eissn": "1932-6203",
		"pissn": "",
		"apc": {"has_apc": true},
		"ref": {"homepage": "https://journals.plos.org/plosone/"}
	},
	"created_date": "2010-01-01T00:00:00Z"
}`

const singleArticlePayload = `{
	"id": "art999",
	"bibjson": {
		"title": "Climate Change Impacts",
		"author": [{"name": "Müller, Hans"}, {"name": "Garcia, Maria"}],
		"journal": {"title": "Environmental Science", "issns": ["9999-0000"]},
		"year": "2022",
		"identifier": [{"type": "doi", "id": "10.9999/env.2022.999"}],
		"link": [{"type": "fulltext", "url": "https://env.example.com/art999"}],
		"subject": [{"term": "Environment"}]
	}
}`

func TestSearchArticlesSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"total":0,"pageSize":10,"page":1,"results":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _ = c.SearchArticles(context.Background(), "test", 5)
	if gotUA == "" {
		t.Error("request carried no User-Agent")
	}
}

func TestSearchArticlesParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(articleSearchPayload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	arts, err := c.SearchArticles(context.Background(), "machine learning", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("got %d articles, want 2", len(arts))
	}

	a := arts[0]
	if a.ID != "art001" {
		t.Errorf("ID = %q, want art001", a.ID)
	}
	if a.Title != "Machine Learning in Science" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.Authors != "Doe, John, Smith, Jane, Kumar, Raj et al." {
		t.Errorf("Authors = %q", a.Authors)
	}
	if a.Journal != "Journal of AI" {
		t.Errorf("Journal = %q", a.Journal)
	}
	if a.Year != "2023" {
		t.Errorf("Year = %q", a.Year)
	}
	if a.DOI != "10.1234/jai.2023.001" {
		t.Errorf("DOI = %q", a.DOI)
	}
	if !a.OpenAccess {
		t.Error("OpenAccess should be true")
	}
	if a.URL != "https://doaj.org/article/art001" {
		t.Errorf("URL = %q", a.URL)
	}
	if a.Rank != 1 {
		t.Errorf("Rank = %d, want 1", a.Rank)
	}

	// Second article has no DOI.
	b := arts[1]
	if b.DOI != "" {
		t.Errorf("expected empty DOI, got %q", b.DOI)
	}
	if b.Authors != "Lee, Anna" {
		t.Errorf("Authors = %q", b.Authors)
	}
}

func TestSearchJournalsParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(journalSearchPayload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	journals, err := c.SearchJournals(context.Background(), "biology", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(journals) != 1 {
		t.Fatalf("got %d journals, want 1", len(journals))
	}

	j := journals[0]
	if j.ISSN != "2345-6789" {
		t.Errorf("ISSN = %q, want 2345-6789 (eissn preferred)", j.ISSN)
	}
	if j.Title != "Open Biology Journal" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Publisher != "Open Science Publishers" {
		t.Errorf("Publisher = %q", j.Publisher)
	}
	if j.Country != "United Kingdom" {
		t.Errorf("Country = %q", j.Country)
	}
	if j.Language != "EN/FR" {
		t.Errorf("Language = %q, want EN/FR", j.Language)
	}
	if j.APC {
		t.Error("APC should be false")
	}
	if j.URL != "https://doaj.org/toc/2345-6789" {
		t.Errorf("URL = %q", j.URL)
	}
}

func TestGetJournalByISSN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/journals/1932-6203" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Return a search response wrapping the single journal.
		payload := `{"total":1,"pageSize":1,"page":1,"results":[` + singleJournalPayload + `]}`
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	j, err := c.GetJournal(context.Background(), "1932-6203")
	if err != nil {
		t.Fatal(err)
	}
	if j.Title != "PLOS ONE" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.ISSN != "1932-6203" {
		t.Errorf("ISSN = %q", j.ISSN)
	}
	if !j.APC {
		t.Error("APC should be true")
	}
}

func TestGetArticleByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/articles/art999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(singleArticlePayload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	a, err := c.GetArticle(context.Background(), "art999")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "art999" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.DOI != "10.9999/env.2022.999" {
		t.Errorf("DOI = %q", a.DOI)
	}
	if a.Authors != "Müller, Hans, Garcia, Maria" {
		t.Errorf("Authors = %q", a.Authors)
	}
}

func TestGetJournalNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total":0,"pageSize":1,"page":1,"results":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetJournal(context.Background(), "9999-9999")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestSearchArticlesRetries(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"total":0,"pageSize":10,"page":1,"results":[]}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	_, err := c.SearchArticles(context.Background(), "test", 5)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		authors []wireAuthor
		want    string
	}{
		{nil, ""},
		{[]wireAuthor{{Name: "Doe, John"}}, "Doe, John"},
		{[]wireAuthor{{Name: "A"}, {Name: "B"}}, "A, B"},
		{[]wireAuthor{{Name: "A"}, {Name: "B"}, {Name: "C"}}, "A, B, C"},
		{[]wireAuthor{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}}, "A, B, C et al."},
	}
	for _, tc := range tests {
		got := formatAuthors(tc.authors)
		if got != tc.want {
			t.Errorf("formatAuthors(%v) = %q, want %q", tc.authors, got, tc.want)
		}
	}
}

func TestExtractDOI(t *testing.T) {
	ids := []wireIdentifier{
		{Type: "purl", ID: "purl:123"},
		{Type: "doi", ID: "10.1234/test"},
		{Type: "pmid", ID: "98765"},
	}
	got := extractDOI(ids)
	if got != "10.1234/test" {
		t.Errorf("extractDOI = %q, want 10.1234/test", got)
	}

	got = extractDOI(nil)
	if got != "" {
		t.Errorf("extractDOI(nil) = %q, want empty", got)
	}
}

func TestJoinLanguages(t *testing.T) {
	tests := []struct {
		langs []string
		want  string
	}{
		{nil, ""},
		{[]string{"EN"}, "EN"},
		{[]string{"EN", "FR"}, "EN/FR"},
		{[]string{"EN", "FR", "ZH"}, "EN/FR/ZH"},
	}
	for _, tc := range tests {
		got := joinLanguages(tc.langs)
		if got != tc.want {
			t.Errorf("joinLanguages(%v) = %q, want %q", tc.langs, got, tc.want)
		}
	}
}

func TestNormaliseISSN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1234-5678", "1234-5678"},
		{"12345678", "1234-5678"},
		{"1932-6203", "1932-6203"},
	}
	for _, tc := range tests {
		got := normaliseISSN(tc.input)
		if got != tc.want {
			t.Errorf("normaliseISSN(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
