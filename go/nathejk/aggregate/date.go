package aggregate

import (
	"log"
	"time"
)

func ReformatDatetimeUTC(datetimeString string, inputFormat, outputFormat string) string {
	if datetimeString == "" {
		return ""
	}
	datetime, err := time.Parse(inputFormat, datetimeString)
	if err != nil {
		panic(err)
	}
	return datetime.UTC().Format(outputFormat)
}

func ReformatDatetime(datetimeString string) string {
	if datetimeString == "" {
		return ""
	}
	layout := "2006-01-02T15:04:05.999999999Z07:00"
	datetime, err := time.Parse(layout, datetimeString)
	if err != nil {
		log.Fatalf("Error parsing %s %#v", datetimeString, err)
	}
	return datetime.UTC().Format("2006-01-02T15:04:05Z07:00")
}
