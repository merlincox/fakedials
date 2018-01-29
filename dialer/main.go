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

type AppData struct {
	Path string `json:"path"`
	Key  string `json:"key"`
	Uri  string `json:"uri"`
	Port int `json:"port"`
}

type DialsData map[string]interface{}

const FIELD_NAME = "smp"

var appData AppData

func main() {

	getAppData()
	miniServe()
}

func handleGet(w http.ResponseWriter, r *http.Request) {

	log.Println("Handling GET")
	dialValue := getDialValue()
	html := getHtml(dialValue)
	io.WriteString(w, html)
}

func handlePost(w http.ResponseWriter, r *http.Request) {

	log.Println("Handling POST")
	dialValue := getDialValue()
	r.ParseForm()
	newDialValue := r.FormValue(FIELD_NAME)

	log.Printf("Received value: %v\n", newDialValue)

	if dialValue != newDialValue {
		writeOut(newDialValue)
	}
	http.Redirect(w, r, appData.Uri, 301)
}

func writeOut(dialValue string) {

	dialsData, err := getDialsData()

	if err != nil {
		log.Fatal("getDialsData: ", err)
	}

	dialsData[appData.Key] = dialValue

	raw, err := json.Marshal(dialsData)

	if err != nil {
		log.Fatal("Cannot parse JSON: ", err)
	}

	log.Printf("Writing: %v\n", stringify(raw))
	ioutil.WriteFile(appData.Path, raw, 0666)
}

func getDialValue() string {

	dialsData, err := getDialsData()

	if err != nil {
		log.Fatal("getDialsData: ", err)
	}

	if val, ok := dialsData[appData.Key]; ok {

		return val.(string)
	}

	panic(fmt.Sprintf("%v does not contain key %v", appData.Path, appData.Key))
}

func miniServe() {

	http.HandleFunc(appData.Uri, func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case "POST":
			handlePost(w, r)
		case "GET":
			handleGet(w, r)
		default:
			panic(fmt.Sprintf("Cannot handle %v requests!", r.Method))
		}
	})

	err := http.ListenAndServe(fmt.Sprintf(":%v", appData.Port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getDialsData() (DialsData, error) {

	var dialsData DialsData
	var data interface{}

	raw, err := ioutil.ReadFile(appData.Path);

	if err != nil {
		return dialsData, err
	}

	json.Unmarshal(raw, &data)

	log.Printf("Read: %v\n", stringify(raw))

	dialsData = DialsData(data.(map[string]interface{}))

	return dialsData, nil
}

func getAppData() {

	configPath := flag.String("config", "./config.json", "defines path for config.json")
	flag.Parse()

	raw, err := ioutil.ReadFile(*configPath);

	if err != nil {
		log.Fatal("getDialsData: ", err)
	}

	json.Unmarshal(raw, &appData)
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
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.2.0/jquery.min.js"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>
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

        <div class="seconds-txt" style="margin-top:32px;">
               0 seconds
        </div>
    </form>
</div>
<script>
function run() {
  var progressContainer = document.getElementsByClassName('seconds-txt')[0];
  var seconds = 0;
  var timer = setInterval(frame, 1000);
  function frame() {
    seconds++;
    if (seconds > 60) {
      progressContainer.style.visibility = "hidden";
      clearInterval(timer);
    } else {
      progressContainer.textContent = seconds + ' seconds';
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
		"{uri}": appData.Uri,
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
