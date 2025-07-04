//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/http"
)

const (
	wikidataEndpoint     = "https://query.wikidata.org/sparql"
	dbpediaEndpoint      = "https://dbpedia.org/sparql"
	mediawikiAPIEndpoint = "https://en.wikipedia.org/w/api.php"
	requestTimeoutMs     = 5000
)

var (
	ErrNotFound       = api.ErrNotFound
	ErrNotImplemented = api.ErrNotImplemented

	client = http.NewHttpService()
)

// SPARQLResult struct for all possible fields
// Only the needed field will be non-nil in each context
// (Sitelink, Wiki, Comment, Img)
type SPARQLResult struct {
	Results struct {
		Bindings []struct {
			Sitelink *struct{ Value string } `json:"sitelink,omitempty"`
			Wiki     *struct{ Value string } `json:"wiki,omitempty"`
			Comment  *struct{ Value string } `json:"comment,omitempty"`
			Img      *struct{ Value string } `json:"img,omitempty"`
		} `json:"bindings"`
	} `json:"results"`
}

// MediaWikiExtractResult is used to unmarshal MediaWiki API extract responses
// (for getWikipediaExtract)
type MediaWikiExtractResult struct {
	Query struct {
		Pages map[string]struct {
			PageID  int    `json:"pageid"`
			Ns      int    `json:"ns"`
			Title   string `json:"title"`
			Extract string `json:"extract"`
			Missing bool   `json:"missing"`
		} `json:"pages"`
	} `json:"query"`
}

// --- SPARQL Query Helper ---
func sparqlQuery(ctx context.Context, client http.HttpService, endpoint, query string) (*SPARQLResult, error) {
	form := url.Values{}
	form.Set("query", query)

	req := &http.HttpRequest{
		Url: endpoint,
		Headers: map[string]string{
			"Accept":       "application/sparql-results+json",
			"Content-Type": "application/x-www-form-urlencoded", // Required by SPARQL endpoints
			"User-Agent":   "NavidromeWikimediaPlugin/0.1",
		},
		Body:      []byte(form.Encode()), // Send encoded form data
		TimeoutMs: requestTimeoutMs,
	}
	log.Printf("[Wikimedia Query] Attempting SPARQL query to %s (query length: %d):\n%s", endpoint, len(query), query)
	resp, err := client.Post(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("SPARQL request error: %w", err)
	}
	if resp.Status != 200 {
		log.Printf("[Wikimedia Query] SPARQL HTTP error %d for query to %s. Body: %s", resp.Status, endpoint, string(resp.Body))
		return nil, fmt.Errorf("SPARQL HTTP error: status %d", resp.Status)
	}
	var result SPARQLResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SPARQL response: %w", err)
	}
	if len(result.Results.Bindings) == 0 {
		return nil, ErrNotFound
	}
	return &result, nil
}

// --- MediaWiki API Helper ---
func mediawikiQuery(ctx context.Context, client http.HttpService, params url.Values) ([]byte, error) {
	apiURL := fmt.Sprintf("%s?%s", mediawikiAPIEndpoint, params.Encode())
	req := &http.HttpRequest{
		Url: apiURL,
		Headers: map[string]string{
			"Accept":     "application/json",
			"User-Agent": "NavidromeWikimediaPlugin/0.1",
		},
		TimeoutMs: requestTimeoutMs,
	}
	resp, err := client.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("MediaWiki request error: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("MediaWiki HTTP error: status %d, body: %s", resp.Status, string(resp.Body))
	}
	return resp.Body, nil
}

// --- Wikidata Fetch Functions ---
func getWikidataWikipediaURL(ctx context.Context, client http.HttpService, mbid, name string) (string, error) {
	var q string
	if mbid != "" {
		// Using property chain: ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>.
		q = fmt.Sprintf(`SELECT ?sitelink WHERE { ?artist wdt:P434 "%s". ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>. } LIMIT 1`, mbid)
	} else if name != "" {
		escapedName := strings.ReplaceAll(name, "\"", "\\\"")
		// Using property chain: ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>.
		q = fmt.Sprintf(`SELECT ?sitelink WHERE { ?artist rdfs:label "%s"@en. ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>. } LIMIT 1`, escapedName)
	} else {
		return "", errors.New("MBID or Name required for Wikidata URL lookup")
	}

	result, err := sparqlQuery(ctx, client, wikidataEndpoint, q)
	if err != nil {
		return "", fmt.Errorf("Wikidata SPARQL query failed: %w", err)
	}
	if result.Results.Bindings[0].Sitelink != nil {
		return result.Results.Bindings[0].Sitelink.Value, nil
	}
	return "", ErrNotFound
}

