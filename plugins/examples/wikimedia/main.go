// Wikimedia plugin for Navidrome - fetches artist metadata from Wikidata, DBpedia and Wikipedia.
//
// Build with:
//
//	tinygo build -o wikimedia.wasm -target wasip1 -buildmode=c-shared .
//
// Install by copying the .ndp file to your Navidrome plugins folder.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// wikimediaPlugin implements the metadata provider interfaces for the methods we support.
type wikimediaPlugin struct{}

// init registers the plugin implementation
func init() {
	metadata.Register(&wikimediaPlugin{})
}

// Ensure wikimediaPlugin implements the provider interfaces
var (
	_ metadata.ArtistURLProvider       = (*wikimediaPlugin)(nil)
	_ metadata.ArtistBiographyProvider = (*wikimediaPlugin)(nil)
	_ metadata.ArtistImagesProvider    = (*wikimediaPlugin)(nil)
)

const (
	wikidataEndpoint     = "https://query.wikidata.org/sparql"
	dbpediaEndpoint      = "https://dbpedia.org/sparql"
	mediawikiAPIEndpoint = "https://en.wikipedia.org/w/api.php"
)

// SPARQL response types
type SPARQLResult struct {
	Results struct {
		Bindings []SPARQLBinding `json:"bindings"`
	} `json:"results"`
}

type SPARQLBinding struct {
	Sitelink *SPARQLValue `json:"sitelink,omitempty"`
	Wiki     *SPARQLValue `json:"wiki,omitempty"`
	Comment  *SPARQLValue `json:"comment,omitempty"`
	Img      *SPARQLValue `json:"img,omitempty"`
}

type SPARQLValue struct {
	Value string `json:"value"`
}

// MediaWiki API response types
type MediaWikiExtractResult struct {
	Query struct {
		Pages map[string]MediaWikiPage `json:"pages"`
	} `json:"query"`
}

type MediaWikiPage struct {
	PageID  int    `json:"pageid"`
	Ns      int    `json:"ns"`
	Title   string `json:"title"`
	Extract string `json:"extract"`
	Missing bool   `json:"missing"`
}

// sparqlQuery executes a SPARQL query and returns the result
func sparqlQuery(endpoint, query string) (*SPARQLResult, error) {
	form := url.Values{}
	form.Set("query", query)

	req := pdk.NewHTTPRequest(pdk.MethodPost, endpoint)
	req.SetHeader("Accept", "application/sparql-results+json")
	req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	req.SetHeader("User-Agent", "NavidromeWikimediaPlugin/1.0")
	req.SetBody([]byte(form.Encode()))

	pdk.Log(pdk.LogDebug, fmt.Sprintf("SPARQL query to %s: %s", endpoint, query))

	resp := req.Send()
	if resp.Status() != 200 {
		return nil, fmt.Errorf("SPARQL HTTP error: status %d", resp.Status())
	}

	var result SPARQLResult
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse SPARQL response: %w", err)
	}
	if len(result.Results.Bindings) == 0 {
		return nil, errors.New("not found")
	}
	return &result, nil
}

// mediawikiQuery executes a MediaWiki API query
func mediawikiQuery(params url.Values) ([]byte, error) {
	apiURL := fmt.Sprintf("%s?%s", mediawikiAPIEndpoint, params.Encode())

	req := pdk.NewHTTPRequest(pdk.MethodGet, apiURL)
	req.SetHeader("Accept", "application/json")
	req.SetHeader("User-Agent", "NavidromeWikimediaPlugin/1.0")

	resp := req.Send()
	if resp.Status() != 200 {
		return nil, fmt.Errorf("MediaWiki HTTP error: status %d", resp.Status())
	}
	return resp.Body(), nil
}

