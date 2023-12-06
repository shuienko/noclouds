package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type MBCloudsResponse struct {
	Metadata Metadata `json:"metadata"`
	Units    Units    `json:"units"`
	Data1H   Data1H   `json:"data_1h"`
}

type Data1H struct {
	Time            []string `json:"time"`
	Sunshinetime    []*int64 `json:"sunshinetime"`
	Lowclouds       []int64  `json:"lowclouds"`
	Midclouds       []int64  `json:"midclouds"`
	Highclouds      []int64  `json:"highclouds"`
	Visibility      []int64  `json:"visibility"`
	Totalcloudcover []int64  `json:"totalcloudcover"`
}

type Metadata struct {
	Name                  string  `json:"name"`
	Latitude              float64 `json:"latitude"`
	Longitude             float64 `json:"longitude"`
	Height                int64   `json:"height"`
	TimezoneAbbrevation   string  `json:"timezone_abbrevation"`
	UTCTimeoffset         float64 `json:"utc_timeoffset"`
	ModelrunUTC           string  `json:"modelrun_utc"`
	ModelrunUpdatetimeUTC string  `json:"modelrun_updatetime_utc"`
}

type Units struct {
	Time         string `json:"time"`
	Cloudcover   string `json:"cloudcover"`
	Sunshinetime string `json:"sunshinetime"`
	Visibility   string `json:"visibility"`
}

type DataPoint struct {
	Time       time.Time
	LowClouds  int64
	MidClouds  int64
	HighClouds int64
}

type DataPoints []DataPoint

func (d DataPoint) isGood() bool {
	maxCloudCoverInt, _ := strconv.Atoi(os.Getenv("MAX_CLOUD_COVER"))
	maxCloudCoverInt64 := int64(maxCloudCoverInt)

	if d.HighClouds <= maxCloudCoverInt64 && d.MidClouds <= maxCloudCoverInt64 && d.LowClouds <= maxCloudCoverInt64 {
		return true
	} else {
		return false
	}
}

func (d DataPoint) atNight() bool {
	nightStart, _ := strconv.Atoi(os.Getenv("NIGHT_START_HOUR"))
	nightEnd, _ := strconv.Atoi(os.Getenv("NIGHT_END_HOUR"))

	if d.Time.Hour() >= nightStart || d.Time.Hour() <= nightEnd {
		return true
	} else {
		return false
	}
}

func (dp DataPoints) Good() DataPoints {
	good := DataPoints{}
	for _, v := range dp {
		if v.Time.After(time.Now()) && v.isGood() && v.atNight() {
			good = append(good, v)
		}
	}

	return good
}

func (dp DataPoints) onlyStartPoints() DataPoints {
	onlyStartPoints := DataPoints{}
	hoursStreak, _ := strconv.Atoi(os.Getenv("NIGHT_HOURS_STREAK"))

	i := 0
	for i < len(dp)-hoursStreak {
		sum := 0
		for j := 1; j <= hoursStreak; j++ {
			diff := dp[i+j].Time.Sub(dp[i+j-1].Time)
			sum += int(diff.Hours())
		}

		if sum > hoursStreak {
			i++
			continue
		} else {
			if i > 0 && dp[i].Time.Sub(dp[i-1].Time).Hours() == 1 {
				i++
				continue
			}
			onlyStartPoints = append(onlyStartPoints, dp[i])
			i = i + hoursStreak
		}
	}

	return onlyStartPoints
}

func (mbresponse *MBCloudsResponse) Init() {
	client := &http.Client{}
	MeteoblueAPIEndpoint := os.Getenv("MB_API_ENDPOINT")

	// Set paramenters
	params := url.Values{}
	params.Add("apikey", os.Getenv("MB_API_KEY"))
	params.Add("lat", os.Getenv("MB_LAT"))
	params.Add("lon", os.Getenv("MB_LON"))
	params.Add("asl", os.Getenv("MB_ALT"))
	params.Add("format", "json")

	// Make request to Meteoblue API
	req, _ := http.NewRequest("GET", MeteoblueAPIEndpoint+params.Encode(), nil)

	parseFormErr := req.ParseForm()
	if parseFormErr != nil {
		fmt.Println(parseFormErr)
	}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Failure : ", err)
	}

	// Read Response Body
	if resp.StatusCode != 200 {
		fmt.Println("response Status : ", resp.Status)
		log.Fatal("Exit. Response from Meteoblue API is not 200 OK")
	}

	respBody, _ := io.ReadAll(resp.Body)

	// Save response as MBCloudsResponse object
	err = json.Unmarshal(respBody, mbresponse)
	if err != nil {
		log.Fatal(err)
	}

}

func (data MBCloudsResponse) Points() DataPoints {
	points := DataPoints{}

	for i := 0; i < len(data.Data1H.Time); i++ {
		timeString := data.Data1H.Time[i]
		time, _ := time.ParseInLocation("2006-01-02 15:04", timeString, time.FixedZone(data.Metadata.TimezoneAbbrevation, 1*60*60))

		point := DataPoint{
			Time:       time,
			LowClouds:  data.Data1H.Lowclouds[i],
			MidClouds:  data.Data1H.Midclouds[i],
			HighClouds: data.Data1H.Highclouds[i],
		}

		points = append(points, point)
	}

	return points
}

func main() {
	data := MBCloudsResponse{}
	data.Init()

	points := data.Points()
	pointsGood := points.Good()
	startPoints := pointsGood.onlyStartPoints()
	fmt.Println(startPoints)
}