// --- DBpedia Fetch Functions ---
func getDBpediaWikipediaURL(ctx context.Context, client http.HttpService, name string) (string, error) {
	if name == "" {
		return "", ErrNotFound
	}
	escapedName := strings.ReplaceAll(name, "\"", "\\\"")
	q := fmt.Sprintf(`SELECT ?wiki WHERE { ?artist foaf:name "%s"@en; foaf:isPrimaryTopicOf ?wiki. FILTER regex(str(?wiki), "^https://en.wikipedia.org/") } LIMIT 1`, escapedName)
	result, err := sparqlQuery(ctx, client, dbpediaEndpoint, q)
	if err != nil {
		return "", fmt.Errorf("DBpedia SPARQL query failed: %w", err)
	}
	if result.Results.Bindings[0].Wiki != nil {
		return result.Results.Bindings[0].Wiki.Value, nil
	}
	return "", ErrNotFound
}

func getDBpediaComment(ctx context.Context, client http.HttpService, name string) (string, error) {
	if name == "" {
		return "", ErrNotFound
	}
	escapedName := strings.ReplaceAll(name, "\"", "\\\"")
	q := fmt.Sprintf(`SELECT ?comment WHERE { ?artist foaf:name "%s"@en; rdfs:comment ?comment. FILTER (lang(?comment) = 'en') } LIMIT 1`, escapedName)
	result, err := sparqlQuery(ctx, client, dbpediaEndpoint, q)
	if err != nil {
		return "", fmt.Errorf("DBpedia comment SPARQL query failed: %w", err)
	}
	if result.Results.Bindings[0].Comment != nil {
		return result.Results.Bindings[0].Comment.Value, nil
	}
	return "", ErrNotFound
}

// --- Wikipedia API Fetch Function ---
func getWikipediaExtract(ctx context.Context, client http.HttpService, pageTitle string) (string, error) {
	if pageTitle == "" {
		return "", errors.New("page title required for Wikipedia API lookup")
	}
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("prop", "extracts")
	params.Set("exintro", "true")     // Intro section only
	params.Set("explaintext", "true") // Plain text
	params.Set("titles", pageTitle)
	params.Set("redirects", "1") // Follow redirects

	body, err := mediawikiQuery(ctx, client, params)
	if err != nil {
		return "", fmt.Errorf("MediaWiki query failed: %w", err)
	}

	var result MediaWikiExtractResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse MediaWiki response: %w", err)
	}

	// Iterate through the pages map (usually only one page)
	for _, page := range result.Query.Pages {
		if page.Missing {
			continue // Skip missing pages
		}
		if page.Extract != "" {
			return strings.TrimSpace(page.Extract), nil
		}
	}

	return "", ErrNotFound
}

