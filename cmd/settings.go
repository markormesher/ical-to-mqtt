package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Settings struct {
	MqttConnectionString  string
	MqttTopicPrefix       string
	UpdateInterval        int
	PublishHistoricEvents bool
	CalendarUrls          []string
}

func getSettings() (*Settings, error) {
	mqttConnectionString := os.Getenv("MQTT_CONNECTION_STRING")
	if len(mqttConnectionString) == 0 {
		mqttConnectionString = "tcp://0.0.0.0:1883"
	}

	mqttTopicPrefix := strings.TrimRight(os.Getenv("MQTT_TOPIC_PREFIX"), "/")
	if len(mqttTopicPrefix) == 0 {
		mqttTopicPrefix = "calendars"
	}

	updateIntervalStr := os.Getenv("UPDATE_INTERVAL")
	if len(updateIntervalStr) == 0 {
		updateIntervalStr = "0"
	}
	updateInterval, err := strconv.Atoi(updateIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("Could not parse update interval as an integer: %w", err)
	}

	publishHistoricEventsStr := os.Getenv("PUBLISH_HISTORIC_EVENTS")
	publishHistoricEvents := publishHistoricEventsStr == "true"

	calendarUrlsRaw := os.Getenv("CALENDAR_URLS")
	calendarUrls := strings.Split(calendarUrlsRaw, ";")

	return &Settings{
		MqttConnectionString:  mqttConnectionString,
		MqttTopicPrefix:       mqttTopicPrefix,
		UpdateInterval:        updateInterval,
		PublishHistoricEvents: publishHistoricEvents,
		CalendarUrls:          calendarUrls,
	}, nil
}
