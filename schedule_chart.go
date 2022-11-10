package main

import (
	"fmt"
	"os"
	"time"

	chart "github.com/wcharczuk/go-chart/v2"
)

func createScheduleChart(lights []*Light) {

	// create a map of schedule names and a light index that uses that schedule
	var schedules = make(map[string]int)
	for index, light := range lights {
		schedules[light.Schedule.name] = index
	}

	// create a chart for each schedule
	for scheduleName, lightIndex := range schedules {
		var light = lights[lightIndex]

		xValues := make([]float64, 1440)
		brightnessValues := make([]float64, 1440)
		temperatureValues := make([]float64, 1440)
		i := 0

		// iterate over a full day and capture what values the lights would be given
		for hour := 0; hour < 24; hour++ {
			for min := 0; min < 60; min++ {
				testTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, min, 1, 1, time.Local)
				light.updateInterval(testTime)
				var result = light.Interval.calculateLightStateInInterval(testTime)
				xValues[i] = float64(testTime.Unix())
				brightnessValues[i] = float64(result.Brightness)
				temperatureValues[i] = float64(result.ColorTemperature)
				i++
			}
		}

		// create X Axis ticks for each of the schedule points
		yr, mth, dy := time.Now().Date()
		startOfDay := float64(time.Date(yr, mth, dy, 0, 0, 1, 1, time.Local).Unix())
		ticks := []chart.Tick{
			{Value: startOfDay, Label: "Start of day"},
		}
		for _, v := range light.Schedule.beforeSunrise {
			ticks = append(ticks, chart.Tick{Value: float64(v.Time.Unix()), Label: v.Time.Format("15:00")})
		}
		for _, v := range light.Schedule.afterSunset {
			ticks = append(ticks, chart.Tick{Value: float64(v.Time.Unix()), Label: v.Time.Format("15:00")})
		}
		ticks = append(ticks, chart.Tick{Value: float64(light.Schedule.endOfDay.Unix()), Label: "End of Day"})

		sunrise := float64(light.Schedule.sunrise.Time.Unix())
		sunset := float64(light.Schedule.sunset.Time.Unix())

		// create the chart
		graph := chart.Chart{
			Title: fmt.Sprintf("Kelvin Schedule (%s)", scheduleName),
			XAxis: chart.XAxis{
				Name:  "Time of Day",
				Ticks: ticks,
			},
			YAxis: chart.YAxis{
				Name: "Brightness",
			},
			YAxisSecondary: chart.YAxis{
				Name: "Temperature",
			},
			Series: []chart.Series{
				chart.ContinuousSeries{
					Name:    "Brightness",
					XValues: xValues,
					YValues: brightnessValues,
				},
				chart.ContinuousSeries{
					Name:    "Temperature",
					XValues: xValues,
					YValues: temperatureValues,
					YAxis:   chart.YAxisSecondary,
				},
				chart.AnnotationSeries{
					Annotations: []chart.Value2{
						{XValue: sunrise, YValue: 100, Label: "Sunrise"},
						{XValue: sunset, YValue: 100, Label: "Sunset"},
					},
				},
			},
			Background: chart.Style{
				Padding: chart.Box{
					Top:    48,
					Left:   36,
					Right:  24,
					Bottom: 16,
				},
			},
		}

		// add a legend
		graph.Elements = []chart.Renderable{
			chart.Legend(&graph),
		}

		// write to png file in the kelvin folder
		pngFile, _ := os.Create(fmt.Sprintf("schedule_%s.png", scheduleName))
		graph.Render(chart.PNG, pngFile)
	}
}
