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

const (
	MeteoblueAPIEndpoint = "https://my.meteoblue.com/packages/clouds-1h?"
	NightStarts          = 22
	NightEnds            = 5
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
	if d.Time.Hour() >= NightStarts || d.Time.Hour() <= NightEnds {
		return true
	} else {
		return false
	}
}

func (mbresponse *MBCloudsResponse) Init() {
	client := &http.Client{}

	// Set paramenters
	params := url.Values{}
	params.Add("apikey", os.Getenv("MB_API_KEY"))
	params.Add("lat", os.Getenv("MB_LAT"))
	params.Add("lon", os.Getenv("MB_LON"))
	params.Add("asl", "51")
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

func main() {
	data := MBCloudsResponse{}

	// data.Init()

	data.Data1H.Highclouds = []int64{100, 100, 74, 38, 11, 2, 1, 0, 0, 0, 1, 22, 51, 73, 84, 88, 85, 60, 25, 0, 0, 2, 12, 39, 74, 100, 100, 100, 100, 74, 38, 12, 7, 10, 13, 6, 0, 0, 29, 70, 100, 100, 100, 100, 98, 95, 93, 96, 99, 100, 73, 35, 6, 1, 0, 0, 33, 78, 100, 77, 31, 0, 4, 18, 30, 34, 33, 31, 26, 19, 14, 12, 12, 19, 44, 76, 100, 100, 100, 100, 90, 77, 64, 53, 42, 32, 20, 8, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 4, 9, 12, 12, 9, 7, 5, 2, 1, 2, 4, 5, 4, 3, 1, 0, 0, 0, 0, 1, 6, 34, 72, 100, 99, 92, 82, 72, 62, 55, 52, 53, 54, 52, 50, 54, 67, 85, 99, 100, 100, 100, 98, 94, 92, 91, 90, 90, 91, 93, 95, 96, 98, 100, 100, 100, 99, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 92, 80, 71, 45, 15, 0, 23, 65, 96, 99, 100, 100, 80, 51, 31}

	data.Data1H.Midclouds = []int64{90, 90, 90, 90, 90, 0, 0, 45, 45, 90, 90, 90, 90, 90, 50, 89, 95, 91, 83, 77, 74, 74, 75, 78, 82, 82, 77, 66, 54, 40, 24, 13, 11, 11, 10, 7, 3, 2, 12, 27, 42, 56, 71, 80, 83, 81, 76, 65, 50, 38, 30, 24, 22, 25, 31, 32, 23, 10, 0, 0, 2, 4, 4, 3, 4, 19, 39, 52, 54, 49, 42, 33, 21, 13, 11, 10, 9, 6, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 3, 4, 6, 8, 9, 7, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 24, 33, 48, 69, 81, 75, 59, 47}

	data.Data1H.Lowclouds = []int64{90, 90, 90, 90, 90, 0, 0, 45, 45, 90, 90, 90, 90, 90, 50, 90, 73, 66, 60, 57, 62, 70, 77, 79, 79, 77, 72, 65, 61, 63, 68, 71, 66, 59, 53, 48, 45, 40, 34, 28, 22, 18, 14, 11, 7, 3, 0, 0, 1, 2, 5, 10, 16, 27, 39, 43, 33, 14, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 2, 3, 4, 6, 7, 8, 8, 9, 9, 8, 6, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 3, 5, 5, 5, 4, 3, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 5, 8, 10, 13, 15, 16, 16, 16, 17, 18, 19, 21, 24, 24, 22, 18, 15, 12, 10, 8}

	data.Data1H.Totalcloudcover = []int64{90, 90, 90, 90, 90, 50, 50, 50, 50, 90, 90, 90, 90, 90, 50, 90, 95, 91, 83, 77, 74, 74, 77, 79, 82, 82, 77, 66, 61, 63, 68, 71, 66, 59, 53, 48, 45, 40, 34, 28, 42, 56, 71, 80, 83, 81, 76, 65, 50, 38, 30, 24, 22, 27, 39, 43, 33, 23, 30, 23, 9, 4, 4, 5, 9, 19, 39, 52, 54, 49, 42, 33, 21, 13, 13, 23, 30, 30, 30, 30, 27, 23, 19, 16, 13, 9, 6, 4, 6, 7, 8, 8, 9, 9, 9, 7, 3, 0, 1, 3, 4, 3, 3, 2, 1, 1, 0, 1, 1, 2, 1, 3, 5, 5, 5, 4, 3, 1, 2, 10, 22, 30, 30, 27, 25, 22, 19, 16, 16, 16, 16, 16, 15, 16, 20, 26, 30, 30, 30, 30, 29, 28, 28, 27, 27, 27, 27, 28, 28, 29, 29, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 27, 24, 21, 17, 18, 19, 21, 24, 33, 48, 69, 81, 75, 59, 47}

	data.Data1H.Time = []string{"2023-12-06 00:00", "2023-12-06 01:00", "2023-12-06 02:00", "2023-12-06 03:00", "2023-12-06 04:00", "2023-12-06 05:00", "2023-12-06 06:00", "2023-12-06 07:00", "2023-12-06 08:00", "2023-12-06 09:00", "2023-12-06 10:00", "2023-12-06 11:00", "2023-12-06 12:00", "2023-12-06 13:00", "2023-12-06 14:00", "2023-12-06 15:00", "2023-12-06 16:00", "2023-12-06 17:00", "2023-12-06 18:00", "2023-12-06 19:00", "2023-12-06 20:00", "2023-12-06 21:00", "2023-12-06 22:00", "2023-12-06 23:00", "2023-12-07 00:00", "2023-12-07 01:00", "2023-12-07 02:00", "2023-12-07 03:00", "2023-12-07 04:00", "2023-12-07 05:00", "2023-12-07 06:00", "2023-12-07 07:00", "2023-12-07 08:00", "2023-12-07 09:00", "2023-12-07 10:00", "2023-12-07 11:00", "2023-12-07 12:00", "2023-12-07 13:00", "2023-12-07 14:00", "2023-12-07 15:00", "2023-12-07 16:00", "2023-12-07 17:00", "2023-12-07 18:00", "2023-12-07 19:00", "2023-12-07 20:00", "2023-12-07 21:00", "2023-12-07 22:00", "2023-12-07 23:00", "2023-12-08 00:00", "2023-12-08 01:00", "2023-12-08 02:00", "2023-12-08 03:00", "2023-12-08 04:00", "2023-12-08 05:00", "2023-12-08 06:00", "2023-12-08 07:00", "2023-12-08 08:00", "2023-12-08 09:00", "2023-12-08 10:00", "2023-12-08 11:00", "2023-12-08 12:00", "2023-12-08 13:00", "2023-12-08 14:00", "2023-12-08 15:00", "2023-12-08 16:00", "2023-12-08 17:00", "2023-12-08 18:00", "2023-12-08 19:00", "2023-12-08 20:00", "2023-12-08 21:00", "2023-12-08 22:00", "2023-12-08 23:00", "2023-12-09 00:00", "2023-12-09 01:00", "2023-12-09 02:00", "2023-12-09 03:00", "2023-12-09 04:00", "2023-12-09 05:00", "2023-12-09 06:00", "2023-12-09 07:00", "2023-12-09 08:00", "2023-12-09 09:00", "2023-12-09 10:00", "2023-12-09 11:00", "2023-12-09 12:00", "2023-12-09 13:00", "2023-12-09 14:00", "2023-12-09 15:00", "2023-12-09 16:00", "2023-12-09 17:00", "2023-12-09 18:00", "2023-12-09 19:00", "2023-12-09 20:00", "2023-12-09 21:00", "2023-12-09 22:00", "2023-12-09 23:00", "2023-12-10 00:00", "2023-12-10 01:00", "2023-12-10 02:00", "2023-12-10 03:00", "2023-12-10 04:00", "2023-12-10 05:00", "2023-12-10 06:00", "2023-12-10 07:00", "2023-12-10 08:00", "2023-12-10 09:00", "2023-12-10 10:00", "2023-12-10 11:00", "2023-12-10 12:00", "2023-12-10 13:00", "2023-12-10 14:00", "2023-12-10 15:00", "2023-12-10 16:00", "2023-12-10 17:00", "2023-12-10 18:00", "2023-12-10 19:00", "2023-12-10 20:00", "2023-12-10 21:00", "2023-12-10 22:00", "2023-12-10 23:00", "2023-12-11 00:00", "2023-12-11 01:00", "2023-12-11 02:00", "2023-12-11 03:00", "2023-12-11 04:00", "2023-12-11 05:00", "2023-12-11 06:00", "2023-12-11 07:00", "2023-12-11 08:00", "2023-12-11 09:00", "2023-12-11 10:00", "2023-12-11 11:00", "2023-12-11 12:00", "2023-12-11 13:00", "2023-12-11 14:00", "2023-12-11 15:00", "2023-12-11 16:00", "2023-12-11 17:00", "2023-12-11 18:00", "2023-12-11 19:00", "2023-12-11 20:00", "2023-12-11 21:00", "2023-12-11 22:00", "2023-12-11 23:00", "2023-12-12 00:00", "2023-12-12 01:00", "2023-12-12 02:00", "2023-12-12 03:00", "2023-12-12 04:00", "2023-12-12 05:00", "2023-12-12 06:00", "2023-12-12 07:00", "2023-12-12 08:00", "2023-12-12 09:00", "2023-12-12 10:00", "2023-12-12 11:00", "2023-12-12 12:00", "2023-12-12 13:00", "2023-12-12 14:00", "2023-12-12 15:00", "2023-12-12 16:00", "2023-12-12 17:00", "2023-12-12 18:00", "2023-12-12 19:00", "2023-12-12 20:00", "2023-12-12 21:00", "2023-12-12 22:00", "2023-12-12 23:00", "2023-12-13 00:00", "2023-12-13 01:00", "2023-12-13 02:00", "2023-12-13 03:00", "2023-12-13 04:00", "2023-12-13 05:00", "2023-12-13 06:00", "2023-12-13 07:00", "2023-12-13 08:00", "2023-12-13 09:00", "2023-12-13 10:00", "2023-12-13 11:00", "2023-12-13 12:00", "2023-12-13 13:00"}

	fmt.Println("data.Data1H.Highclouds -", len(data.Data1H.Highclouds))
	fmt.Println("data.Data1H.Lowclouds -", len(data.Data1H.Lowclouds))
	fmt.Println("data.Data1H.Midclouds -", len(data.Data1H.Midclouds))
	fmt.Println("data.Data1H.Time -", len(data.Data1H.Time))

	dataPoints := []DataPoint{}
	for i := 0; i < len(data.Data1H.Time); i++ {
		timeString := data.Data1H.Time[i]
		time, _ := time.ParseInLocation("2006-01-02 15:04", timeString, time.FixedZone(data.Metadata.TimezoneAbbrevation, 1*60*60))

		point := DataPoint{
			Time:       time,
			LowClouds:  data.Data1H.Lowclouds[i],
			MidClouds:  data.Data1H.Midclouds[i],
			HighClouds: data.Data1H.Highclouds[i],
		}

		dataPoints = append(dataPoints, point)
	}

	dataPointsGood := []DataPoint{}
	for _, v := range dataPoints {
		if v.Time.After(time.Now()) && v.isGood() && v.atNight() {
			dataPointsGood = append(dataPointsGood, v)
			fmt.Println("Good Weather:", v)
		}
	}

	dataPointsStart := []DataPoint{}
	i := 0
	for i < len(dataPointsGood)-4 {
		diff1 := dataPointsGood[i+1].Time.Sub(dataPointsGood[i].Time)
		diff2 := dataPointsGood[i+2].Time.Sub(dataPointsGood[i+1].Time)
		diff3 := dataPointsGood[i+3].Time.Sub(dataPointsGood[i+2].Time)
		diff4 := dataPointsGood[i+4].Time.Sub(dataPointsGood[i+3].Time)

		if diff1.Hours()+diff2.Hours()+diff3.Hours()+diff4.Hours() > 4 {
			i++
			continue
		} else {
			fmt.Println("Good for astrophoto staring from:", dataPointsGood[i])
			dataPointsStart = append(dataPointsStart, dataPointsGood[i])
			i = i + 4
		}
	}
}
