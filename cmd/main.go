package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/teambition/rrule-go"
)

var jsonHandler = slog.NewJSONHandler(os.Stdout, nil)
var l = slog.New(jsonHandler)

var recurrenceLimit = time.Now().AddDate(0, 6, 0)

type PublishEvent struct {
	Uid         string    `json:"uid"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	WholeDays   bool      `json:"wholeDays"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	CalendarUrl string    `json:"calendar"`
}

func fetchEvents(url string) ([]PublishEvent, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to download calendar: %w", err)
	}

	events, err := parseCalendar(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse calendar: %w", err)
	}

	publishEvents := make([]PublishEvent, 0)

	for _, event := range events {
		id := event.GetSingleProperty("UID").Value

		// basic properties
		summary := event.GetSingleProperty("SUMMARY").GetValueOrDefault("")
		description := event.GetSingleProperty("DESCRIPTION").GetValueOrDefault("")
		location := event.GetSingleProperty("LOCATION").GetValueOrDefault("")

		// dates
		startProp := event.GetSingleProperty("DTSTART")
		start, err := startProp.ParseAsDate()
		if err != nil {
			return nil, fmt.Errorf("Failed to parse event start time: %w", err)
		}

		endProp := event.GetSingleProperty("DTEND")
		var end *time.Time
		if endProp == nil {
			end = start
		} else {
			end, err = endProp.ParseAsDate()
			if err != nil {
				return nil, fmt.Errorf("Failed to parse event end time: %w", err)
			}
		}

		startProp.ParseParameters()
		dateType, dateTypeOk := startProp.Parameters["VALUE"]
		wholeDays := dateTypeOk && dateType == "DATE"

		// recurrence
		rruleProp := event.GetSingleProperty("RRULE")
		if rruleProp == nil {
			publishEvent := PublishEvent{
				Uid:         id,
				Start:       *start,
				End:         *end,
				WholeDays:   wholeDays,
				Summary:     summary,
				Description: description,
				Location:    location,
				CalendarUrl: url,
			}
			publishEvents = append(publishEvents, publishEvent)
		} else {
			parsedRrule, err := rrule.StrToRRule(rruleProp.Value)
			if err != nil {
				return nil, fmt.Errorf("Failed to parse recurrence rule: %w", err)
			}

			rruleSet := rrule.Set{}
			rruleSet.DTStart(*start)
			rruleSet.RRule(parsedRrule)

			for _, exDateProp := range event.GetProperties("EXDATE") {
				exDate, err := exDateProp.ParseAsDate()
				if err != nil {
					return nil, fmt.Errorf("Failed to parse exdate as date: %w", err)
				}
				rruleSet.ExDate(*exDate)
			}

			duration := end.Sub(*start)

			for _, rStart := range rruleSet.Between(*start, recurrenceLimit, true) {
				publishEvent := PublishEvent{
					Uid:         id,
					Summary:     summary,
					Description: description,
					Location:    location,
					CalendarUrl: url,
					Start:       rStart,
					End:         rStart.Add(duration),
					WholeDays:   wholeDays,
				}
				publishEvents = append(publishEvents, publishEvent)
			}
		}
	}

	return publishEvents, nil
}

func doUpdate(settings *Settings) {
	now := time.Now()

	mqttClient, err := setupMqttClient(*settings)
	if err != nil {
		l.Error("Failed to setup MQTT client")
		panic(err)
	}

	allEvents := make([]PublishEvent, 0)
	for _, url := range settings.CalendarUrls {
		if url == "" {
			continue
		}

		events, err := fetchEvents(url)
		if err != nil {
			l.Error("Failed to fetch events from calendar", "url", url)
			panic(err)
		}
		for _, e := range events {
			allEvents = append(allEvents, e)
		}
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Start.Before(allEvents[j].Start)
	})

	todayEvents := make([]PublishEvent, 0)
	todayAndFutureEvents := make([]PublishEvent, 0)
	for _, e := range allEvents {
		// at exactly 00:00:00.000 this will think that a whole-day event that ended yesterday is still live, but I can live with that

		if e.Start.Compare(now) <= 0 && e.End.Compare(now) >= 0 {
			todayEvents = append(todayEvents, e)
		}

		if (e.Start.Compare(now) <= 0 && e.End.Compare(now) >= 0) || e.Start.Compare(now) >= 0 {
			todayAndFutureEvents = append(todayAndFutureEvents, e)
		}
	}

	mqttClient.publish("_meta/last_seen", now.Format(time.RFC3339))
	mqttClient.publish("state/today_events", todayEvents)
	mqttClient.publish("state/today_and_future_events", todayAndFutureEvents)
	if settings.PublishHistoricEvents {
		mqttClient.publish("state/all_events", allEvents)
	}
}

func main() {
	settings, err := getSettings()
	if err != nil {
		l.Error("Failed to get settings")
		panic(err)
	}

	if settings.UpdateInterval <= 0 {
		l.Info("Running once then exiting because update interval is <= 0")
		doUpdate(settings)
	} else {
		l.Info("Running forever", "interval", settings.UpdateInterval)
		for {
			doUpdate(settings)
			time.Sleep(time.Duration(settings.UpdateInterval * int(time.Second)))
		}
	}
}
