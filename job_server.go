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

func JobServer(engines []string) {
	common.LogMessage(JOBTAG, "Starting Job Server")

	//start chrome_server
	go chrome_server.RemoteServer()

	//TODO: add cache functionality
	jobs := make(map[string]JobRequest)

	for {
		select {
		case x := <-JobChan:
			switch x.Type {
			case "new":
				common.LogMessage(JOBTAG, fmt.Sprintf("Starting New Job: %s", x.ID))
				//TODO: add cache filter on |x| if there are other topics like it
				jobs[x.ID] = x
				//TODO: async results
				for _, engine := range engines {
					go AsyncGetResults(x.ID, engine, url.QueryEscape(x.Topic), x.Num(len(engines)), x.ThumbSize, JobChan)
				}
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

func AsyncGetResults(job_id string, engine string, topic string, num int, thumb_size string, rec_chan chan JobRequest) {
	fmt.Println(fmt.Sprintf("Getting %s on %s for %s", topic, engine, job_id))
	js := GetJSString(engine)

	res := chrome_server.Search(fmt.Sprintf("https://%s", GetSearchString(engine, topic)), js)

	fmt.Println(fmt.Sprintf("%s for %s %s", res, job_id, engine))
	r := res.([]interface{})

	c := 0
	done_chan := make(chan int)
	// pause_chan := make(chan bool)
	for i, l := range r {
		fmt.Println(fmt.Sprintf("%d: %s %s", i, l, engine))
		if l == nil {
			continue
		}
		if strings.Contains(l.(string), "duckduck") {
			continue
		}
		if c == num {
			break
		}

		l_str := l.(string)
		if strings.Contains(l_str, ":~:text=") {
			l_str = l_str[:strings.Index(l_str, "#")]
		}
		// if i > 0 {
		// 	<-pause_chan
		// }
		go func(i int) {
			x := make(chan interface{})
			go GetWebPage(l_str, thumb_size, x)
			select {
			case loc := <-x:
				loca := loc.(chrome_server.JSEval)
				fmt.Println("update for ", engine, topic)
				rec_chan <- JobRequest{ID: job_id, Type: "update", List: []SearchResult{SearchResult{URL: l_str, Location: loca.Loc, LoadTime: loca.Res.DomComplete - loca.Res.NavigationStart, SpeedIndex: loca.Res.SpeedIndex, FirstPaint: loca.Res.FirstPaint, SearchEngine: engine, Images: loca.Res.Images, Words: loca.Res.Words, Scripts: loca.Res.Scripts}}}
				// pause_chan <- true
				done_chan <- i
			case <-time.Tick(time.Second * 30):
				rec_chan <- JobRequest{ID: job_id, Type: "update", List: []SearchResult{SearchResult{URL: l_str, Location: "null", SearchEngine: engine}}}
				done_chan <- i
				// pause_chan <- true
			}
		}(c)
		c += 1
	}
	for {
		select {
		case z := <-done_chan:
			if z == num {
				return
			}
		}
	}
}
