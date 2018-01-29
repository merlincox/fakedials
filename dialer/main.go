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
)

type AppConfig struct {
	Path string `json:"path"`
	Key  string `json:"key"`
	Uri  string `json:"uri"`
	Port int `json:"port"`
}

type AppState struct {
	changed bool
}

type DialsData map[string]interface{}

const FIELD_NAME = "smp"

var appConfig AppConfig
var appState AppState

func main() {
	getAppConfig()
	miniServer()
}

func getAppConfig() {

	configPath := flag.String("config", "./config.json", "defines path for config.json")
	flag.Parse()

	raw, err := ioutil.ReadFile(*configPath);

	if err != nil {
		log.Fatal("ReadFile failed with error: ", err)
	}

	json.Unmarshal(raw, &appConfig)
}

func miniServer() {

	http.HandleFunc(appConfig.Uri, func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case "POST":
			handlePost(w, r)
		case "GET":
			handleGet(w, r)
		default:
			log.Fatalf("Cannot handle %v requests!", r.Method)
		}
	})

	err := http.ListenAndServe(fmt.Sprintf(":%v", appConfig.Port), nil)

	if err != nil {
		log.Fatal("ListenAndServe failed with error: ", err)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {

	log.Println("Handling GET request")
	
	dialValue := getDialValue()
	
	html := getHtml(dialValue)
	
	io.WriteString(w, html)

	appState.changed = false
}

func handlePost(w http.ResponseWriter, r *http.Request) {

	log.Println("Handling POST request")
	
	dialValue := getDialValue()
	
	r.ParseForm()
	
	newDialValue := r.FormValue(FIELD_NAME)

	log.Printf("Received value: %v\n", newDialValue)

	if dialValue != newDialValue {
		writeOut(newDialValue)
		appState.changed = true
	}
	
	http.Redirect(w, r, appConfig.Uri, 301)
}

func writeOut(dialValue string) {

	dialsData := getDialsData()

	dialsData[appConfig.Key] = dialValue

	raw, err := json.Marshal(dialsData)

	if err != nil {
		log.Fatal("Cannot parse JSON: ", err)
	}

	log.Printf("Writing: %v\n", stringify(raw))
	
	ioutil.WriteFile(appConfig.Path, raw, 0666)
}

func getDialValue() string {

	dialsData := getDialsData()

	if value, found := dialsData[appConfig.Key]; found {

		return value.(string)
	}

	log.Fatalf("%v does not contain key %v", appConfig.Path, appConfig.Key)

	return ""
}

func getDialsData() (DialsData) {

	var data interface{}

	raw, err := ioutil.ReadFile(appConfig.Path);

	if err != nil {
		log.Fatal("ReadFile failed with error: ", err)
	}

	json.Unmarshal(raw, &data)

	log.Printf("Read: %v\n", stringify(raw))

	return DialsData(data.(map[string]interface{}))
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
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
    <link href="data:image/x-icon;base64,{icon}" rel="icon" type="image/x-icon">
</head>
<body>

<div class="container">
    <h3>Dials Simulator</h3>
    <p>SMP status is {value}</p>

    <form method="POST" action="{uri}">
        <div class="radio">
            <label><input type="radio" name="{field}" {enabled} value="enabled">Enabled</label>
        </div>
        <div class="radio">
            <label><input type="radio" name="{field}" {disabled} value="disabled">Disabled</label>
        </div>
        <button type="submit" class="btn btn-default">Submit</button>
        {changed}
    </form>
</div>
<script>
    function run() {
        var secondsDiv = document.getElementById('seconds-txt');
        if (secondsDiv) {
            var seconds = 0;
            var timer = setInterval(frame, 1000);
            function frame() {
                seconds++;
                if (seconds > 60) {
                    secondsDiv.style.visibility = "hidden";
                    clearInterval(timer);
                } else {
                    secondsDiv.textContent = seconds + ' seconds';
                }
            }
        }
    }
    run();
</script>

</body>
</html>
`
	enabled, disabled := getChecked(dialValue)

	replacements := map[string]string{
		"{changed}": getSecondsDiv(),
		"{icon}": getBase64Icon(),
		"{uri}": appConfig.Uri,
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

func getChecked(value string) (string, string) {

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


//insert a timer if the data was changed in the last call

func getSecondsDiv() string {

	if appState.changed {
		return `<div id="seconds-txt" style="margin-top:32px;">0 seconds</div>`
	}

	return ""
}
