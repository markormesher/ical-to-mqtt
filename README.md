# iCal to MQTT

This project fetches events from one or more iCal links (.ics) and publishes them to MQTT.

## Configuration

Configuration is via environment variables:

- `MQTT_CONNECTION_STRING` - MQTT connection string, including protocol, host and port (default: `mqtt://0.0.0.0:1883`).
- `MQTT_TOPIC_PREFIX` - topix prefix (default: `calendars`).
- `UPDATE_INTERVAL` - interval in seconds for updates; if this is <= 0 then the program will run once and exit (default: `0`).
- `PUBLISH_HISTORIC_EVENTS` - set to `true` to publish historic events, `false` to publish current and future events only (default: `false`).
- `CALENDAR_URLS` - semicolon-separated list of iCal URLs to fetch (if the URL happens to include a colon, replace it with `%3B`).

## MQTT Topics

- `${prefix}/_meta/last_seen` - RFC3339 timestamp of when the program most last ran.
- `${prefix}/state/all_events` - JSON array of all events (if enabled).
- `${prefix}/state/today_events` - JSON array of events that include today.
- `${prefix}/state/today_and_future_events` - JSON array of events that include today or are in the future.

Each event object takes the following form:

```jsonc
{
  "uid": "abcd1234", // event ID
  "start": "2023-03-14T00:00:00.000Z", // ISO timestamp
  "end": "2023-03-14T00:00:00.000Z", // ISO timestamp
  "wholeDays": true, // whether the event start/end includes a time component
  "summary": "Happy Pi Day!", // event title
  "description": "...", // event description (optional)
  "location": "...", // event location (optional)
  "calendar": "https://..." // calendar URL
}
```

For whole-day events the start time will usually be set to 00:00:00.000 on the first day of the event, and the end time will be set to 00:00:00.000 on the day _after_ the last day of the event.
