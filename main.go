// main.go: Brainz command.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

// ListenBrainzAPI points to the root of the ListenBrainz REST API.
// https://listenbrainz.readthedocs.io/en/latest/users/api
const ListenBrainzAPI = "https://api.listenbrainz.org/1"

// Track describes a music track
type Track struct {
	Name   string `json:"track_name"`
	Artist string `json:"artist_name"`
}

// Listen describes the Recording of a Track listened at a given ListenedAt time.
type Listen struct {
	Recording  string `json:"recording_msid"`
	Track      Track  `json:"track_metadata"`
	ListenedAt int64  `json:"listened_at"`
}

// Time the Track/Recording was listened to.
func (listen Listen) Time() time.Time {
	return time.Unix(listen.ListenedAt, 0)
}

func (listen Listen) String() string {
	return "<" + listen.Recording + "> " + listen.Track.Artist + " - \"" + listen.Track.Name + "\""
}

// Payload contains a set of Listen's.
type Payload struct {
	Count   int      `json:"count"`
	Latest  int      `json:"latest_listen_ts"`
	Listens []Listen `json:"listens"`
}

// Listens contains a Payload describing a set of Listen's.
type Listens struct {
	Payload Payload `json:"payload"`
}

func deleteListen(listen Listen) bool {
	url := ListenBrainzAPI + "/delete-listen"

	// Create a payload to send in the request
	payload := map[string]string{
		"listened_at":    fmt.Sprintf("%d", listen.ListenedAt),
		"recording_msid": listen.Recording,
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	// Create a new http get request
	req, err := http.NewRequest("post", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		fmt.Println("error creating request:", err)
		return false
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("token %s", os.Getenv("brainz_token")))

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if verbosePrint {
		fmt.Printf("(debug) deletelisten(%s, %s): response status: %s\n",
			listen.Time(), listen.Recording, resp.Status)
	}

	return resp.Status == "200 ok"
}

func getListens(max int64) Listens {
	url := fmt.Sprintf("%s/user/%s/listens?count=1000", ListenBrainzAPI, userName)
	if max > 0 {
		url = fmt.Sprintf("%s&max_ts=%d", url, max)
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return Listens{}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return Listens{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return Listens{}
	}

	var listens Listens

	err = json.Unmarshal(body, &listens)
	if err != nil {
		fmt.Println("Error:", err)
		return Listens{}
	}

	return listens
}

var (
	listListens   bool
	deleteListens bool
	userName      string
	searchPattern string
	verbosePrint  bool
	showUsage     bool
)

func init() {
	flag.BoolVar(&listListens, "l", false, "Print matched listens.")
	flag.BoolVar(&deleteListens, "d", false, "Delete matched listens.")
	flag.BoolVar(&verbosePrint, "v", false, "Debug/verbose output.")
	flag.StringVar(&userName, "u", "", "The user name or login ID.")
	flag.StringVar(&searchPattern, "s", "", "The search pattern.")
	flag.BoolVar(&showUsage, "h", false, "Show usage help.")
}

func usage() {
	fmt.Printf("This program requires the environment variable LISTENBRAINZ_TOKEN to be defined.\n\n")
	fmt.Println("Usage: go run main.go [-ldvh] -u <username> -s <regexp>")
	fmt.Println("   -l: List matched listens.")
	fmt.Println("   -d: Delete matched listens.")
	fmt.Println("   -u: The user name or login ID.")
	fmt.Println("   -s: Search regexp pattern.")
	fmt.Println("   -v: Debug/verbose output.")
	fmt.Println("   -h: Show this help.")
	os.Exit(2)
}

func brainz() {
	var listens Listens = getListens(0)

	for _, listen := range listens.Payload.Listens {
		match, err := regexp.MatchString("(?i)"+searchPattern, listen.String())
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		if match {
			if listListens {
				fmt.Println(listen)
			}
			if deleteListens && !deleteListen(listen) {
				fmt.Printf("Warning: failed deleting listen: %s", listen)
			}
		}
	}
}

func main() {
	flag.Parse()

	if showUsage {
		usage()
	}

	if os.Getenv("LISTENBRAINZ_TOKEN") == "" {
		fmt.Println("Error: please define LISTENBRAINZ_TOKEN.")
		os.Exit(1)
	}

	os.Setenv("BRAINZ_TOKEN", os.Getenv("LISTENBRAINZ_TOKEN"))

	if userName == "" {
		fmt.Println("Error: username is missing.")
		usage()
	}

	if searchPattern == "" {
		fmt.Println("Error: search pattern not provided.")
		usage()
	}

	if !(listListens || deleteListens) {
		fmt.Println("Error: you must provide at least one of the commands -l and/or -d.")
		usage()
	}

	brainz()
}
