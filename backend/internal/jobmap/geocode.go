package jobmap

import (
	"context"
	"net/url"
	"strings"
)

func (c *Client) Geocode(ctx context.Context, query string) ([]Place, error) {
	var body struct {
		Results []struct {
			ID          string  `json:"place_id"`
			City        string  `json:"city"`
			Name        string  `json:"name"`
			Formatted   string  `json:"formatted"`
			Country     string  `json:"country"`
			CountryCode string  `json:"country_code"`
			Lat         float64 `json:"lat"`
			Lon         float64 `json:"lon"`
		} `json:"results"`
	}
	values := url.Values{"text": {query}, "type": {"city"}, "format": {"json"}, "limit": {"6"}, "apiKey": {c.Config.GeoapifyKey}}
	if err := c.get(ctx, "https://api.geoapify.com/v1/geocode/search?"+values.Encode(), nil, &body); err != nil {
		return nil, err
	}
	places := []Place{}
	for _, p := range body.Results {
		name := p.City
		if name == "" {
			name = p.Name
		}
		if name == "" {
			name = p.Formatted
		}
		places = append(places, Place{ID: p.ID, Name: name, Country: p.Country, CountryCode: strings.ToUpper(p.CountryCode), Latitude: p.Lat, Longitude: p.Lon, Attribution: "Geoapify / OpenStreetMap"})
	}
	return places, nil
}
