package locale

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestRouter() http.Handler {
	r := chi.NewRouter()
	h := NewHandler()
	r.Mount("/", Routes(h))
	return r
}

// countriesResponse mirrors the JSON shape returned by listCountries.
type countriesResponse struct {
	Total     int       `json:"total"`
	Countries []Country `json:"countries"`
}

type continentsResponse struct {
	Total      int              `json:"total"`
	Continents []map[string]string `json:"continents"`
}

type currenciesResponse struct {
	Total      int        `json:"total"`
	Currencies []Currency `json:"currencies"`
}

type languagesResponse struct {
	Total     int        `json:"total"`
	Languages []Language `json:"languages"`
}

func getJSON(t *testing.T, srv http.Handler, path string, dest interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d", path, rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(dest); err != nil {
		t.Fatalf("GET %s: failed to decode JSON: %v", path, err)
	}
}

func TestCountries_Count(t *testing.T) {
	srv := newTestRouter()
	var resp countriesResponse
	getJSON(t, srv, "/countries", &resp)
	if resp.Total < 190 {
		t.Fatalf("expected >= 190 countries, got %d", resp.Total)
	}
}

func TestCountries_HasUSA(t *testing.T) {
	srv := newTestRouter()
	var resp countriesResponse
	getJSON(t, srv, "/countries", &resp)

	found := false
	for _, c := range resp.Countries {
		if c.Code == "US" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find country with code US")
	}
}

func TestCountries_AllHaveCodes(t *testing.T) {
	srv := newTestRouter()
	var resp countriesResponse
	getJSON(t, srv, "/countries", &resp)

	for i, c := range resp.Countries {
		if c.Name == "" {
			t.Fatalf("country at index %d has empty Name", i)
		}
		if c.Code == "" {
			t.Fatalf("country at index %d (%s) has empty Code", i, c.Name)
		}
	}
}

func TestContinents_Count(t *testing.T) {
	srv := newTestRouter()
	var resp continentsResponse
	getJSON(t, srv, "/continents", &resp)
	if resp.Total != 7 {
		t.Fatalf("expected 7 continents, got %d", resp.Total)
	}
}

func TestCurrencies_Count(t *testing.T) {
	srv := newTestRouter()
	var resp currenciesResponse
	getJSON(t, srv, "/currencies", &resp)
	if resp.Total < 40 {
		t.Fatalf("expected >= 40 currencies, got %d", resp.Total)
	}
}

func TestCurrencies_HasUSD(t *testing.T) {
	srv := newTestRouter()
	var resp currenciesResponse
	getJSON(t, srv, "/currencies", &resp)

	found := false
	for _, c := range resp.Currencies {
		if c.Code == "USD" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find currency with code USD")
	}
}

func TestLanguages_Count(t *testing.T) {
	srv := newTestRouter()
	var resp languagesResponse
	getJSON(t, srv, "/languages", &resp)
	if resp.Total < 40 {
		t.Fatalf("expected >= 40 languages, got %d", resp.Total)
	}
}

func TestLanguages_HasEnglish(t *testing.T) {
	srv := newTestRouter()
	var resp languagesResponse
	getJSON(t, srv, "/languages", &resp)

	found := false
	for _, l := range resp.Languages {
		if l.Code == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find language with code en")
	}
}
