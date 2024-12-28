# noclouds [![Docker Build and Push](https://github.com/shuienko/noclouds/actions/workflows/docker-image.yml/badge.svg)](https://github.com/shuienko/noclouds/actions/workflows/docker-image.yml)

Platform to monitor cloud cover and alert when sky is going to be clear.

## build
```
docker build -t noclouds .
```

## deploy
Run as a docker container. TG_BOT_TOKEN, CHAT_ID, LAT and LON must be set.
```
docker run -it -e <see environment variables below> noclouds
```

### Environment variables
#### Mandatory
- `LAT` - Latitude of your chosen location. Default: ``
- `LON` - Longitude of your chosen location. Default: ``
- `CHAT_ID` - Telegram ChatID where all notifications will be sent. Default: ``
- `TG_BOT_TOKEN` - Telegram bot token. Default: ``

#### Optional
- `API_ENDPOINT` - Open-Meteo API Endpoint. Default: `https://api.open-meteo.com/v1/forecast?`
- `REQUEST_PARAMS` - Open-Meteo API request parameters. Default: `temperature_2m,cloud_cover_low,cloud_cover_mid,cloud_cover_high,wind_speed_10m,wind_gusts_10m`
- `STATE_FILE_PATH` - Path to text file with the status. Default: `state.txt`
- `MAX_CLOUD_COVER` - Maximum acceptable percentage of clouds per layer. Default: `25`
- `MAX_WIND` - Maximum acceptable wind gusts speed in km/h. Default: `20`
- `NIGHT_STARTS_AT` - All events will be taked into account starting from this hour if the day. Dafault: `22`
- `NIGHT_ENDS_AT` - All events will be taken into account before this hour if the day. Dafault: `5`
- `GOOD_WEATHER_WINDOW` - Weather will be considered as "good" only if GOOD_WEATHER_WINDOW hours in a row weather is good. Default: `4`
- `CRON_EXPRESSION` - Background weather checks will be performed with this schedule. Default: `0 10,20 * * *`
