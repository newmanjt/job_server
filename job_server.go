package job_server

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/newmanjt/chrome_server"
	"github.com/newmanjt/common"
	"os"
	"os/exec"
	"time"
)

type Config struct {
	SupportedEngines []string `json:"supported_engines"`
	ComboEngines     []string `json:"combo_engines"`
	Browser          string   `json:"browser"`
	User             string   `json:"user"`
}

const JOBTAG = "JOBSERVER"

var JobChan chan JobRequest
var RespChan chan Job

type Job struct {
	ID      string              `json:"id"`
	JobReq  JobRequest          `json:"req"`
	State   int                 `json:"state"`
	List    []SearchResult      `json:"list"`
	History [][]SearchResult    `json:"history"`
	ReqChan chan []SearchResult `json:"-"`
}

//request for a topic
type SearchResult struct {
	URL          string  `json:"url"`
	SearchEngine string  `json:"engine"`
	Location     string  `json:"location"`
	LoadTime     float64 `json:"loadtime"`
	SpeedIndex   float64 `json:"rum_si"`
	FirstPaint   float64 `json:"rum_fp"`
	Images       float64 `json:"rects"`
	Words        float64 `json:"words"`
	Scripts      float64 `json:"scripts"`
}

type JobRequest struct {
	ID           string `json:"id"`
	Topic        string `json:"topic"`
	SearchEngine string `json:"search_engine"`
	Num          int    `json:"num"`
	Type         string `json:"type"`
	Active       bool   `json:"active"`
	ThumbSize    string `json:"thumb_size"`
}

func JobServer() {
	common.LogMessage(JOBTAG, "Starting Job Server")

	//start chrome_server
	go chrome_server.RemoteServer()

	//TODO: add cache functionality
	for {
		select {
		case x := <-JobChan:
			switch x.Type {
			case "new":
				common.LogMessage(JOBTAG, fmt.Sprintf("Starting New Job: %s", x.ID))
				//TODO: add cache filter on |x| if there are other topics like it
				// jobs[x.ID] = x
				//TODO: async results
			case "update":

			case "request":

			case "get":

			}
		}
	}

}

func (j *Job) New(req JobRequest) {

}

func Setup(browser string, user string) {
	KillBrowser(browser)
	time.Sleep(2 * time.Second)
	OpenBrowser(user, browser)
	time.Sleep(time.Second * 2)
	chrome_server.RemoteChan = make(chan chrome_server.GlobalRequest)
	chrome_server.NewTabChan = make(chan chrome_server.GlobalResponse)
	chrome_server.EvaluateJSChan = make(chan chrome_server.GlobalResponse)
	JobChan = make(chan JobRequest)
}

//******************************
//
//   UTIL Function
//
//
//******************************

//Kill any instance of |Browser|
func KillBrowser(browser string) {
	cmd := exec.Command("sudo", "killall", browser)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}
	if string(out) != "" {
		fmt.Println(string(out))
	}
}

//open an instance of |Browser| as |User|
func OpenBrowser(user string, browser string) {
	if browser == "brave" {
		browser = "brave-browser"
	}
	// cmd := exec.Command("sudo", "-u", user, browser, "--disk-cache-dir=/dev/null", "--disk-cache-size=1", "--media-cache-size=1", "--remote-debugging-port=9222")
	cmd := exec.Command("sudo", "-u", user, "xvfb-run", browser, "--window-size=1000,1000", "--disk-cache-dir=/dev/null", "--disk-cache-size=1", "--media-cache-size=1", "--remote-debugging-port=9222")
	go func() {
		out, err := cmd.CombinedOutput()
		common.CheckError(err)
		if string(out) != "" {
			fmt.Println(string(out))
		}
	}()
	return
}

func ProcessFlags() Config {
	var configFlag = flag.String("c", "consolidated.json", "Config File")
	var usageFlag = flag.Bool("h", false, "Show Usage")

	flag.Parse()

	if *usageFlag {
		fmt.Println("Job Server")
		fmt.Println("\t'-h | display this text'")
		fmt.Println("\t'-c | config file'")
		os.Exit(1)
	}

	return LoadConfig(*configFlag)
}

func LoadConfig(config_file string) Config {
	var cd Config
	body, err := common.LoadFile(config_file)
	common.CheckError(err)
	err = json.Unmarshal(body, &cd)
	common.CheckError(err)
	return cd
}
