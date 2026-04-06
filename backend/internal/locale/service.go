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
	PhoneCode string `json:"phoneCode"`
}

// Currency represents a currency.
type Currency struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	Symbol        string `json:"symbol"`
	SymbolNative  string `json:"symbolNative"`
	DecimalDigits int    `json:"decimalDigits"`
}

// Language represents a language.
type Language struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	NativeName string `json:"nativeName"`
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
		{Name: "Afghanistan", Code: "AF", ISO3: "AFG", Continent: "Asia", Currency: "AFN", PhoneCode: "+93"},
		{Name: "Albania", Code: "AL", ISO3: "ALB", Continent: "Europe", Currency: "ALL", PhoneCode: "+355"},
		{Name: "Algeria", Code: "DZ", ISO3: "DZA", Continent: "Africa", Currency: "DZD", PhoneCode: "+213"},
		{Name: "Andorra", Code: "AD", ISO3: "AND", Continent: "Europe", Currency: "EUR", PhoneCode: "+376"},
		{Name: "Angola", Code: "AO", ISO3: "AGO", Continent: "Africa", Currency: "AOA", PhoneCode: "+244"},
		{Name: "Antigua and Barbuda", Code: "AG", ISO3: "ATG", Continent: "North America", Currency: "XCD", PhoneCode: "+1-268"},
		{Name: "Argentina", Code: "AR", ISO3: "ARG", Continent: "South America", Currency: "ARS", PhoneCode: "+54"},
		{Name: "Armenia", Code: "AM", ISO3: "ARM", Continent: "Asia", Currency: "AMD", PhoneCode: "+374"},
		{Name: "Australia", Code: "AU", ISO3: "AUS", Continent: "Oceania", Currency: "AUD", PhoneCode: "+61"},
		{Name: "Austria", Code: "AT", ISO3: "AUT", Continent: "Europe", Currency: "EUR", PhoneCode: "+43"},
		{Name: "Azerbaijan", Code: "AZ", ISO3: "AZE", Continent: "Asia", Currency: "AZN", PhoneCode: "+994"},
		{Name: "Bahamas", Code: "BS", ISO3: "BHS", Continent: "North America", Currency: "BSD", PhoneCode: "+1-242"},
		{Name: "Bahrain", Code: "BH", ISO3: "BHR", Continent: "Asia", Currency: "BHD", PhoneCode: "+973"},
		{Name: "Bangladesh", Code: "BD", ISO3: "BGD", Continent: "Asia", Currency: "BDT", PhoneCode: "+880"},
		{Name: "Barbados", Code: "BB", ISO3: "BRB", Continent: "North America", Currency: "BBD", PhoneCode: "+1-246"},
		{Name: "Belarus", Code: "BY", ISO3: "BLR", Continent: "Europe", Currency: "BYN", PhoneCode: "+375"},
		{Name: "Belgium", Code: "BE", ISO3: "BEL", Continent: "Europe", Currency: "EUR", PhoneCode: "+32"},
		{Name: "Belize", Code: "BZ", ISO3: "BLZ", Continent: "North America", Currency: "BZD", PhoneCode: "+501"},
		{Name: "Benin", Code: "BJ", ISO3: "BEN", Continent: "Africa", Currency: "XOF", PhoneCode: "+229"},
		{Name: "Bhutan", Code: "BT", ISO3: "BTN", Continent: "Asia", Currency: "BTN", PhoneCode: "+975"},
		{Name: "Bolivia", Code: "BO", ISO3: "BOL", Continent: "South America", Currency: "BOB", PhoneCode: "+591"},
		{Name: "Bosnia and Herzegovina", Code: "BA", ISO3: "BIH", Continent: "Europe", Currency: "BAM", PhoneCode: "+387"},
		{Name: "Botswana", Code: "BW", ISO3: "BWA", Continent: "Africa", Currency: "BWP", PhoneCode: "+267"},
		{Name: "Brazil", Code: "BR", ISO3: "BRA", Continent: "South America", Currency: "BRL", PhoneCode: "+55"},
		{Name: "Brunei", Code: "BN", ISO3: "BRN", Continent: "Asia", Currency: "BND", PhoneCode: "+673"},
		{Name: "Bulgaria", Code: "BG", ISO3: "BGR", Continent: "Europe", Currency: "BGN", PhoneCode: "+359"},
		{Name: "Burkina Faso", Code: "BF", ISO3: "BFA", Continent: "Africa", Currency: "XOF", PhoneCode: "+226"},
		{Name: "Burundi", Code: "BI", ISO3: "BDI", Continent: "Africa", Currency: "BIF", PhoneCode: "+257"},
		{Name: "Cabo Verde", Code: "CV", ISO3: "CPV", Continent: "Africa", Currency: "CVE", PhoneCode: "+238"},
		{Name: "Cambodia", Code: "KH", ISO3: "KHM", Continent: "Asia", Currency: "KHR", PhoneCode: "+855"},
		{Name: "Cameroon", Code: "CM", ISO3: "CMR", Continent: "Africa", Currency: "XAF", PhoneCode: "+237"},
		{Name: "Canada", Code: "CA", ISO3: "CAN", Continent: "North America", Currency: "CAD", PhoneCode: "+1"},
		{Name: "Central African Republic", Code: "CF", ISO3: "CAF", Continent: "Africa", Currency: "XAF", PhoneCode: "+236"},
		{Name: "Chad", Code: "TD", ISO3: "TCD", Continent: "Africa", Currency: "XAF", PhoneCode: "+235"},
		{Name: "Chile", Code: "CL", ISO3: "CHL", Continent: "South America", Currency: "CLP", PhoneCode: "+56"},
		{Name: "China", Code: "CN", ISO3: "CHN", Continent: "Asia", Currency: "CNY", PhoneCode: "+86"},
		{Name: "Colombia", Code: "CO", ISO3: "COL", Continent: "South America", Currency: "COP", PhoneCode: "+57"},
		{Name: "Comoros", Code: "KM", ISO3: "COM", Continent: "Africa", Currency: "KMF", PhoneCode: "+269"},
		{Name: "Congo", Code: "CG", ISO3: "COG", Continent: "Africa", Currency: "XAF", PhoneCode: "+242"},
		{Name: "Congo (Democratic Republic)", Code: "CD", ISO3: "COD", Continent: "Africa", Currency: "CDF", PhoneCode: "+243"},
		{Name: "Costa Rica", Code: "CR", ISO3: "CRI", Continent: "North America", Currency: "CRC", PhoneCode: "+506"},
		{Name: "Croatia", Code: "HR", ISO3: "HRV", Continent: "Europe", Currency: "EUR", PhoneCode: "+385"},
		{Name: "Cuba", Code: "CU", ISO3: "CUB", Continent: "North America", Currency: "CUP", PhoneCode: "+53"},
		{Name: "Cyprus", Code: "CY", ISO3: "CYP", Continent: "Europe", Currency: "EUR", PhoneCode: "+357"},
		{Name: "Czech Republic", Code: "CZ", ISO3: "CZE", Continent: "Europe", Currency: "CZK", PhoneCode: "+420"},
		{Name: "Denmark", Code: "DK", ISO3: "DNK", Continent: "Europe", Currency: "DKK", PhoneCode: "+45"},
		{Name: "Djibouti", Code: "DJ", ISO3: "DJI", Continent: "Africa", Currency: "DJF", PhoneCode: "+253"},
		{Name: "Dominica", Code: "DM", ISO3: "DMA", Continent: "North America", Currency: "XCD", PhoneCode: "+1-767"},
		{Name: "Dominican Republic", Code: "DO", ISO3: "DOM", Continent: "North America", Currency: "DOP", PhoneCode: "+1-809"},
		{Name: "Ecuador", Code: "EC", ISO3: "ECU", Continent: "South America", Currency: "USD", PhoneCode: "+593"},
		{Name: "Egypt", Code: "EG", ISO3: "EGY", Continent: "Africa", Currency: "EGP", PhoneCode: "+20"},
		{Name: "El Salvador", Code: "SV", ISO3: "SLV", Continent: "North America", Currency: "USD", PhoneCode: "+503"},
		{Name: "Equatorial Guinea", Code: "GQ", ISO3: "GNQ", Continent: "Africa", Currency: "XAF", PhoneCode: "+240"},
		{Name: "Eritrea", Code: "ER", ISO3: "ERI", Continent: "Africa", Currency: "ERN", PhoneCode: "+291"},
		{Name: "Estonia", Code: "EE", ISO3: "EST", Continent: "Europe", Currency: "EUR", PhoneCode: "+372"},
		{Name: "Eswatini", Code: "SZ", ISO3: "SWZ", Continent: "Africa", Currency: "SZL", PhoneCode: "+268"},
		{Name: "Ethiopia", Code: "ET", ISO3: "ETH", Continent: "Africa", Currency: "ETB", PhoneCode: "+251"},
		{Name: "Fiji", Code: "FJ", ISO3: "FJI", Continent: "Oceania", Currency: "FJD", PhoneCode: "+679"},
		{Name: "Finland", Code: "FI", ISO3: "FIN", Continent: "Europe", Currency: "EUR", PhoneCode: "+358"},
		{Name: "France", Code: "FR", ISO3: "FRA", Continent: "Europe", Currency: "EUR", PhoneCode: "+33"},
		{Name: "Gabon", Code: "GA", ISO3: "GAB", Continent: "Africa", Currency: "XAF", PhoneCode: "+241"},
		{Name: "Gambia", Code: "GM", ISO3: "GMB", Continent: "Africa", Currency: "GMD", PhoneCode: "+220"},
		{Name: "Georgia", Code: "GE", ISO3: "GEO", Continent: "Asia", Currency: "GEL", PhoneCode: "+995"},
		{Name: "Germany", Code: "DE", ISO3: "DEU", Continent: "Europe", Currency: "EUR", PhoneCode: "+49"},
		{Name: "Ghana", Code: "GH", ISO3: "GHA", Continent: "Africa", Currency: "GHS", PhoneCode: "+233"},
		{Name: "Greece", Code: "GR", ISO3: "GRC", Continent: "Europe", Currency: "EUR", PhoneCode: "+30"},
		{Name: "Grenada", Code: "GD", ISO3: "GRD", Continent: "North America", Currency: "XCD", PhoneCode: "+1-473"},
		{Name: "Guatemala", Code: "GT", ISO3: "GTM", Continent: "North America", Currency: "GTQ", PhoneCode: "+502"},
		{Name: "Guinea", Code: "GN", ISO3: "GIN", Continent: "Africa", Currency: "GNF", PhoneCode: "+224"},
		{Name: "Guinea-Bissau", Code: "GW", ISO3: "GNB", Continent: "Africa", Currency: "XOF", PhoneCode: "+245"},
		{Name: "Guyana", Code: "GY", ISO3: "GUY", Continent: "South America", Currency: "GYD", PhoneCode: "+592"},
		{Name: "Haiti", Code: "HT", ISO3: "HTI", Continent: "North America", Currency: "HTG", PhoneCode: "+509"},
		{Name: "Honduras", Code: "HN", ISO3: "HND", Continent: "North America", Currency: "HNL", PhoneCode: "+504"},
		{Name: "Hungary", Code: "HU", ISO3: "HUN", Continent: "Europe", Currency: "HUF", PhoneCode: "+36"},
		{Name: "Iceland", Code: "IS", ISO3: "ISL", Continent: "Europe", Currency: "ISK", PhoneCode: "+354"},
		{Name: "India", Code: "IN", ISO3: "IND", Continent: "Asia", Currency: "INR", PhoneCode: "+91"},
		{Name: "Indonesia", Code: "ID", ISO3: "IDN", Continent: "Asia", Currency: "IDR", PhoneCode: "+62"},
		{Name: "Iran", Code: "IR", ISO3: "IRN", Continent: "Asia", Currency: "IRR", PhoneCode: "+98"},
		{Name: "Iraq", Code: "IQ", ISO3: "IRQ", Continent: "Asia", Currency: "IQD", PhoneCode: "+964"},
		{Name: "Ireland", Code: "IE", ISO3: "IRL", Continent: "Europe", Currency: "EUR", PhoneCode: "+353"},
		{Name: "Israel", Code: "IL", ISO3: "ISR", Continent: "Asia", Currency: "ILS", PhoneCode: "+972"},
		{Name: "Italy", Code: "IT", ISO3: "ITA", Continent: "Europe", Currency: "EUR", PhoneCode: "+39"},
		{Name: "Ivory Coast", Code: "CI", ISO3: "CIV", Continent: "Africa", Currency: "XOF", PhoneCode: "+225"},
		{Name: "Jamaica", Code: "JM", ISO3: "JAM", Continent: "North America", Currency: "JMD", PhoneCode: "+1-876"},
		{Name: "Japan", Code: "JP", ISO3: "JPN", Continent: "Asia", Currency: "JPY", PhoneCode: "+81"},
		{Name: "Jordan", Code: "JO", ISO3: "JOR", Continent: "Asia", Currency: "JOD", PhoneCode: "+962"},
		{Name: "Kazakhstan", Code: "KZ", ISO3: "KAZ", Continent: "Asia", Currency: "KZT", PhoneCode: "+7"},
		{Name: "Kenya", Code: "KE", ISO3: "KEN", Continent: "Africa", Currency: "KES", PhoneCode: "+254"},
		{Name: "Kiribati", Code: "KI", ISO3: "KIR", Continent: "Oceania", Currency: "AUD", PhoneCode: "+686"},
		{Name: "Kuwait", Code: "KW", ISO3: "KWT", Continent: "Asia", Currency: "KWD", PhoneCode: "+965"},
		{Name: "Kyrgyzstan", Code: "KG", ISO3: "KGZ", Continent: "Asia", Currency: "KGS", PhoneCode: "+996"},
		{Name: "Laos", Code: "LA", ISO3: "LAO", Continent: "Asia", Currency: "LAK", PhoneCode: "+856"},
		{Name: "Latvia", Code: "LV", ISO3: "LVA", Continent: "Europe", Currency: "EUR", PhoneCode: "+371"},
		{Name: "Lebanon", Code: "LB", ISO3: "LBN", Continent: "Asia", Currency: "LBP", PhoneCode: "+961"},
		{Name: "Lesotho", Code: "LS", ISO3: "LSO", Continent: "Africa", Currency: "LSL", PhoneCode: "+266"},
		{Name: "Liberia", Code: "LR", ISO3: "LBR", Continent: "Africa", Currency: "LRD", PhoneCode: "+231"},
		{Name: "Libya", Code: "LY", ISO3: "LBY", Continent: "Africa", Currency: "LYD", PhoneCode: "+218"},
		{Name: "Liechtenstein", Code: "LI", ISO3: "LIE", Continent: "Europe", Currency: "CHF", PhoneCode: "+423"},
		{Name: "Lithuania", Code: "LT", ISO3: "LTU", Continent: "Europe", Currency: "EUR", PhoneCode: "+370"},
		{Name: "Luxembourg", Code: "LU", ISO3: "LUX", Continent: "Europe", Currency: "EUR", PhoneCode: "+352"},
		{Name: "Madagascar", Code: "MG", ISO3: "MDG", Continent: "Africa", Currency: "MGA", PhoneCode: "+261"},
		{Name: "Malawi", Code: "MW", ISO3: "MWI", Continent: "Africa", Currency: "MWK", PhoneCode: "+265"},
		{Name: "Malaysia", Code: "MY", ISO3: "MYS", Continent: "Asia", Currency: "MYR", PhoneCode: "+60"},
		{Name: "Maldives", Code: "MV", ISO3: "MDV", Continent: "Asia", Currency: "MVR", PhoneCode: "+960"},
		{Name: "Mali", Code: "ML", ISO3: "MLI", Continent: "Africa", Currency: "XOF", PhoneCode: "+223"},
		{Name: "Malta", Code: "MT", ISO3: "MLT", Continent: "Europe", Currency: "EUR", PhoneCode: "+356"},
		{Name: "Marshall Islands", Code: "MH", ISO3: "MHL", Continent: "Oceania", Currency: "USD", PhoneCode: "+692"},
		{Name: "Mauritania", Code: "MR", ISO3: "MRT", Continent: "Africa", Currency: "MRU", PhoneCode: "+222"},
		{Name: "Mauritius", Code: "MU", ISO3: "MUS", Continent: "Africa", Currency: "MUR", PhoneCode: "+230"},
		{Name: "Mexico", Code: "MX", ISO3: "MEX", Continent: "North America", Currency: "MXN", PhoneCode: "+52"},
		{Name: "Micronesia", Code: "FM", ISO3: "FSM", Continent: "Oceania", Currency: "USD", PhoneCode: "+691"},
		{Name: "Moldova", Code: "MD", ISO3: "MDA", Continent: "Europe", Currency: "MDL", PhoneCode: "+373"},
		{Name: "Monaco", Code: "MC", ISO3: "MCO", Continent: "Europe", Currency: "EUR", PhoneCode: "+377"},
		{Name: "Mongolia", Code: "MN", ISO3: "MNG", Continent: "Asia", Currency: "MNT", PhoneCode: "+976"},
		{Name: "Montenegro", Code: "ME", ISO3: "MNE", Continent: "Europe", Currency: "EUR", PhoneCode: "+382"},
		{Name: "Morocco", Code: "MA", ISO3: "MAR", Continent: "Africa", Currency: "MAD", PhoneCode: "+212"},
		{Name: "Mozambique", Code: "MZ", ISO3: "MOZ", Continent: "Africa", Currency: "MZN", PhoneCode: "+258"},
		{Name: "Myanmar", Code: "MM", ISO3: "MMR", Continent: "Asia", Currency: "MMK", PhoneCode: "+95"},
		{Name: "Namibia", Code: "NA", ISO3: "NAM", Continent: "Africa", Currency: "NAD", PhoneCode: "+264"},
		{Name: "Nauru", Code: "NR", ISO3: "NRU", Continent: "Oceania", Currency: "AUD", PhoneCode: "+674"},
		{Name: "Nepal", Code: "NP", ISO3: "NPL", Continent: "Asia", Currency: "NPR", PhoneCode: "+977"},
		{Name: "Netherlands", Code: "NL", ISO3: "NLD", Continent: "Europe", Currency: "EUR", PhoneCode: "+31"},
		{Name: "New Zealand", Code: "NZ", ISO3: "NZL", Continent: "Oceania", Currency: "NZD", PhoneCode: "+64"},
		{Name: "Nicaragua", Code: "NI", ISO3: "NIC", Continent: "North America", Currency: "NIO", PhoneCode: "+505"},
		{Name: "Niger", Code: "NE", ISO3: "NER", Continent: "Africa", Currency: "XOF", PhoneCode: "+227"},
		{Name: "Nigeria", Code: "NG", ISO3: "NGA", Continent: "Africa", Currency: "NGN", PhoneCode: "+234"},
		{Name: "North Korea", Code: "KP", ISO3: "PRK", Continent: "Asia", Currency: "KPW", PhoneCode: "+850"},
		{Name: "North Macedonia", Code: "MK", ISO3: "MKD", Continent: "Europe", Currency: "MKD", PhoneCode: "+389"},
		{Name: "Norway", Code: "NO", ISO3: "NOR", Continent: "Europe", Currency: "NOK", PhoneCode: "+47"},
		{Name: "Oman", Code: "OM", ISO3: "OMN", Continent: "Asia", Currency: "OMR", PhoneCode: "+968"},
		{Name: "Pakistan", Code: "PK", ISO3: "PAK", Continent: "Asia", Currency: "PKR", PhoneCode: "+92"},
		{Name: "Palau", Code: "PW", ISO3: "PLW", Continent: "Oceania", Currency: "USD", PhoneCode: "+680"},
		{Name: "Palestine", Code: "PS", ISO3: "PSE", Continent: "Asia", Currency: "ILS", PhoneCode: "+970"},
		{Name: "Panama", Code: "PA", ISO3: "PAN", Continent: "North America", Currency: "PAB", PhoneCode: "+507"},
		{Name: "Papua New Guinea", Code: "PG", ISO3: "PNG", Continent: "Oceania", Currency: "PGK", PhoneCode: "+675"},
		{Name: "Paraguay", Code: "PY", ISO3: "PRY", Continent: "South America", Currency: "PYG", PhoneCode: "+595"},
		{Name: "Peru", Code: "PE", ISO3: "PER", Continent: "South America", Currency: "PEN", PhoneCode: "+51"},
		{Name: "Philippines", Code: "PH", ISO3: "PHL", Continent: "Asia", Currency: "PHP", PhoneCode: "+63"},
		{Name: "Poland", Code: "PL", ISO3: "POL", Continent: "Europe", Currency: "PLN", PhoneCode: "+48"},
		{Name: "Portugal", Code: "PT", ISO3: "PRT", Continent: "Europe", Currency: "EUR", PhoneCode: "+351"},
		{Name: "Qatar", Code: "QA", ISO3: "QAT", Continent: "Asia", Currency: "QAR", PhoneCode: "+974"},
		{Name: "Romania", Code: "RO", ISO3: "ROU", Continent: "Europe", Currency: "RON", PhoneCode: "+40"},
		{Name: "Russia", Code: "RU", ISO3: "RUS", Continent: "Europe", Currency: "RUB", PhoneCode: "+7"},
		{Name: "Rwanda", Code: "RW", ISO3: "RWA", Continent: "Africa", Currency: "RWF", PhoneCode: "+250"},
		{Name: "Saint Kitts and Nevis", Code: "KN", ISO3: "KNA", Continent: "North America", Currency: "XCD", PhoneCode: "+1-869"},
		{Name: "Saint Lucia", Code: "LC", ISO3: "LCA", Continent: "North America", Currency: "XCD", PhoneCode: "+1-758"},
		{Name: "Saint Vincent and the Grenadines", Code: "VC", ISO3: "VCT", Continent: "North America", Currency: "XCD", PhoneCode: "+1-784"},
		{Name: "Samoa", Code: "WS", ISO3: "WSM", Continent: "Oceania", Currency: "WST", PhoneCode: "+685"},
		{Name: "San Marino", Code: "SM", ISO3: "SMR", Continent: "Europe", Currency: "EUR", PhoneCode: "+378"},
		{Name: "Sao Tome and Principe", Code: "ST", ISO3: "STP", Continent: "Africa", Currency: "STN", PhoneCode: "+239"},
		{Name: "Saudi Arabia", Code: "SA", ISO3: "SAU", Continent: "Asia", Currency: "SAR", PhoneCode: "+966"},
		{Name: "Senegal", Code: "SN", ISO3: "SEN", Continent: "Africa", Currency: "XOF", PhoneCode: "+221"},
		{Name: "Serbia", Code: "RS", ISO3: "SRB", Continent: "Europe", Currency: "RSD", PhoneCode: "+381"},
		{Name: "Seychelles", Code: "SC", ISO3: "SYC", Continent: "Africa", Currency: "SCR", PhoneCode: "+248"},
		{Name: "Sierra Leone", Code: "SL", ISO3: "SLE", Continent: "Africa", Currency: "SLL", PhoneCode: "+232"},
		{Name: "Singapore", Code: "SG", ISO3: "SGP", Continent: "Asia", Currency: "SGD", PhoneCode: "+65"},
		{Name: "Slovakia", Code: "SK", ISO3: "SVK", Continent: "Europe", Currency: "EUR", PhoneCode: "+421"},
		{Name: "Slovenia", Code: "SI", ISO3: "SVN", Continent: "Europe", Currency: "EUR", PhoneCode: "+386"},
		{Name: "Solomon Islands", Code: "SB", ISO3: "SLB", Continent: "Oceania", Currency: "SBD", PhoneCode: "+677"},
		{Name: "Somalia", Code: "SO", ISO3: "SOM", Continent: "Africa", Currency: "SOS", PhoneCode: "+252"},
		{Name: "South Africa", Code: "ZA", ISO3: "ZAF", Continent: "Africa", Currency: "ZAR", PhoneCode: "+27"},
		{Name: "South Korea", Code: "KR", ISO3: "KOR", Continent: "Asia", Currency: "KRW", PhoneCode: "+82"},
		{Name: "South Sudan", Code: "SS", ISO3: "SSD", Continent: "Africa", Currency: "SSP", PhoneCode: "+211"},
		{Name: "Spain", Code: "ES", ISO3: "ESP", Continent: "Europe", Currency: "EUR", PhoneCode: "+34"},
		{Name: "Sri Lanka", Code: "LK", ISO3: "LKA", Continent: "Asia", Currency: "LKR", PhoneCode: "+94"},
		{Name: "Sudan", Code: "SD", ISO3: "SDN", Continent: "Africa", Currency: "SDG", PhoneCode: "+249"},
		{Name: "Suriname", Code: "SR", ISO3: "SUR", Continent: "South America", Currency: "SRD", PhoneCode: "+597"},
		{Name: "Sweden", Code: "SE", ISO3: "SWE", Continent: "Europe", Currency: "SEK", PhoneCode: "+46"},
		{Name: "Switzerland", Code: "CH", ISO3: "CHE", Continent: "Europe", Currency: "CHF", PhoneCode: "+41"},
		{Name: "Syria", Code: "SY", ISO3: "SYR", Continent: "Asia", Currency: "SYP", PhoneCode: "+963"},
		{Name: "Taiwan", Code: "TW", ISO3: "TWN", Continent: "Asia", Currency: "TWD", PhoneCode: "+886"},
		{Name: "Tajikistan", Code: "TJ", ISO3: "TJK", Continent: "Asia", Currency: "TJS", PhoneCode: "+992"},
		{Name: "Tanzania", Code: "TZ", ISO3: "TZA", Continent: "Africa", Currency: "TZS", PhoneCode: "+255"},
		{Name: "Thailand", Code: "TH", ISO3: "THA", Continent: "Asia", Currency: "THB", PhoneCode: "+66"},
		{Name: "Timor-Leste", Code: "TL", ISO3: "TLS", Continent: "Asia", Currency: "USD", PhoneCode: "+670"},
		{Name: "Togo", Code: "TG", ISO3: "TGO", Continent: "Africa", Currency: "XOF", PhoneCode: "+228"},
		{Name: "Tonga", Code: "TO", ISO3: "TON", Continent: "Oceania", Currency: "TOP", PhoneCode: "+676"},
		{Name: "Trinidad and Tobago", Code: "TT", ISO3: "TTO", Continent: "North America", Currency: "TTD", PhoneCode: "+1-868"},
		{Name: "Tunisia", Code: "TN", ISO3: "TUN", Continent: "Africa", Currency: "TND", PhoneCode: "+216"},
		{Name: "Turkey", Code: "TR", ISO3: "TUR", Continent: "Asia", Currency: "TRY", PhoneCode: "+90"},
		{Name: "Turkmenistan", Code: "TM", ISO3: "TKM", Continent: "Asia", Currency: "TMT", PhoneCode: "+993"},
		{Name: "Tuvalu", Code: "TV", ISO3: "TUV", Continent: "Oceania", Currency: "AUD", PhoneCode: "+688"},
		{Name: "Uganda", Code: "UG", ISO3: "UGA", Continent: "Africa", Currency: "UGX", PhoneCode: "+256"},
		{Name: "Ukraine", Code: "UA", ISO3: "UKR", Continent: "Europe", Currency: "UAH", PhoneCode: "+380"},
		{Name: "United Arab Emirates", Code: "AE", ISO3: "ARE", Continent: "Asia", Currency: "AED", PhoneCode: "+971"},
		{Name: "United Kingdom", Code: "GB", ISO3: "GBR", Continent: "Europe", Currency: "GBP", PhoneCode: "+44"},
		{Name: "United States", Code: "US", ISO3: "USA", Continent: "North America", Currency: "USD", PhoneCode: "+1"},
		{Name: "Uruguay", Code: "UY", ISO3: "URY", Continent: "South America", Currency: "UYU", PhoneCode: "+598"},
		{Name: "Uzbekistan", Code: "UZ", ISO3: "UZB", Continent: "Asia", Currency: "UZS", PhoneCode: "+998"},
		{Name: "Vanuatu", Code: "VU", ISO3: "VUT", Continent: "Oceania", Currency: "VUV", PhoneCode: "+678"},
		{Name: "Vatican City", Code: "VA", ISO3: "VAT", Continent: "Europe", Currency: "EUR", PhoneCode: "+379"},
		{Name: "Venezuela", Code: "VE", ISO3: "VEN", Continent: "South America", Currency: "VES", PhoneCode: "+58"},
		{Name: "Vietnam", Code: "VN", ISO3: "VNM", Continent: "Asia", Currency: "VND", PhoneCode: "+84"},
		{Name: "Yemen", Code: "YE", ISO3: "YEM", Continent: "Asia", Currency: "YER", PhoneCode: "+967"},
		{Name: "Zambia", Code: "ZM", ISO3: "ZMB", Continent: "Africa", Currency: "ZMW", PhoneCode: "+260"},
		{Name: "Zimbabwe", Code: "ZW", ISO3: "ZWE", Continent: "Africa", Currency: "ZWL", PhoneCode: "+263"},
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
	currencies := []Currency{
		{Name: "United States Dollar", Code: "USD", Symbol: "$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Euro", Code: "EUR", Symbol: "\u20ac", SymbolNative: "\u20ac", DecimalDigits: 2},
		{Name: "British Pound", Code: "GBP", Symbol: "\u00a3", SymbolNative: "\u00a3", DecimalDigits: 2},
		{Name: "Japanese Yen", Code: "JPY", Symbol: "\u00a5", SymbolNative: "\uffe5", DecimalDigits: 0},
		{Name: "Australian Dollar", Code: "AUD", Symbol: "A$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Canadian Dollar", Code: "CAD", Symbol: "CA$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Swiss Franc", Code: "CHF", Symbol: "CHF", SymbolNative: "CHF", DecimalDigits: 2},
		{Name: "Chinese Yuan", Code: "CNY", Symbol: "CN\u00a5", SymbolNative: "\u00a5", DecimalDigits: 2},
		{Name: "Indian Rupee", Code: "INR", Symbol: "\u20b9", SymbolNative: "\u20b9", DecimalDigits: 2},
		{Name: "Brazilian Real", Code: "BRL", Symbol: "R$", SymbolNative: "R$", DecimalDigits: 2},
		{Name: "South Korean Won", Code: "KRW", Symbol: "\u20a9", SymbolNative: "\u20a9", DecimalDigits: 0},
		{Name: "Mexican Peso", Code: "MXN", Symbol: "MX$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Singapore Dollar", Code: "SGD", Symbol: "S$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Hong Kong Dollar", Code: "HKD", Symbol: "HK$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Norwegian Krone", Code: "NOK", Symbol: "Nkr", SymbolNative: "kr", DecimalDigits: 2},
		{Name: "Swedish Krona", Code: "SEK", Symbol: "Skr", SymbolNative: "kr", DecimalDigits: 2},
		{Name: "Danish Krone", Code: "DKK", Symbol: "Dkr", SymbolNative: "kr", DecimalDigits: 2},
		{Name: "New Zealand Dollar", Code: "NZD", Symbol: "NZ$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "South African Rand", Code: "ZAR", Symbol: "R", SymbolNative: "R", DecimalDigits: 2},
		{Name: "Russian Ruble", Code: "RUB", Symbol: "RUB", SymbolNative: "\u20bd", DecimalDigits: 2},
		{Name: "Turkish Lira", Code: "TRY", Symbol: "TL", SymbolNative: "\u20ba", DecimalDigits: 2},
		{Name: "Polish Zloty", Code: "PLN", Symbol: "z\u0142", SymbolNative: "z\u0142", DecimalDigits: 2},
		{Name: "Thai Baht", Code: "THB", Symbol: "\u0e3f", SymbolNative: "\u0e3f", DecimalDigits: 2},
		{Name: "Indonesian Rupiah", Code: "IDR", Symbol: "Rp", SymbolNative: "Rp", DecimalDigits: 0},
		{Name: "Malaysian Ringgit", Code: "MYR", Symbol: "RM", SymbolNative: "RM", DecimalDigits: 2},
		{Name: "Philippine Peso", Code: "PHP", Symbol: "\u20b1", SymbolNative: "\u20b1", DecimalDigits: 2},
		{Name: "Czech Koruna", Code: "CZK", Symbol: "K\u010d", SymbolNative: "K\u010d", DecimalDigits: 2},
		{Name: "Hungarian Forint", Code: "HUF", Symbol: "Ft", SymbolNative: "Ft", DecimalDigits: 0},
		{Name: "Israeli New Shekel", Code: "ILS", Symbol: "\u20aa", SymbolNative: "\u20aa", DecimalDigits: 2},
		{Name: "Chilean Peso", Code: "CLP", Symbol: "CL$", SymbolNative: "$", DecimalDigits: 0},
		{Name: "Nigerian Naira", Code: "NGN", Symbol: "\u20a6", SymbolNative: "\u20a6", DecimalDigits: 2},
		{Name: "Colombian Peso", Code: "COP", Symbol: "CO$", SymbolNative: "$", DecimalDigits: 0},
		{Name: "Argentine Peso", Code: "ARS", Symbol: "AR$", SymbolNative: "$", DecimalDigits: 2},
		{Name: "Egyptian Pound", Code: "EGP", Symbol: "EGP", SymbolNative: "\u062c.\u0645.", DecimalDigits: 2},
		{Name: "Vietnamese Dong", Code: "VND", Symbol: "\u20ab", SymbolNative: "\u20ab", DecimalDigits: 0},
		{Name: "Ukrainian Hryvnia", Code: "UAH", Symbol: "\u20b4", SymbolNative: "\u20b4", DecimalDigits: 2},
		{Name: "Peruvian Sol", Code: "PEN", Symbol: "S/.", SymbolNative: "S/.", DecimalDigits: 2},
		{Name: "Romanian Leu", Code: "RON", Symbol: "RON", SymbolNative: "RON", DecimalDigits: 2},
		{Name: "Bangladeshi Taka", Code: "BDT", Symbol: "Tk", SymbolNative: "\u09f3", DecimalDigits: 2},
		{Name: "Pakistani Rupee", Code: "PKR", Symbol: "PKRs", SymbolNative: "\u20a8", DecimalDigits: 0},
		{Name: "Kenyan Shilling", Code: "KES", Symbol: "Ksh", SymbolNative: "Ksh", DecimalDigits: 2},
		{Name: "Qatari Riyal", Code: "QAR", Symbol: "QR", SymbolNative: "\u0631.\u0642.", DecimalDigits: 2},
		{Name: "Saudi Riyal", Code: "SAR", Symbol: "SR", SymbolNative: "\u0631.\u0633.", DecimalDigits: 2},
		{Name: "United Arab Emirates Dirham", Code: "AED", Symbol: "AED", SymbolNative: "\u062f.\u0625.", DecimalDigits: 2},
		{Name: "Kuwaiti Dinar", Code: "KWD", Symbol: "KD", SymbolNative: "\u062f.\u0643.", DecimalDigits: 3},
		{Name: "Bahraini Dinar", Code: "BHD", Symbol: "BD", SymbolNative: "\u062f.\u0628.", DecimalDigits: 3},
		{Name: "Omani Rial", Code: "OMR", Symbol: "OMR", SymbolNative: "\u0631.\u0639.", DecimalDigits: 3},
		{Name: "Moroccan Dirham", Code: "MAD", Symbol: "MAD", SymbolNative: "\u062f.\u0645.", DecimalDigits: 2},
		{Name: "Tanzanian Shilling", Code: "TZS", Symbol: "TSh", SymbolNative: "TSh", DecimalDigits: 0},
		{Name: "Ghanaian Cedi", Code: "GHS", Symbol: "GH\u20b5", SymbolNative: "GH\u20b5", DecimalDigits: 2},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      len(currencies),
		"currencies": currencies,
	})
}