// --- Helper to get Wikipedia Page Title from URL ---
func extractPageTitleFromURL(wikiURL string) (string, error) {
	parsedURL, err := url.Parse(wikiURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Host != "en.wikipedia.org" {
		return "", fmt.Errorf("URL host is not en.wikipedia.org: %s", parsedURL.Host)
	}
	pathParts := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "wiki" {
		return "", fmt.Errorf("URL path does not match /wiki/<title> format: %s", parsedURL.Path)
	}
	title := pathParts[1]
	if title == "" {
		return "", errors.New("extracted title is empty")
	}
	decodedTitle, err := url.PathUnescape(title)
	if err != nil {
		return "", fmt.Errorf("failed to decode title '%s': %w", title, err)
	}
	return decodedTitle, nil
}

// --- Agent Implementation ---
type WikimediaAgent struct{}

// GetArtistURL fetches the Wikipedia URL.
// Order: Wikidata(MBID/Name) -> DBpedia(Name) -> Search URL
func (WikimediaAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	var wikiURL string
	var err error

	// 1. Try Wikidata (MBID first, then name)
	wikiURL, err = getWikidataWikipediaURL(ctx, client, req.Mbid, req.Name)
	if err == nil && wikiURL != "" {
		return &api.ArtistURLResponse{Url: wikiURL}, nil
	}
	if err != nil && err != ErrNotFound {
		log.Printf("[Wikimedia] Error fetching Wikidata URL: %v\n", err)
		// Don't stop, try DBpedia
	}

	// 2. Try DBpedia (Name only)
	if req.Name != "" {
		wikiURL, err = getDBpediaWikipediaURL(ctx, client, req.Name)
		if err == nil && wikiURL != "" {
			return &api.ArtistURLResponse{Url: wikiURL}, nil
		}
		if err != nil && err != ErrNotFound {
			log.Printf("[Wikimedia] Error fetching DBpedia URL: %v\n", err)
			// Don't stop, generate search URL
		}
	}

	// 3. Fallback to search URL
	if req.Name != "" {
		searchURL := fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s", url.QueryEscape(req.Name))
		log.Printf("[Wikimedia] URL not found, falling back to search URL: %s\n", searchURL)
		return &api.ArtistURLResponse{Url: searchURL}, nil
	}

	log.Printf("[Wikimedia] Could not determine Wikipedia URL for: %s (%s)\n", req.Name, req.Mbid)
	return nil, ErrNotFound
}

// GetArtistBiography fetches the long biography.
// Order: Wikipedia API (via Wikidata/DBpedia URL) -> DBpedia Comment (Name)
func (WikimediaAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	var bio string
	var err error

	log.Printf("[Wikimedia Bio] Fetching for Name: %s, MBID: %s", req.Name, req.Mbid)

	// 1. Get Wikipedia URL (using the logic from GetArtistURL)
	wikiURL := ""
	// Try Wikidata first
	tempURL, wdErr := getWikidataWikipediaURL(ctx, client, req.Mbid, req.Name)
	if wdErr == nil && tempURL != "" {
		log.Printf("[Wikimedia Bio] Found Wikidata URL: %s", tempURL)
		wikiURL = tempURL
	} else if req.Name != "" {
		// Try DBpedia if Wikidata failed or returned not found
		log.Printf("[Wikimedia Bio] Wikidata URL failed (%v), trying DBpedia URL", wdErr)
		tempURL, dbErr := getDBpediaWikipediaURL(ctx, client, req.Name)
		if dbErr == nil && tempURL != "" {
			log.Printf("[Wikimedia Bio] Found DBpedia URL: %s", tempURL)
			wikiURL = tempURL
		} else {
			log.Printf("[Wikimedia Bio] DBpedia URL failed (%v)", dbErr)
		}
	}

	// 2. If Wikipedia URL found, try MediaWiki API
	if wikiURL != "" {
		pageTitle, err := extractPageTitleFromURL(wikiURL)
		if err == nil {
			log.Printf("[Wikimedia Bio] Extracted page title: %s", pageTitle)
			bio, err = getWikipediaExtract(ctx, client, pageTitle)
			if err == nil && bio != "" {
				log.Printf("[Wikimedia Bio] Found Wikipedia extract.")
				return &api.ArtistBiographyResponse{Biography: bio}, nil
			}
			log.Printf("[Wikimedia Bio] Wikipedia extract failed: %v", err)
			if err != nil && err != ErrNotFound {
				log.Printf("[Wikimedia Bio] Error fetching Wikipedia extract for '%s': %v", pageTitle, err)
				// Don't stop, try DBpedia comment
			}
		} else {
			log.Printf("[Wikimedia Bio] Error extracting page title from URL '%s': %v", wikiURL, err)
			// Don't stop, try DBpedia comment
		}
	}

	// 3. Fallback to DBpedia Comment (Name only)
	if req.Name != "" {
		log.Printf("[Wikimedia Bio] Falling back to DBpedia comment for name: %s", req.Name)
		bio, err = getDBpediaComment(ctx, client, req.Name)
		if err == nil && bio != "" {
			log.Printf("[Wikimedia Bio] Found DBpedia comment.")
			return &api.ArtistBiographyResponse{Biography: bio}, nil
		}
		log.Printf("[Wikimedia Bio] DBpedia comment failed: %v", err)
		if err != nil && err != ErrNotFound {
			log.Printf("[Wikimedia Bio] Error fetching DBpedia comment for '%s': %v", req.Name, err)
		}
	}

	log.Printf("[Wikimedia Bio] Final: Biography not found for: %s (%s)", req.Name, req.Mbid)
	return nil, ErrNotFound
}

// GetArtistImages fetches images (Wikidata only for now)
func (WikimediaAgent) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	var q string
	if req.Mbid != "" {
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist wdt:P434 "%s"; wdt:P18 ?img } LIMIT 1`, req.Mbid)
	} else if req.Name != "" {
		escapedName := strings.ReplaceAll(req.Name, "\"", "\\\"")
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist rdfs:label "%s"@en; wdt:P18 ?img } LIMIT 1`, escapedName)
	} else {
		return nil, errors.New("MBID or Name required for Wikidata Image lookup")
	}

	result, err := sparqlQuery(ctx, client, wikidataEndpoint, q)
	if err != nil {
		log.Printf("[Wikimedia] Image not found for: %s (%s)\n", req.Name, req.Mbid)
		return nil, ErrNotFound
	}
	if result.Results.Bindings[0].Img != nil {
		return &api.ArtistImageResponse{Images: []*api.ExternalImage{{Url: result.Results.Bindings[0].Img.Value, Size: 0}}}, nil
	}
	log.Printf("[Wikimedia] Image not found for: %s (%s)\n", req.Name, req.Mbid)
	return nil, ErrNotFound
}

// Not implemented methods
func (WikimediaAgent) GetArtistMBID(context.Context, *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	return nil, ErrNotImplemented
}
func (WikimediaAgent) GetSimilarArtists(context.Context, *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return nil, ErrNotImplemented
}
func (WikimediaAgent) GetArtistTopSongs(context.Context, *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return nil, ErrNotImplemented
}
func (WikimediaAgent) GetAlbumInfo(context.Context, *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	return nil, ErrNotImplemented
}

func (WikimediaAgent) GetAlbumImages(context.Context, *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return nil, ErrNotImplemented
}

func main() {}

func init() {
	// Configure logging: No timestamps, no source file/line
	log.SetFlags(0)
	log.SetPrefix("[Wikimedia] ")

	api.RegisterMetadataAgent(WikimediaAgent{})
}
