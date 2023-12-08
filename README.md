# noclouds [![Docker Build and Push](https://github.com/shuienko/noclouds/actions/workflows/docker-image.yml/badge.svg)](https://github.com/shuienko/noclouds/actions/workflows/docker-image.yml)

Platform to monitor cloud cover and alert when sky is going to be clear.

## build
```
docker build -t noclouds .
```

## deploy
Simply run as docker container.
```
docker run -it noclouds -e <see environment variables below>
```

### Environment variables
- `MB_API_ENDPOINT` - Meteoblue Cloud API Endpoint. Default: `https://my.meteoblue.com/packages/clouds-1h?`
- `MB_API_KEY` - Meteoblue Cloud API Key. Default: `none`
- `MB_LAT` - Latitude of your chosen location. Default: `none`
- `MB_LON` - Longitude of your chisen location. Default: `none`
- `MB_ALT` - Altitude above the sea level. Default: `none`
- `TG_BOT_TOKEN` - Telegram bot token. Default: `none`
- `MAX_CLOUD_COVER` - Maximum acceptable percentage of clouds per layer. Default: `25`
- `NIGHT_START_HOUR` - All events will be taked into account starting from this hour if the day. Dafault: `22`
- `NIGHT_END_HOUR` - All events will be taken into account before this hour if the day. Dafault: `5`
- `NIGHT_HOURS_STREAK` - Weather will be considered as "good" only if NIGHT_HOURS_STREAK hours in a row weather is good. Default: `4`
- `CRON_EXPRESSION` - Background weather checks will be performed with this schedule. Default: `0 8,12,16,20 * * *`
- `CHAT_ID` - Telegram ChatID where all notifications will be sent. Default: `none`
