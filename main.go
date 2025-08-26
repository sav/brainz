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

// Maximum value of an uint16.
const MaxUint16 int64 = int64(uint16(1<<16 - 1))

// ListenBrainzAPI points to the root of the ListenBrainz REST API.
// https://listenbrainz.readthedocs.io/en/latest/users/api
const ListenBrainzAPI = "https://api.listenbrainz.org/1"

// ItemsPerPage determines how many items to retrieve per request.
// Defaults to the maximum of MAX_ITEMS_PER_GET (1000).
const ItemsPerPage = 1000

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
func (listen Listen) Time() string {
	return time.Unix(listen.ListenedAt, 0).Format(time.RFC3339)
}

func (listen Listen) String() string {
	return "[" + listen.Time() + "] " + listen.Track.Artist + " - \"" + listen.Track.Name + "\""
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

func (listens *Listens) length() int {
	if listens != nil {
		return len(listens.Payload.Listens)
	}
	return 0
}

func log(format string, args ...any) {
	if verbosePrint {
		fmt.Fprintf(os.Stderr, format, args...)
	}
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
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		fmt.Println("error creating request:", err)
		return false
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("token %s", os.Getenv("LISTENBRAINZ_TOKEN")))

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log("deletelisten(%s, %s): response status: %s\n", listen.Time(), listen.Recording, resp.Status)

	return resp.StatusCode == http.StatusOK
}

func lastTimestamp(listens []Listen) int64 {
	return listens[len(listens)-1].ListenedAt
}

func getAllListens() []Listen {
	var listens []Listen
	timestamp := int64(0)
	for {
		page := getListens(timestamp)
		if page.length() == 0 {
			break
		}
		timestamp = lastTimestamp(page.Payload.Listens)
		for _, listen := range page.Payload.Listens {
			if cutOffTime > 0 && listen.ListenedAt < cutOffTime {
				return listens
			}
			listens = append(listens, listen)
			if int64(len(listens)) >= maxCount {
				return listens
			}
		}
	}
	return listens
}

func getListens(max int64) Listens {
	url := fmt.Sprintf("%s/user/%s/listens?count=%d",
		ListenBrainzAPI, userName, ItemsPerPage)

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
	maxCount      int64
	deleteListens bool
	userName      string
	searchPattern string
	verbosePrint  bool
	showUsage     bool
	timeFilter    string
	cutOffTime    int64
)

func init() {
	flag.Int64Var(&maxCount, "c", MaxUint16, "Maxium number of items.")
	flag.BoolVar(&deleteListens, "d", false, "Delete matched listens.")
	flag.BoolVar(&verbosePrint, "v", false, "Debug/verbose output.")
	flag.StringVar(&userName, "u", "", "The user name or login ID.")
	flag.StringVar(&searchPattern, "s", ".+", "The search pattern.")
	flag.BoolVar(&showUsage, "h", false, "Show usage help.")
	flag.StringVar(&timeFilter, "t", "", "Only listens within the range (e.g. 10m, 5h, 1d, 1y).")
}

func usage() {
	flag.Usage()
	os.Exit(2)
}

func parseTimeFilter(input string) (int64, error) {
	if input == "" {
		return 0, nil
	}
	if len(input) < 2 {
		return 0, fmt.Errorf("invalid time filter: %s", input)
	}
	nVal := input[:len(input)-1]
	unit := input[len(input)-1]
	var amount int64
	_, err := fmt.Sscanf(nVal, "%d", &amount)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid duration: %s", input)
	}
	var duration time.Duration
	switch unit {
	case 'm':
		duration = time.Duration(amount) * time.Minute
	case 'h':
		duration = time.Duration(amount) * time.Hour
	case 'd':
		duration = time.Duration(amount) * 24 * time.Hour
	case 'y':
		duration = time.Duration(amount) * 365 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("invalid duration unit: %c", unit)
	}
	cutoff := time.Now().Add(-duration).Unix()
	return cutoff, nil
}

func brainz() {
	var listens []Listen = getAllListens()
	for _, listen := range listens {
		if cutOffTime > 0 && listen.ListenedAt < cutOffTime {
			continue
		}
		match, err := regexp.MatchString("(?i)"+searchPattern, listen.String())
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		if match {
			fmt.Println(listen)
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

	if userName == "" {
		fmt.Println("Error: username is missing.")
		usage()
	}

	if maxCount < 1 {
		fmt.Println("Error: invalid maxCount:", maxCount)
		usage()
	}

	var err error
	cutOffTime, err = parseTimeFilter(timeFilter)
	if err != nil {
		fmt.Println("Error:", err)
		usage()
	}

	brainz()
}
