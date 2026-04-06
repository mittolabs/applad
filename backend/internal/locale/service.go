// Package locale implements Applad's locale service:
// countries, currencies, languages, continents, and locale detection.
package locale

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Country represents a country.
type Country struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	ISO3      string `json:"iso3"`
	Continent string `json:"continent"`
	Currency  string `json:"currency"`
}

// Locale represents detected locale info for a request.
type Locale struct {
	IP        string `json:"ip"`
	Country   string `json:"country"`
	Continent string `json:"continent"`
}

// Handler handles locale HTTP requests.
type Handler struct{}

// NewHandler creates a new locale Handler.
func NewHandler() *Handler { return &Handler{} }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the locale router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.getLocale)
	r.Get("/countries", h.listCountries)
	r.Get("/continents", h.listContinents)
	r.Get("/currencies", h.listCurrencies)
	r.Get("/languages", h.listLanguages)
	return r
}

func (h *Handler) getLocale(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}
	// Basic locale detection — in production would use a GeoIP database
	writeJSON(w, http.StatusOK, Locale{
		IP:        ip,
		Country:   "Unknown",
		Continent: "Unknown",
	})
}

func (h *Handler) listCountries(w http.ResponseWriter, r *http.Request) {
	countries := []Country{
		{Name: "United States", Code: "US", ISO3: "USA", Continent: "North America", Currency: "USD"},
		{Name: "United Kingdom", Code: "GB", ISO3: "GBR", Continent: "Europe", Currency: "GBP"},
		{Name: "Germany", Code: "DE", ISO3: "DEU", Continent: "Europe", Currency: "EUR"},
		{Name: "France", Code: "FR", ISO3: "FRA", Continent: "Europe", Currency: "EUR"},
		{Name: "Japan", Code: "JP", ISO3: "JPN", Continent: "Asia", Currency: "JPY"},
		{Name: "Australia", Code: "AU", ISO3: "AUS", Continent: "Oceania", Currency: "AUD"},
		{Name: "Canada", Code: "CA", ISO3: "CAN", Continent: "North America", Currency: "CAD"},
		{Name: "Brazil", Code: "BR", ISO3: "BRA", Continent: "South America", Currency: "BRL"},
		{Name: "India", Code: "IN", ISO3: "IND", Continent: "Asia", Currency: "INR"},
		{Name: "China", Code: "CN", ISO3: "CHN", Continent: "Asia", Currency: "CNY"},
		{Name: "South Korea", Code: "KR", ISO3: "KOR", Continent: "Asia", Currency: "KRW"},
		{Name: "Mexico", Code: "MX", ISO3: "MEX", Continent: "North America", Currency: "MXN"},
		{Name: "Nigeria", Code: "NG", ISO3: "NGA", Continent: "Africa", Currency: "NGN"},
		{Name: "South Africa", Code: "ZA", ISO3: "ZAF", Continent: "Africa", Currency: "ZAR"},
		{Name: "Kenya", Code: "KE", ISO3: "KEN", Continent: "Africa", Currency: "KES"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(countries),
		"countries": countries,
	})
}

func (h *Handler) listContinents(w http.ResponseWriter, r *http.Request) {
	continents := []map[string]string{
		{"name": "Africa", "code": "AF"},
		{"name": "Antarctica", "code": "AN"},
		{"name": "Asia", "code": "AS"},
		{"name": "Europe", "code": "EU"},
		{"name": "North America", "code": "NA"},
		{"name": "Oceania", "code": "OC"},
		{"name": "South America", "code": "SA"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      len(continents),
		"continents": continents,
	})
}

func (h *Handler) listCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies := []map[string]string{
		{"code": "USD", "name": "United States Dollar", "symbol": "$"},
		{"code": "EUR", "name": "Euro", "symbol": "\u20ac"},
		{"code": "GBP", "name": "British Pound", "symbol": "\u00a3"},
		{"code": "JPY", "name": "Japanese Yen", "symbol": "\u00a5"},
		{"code": "AUD", "name": "Australian Dollar", "symbol": "A$"},
		{"code": "CAD", "name": "Canadian Dollar", "symbol": "C$"},
		{"code": "CHF", "name": "Swiss Franc", "symbol": "CHF"},
		{"code": "CNY", "name": "Chinese Yuan", "symbol": "\u00a5"},
		{"code": "INR", "name": "Indian Rupee", "symbol": "\u20b9"},
		{"code": "BRL", "name": "Brazilian Real", "symbol": "R$"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      len(currencies),
		"currencies": currencies,
	})
}

func (h *Handler) listLanguages(w http.ResponseWriter, r *http.Request) {
	languages := []map[string]string{
		{"code": "en", "name": "English", "nativeName": "English"},
		{"code": "es", "name": "Spanish", "nativeName": "Espa\u00f1ol"},
		{"code": "fr", "name": "French", "nativeName": "Fran\u00e7ais"},
		{"code": "de", "name": "German", "nativeName": "Deutsch"},
		{"code": "ja", "name": "Japanese", "nativeName": "\u65e5\u672c\u8a9e"},
		{"code": "zh", "name": "Chinese", "nativeName": "\u4e2d\u6587"},
		{"code": "pt", "name": "Portuguese", "nativeName": "Portugu\u00eas"},
		{"code": "ar", "name": "Arabic", "nativeName": "\u0627\u0644\u0639\u0631\u0628\u064a\u0629"},
		{"code": "hi", "name": "Hindi", "nativeName": "\u0939\u093f\u0928\u094d\u0926\u0940"},
		{"code": "ko", "name": "Korean", "nativeName": "\ud55c\uad6d\uc5b4"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(languages),
		"languages": languages,
	})
}
