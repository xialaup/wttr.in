package prometheus

import (
	"fmt"
	"strings"
	"time"

	"github.com/chubin/wttr.in/internal/domain"
)

// Description maps field names to their Prometheus metric name and help text
var Description = map[string][2]string{
	"FeelsLikeC":       {"feels_like_celsius", "Feels like temperature in Celsius"},
	"TempC":            {"temperature_celsius", "Current temperature in Celsius"},
	"humidity":         {"humidity_percent", "Humidity percentage"},
	"precipMM":         {"precipitation_mm", "Precipitation in millimeters"},
	"pressure":         {"pressure_hpa", "Atmospheric pressure in hPa"},
	"visibility":       {"visibility_km", "Visibility in kilometers"},
	"windspeedKmph":    {"wind_speed_kmph", "Wind speed in kilometers per hour"},
	"winddir16Point":   {"wind_direction", "Wind direction in 16-point compass"},
	"weatherDesc":      {"weather_description", "Weather condition description"},
	"observation_time": {"observation_time_minutes", "Minutes since midnight for observation time"},
	"sunrise":          {"sunrise_minutes", "Minutes since midnight for sunrise"},
	"sunset":           {"sunset_minutes", "Minutes since midnight for sunset"},
	"moonrise":         {"moonrise_minutes", "Minutes since midnight for moonrise"},
	"moonset":          {"moonset_minutes", "Minutes since midnight for moonset"},
}

// PrometheusRenderer implements the Renderer interface for Prometheus format output
type PrometheusRenderer struct{}

// NewPrometheusRenderer creates a new instance of PrometheusRenderer
func NewPrometheusRenderer() *PrometheusRenderer {
	return &PrometheusRenderer{}
}

// Render converts weather data into Prometheus format
func (r *PrometheusRenderer) Render(query domain.Query) (domain.RenderOutput, error) {
	var output strings.Builder
	alreadySeen := make(map[string]bool)

	// Render current conditions
	if len(query.Weather.CurrentCondition) > 0 {
		currentData := query.Weather.CurrentCondition[0]
		rendered := r.renderCurrent(currentData, "current", alreadySeen)
		output.WriteString(rendered)
	}

	// Render forecast for next 3 days
	for i := 0; i < 3 && i < len(query.Weather.Weather); i++ {
		dayData := query.Weather.Weather[i]
		rendered := r.renderCurrent(dayData, fmt.Sprintf("%dd", i), alreadySeen)
		output.WriteString(rendered)
	}

	return domain.RenderOutput{
		Content: []byte(output.String()),
	}, nil
}

// renderCurrent converts data for a specific day or current condition into Prometheus format
func (r *PrometheusRenderer) renderCurrent(data interface{}, forDay string, alreadySeen map[string]bool) string {
	var output []string

	// Handle different types of input data (CurrentCondition or WeatherDay)
	switch d := data.(type) {
	case domain.CurrentCondition:
		for fieldName, val := range Description {
			helpText, metricName := val[0], val[1]

			value := r.extractValueFromCurrentCondition(d, fieldName)
			if value == "" {
				continue
			}

			if fieldName == "observation_time" {
				value = r.convertTimeToMinutes(value)
				if value == "" {
					continue
				}
			}

			description := ""
			if !r.isNumeric(value) {
				description = fmt.Sprintf(`, description="%s"`, value)
				value = "1"
			}

			if !alreadySeen[metricName] {
				output = append(output, fmt.Sprintf("# HELP %s %s", metricName, helpText))
				alreadySeen[metricName] = true
			}

			output = append(output, fmt.Sprintf(`%s{forecast="%s"%s} %s`, metricName, forDay, description, value))
		}
	case domain.WeatherDay:
		for fieldName, val := range Description {
			helpText, metricName := val[0], val[1]

			value := r.extractValueFromWeatherDay(d, fieldName)
			if value == "" {
				continue
			}

			if strings.HasSuffix(fieldName, "rise") || strings.HasSuffix(fieldName, "set") {
				value = r.convertTimeToMinutes(value)
				if value == "" {
					continue
				}
			}

			description := ""
			if !r.isNumeric(value) {
				description = fmt.Sprintf(`, description="%s"`, value)
				value = "1"
			}

			if !alreadySeen[metricName] {
				output = append(output, fmt.Sprintf("# HELP %s %s", metricName, helpText))
				alreadySeen[metricName] = true
			}

			output = append(output, fmt.Sprintf(`%s{forecast="%s"%s} %s`, metricName, forDay, description, value))
		}
	}

	return strings.Join(output, "\n") + "\n"
}

// extractValueFromCurrentCondition extracts the value for a field from CurrentCondition
func (r *PrometheusRenderer) extractValueFromCurrentCondition(data domain.CurrentCondition, fieldName string) string {
	switch fieldName {
	case "FeelsLikeC":
		return data.FeelsLikeC
	case "TempC":
		return data.TempC
	case "humidity":
		return data.Humidity
	case "precipMM":
		return data.PrecipMM
	case "pressure":
		return data.Pressure
	case "visibility":
		return data.Visibility
	case "windspeedKmph":
		return data.WindspeedKmph
	case "winddir16Point":
		return data.Winddir16Point
	case "weatherDesc":
		if len(data.WeatherDesc) > 0 {
			return data.WeatherDesc[0].Value
		}
	case "observation_time":
		return data.ObservationTime
	}
	return ""
}

// extractValueFromWeatherDay extracts the value for a field from WeatherDay
func (r *PrometheusRenderer) extractValueFromWeatherDay(data domain.WeatherDay, fieldName string) string {
	switch fieldName {
	case "sunrise", "sunset", "moonrise", "moonset":
		if len(data.Astronomy) > 0 {
			astro := data.Astronomy[0]
			switch fieldName {
			case "sunrise":
				return astro.Sunrise
			case "sunset":
				return astro.Sunset
			case "moonrise":
				return astro.Moonrise
			case "moonset":
				return astro.Moonset
			}
		}
	case "TempC":
		return data.AvgTempC
	case "FeelsLikeC":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].FeelsLikeC
		}
	case "humidity":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].Humidity
		}
	case "precipMM":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].PrecipMM
		}
	case "pressure":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].Pressure
		}
	case "visibility":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].Visibility
		}
	case "windspeedKmph":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].WindspeedKmph
		}
	case "winddir16Point":
		if len(data.Hourly) > 0 {
			return data.Hourly[0].Winddir16Point
		}
	case "weatherDesc":
		if len(data.Hourly) > 0 && len(data.Hourly[0].WeatherDesc) > 0 {
			return data.Hourly[0].WeatherDesc[0].Value
		}
	}
	return ""
}

// convertTimeToMinutes converts a time string to minutes since midnight
func (r *PrometheusRenderer) convertTimeToMinutes(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	// Try different time formats that might appear in the data
	formats := []string{
		"03:04 PM",
		"3:04 PM",
		"03:04 AM",
		"3:04 AM",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, timeStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return ""
	}

	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	duration := t.Sub(midnight)
	return fmt.Sprintf("%d", int(duration.Minutes()))
}

// isNumeric checks if a string can be converted to a float
func (r *PrometheusRenderer) isNumeric(value string) bool {
	if value == "" {
		return false
	}
	_, err := fmt.Sscanf(value, "%f", new(float64))
	return err == nil
}
