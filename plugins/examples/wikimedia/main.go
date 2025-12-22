// Wikimedia plugin for Navidrome - fetches artist metadata from Wikidata, DBpedia and Wikipedia.
//
// Build with:
//
//	tinygo build -o wikimedia.wasm -target wasip1 -buildmode=c-shared ./main.go
//
// Install by copying wikimedia.wasm to your Navidrome plugins folder.
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/extism/go-pdk"
)

const (
	wikidataEndpoint     = "https://query.wikidata.org/sparql"
	dbpediaEndpoint      = "https://dbpedia.org/sparql"
	mediawikiAPIEndpoint = "https://en.wikipedia.org/w/api.php"
)

// Manifest types
type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Website     string       `json:"website,omitempty"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	HTTP *HTTPPermission `json:"http,omitempty"`
}

type HTTPPermission struct {
	Reason       string   `json:"reason,omitempty"`
	AllowedHosts []string `json:"allowedHosts,omitempty"`
}

// Input types
type ArtistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

// Output types
type URLOutput struct {
	URL string `json:"url"`
}

type BiographyOutput struct {
	Biography string `json:"biography"`
}

type ImageInfo struct {
	URL  string `json:"url"`
	Size int    `json:"size"`
}

type ImagesOutput struct {
	Images []ImageInfo `json:"images"`
}

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

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Wikimedia",
		Author:      "Navidrome",
		Version:     "1.0.0",
		Description: "Fetches artist metadata from Wikidata, DBpedia and Wikipedia",
		Website:     "https://navidrome.org",
		Permissions: &Permissions{
			HTTP: &HTTPPermission{
				Reason: "Fetch metadata from Wikimedia APIs",
				AllowedHosts: []string{
					"query.wikidata.org",
					"dbpedia.org",
					"en.wikipedia.org",
				},
			},
		},
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
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
		return nil, fmt.Errorf("not found")
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
		return "", fmt.Errorf("MBID or Name required for Wikidata URL lookup")
	}

	result, err := sparqlQuery(wikidataEndpoint, q)
	if err != nil {
		return "", err
	}
	if result.Results.Bindings[0].Sitelink != nil {
		return result.Results.Bindings[0].Sitelink.Value, nil
	}
	return "", fmt.Errorf("not found")
}

// getDBpediaWikipediaURL fetches the Wikipedia URL from DBpedia using name
func getDBpediaWikipediaURL(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("not found")
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
	return "", fmt.Errorf("not found")
}

// getDBpediaComment fetches the DBpedia comment (short bio) for an artist
func getDBpediaComment(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("not found")
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
	return "", fmt.Errorf("not found")
}

// getWikipediaExtract fetches the intro text from Wikipedia
func getWikipediaExtract(pageTitle string) (string, error) {
	if pageTitle == "" {
		return "", fmt.Errorf("page title required")
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
	return "", fmt.Errorf("not found")
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
		return "", fmt.Errorf("extracted title is empty")
	}
	decodedTitle, err := url.PathUnescape(title)
	if err != nil {
		return "", fmt.Errorf("failed to decode title '%s': %w", title, err)
	}
	return decodedTitle, nil
}

//go:wasmexport nd_get_artist_url
func ndGetArtistURL() int32 {
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	pdk.Log(pdk.LogDebug, fmt.Sprintf("GetArtistURL: name=%s, mbid=%s", input.Name, input.MBID))

	// 1. Try Wikidata (MBID first, then name)
	wikiURL, err := getWikidataWikipediaURL(input.MBID, input.Name)
	if err == nil && wikiURL != "" {
		output := URLOutput{URL: wikiURL}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}
	if err != nil {
		pdk.Log(pdk.LogDebug, fmt.Sprintf("Wikidata URL failed: %v", err))
	}

	// 2. Try DBpedia (Name only)
	if input.Name != "" {
		wikiURL, err = getDBpediaWikipediaURL(input.Name)
		if err == nil && wikiURL != "" {
			output := URLOutput{URL: wikiURL}
			if err := pdk.OutputJSON(output); err != nil {
				pdk.SetError(err)
				return 1
			}
			return 0
		}
		if err != nil {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("DBpedia URL failed: %v", err))
		}
	}

	// 3. Fallback to search URL
	if input.Name != "" {
		searchURL := fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s", url.QueryEscape(input.Name))
		pdk.Log(pdk.LogInfo, fmt.Sprintf("URL not found, falling back to search URL: %s", searchURL))
		output := URLOutput{URL: searchURL}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}

	pdk.SetErrorString("could not determine Wikipedia URL")
	return 1
}

//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

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
				output := BiographyOutput{Biography: bio}
				if err := pdk.OutputJSON(output); err != nil {
					pdk.SetError(err)
					return 1
				}
				return 0
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
			output := BiographyOutput{Biography: bio}
			if err := pdk.OutputJSON(output); err != nil {
				pdk.SetError(err)
				return 1
			}
			return 0
		}
		pdk.Log(pdk.LogDebug, fmt.Sprintf("DBpedia comment failed: %v", err))
	}

	pdk.Log(pdk.LogInfo, fmt.Sprintf("Biography not found for: %s (%s)", input.Name, input.MBID))
	pdk.SetErrorString("biography not found")
	return 1
}

//go:wasmexport nd_get_artist_images
func ndGetArtistImages() int32 {
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	pdk.Log(pdk.LogDebug, fmt.Sprintf("GetArtistImages: name=%s, mbid=%s", input.Name, input.MBID))

	var q string
	if input.MBID != "" {
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist wdt:P434 "%s"; wdt:P18 ?img } LIMIT 1`, input.MBID)
	} else if input.Name != "" {
		escapedName := strings.ReplaceAll(input.Name, "\"", "\\\"")
		q = fmt.Sprintf(`SELECT ?img WHERE { ?artist rdfs:label "%s"@en; wdt:P18 ?img } LIMIT 1`, escapedName)
	} else {
		pdk.SetErrorString("MBID or Name required for Wikidata Image lookup")
		return 1
	}

	result, err := sparqlQuery(wikidataEndpoint, q)
	if err != nil {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("Image not found for: %s (%s)", input.Name, input.MBID))
		pdk.SetErrorString("image not found")
		return 1
	}
	if result.Results.Bindings[0].Img != nil {
		output := ImagesOutput{
			Images: []ImageInfo{{URL: result.Results.Bindings[0].Img.Value, Size: 0}},
		}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}

	pdk.Log(pdk.LogInfo, fmt.Sprintf("Image not found for: %s (%s)", input.Name, input.MBID))
	pdk.SetErrorString("image not found")
	return 1
}

func main() {}
