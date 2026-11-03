package jobmap

import (
	_ "embed"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
)

//go:embed places.json
var placeData []byte

type Country struct {
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Continent  string    `json:"continent"`
	Aliases    []string  `json:"aliases"`
	UTCOffsets []float64 `json:"utcOffsets"`
}

type Place struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	CountryCode string   `json:"countryCode"`
	Country     string   `json:"country"`
	Population  int      `json:"population,omitempty"`
	Attribution string   `json:"attribution"`
}

var gazetteer struct {
	Source    string    `json:"source"`
	Countries []Country `json:"countries"`
	Places    []Place   `json:"places"`
}

func init() {
	if err := json.Unmarshal(placeData, &gazetteer); err != nil {
		panic(err)
	}
	for i := range gazetteer.Places {
		p := &gazetteer.Places[i]
		if c, ok := FindCountry(p.CountryCode); ok {
			p.Country = c.Name
		}
		p.Attribution = "GeoNames (CC BY 4.0)"
	}
	// Prefer the longest city name when a location contains several matching aliases.
	sort.SliceStable(gazetteer.Places, func(i, j int) bool { return len(gazetteer.Places[i].Name) > len(gazetteer.Places[j].Name) })
}

func Countries() []Country { return gazetteer.Countries }

func FindCountry(value string) (Country, bool) {
	value = strings.TrimSpace(value)
	for _, c := range gazetteer.Countries {
		if strings.EqualFold(value, c.Code) || strings.EqualFold(value, c.Name) {
			return c, true
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(value, a) {
				return c, true
			}
		}
	}
	return Country{}, false
}

func CountriesInText(text string) []Country {
	var out []Country
	for _, c := range gazetteer.Countries {
		matched := containsWord(text, c.Name)
		for _, alias := range c.Aliases {
			if len(alias) >= 3 && containsWord(text, alias) {
				matched = true
			}
		}
		// Avoid IN (a preposition) and CA (a US state). Explicit country fields use FindCountry.
		if c.Code == "US" && (containsWord(text, "US") || containsWord(text, "USA")) {
			matched = true
		}
		if c.Code == "GB" && containsWord(text, "UK") {
			matched = true
		}
		if matched {
			out = append(out, c)
		}
	}
	return out
}

func SearchPlaces(query string, limit int) []Place {
	query = strings.ToLower(strings.TrimSpace(query))
	result := []Place{}
	if len(query) < 2 {
		return result
	}
	for _, p := range gazetteer.Places {
		values := append([]string{p.Name, p.Name + ", " + p.Country}, p.Aliases...)
		for _, name := range values {
			if strings.Contains(strings.ToLower(name), query) {
				result = append(result, p)
				break
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		iPrefix, jPrefix := strings.HasPrefix(strings.ToLower(result[i].Name), query), strings.HasPrefix(strings.ToLower(result[j].Name), query)
		if iPrefix != jPrefix {
			return iPrefix
		}
		return result[i].Population > result[j].Population
	})
	return result[:min(limit, len(result))]
}

var locationCache sync.Map

// ResolveLocation uses an attributed offline gazetteer. These are city centroids,
// never office addresses. Unknown locations remain unmapped rather than guessed.
func ResolveLocation(name, countryHint string) Location {
	key := name + "\x00" + countryHint
	if v, ok := locationCache.Load(key); ok {
		return v.(Location)
	}
	out := Location{Name: name, Precision: "unknown"}
	if c, ok := FindCountry(countryHint); ok {
		out.Country, out.CountryCode = c.Name, c.Code
	}
	if out.CountryCode == "" {
		countries := CountriesInText(name)
		if len(countries) == 1 {
			out.Country, out.CountryCode = countries[0].Name, countries[0].Code
		}
	}
	lower := strings.ToLower(name)
	if !strings.Contains(lower, "remote") {
		for _, p := range gazetteer.Places {
			if out.CountryCode != "" && p.CountryCode != out.CountryCode {
				continue
			}
			matched := len([]rune(p.Name)) >= 4 && strings.Contains(lower, strings.ToLower(p.Name)) && containsWord(lower, p.Name)
			for _, a := range p.Aliases {
				if len(a) >= 4 && strings.Contains(lower, strings.ToLower(a)) && containsWord(lower, a) {
					matched = true
				}
			}
			if matched {
				out.Country, out.CountryCode = p.Country, p.CountryCode
				lat, lng := p.Latitude, p.Longitude
				out.Latitude, out.Longitude, out.Precision, out.Provenance = &lat, &lng, "city", p.Attribution
				break
			}
		}
	}
	locationCache.Store(key, out)
	return out
}

var eu = map[string]bool{"AT": true, "BE": true, "BG": true, "HR": true, "CY": true, "CZ": true, "DK": true, "EE": true, "FI": true, "FR": true, "DE": true, "GR": true, "HU": true, "IE": true, "IT": true, "LV": true, "LT": true, "LU": true, "MT": true, "NL": true, "PL": true, "PT": true, "RO": true, "SK": true, "SI": true, "ES": true, "SE": true}

func RemoteMatches(policy RemotePolicy, countryCode string, includeUnknown bool) bool {
	if countryCode == "" {
		return true
	}
	c, ok := FindCountry(countryCode)
	if !ok {
		return false
	}
	if len(policy.Timezones) > 0 {
		matched := false
		for _, offset := range c.UTCOffsets {
			for _, allowed := range policy.Timezones {
				if math.Abs(offset-allowed) < 0.01 {
					matched = true
				}
			}
		}
		if !matched {
			return false
		}
	}
	if policy.Scope == "worldwide" {
		return true
	}
	if policy.Scope == "unspecified" || len(policy.Regions) == 0 {
		return includeUnknown
	}
	for _, region := range policy.Regions {
		if rc, ok := FindCountry(region); ok && rc.Code == c.Code {
			return true
		}
		switch strings.ToLower(region) {
		case "europe":
			if c.Continent == "EU" {
				return true
			}
		case "european union":
			if eu[c.Code] {
				return true
			}
		case "asia":
			if c.Continent == "AS" {
				return true
			}
		case "africa":
			if c.Continent == "AF" {
				return true
			}
		case "north america":
			if c.Continent == "NA" {
				return true
			}
		case "south america":
			if c.Continent == "SA" {
				return true
			}
		case "americas":
			if c.Continent == "NA" || c.Continent == "SA" {
				return true
			}
		case "apac":
			if c.Continent == "AS" || c.Continent == "OC" {
				return true
			}
		case "emea":
			if c.Continent == "EU" || c.Continent == "AF" || strings.Contains(" AE SA BH KW OM QA IL JO LB TR ", " "+c.Code+" ") {
				return true
			}
		case "latam":
			if c.Continent == "SA" || strings.Contains(" MX GT HN SV NI CR PA CU DO PR ", " "+c.Code+" ") {
				return true
			}
		}
	}
	return false
}