// getWikidataWikipediaURL fetches the Wikipedia URL from Wikidata using MBID or name
func getWikidataWikipediaURL(mbid, name string) (string, error) {
	var q string
	if mbid != "" {
		q = fmt.Sprintf(`SELECT ?sitelink WHERE { ?artist wdt:P434 "%s". ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>. } LIMIT 1`, mbid)
	} else if name != "" {
		escapedName := strings.ReplaceAll(name, "\"", "\\\"")
		q = fmt.Sprintf(`SELECT ?sitelink WHERE { ?artist rdfs:label "%s"@en. ?sitelink schema:about ?artist; schema:isPartOf <https://en.wikipedia.org/>. } LIMIT 1`, escapedName)
	} else {
		return "", errors.New("MBID or Name required for Wikidata URL lookup")
	}

	result, err := sparqlQuery(wikidataEndpoint, q)
	if err != nil {
		return "", err
	}
	if result.Results.Bindings[0].Sitelink != nil {
		return result.Results.Bindings[0].Sitelink.Value, nil
	}
	return "", errors.New("not found")
}

// getDBpediaWikipediaURL fetches the Wikipedia URL from DBpedia using name
func getDBpediaWikipediaURL(name string) (string, error) {
	if name == "" {
		return "", errors.New("not found")
	}
	escapedName := strings.ReplaceAll(name, "\"", "\\\"")
	q := fmt.Sprintf(`SELECT ?wiki WHERE { ?artist foaf:name "%s"@en; foaf:isPrimaryTopicOf ?wiki. FILTER regex(str(?wiki), "^https://en.wikipedia.org/") } LIMIT 1`, escapedName)

	result, err := sparqlQuery(dbpediaEndpoint, q)
	if err != nil {
		return "", err
	}
	if result.Results.Bindings[0].Wiki != nil {
		return result.Results.Bindings[0].Wiki.Value, nil
	}
	return "", errors.New("not found")
}

// getDBpediaComment fetches the DBpedia comment (short bio) for an artist
func getDBpediaComment(name string) (string, error) {
	if name == "" {
		return "", errors.New("not found")
	}
	escapedName := strings.ReplaceAll(name, "\"", "\\\"")
	q := fmt.Sprintf(`SELECT ?comment WHERE { ?artist foaf:name "%s"@en; rdfs:comment ?comment. FILTER (lang(?comment) = 'en') } LIMIT 1`, escapedName)

	result, err := sparqlQuery(dbpediaEndpoint, q)
	if err != nil {
		return "", err
	}
	if result.Results.Bindings[0].Comment != nil {
		return result.Results.Bindings[0].Comment.Value, nil
	}
	return "", errors.New("not found")
}

// getWikipediaExtract fetches the intro text from Wikipedia
func getWikipediaExtract(pageTitle string) (string, error) {
	if pageTitle == "" {
		return "", errors.New("page title required")
	}
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("prop", "extracts")
	params.Set("exintro", "true")
	params.Set("explaintext", "true")
	params.Set("titles", pageTitle)
	params.Set("redirects", "1")

	body, err := mediawikiQuery(params)
	if err != nil {
		return "", err
	}

	var result MediaWikiExtractResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse MediaWiki response: %w", err)
	}

	for _, page := range result.Query.Pages {
		if page.Missing {
			continue
		}
		if page.Extract != "" {
			return strings.TrimSpace(page.Extract), nil
		}
	}
	return "", errors.New("not found")
}

// extractPageTitleFromURL extracts the page title from a Wikipedia URL
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

// GetArtistURL returns the Wikipedia URL for an artist
func (*wikimediaPlugin) GetArtistURL(input metadata.ArtistRequest) (*metadata.ArtistURLResponse, error) {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("GetArtistURL: name=%s, mbid=%s", input.Name, input.MBID))

	// 1. Try Wikidata (MBID first, then name)
	wikiURL, err := getWikidataWikipediaURL(input.MBID, input.Name)
	if err == nil && wikiURL != "" {
		return &metadata.ArtistURLResponse{URL: wikiURL}, nil
	}
	if err != nil {
		pdk.Log(pdk.LogDebug, fmt.Sprintf("Wikidata URL failed: %v", err))
	}

	// 2. Try DBpedia (Name only)
	if input.Name != "" {
		wikiURL, err = getDBpediaWikipediaURL(input.Name)
		if err == nil && wikiURL != "" {
			return &metadata.ArtistURLResponse{URL: wikiURL}, nil
		}
		if err != nil {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("DBpedia URL failed: %v", err))
		}
	}

	// 3. Fallback to search URL
	if input.Name != "" {
		searchURL := fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s", url.QueryEscape(input.Name))
		pdk.Log(pdk.LogInfo, fmt.Sprintf("URL not found, falling back to search URL: %s", searchURL))
		return &metadata.ArtistURLResponse{URL: searchURL}, nil
	}

	return nil, errors.New("could not determine Wikipedia URL")
}

