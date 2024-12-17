package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type OpenMeteoAPIResponse struct {
	Latitude             float64     `json:"latitude"`
	Longitude            float64     `json:"longitude"`
	GenerationtimeMS     float64     `json:"generationtime_ms"`
	UTCOffsetSeconds     int64       `json:"utc_offset_seconds"`
	Timezone             string      `json:"timezone"`
	TimezoneAbbreviation string      `json:"timezone_abbreviation"`
	Elevation            float64     `json:"elevation"`
	HourlyUnits          HourlyUnits `json:"hourly_units"`
	Hourly               Hourly      `json:"hourly"`
}

type Hourly struct {
	Time           []string  `json:"time"`
	Temperature2M  []float64 `json:"temperature_2m"`
	CloudCoverLow  []int64   `json:"cloud_cover_low"`
	CloudCoverMid  []int64   `json:"cloud_cover_mid"`
	CloudCoverHigh []int64   `json:"cloud_cover_high"`
	WindSpeed10M   []float64 `json:"wind_speed_10m"`
	WindGusts10M   []float64 `json:"wind_gusts_10m"`
}

type HourlyUnits struct {
	Time           string `json:"time"`
	Temperature2M  string `json:"temperature_2m"`
	CloudCoverLow  string `json:"cloud_cover_low"`
	CloudCoverMid  string `json:"cloud_cover_mid"`
	CloudCoverHigh string `json:"cloud_cover_high"`
	WindSpeed10M   string `json:"wind_speed_10m"`
	WindGusts10M   string `json:"wind_gusts_10m"`
}

type DataPoint struct {
	Time       time.Time
	LowClouds  int64
	MidClouds  int64
	HighClouds int64
	MoonIllum  int64
	WindSpeed  float64
	WindGusts  float64
}

// Init() goes to OpenMeteoEndpoint makes HTTPS request and stores result as OpenMeteoAPIResponse object
func (response *OpenMeteoAPIResponse) Init(apiEndpoint, parameters, lat, lon string) {
	log.Println("INFO: Making request to Open-Meteo API and parsing response")
	client := &http.Client{}

	// Set paramenters
	params := url.Values{}
	params.Add("latitude", lat)
	params.Add("longitude", lon)
	params.Add("hourly", parameters)
	params.Add("timezone", "auto")

	// Make request to Open-Meteo API
	req, err := http.NewRequest("GET", apiEndpoint+params.Encode(), nil)
	if err != nil {
		log.Println("ERROR: Couldn't create new Open-Meteo API request", err)
		return
	}

	parseFormErr := req.ParseForm()
	if parseFormErr != nil {
		log.Println(parseFormErr)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}

	// Read Response Body
	if resp.StatusCode != 200 {
		fmt.Println("ERROR: Open-Meteo API response code:", resp.Status)
		return
	}

	log.Println("INFO: Got API response", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)

	// Save response as OpenMeteoAPIResponse object
	err = json.Unmarshal(respBody, response)
	if err != nil {
		log.Println("ERROR: cannot Unmarshal JSON", err)
		return
	}
}

// Points() return DataPoints object based on OpenMeteoAPIResponse fields
func (data OpenMeteoAPIResponse) Points() DataPoints {
	points := DataPoints{}

	for i := 0; i < len(data.Hourly.Time); i++ {
		timeString := data.Hourly.Time[i]
		time, _ := time.ParseInLocation("2006-01-02T15:04", timeString, time.FixedZone(data.TimezoneAbbreviation, 1*60*60))

		point := DataPoint{
			Time:       time,
			LowClouds:  data.Hourly.CloudCoverLow[i],
			MidClouds:  data.Hourly.CloudCoverMid[i],
			HighClouds: data.Hourly.CloudCoverHigh[i],
			WindSpeed:  data.Hourly.WindSpeed10M[i],
			WindGusts:  data.Hourly.WindGusts10M[i],
		}

		points = append(points, point)
	}

	return points
}
