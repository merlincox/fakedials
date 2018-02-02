package main

import (
	"fmt"
	"net/http"
	"strings"
	"log"
	"flag"
	"io/ioutil"
	"encoding/json"
	"io"
	"time"
	"strconv"
)

type Props struct {
	Path string `json:"path"`
	Key  string `json:"key"`
	Uri  string `json:"uri"`
	Port int `json:"port"`
}

type State struct {
	changed  bool
	decisecs int
}

type App struct {
	state State
	props Props
}

type DialsData map[string]interface{}

const FIELD_NAME = "smp"
const REFRESH_SECS = 60

var app App

func main() {
	getProps()
	miniServer()
}

func getProps() {

	configPath := flag.String("config", "./config.json", "defines path for config.json")
	flag.Parse()

	raw, err := ioutil.ReadFile(*configPath);

	if err != nil {
		log.Fatal("ReadFile failed with error: ", err)
	}

	json.Unmarshal(raw, &app.props)
}

func miniServer() {

	http.HandleFunc(app.props.Uri, func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != app.props.Uri {

			log.Printf("Request for %v not found\n", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		switch r.Method {

		case "POST":
			handlePost(w, r)

		case "GET", "HEAD":
			handleGetOrHead(w, r)

		default:
			errorMsg := fmt.Sprintf("%v HTTP method %v not allowed", http.StatusMethodNotAllowed, r.Method)
			log.Println(errorMsg)
			http.Error(w, errorMsg, http.StatusMethodNotAllowed)
		}
	})

	log.Println("Listening..")

	err := http.ListenAndServe(fmt.Sprintf(":%v", app.props.Port), nil)

	if err != nil {
		log.Fatal("ListenAndServe failed with error: ", err)
	}
}

func handleGetOrHead(w http.ResponseWriter, r *http.Request) {

	log.Printf("Handling %v request\n", r.Method)

	dialsData := getDialsData()

	// NB the library correctly handles HEAD requests by using the body to calculate Content-Length
	// without actually transmitting it

	io.WriteString(w, getHtml(dialsData[app.props.Key].(string)))
}

func handlePost(w http.ResponseWriter, r *http.Request) {

	log.Println("Handling POST request")
	
	dialsData := getDialsData()
	
	r.ParseForm()
	
	newDialValue := r.FormValue(FIELD_NAME)

	log.Printf("Received value: %v\n", newDialValue)

	if dialsData[app.props.Key] != newDialValue {
		writeOut(newDialValue, dialsData)
		tickTime()
	}
	
	http.Redirect(w, r, app.props.Uri, http.StatusSeeOther)
}

// Runs for 60 seconds representing Mozart Controller Dials refresh interval

func tickTime() {
	app.state.changed = true
	app.state.decisecs = REFRESH_SECS * 10
	ticker := time.NewTicker(time.Millisecond * 100)
	go func() {
		for range ticker.C {
			app.state.decisecs--
			if app.state.decisecs <= 0 {
				ticker.Stop()
				app.state.changed = false
			}
		}
	}()
}

func writeOut(dialValue string, dialsData DialsData) {

	dialsData[app.props.Key] = dialValue

	raw, err := json.Marshal(dialsData)

	if err != nil {
		log.Fatal("Cannot parse JSON: ", err)
	}

	log.Printf("Writing: %v\n", stringify(raw))
	
	ioutil.WriteFile(app.props.Path, raw, 0666)
}

func getDialsData() (DialsData) {

	var data interface{}

	raw, err := ioutil.ReadFile(app.props.Path);

	if err != nil {
		log.Fatal("ReadFile failed with error: ", err)
	}

	json.Unmarshal(raw, &data)

	log.Printf("Read: %v\n", stringify(raw))

	dialsData := DialsData(data.(map[string]interface{}))

	value, found := dialsData[app.props.Key]

	if !found {

		log.Fatalf("%v does not contain key %v\n", app.props.Path, app.props.Key)
	}

	if _, isAString := value.(string); !isAString {
		log.Fatalf("value for key %v in %v is not a string\n", app.props.Key, app.props.Path)

	}

	return dialsData
}

func stringify(raw []byte) string {
	return string(raw[:])
}

func getHtml(dialValue string) string {

	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Dials Simulator</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css">
    <link href="data:image/x-icon;base64,{icon}" rel="icon" type="image/x-icon">
</head>
<body>

<div class="container mx-auto p-5 mt-5 w-25 border border-primary rounded bg-light">
    <h3>Dials Simulator</h3>
    <p>SMP status is {value}</p>

    <form method="POST" action="{uri}">
        <div class="form-check">
            <label><input class="form-check-input" type="radio" name="{field}" {enabled} value="enabled">Enabled</label>
        </div>
        <div class="form-check">
            <label><input class="form-check-input" type="radio" name="{field}" {disabled} value="disabled">Disabled</label>
        </div>
        <button type="submit" class="btn btn-default">Submit</button>
    </form>
    <div class="mt-3" style="min-height: 24px">{changed}</div>
</div>
<script>
    "option strict"
    var timerDiv = document.getElementById('dials-timer');
    if (timerDiv) {

        var original = {decisecs} * 100;
        var start = Date.now();
        var secs = Math.floor(original/1000);
        var ticker = setInterval(frame, 100);

        function frame() {

            var remaining = original + start - Date.now();
            if (remaining <= 0) {
                clearInterval(ticker);
                timerDiv.parentNode.removeChild(timerDiv);
            } else {
                var newSecs = Math.floor(remaining/1000)
                if (newSecs !== secs) {
                    secs = newSecs
                    timerDiv.textContent = secs + ' seconds';
                }
            }
        }
    }
</script>

</body>
</html>
`
	enabled, disabled := renderChecked(dialValue)

	replacements := map[string]string{
		"{decisecs}": strconv.Itoa(app.state.decisecs),
		"{changed}": getSecondsDiv(),
		"{icon}": getBase64Icon(),
		"{uri}": app.props.Uri,
		"{enabled}": enabled,
		"{disabled}": disabled,
		"{value}": dialValue,
		"{field}": FIELD_NAME,
	}

	for key, value := range replacements {
		html = strings.Replace(html, key, value, -1)
	}

	return html
}

func renderChecked(value string) (string, string) {

	checked, unchecked := `checked="checked"`, ""

	if value == "enabled" {
		return checked, unchecked
	}

	return unchecked, checked
}

// using an in-line icon prevents an additional browser favicon request for every GET //

func getBase64Icon() string {
	return `iVBORw0KGgoAAAANSUhEUgAAABgAAAAeCAYAAAA2Lt7lAAAAOklEQVRIS
	+3SMQoAAAjDQPv/R9cnZHIyzgHhaNp2Di8+IF2JSGgkkggFMHBFEqEABq5IIhTAwB
	U9IFq9Cnen3UNVJgAAAABJRU5ErkJggg==`
}

//insert a timer into the HTML if the data was changed within REFRESH_SECS

func getSecondsDiv() string {

	if app.state.changed {
		return fmt.Sprintf(
			`<div id="dials-timer">%v seconds</div>`,
			(app.state.decisecs / 10),
		)
	}

	return ""
}