// GetArtistBiography returns the biography for an artist from Wikipedia
func (*wikimediaPlugin) GetArtistBiography(input metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("GetArtistBiography: name=%s, mbid=%s", input.Name, input.MBID))

	// 1. Get Wikipedia URL (using the logic from GetArtistURL)
	wikiURL := ""
	tempURL, wdErr := getWikidataWikipediaURL(input.MBID, input.Name)
	if wdErr == nil && tempURL != "" {
		pdk.Log(pdk.LogDebug, fmt.Sprintf("Found Wikidata URL: %s", tempURL))
		wikiURL = tempURL
	} else if input.Name != "" {
		pdk.Log(pdk.LogDebug, fmt.Sprintf("Wikidata URL failed (%v), trying DBpedia", wdErr))
		tempURL, dbErr := getDBpediaWikipediaURL(input.Name)
		if dbErr == nil && tempURL != "" {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Found DBpedia URL: %s", tempURL))
			wikiURL = tempURL
		} else {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("DBpedia URL failed: %v", dbErr))
		}
	}

	// 2. If Wikipedia URL found, try MediaWiki API
	if wikiURL != "" {
		pageTitle, err := extractPageTitleFromURL(wikiURL)
		if err == nil {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Extracted page title: %s", pageTitle))
			bio, err := getWikipediaExtract(pageTitle)
			if err == nil && bio != "" {
				pdk.Log(pdk.LogDebug, "Found Wikipedia extract")
				return &metadata.ArtistBiographyResponse{Biography: bio}, nil
			}
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Wikipedia extract failed: %v", err))
		} else {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Error extracting page title from URL '%s': %v", wikiURL, err))
		}
	}

	// 3. Fallback to DBpedia Comment (Name only)
	if input.Name != "" {
		pdk.Log(pdk.LogDebug, fmt.Sprintf("Falling back to DBpedia comment for name: %s", input.Name))
		bio, err := getDBpediaComment(input.Name)
		if err == nil && bio != "" {
			pdk.Log(pdk.LogDebug, "Found DBpedia comment")
			return &metadata.ArtistBiographyResponse{Biography: bio}, nil
		}
		pdk.Log(pdk.LogDebug, fmt.Sprintf("DBpedia comment failed: %v", err))
	}

	pdk.Log(pdk.LogInfo, fmt.Sprintf("Biography not found for: %s (%s)", input.Name, input.MBID))
	return nil, errors.New("biography not found")
}

// GetArtistImages returns artist images from Wikidata
func (*wikimediaPlugin) GetArtistImages(input metadata.ArtistRequest) (*metadata.ArtistImagesResponse, error) {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("GetArtistImages: name=%s, mbid=%s", input.Name, input.MBID))

	var q string
	if input.MBID != "" {
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist wdt:P434 "%s"; wdt:P18 ?img } LIMIT 1`, input.MBID)
	} else if input.Name != "" {
		escapedName := strings.ReplaceAll(input.Name, "\"", "\\\"")
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist rdfs:label "%s"@en; wdt:P18 ?img } LIMIT 1`, escapedName)
	} else {
		return nil, errors.New("MBID or Name required for Wikidata Image lookup")
	}

	result, err := sparqlQuery(wikidataEndpoint, q)
	if err != nil {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("Image not found for: %s (%s)", input.Name, input.MBID))
		return nil, errors.New("image not found")
	}
	if result.Results.Bindings[0].Img != nil {
		return &metadata.ArtistImagesResponse{
			Images: []metadata.ImageInfo{{URL: result.Results.Bindings[0].Img.Value, Size: 0}},
		}, nil
	}

	pdk.Log(pdk.LogInfo, fmt.Sprintf("Image not found for: %s (%s)", input.Name, input.MBID))
	return nil, errors.New("image not found")
}

// Required main function - init() handles registration
func main() {}
