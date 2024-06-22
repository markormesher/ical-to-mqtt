package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type EventProperty struct {
	Key        string
	Value      string
	Parameters map[string]string
}

func (p *EventProperty) GetValueOrDefault(def string) string {
	if p == nil {
		return def
	} else {
		return p.Value
	}
}

func (p *EventProperty) ParseParameters() {
	if p == nil {
		return
	}

	if p.Parameters != nil {
		// already done
		return
	}

	p.Parameters = make(map[string]string)

	rawParams := strings.Split(p.Key, ";")
	for _, param := range rawParams[1:] {
		paramParts := strings.SplitN(param, "=", 2)
		p.Parameters[paramParts[0]] = paramParts[1]
	}
}

func (p *EventProperty) ParseAsDate() (*time.Time, error) {
	if p == nil {
		return nil, nil
	}

	var err error

	p.ParseParameters()

	timeLoc := time.Local
	if val, ok := p.Parameters["TZID"]; ok {
		timeLoc, err = time.LoadLocation(val)
		if err != nil {
			return nil, err
		}
	}

	if p.Parameters["VALUE"] == "DATE" {
		// format should be YYYYMMDD
		t, err := time.ParseInLocation("20060102", p.Value, timeLoc)
		if err != nil {
			return nil, err
		} else {
			return &t, nil
		}
	} else {
		// format should be YYYYMMDDTHHMMSS(Z)?
		var t time.Time
		if strings.HasSuffix(p.Value, "Z") {
			t, err = time.ParseInLocation("20060102T150405Z", p.Value, time.UTC)
		} else {
			t, err = time.ParseInLocation("20060102T150405", p.Value, timeLoc)
		}

		if err != nil {
			return nil, err
		} else {
			return &t, nil
		}
	}
}

type Event struct {
	Properties []EventProperty
}

func (e *Event) GetProperties(key string) []EventProperty {
	props := make([]EventProperty, 0)
	for _, p := range e.Properties {
		if p.Key == key || strings.HasPrefix(p.Key, key+";") {
			props = append(props, p)
		}
	}
	return props
}

func (e *Event) GetSingleProperty(key string) *EventProperty {
	props := e.GetProperties(key)

	if len(props) == 0 {
		return nil
	}

	if len(props) > 1 {
		l.Warn("Requested single property but multiple properties exist - returning on the first one", "key", key)
	}

	return &props[0]
}

func parseCalendar(reader io.Reader) ([]Event, error) {
	// first, unfold all the lines

	lines := make([]string, 0)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")

		if line[0] == ' ' || line[0] == '\u0020' {
			// this is a continuation of the current line, so append it and do nothing else
			if len(lines) == 0 {
				return nil, fmt.Errorf("Encountered a continuation line as the first element")
			}
			lines[len(lines)-1] += line[1:]
		} else {
			lines = append(lines, line)
		}
	}

	// second, convert lines into properties

	allProps := make([]EventProperty, 0)
	for _, line := range lines {
		lineParts := strings.SplitN(line, ":", 2)
		if len(lineParts) == 1 {
			allProps = append(allProps, EventProperty{
				Key:   lineParts[0],
				Value: "",
			})
		} else {
			allProps = append(allProps, EventProperty{
				Key:   lineParts[0],
				Value: lineParts[1],
			})
		}
	}

	// finally, condense properties into events

	events := make([]Event, 0)
	var currEvent *Event

	for _, prop := range allProps {
		switch {
		case prop.Key == "BEGIN" && prop.Value == "VEVENT":
			if currEvent != nil {
				l.Warn("Saw event begin while another event was still open - discarding the previous event")
			}
			currEvent = &Event{}

		case prop.Key == "END" && prop.Value == "VEVENT":
			if currEvent == nil {
				l.Warn("Saw event end while no event was open - ignoring this")
			} else {
				events = append(events, *currEvent)
				currEvent = nil
			}

		case currEvent == nil:
			l.Warn("Discarding property seen while no event was open", "key", prop.Key, "value", prop.Value)

		default:
			currEvent.Properties = append(currEvent.Properties, prop)
		}
	}

	return events, nil
}
