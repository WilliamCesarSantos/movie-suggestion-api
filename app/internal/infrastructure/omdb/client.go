package omdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type SearchResult struct {
	ImdbID string `json:"imdbID"`
	Title  string `json:"Title"`
}

type SearchResponse struct {
	Search   []SearchResult `json:"Search"`
	Response string         `json:"Response"`
	Error    string         `json:"Error"`
}

type MovieResponse struct {
	ImdbID     string `json:"imdbID"`
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	Plot       string `json:"Plot"`
	Runtime    string `json:"Runtime"`
	Poster     string `json:"Poster"`
	ImdbRating string `json:"imdbRating"`
	Genre      string `json:"Genre"`
	Actors     string `json:"Actors"`
	Director   string `json:"Director"`
	Response   string `json:"Response"`
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, timeoutSeconds int) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
	}
}

func (c *Client) Search(ctx context.Context, term string, page int) ([]SearchResult, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	q := base.Query()
	q.Set("apikey", c.apiKey)
	q.Set("s", term)
	q.Set("page", strconv.Itoa(page))
	base.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Response != "True" {
		if strings.EqualFold(result.Error, "Movie not found!") {
			return []SearchResult{}, nil
		}
		return nil, fmt.Errorf("OMDB search failed for term %q", term)
	}
	return result.Search, nil
}

func (c *Client) FetchByImdbID(ctx context.Context, imdbID string) (*entity.Movie, error) {
	url := fmt.Sprintf("%s/?apikey=%s&i=%s&plot=full", c.baseURL, c.apiKey, imdbID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result MovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Response != "True" {
		return nil, fmt.Errorf("OMDB movie not found: %s", imdbID)
	}

	rating, _ := strconv.ParseFloat(result.ImdbRating, 64)

	genres := parseCSV(result.Genre)
	entityGenres := make([]entity.Genre, len(genres))
	for i, g := range genres {
		entityGenres[i] = entity.Genre{Name: g}
	}

	actors := parseCSV(result.Actors)
	entityActors := make([]entity.Actor, len(actors))
	for i, a := range actors {
		entityActors[i] = entity.Actor{Name: a}
	}

	directors := parseCSV(result.Director)
	entityDirectors := make([]entity.Director, len(directors))
	for i, d := range directors {
		entityDirectors[i] = entity.Director{Name: d}
	}

	return &entity.Movie{
		ImdbID:     result.ImdbID,
		Title:      result.Title,
		Year:       result.Year,
		Plot:       result.Plot,
		Runtime:    result.Runtime,
		Poster:     result.Poster,
		ImdbRating: rating,
		Genres:     entityGenres,
		Actors:     entityActors,
		Directors:  entityDirectors,
	}, nil
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "N/A" {
			result = append(result, p)
		}
	}
	return result
}
