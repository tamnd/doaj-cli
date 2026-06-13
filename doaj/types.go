package doaj

import "strings"

// Article is the record emitted for DOAJ article search results and single-article fetches.
type Article struct {
	Rank       int    `json:"rank"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Authors    string `json:"authors"`
	Journal    string `json:"journal"`
	Year       string `json:"year"`
	DOI        string `json:"doi"`
	OpenAccess bool   `json:"open_access"`
	URL        string `json:"url"`
}

// Journal is the record emitted for DOAJ journal search results and single-journal fetches.
type Journal struct {
	Rank      int    `json:"rank"`
	ISSN      string `json:"issn"`
	Title     string `json:"title"`
	Publisher string `json:"publisher"`
	Country   string `json:"country"`
	Language  string `json:"language"`
	APC       bool   `json:"has_apc"`
	URL       string `json:"url"`
}

// ─── wire types for journals ─────────────────────────────────────────────────

type wireJournalResult struct {
	ID          string         `json:"id"`
	Bibjson     wireJournalBib `json:"bibjson"`
	CreatedDate string         `json:"created_date"`
}

type wireJournalBib struct {
	Title            string        `json:"title"`
	AlternativeTitle string        `json:"alternative_title"`
	Publisher        wirePub       `json:"publisher"`
	Subject          []wireSubject `json:"subject"`
	Language         []string      `json:"language"`
	Country          wireCountry   `json:"country"`
	EISSN            string        `json:"eissn"`
	PISSN            string        `json:"pissn"`
	APC              wireAPC       `json:"apc"`
	License          []wireLicense `json:"license"`
	Ref              wireRef       `json:"ref"`
}

type wirePub struct {
	Name string `json:"name"`
}

type wireSubject struct {
	Term string `json:"term"`
	Code string `json:"code"`
}

type wireCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type wireAPC struct {
	HasAPC bool `json:"has_apc"`
}

type wireLicense struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type wireRef struct {
	Homepage string `json:"homepage"`
}

// ─── wire types for articles ─────────────────────────────────────────────────

type wireArticleResult struct {
	ID      string         `json:"id"`
	Bibjson wireArticleBib `json:"bibjson"`
}

type wireArticleBib struct {
	Title      string           `json:"title"`
	Author     []wireAuthor     `json:"author"`
	Journal    wireJournalRef   `json:"journal"`
	Year       string           `json:"year"`
	Month      string           `json:"month"`
	Abstract   string           `json:"abstract"`
	Identifier []wireIdentifier `json:"identifier"`
	Link       []wireLink       `json:"link"`
	Subject    []wireSubject    `json:"subject"`
}

type wireAuthor struct {
	Name string `json:"name"`
}

type wireJournalRef struct {
	Title string   `json:"title"`
	ISSNs []string `json:"issns"`
}

type wireIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type wireLink struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// ─── response envelopes ───────────────────────────────────────────────────────

type journalSearchResp struct {
	Total    int                 `json:"total"`
	PageSize int                 `json:"pageSize"`
	Page     int                 `json:"page"`
	Results  []wireJournalResult `json:"results"`
}

type articleSearchResp struct {
	Total    int                 `json:"total"`
	PageSize int                 `json:"pageSize"`
	Page     int                 `json:"page"`
	Results  []wireArticleResult `json:"results"`
}

// ─── mapping helpers ─────────────────────────────────────────────────────────

func wireJournalToJournal(w wireJournalResult, rank int) Journal {
	issn := preferredISSN(w.Bibjson)
	u := "https://doaj.org/toc/" + issn
	if issn == "" {
		u = "https://doaj.org/"
	}
	return Journal{
		Rank:      rank,
		ISSN:      issn,
		Title:     w.Bibjson.Title,
		Publisher: w.Bibjson.Publisher.Name,
		Country:   w.Bibjson.Country.Name,
		Language:  joinLanguages(w.Bibjson.Language),
		APC:       w.Bibjson.APC.HasAPC,
		URL:       u,
	}
}

func wireArticleToArticle(w wireArticleResult, rank int) Article {
	return Article{
		Rank:       rank,
		ID:         w.ID,
		Title:      w.Bibjson.Title,
		Authors:    formatAuthors(w.Bibjson.Author),
		Journal:    w.Bibjson.Journal.Title,
		Year:       w.Bibjson.Year,
		DOI:        extractDOI(w.Bibjson.Identifier),
		OpenAccess: true,
		URL:        "https://doaj.org/article/" + w.ID,
	}
}

func formatAuthors(authors []wireAuthor) string {
	const maxShown = 3
	names := make([]string, 0, len(authors))
	for i, a := range authors {
		if i >= maxShown {
			break
		}
		names = append(names, a.Name)
	}
	s := strings.Join(names, ", ")
	if len(authors) > maxShown {
		s += " et al."
	}
	return s
}

func extractDOI(ids []wireIdentifier) string {
	for _, id := range ids {
		if strings.EqualFold(id.Type, "doi") {
			return id.ID
		}
	}
	return ""
}

func joinLanguages(langs []string) string {
	return strings.Join(langs, "/")
}

func preferredISSN(bib wireJournalBib) string {
	if bib.EISSN != "" {
		return bib.EISSN
	}
	return bib.PISSN
}
