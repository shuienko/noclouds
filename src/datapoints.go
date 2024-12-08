package main

import (
	"fmt"
	"noclouds/config"
	"time"
)

type DataPoints []DataPoint

// isGood() returns true if Low, Mid and High clouds percentage is less than maxCloudCover and wind is less than maxWind
func (d DataPoint) isGood(maxCloudCover int64, maxWind float64) bool {
	if d.HighClouds <= maxCloudCover && d.MidClouds <= maxCloudCover && d.LowClouds <= maxCloudCover && d.WindSpeed <= maxWind && d.WindGusts <= maxWind {
		return true
	} else {
		return false
	}
}

// atNight() returns true if time is between nightStart and nightEnd
func (d DataPoint) atNight(nightStart, nightEnd int) bool {
	if d.Time.Hour() >= nightStart || d.Time.Hour() <= nightEnd {
		return true
	} else {
		return false
	}
}

// Good() returns DataPoints which are after "Now", are "Good" and within "Night" defined by in config
func (dp DataPoints) Good(config config.Config) DataPoints {
	good := DataPoints{}
	for _, v := range dp {
		if v.Time.After(time.Now()) && v.isGood(config.MaxCloudCover, config.MaxWind) && v.atNight(config.NightStartsAt, config.NightEndsAt) {
			good = append(good, v)
		}
	}

	return good
}

// onlyStartPoints() returns DataPoints correstonding to the beginning of GoodWeatherWindow sets
// For example, if GoodWeatherWindow = 4 then this algorythm will try to find all 4-hours long sets of points.
// Should be applied to "Good" points.
func (dp DataPoints) onlyStartPoints(goodWeatherWindow int) DataPoints {
	onlyStartPoints := DataPoints{}

	i := 0
	for i < len(dp)-goodWeatherWindow {
		sum := 0
		for j := 1; j <= goodWeatherWindow; j++ {
			diff := dp[i+j].Time.Sub(dp[i+j-1].Time)
			sum += int(diff.Hours())
		}

		// sum > hoursStreak means interval between points is not 1 hour long.
		if sum > goodWeatherWindow {
			i++
			continue
		} else {
			// This allows to exclude 2 sequential 4-hours streaks on the same night.
			if i > 0 && dp[i].Time.Sub(dp[i-1].Time).Hours() == 1 {
				i++
				continue
			}
			onlyStartPoints = append(onlyStartPoints, dp[i])
			i = i + goodWeatherWindow
		}
	}

	return onlyStartPoints
}

// next24H() returns DataPoints within 24 hour range from now
func (dp DataPoints) next24H() DataPoints {
	lessThan24H := DataPoints{}
	now := time.Now()

	for _, point := range dp {
		diff := point.Time.Sub(now)
		if diff.Hours() < 24 {
			lessThan24H = append(lessThan24H, point)
		}
	}

	return lessThan24H
}

// setMoonIllumination() sets MoonIllum value for point in DataPoints
func (dp DataPoints) setMoonIllumination() DataPoints {
	updatedPoints := DataPoints{}

	for _, point := range dp {
		point.MoonIllum = int64(moonIllumination(point.Time))
		updatedPoints = append(updatedPoints, point)
	}

	return updatedPoints
}

// Print() returns Markdown string which represents DataPoints
func (dp DataPoints) Print() string {
	out := ""
	for _, point := range dp {
		out += fmt.Sprintf("%3d%% | %4.1f | %s |%2d %2d %2d\n", point.MoonIllum, point.WindGusts, point.Time.Format("Mon - 02 15h"), point.LowClouds, point.MidClouds, point.HighClouds)
	}

	return out
}

// getAllStartPoints() return all time DataPoints with good weather
func getAllStartPoints(config config.Config) DataPoints {
	data := OpenMeteoAPIResponse{}
	data.Init(config.OpenMeteoApiEndpoint, config.OpenMeteoRequestParams, config.Latitude, config.Lognitude)

	points := data.Points().Good(config).onlyStartPoints(config.GoodWeatherWindow)

	return points
}