func (h *Handler) listLanguages(w http.ResponseWriter, r *http.Request) {
	languages := []Language{
		{Name: "English", Code: "en", NativeName: "English"},
		{Name: "Spanish", Code: "es", NativeName: "Espa\u00f1ol"},
		{Name: "French", Code: "fr", NativeName: "Fran\u00e7ais"},
		{Name: "German", Code: "de", NativeName: "Deutsch"},
		{Name: "Japanese", Code: "ja", NativeName: "\u65e5\u672c\u8a9e"},
		{Name: "Chinese", Code: "zh", NativeName: "\u4e2d\u6587"},
		{Name: "Portuguese", Code: "pt", NativeName: "Portugu\u00eas"},
		{Name: "Arabic", Code: "ar", NativeName: "\u0627\u0644\u0639\u0631\u0628\u064a\u0629"},
		{Name: "Hindi", Code: "hi", NativeName: "\u0939\u093f\u0928\u094d\u0926\u0940"},
		{Name: "Korean", Code: "ko", NativeName: "\ud55c\uad6d\uc5b4"},
		{Name: "Russian", Code: "ru", NativeName: "\u0420\u0443\u0441\u0441\u043a\u0438\u0439"},
		{Name: "Italian", Code: "it", NativeName: "Italiano"},
		{Name: "Dutch", Code: "nl", NativeName: "Nederlands"},
		{Name: "Turkish", Code: "tr", NativeName: "T\u00fcrk\u00e7e"},
		{Name: "Polish", Code: "pl", NativeName: "Polski"},
		{Name: "Ukrainian", Code: "uk", NativeName: "\u0423\u043a\u0440\u0430\u0457\u043d\u0441\u044c\u043a\u0430"},
		{Name: "Romanian", Code: "ro", NativeName: "Rom\u00e2n\u0103"},
		{Name: "Greek", Code: "el", NativeName: "\u0395\u03bb\u03bb\u03b7\u03bd\u03b9\u03ba\u03ac"},
		{Name: "Czech", Code: "cs", NativeName: "\u010ce\u0161tina"},
		{Name: "Swedish", Code: "sv", NativeName: "Svenska"},
		{Name: "Hungarian", Code: "hu", NativeName: "Magyar"},
		{Name: "Finnish", Code: "fi", NativeName: "Suomi"},
		{Name: "Danish", Code: "da", NativeName: "Dansk"},
		{Name: "Norwegian", Code: "no", NativeName: "Norsk"},
		{Name: "Thai", Code: "th", NativeName: "\u0e44\u0e17\u0e22"},
		{Name: "Vietnamese", Code: "vi", NativeName: "Ti\u1ebfng Vi\u1ec7t"},
		{Name: "Indonesian", Code: "id", NativeName: "Bahasa Indonesia"},
		{Name: "Malay", Code: "ms", NativeName: "Bahasa Melayu"},
		{Name: "Filipino", Code: "tl", NativeName: "Filipino"},
		{Name: "Bengali", Code: "bn", NativeName: "\u09ac\u09be\u0982\u09b2\u09be"},
		{Name: "Tamil", Code: "ta", NativeName: "\u0ba4\u0bae\u0bbf\u0bb4\u0bcd"},
		{Name: "Telugu", Code: "te", NativeName: "\u0c24\u0c46\u0c32\u0c41\u0c17\u0c41"},
		{Name: "Marathi", Code: "mr", NativeName: "\u092e\u0930\u093e\u0920\u0940"},
		{Name: "Urdu", Code: "ur", NativeName: "\u0627\u0631\u062f\u0648"},
		{Name: "Gujarati", Code: "gu", NativeName: "\u0a97\u0ac1\u0a9c\u0ab0\u0abe\u0aa4\u0ac0"},
		{Name: "Kannada", Code: "kn", NativeName: "\u0c95\u0ca8\u0ccd\u0ca8\u0ca1"},
		{Name: "Malayalam", Code: "ml", NativeName: "\u0d2e\u0d32\u0d2f\u0d3e\u0d33\u0d02"},
		{Name: "Punjabi", Code: "pa", NativeName: "\u0a2a\u0a70\u0a1c\u0a3e\u0a2c\u0a40"},
		{Name: "Persian", Code: "fa", NativeName: "\u0641\u0627\u0631\u0633\u06cc"},
		{Name: "Swahili", Code: "sw", NativeName: "Kiswahili"},
		{Name: "Hebrew", Code: "he", NativeName: "\u05e2\u05d1\u05e8\u05d9\u05ea"},
		{Name: "Catalan", Code: "ca", NativeName: "Catal\u00e0"},
		{Name: "Croatian", Code: "hr", NativeName: "Hrvatski"},
		{Name: "Slovak", Code: "sk", NativeName: "Sloven\u010dina"},
		{Name: "Bulgarian", Code: "bg", NativeName: "\u0411\u044a\u043b\u0433\u0430\u0440\u0441\u043a\u0438"},
		{Name: "Serbian", Code: "sr", NativeName: "\u0421\u0440\u043f\u0441\u043a\u0438"},
		{Name: "Lithuanian", Code: "lt", NativeName: "Lietuvi\u0173"},
		{Name: "Latvian", Code: "lv", NativeName: "Latvie\u0161u"},
		{Name: "Estonian", Code: "et", NativeName: "Eesti"},
		{Name: "Slovenian", Code: "sl", NativeName: "Sloven\u0161\u010dina"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(languages),
		"languages": languages,
	})
}
