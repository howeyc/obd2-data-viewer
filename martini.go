package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/staticbin"
)

type Record struct {
	Vin       string
	Timestamp int64
	DateTime  time.Time
	Latitude  float64
	Longitude float64
	Readings  map[string]string

	BarometricPressure     string
	IntakeManifoldPressure string
	EngineRPM              string
	EngineCoolantTemp      string
	AmbientAirTemp         string
	Speed                  string
	ThrottlePos            string
	AirIntakeTemp          string
}

// A valid Value is of the form "<name><unit>" but unit can have spaces
var validValue *regexp.Regexp

func init() {
	validValue = regexp.MustCompile(`^(?P<Number>\d+\.?\d*)(?P<Units>\D*)$`)
}

func main() {
	var serverPort int
	var localhost bool

	flag.IntVar(&serverPort, "port", 8056, "Port to listen on.")
	flag.BoolVar(&localhost, "localhost", false, "Listen on localhost only.")

	flag.Parse()

	var cardata []Record
	var carmutex sync.RWMutex

	m := martini.Classic()

	m.Use(gzip.All())
	m.Use(staticbin.Static("public", Asset))

	carfunc := func(res http.ResponseWriter, r *http.Request) {
		fmt.Println(time.Now())
		decoder := json.NewDecoder(r.Body)
		var rec Record

		err := decoder.Decode(&rec)

		if err != nil {
			log.Println(err)
		}

		recTime := time.Unix(0, 0)
		recTime = recTime.Add(time.Millisecond * time.Duration(rec.Timestamp))
		rec.DateTime = recTime
		fmt.Println(recTime)

		for rName, reading := range rec.Readings {
			if reading != "" && reading != "NO DATA" {
				segs := validValue.FindAllStringSubmatch(reading, -1)
				if len(segs) > 0 && len(segs[0]) > 2 {
					val := segs[0][1]
					//fmt.Println(rName, strings.TrimSpace(segs[0][1]), strings.TrimSpace(segs[0][2]))
					switch rName {
					case "BAROMETRIC_PRESSURE":
						rec.BarometricPressure = val
					case "INTAKE_MANIFOLD_PRESSURE":
						rec.IntakeManifoldPressure = val
					case "ENGINE_RPM":
						rec.EngineRPM = val
					case "ENGINE_COOLANT_TEMP":
						rec.EngineCoolantTemp = val
					case "AMBIENT_AIR_TEMP":
						rec.AmbientAirTemp = val
					case "SPEED":
						rec.Speed = val
					case "THROTTLE_POS":
						rec.ThrottlePos = val
					case "AIR_INTAKE_TEMP":
						rec.AirIntakeTemp = val
					}
				}
			}
		}

		fmt.Println(rec)

		carmutex.Lock()
		cardata = append(cardata, rec)
		carmutex.Unlock()
	}

	m.Put("/", carfunc)
	m.Put("/data", carfunc)
	m.Put("/data/", carfunc)

	carview := func(w http.ResponseWriter, r *http.Request) {
		type DataSet struct {
			FieldName string
			RGBColor  string
			Values    []string
		}
		var chartData struct {
			RangeStart, RangeEnd time.Time
			Labels               []string
			DataSets             []DataSet
		}

		carmutex.RLock()
		chartData.DataSets = []DataSet{
			{FieldName: "Speed (km/h)", RGBColor: "220,220,220", Values: make([]string, len(cardata))},
			{FieldName: "Throttle (%)", RGBColor: "151,187,205", Values: make([]string, len(cardata))},
			{FieldName: "RPM (x100)", RGBColor: "70, 191, 189", Values: make([]string, len(cardata))},
		}
		for rIdx, rec := range cardata {
			if chartData.RangeStart.IsZero() {
				chartData.RangeStart = rec.DateTime
			}
			chartData.RangeEnd = rec.DateTime
			chartData.Labels = append(chartData.Labels, rec.DateTime.Format("15:04:05"))
			chartData.DataSets[0].Values[rIdx] = rec.Speed
			chartData.DataSets[1].Values[rIdx] = rec.ThrottlePos
			rpm, _ := strconv.ParseInt(rec.EngineRPM, 10, 64)
			chartData.DataSets[2].Values[rIdx] = fmt.Sprint(rpm/100)
		}
		carmutex.RUnlock()

		t, err := parseAssets("templates/template.linechart.html", "templates/template.nav.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		err = t.Execute(w, chartData)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	}

	m.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/carview", http.StatusFound)
	})
	m.Get("/carview", carview)

	fmt.Println("Listening on port", serverPort)
	listenAddress := ""
	if localhost {
		listenAddress = fmt.Sprintf("127.0.0.1:%d", serverPort)
	} else {
		listenAddress = fmt.Sprintf(":%d", serverPort)
	}
	http.ListenAndServe(listenAddress, m)
}
