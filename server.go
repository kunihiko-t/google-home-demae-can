package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/k0kubun/pp"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var port *int

type Site struct {
	Name string
	URL  string
	Time *int
}

type IndexHandler struct {
}

type DialogFlowRequest struct {
	ID              string `json:"id"`
	Lang            string `json:"lang"`
	OriginalRequest struct {
		Data struct {
			AvailableSurfaces []struct {
				Capabilities []struct {
					Name string `json:"name"`
				} `json:"capabilities"`
			} `json:"availableSurfaces"`
			Conversation struct {
				ConversationID    string `json:"conversationId"`
				ConversationToken string `json:"conversationToken"`
				Type              string `json:"type"`
			} `json:"conversation"`
			Device struct{} `json:"device"`
			Inputs []struct {
				Arguments []struct {
					Name      string `json:"name"`
					RawText   string `json:"rawText"`
					TextValue string `json:"textValue"`
				} `json:"arguments"`
				Intent    string `json:"intent"`
				RawInputs []struct {
					InputType string `json:"inputType"`
					Query     string `json:"query"`
				} `json:"rawInputs"`
			} `json:"inputs"`
			IsInSandbox bool `json:"isInSandbox"`
			Surface     struct {
				Capabilities []struct {
					Name string `json:"name"`
				} `json:"capabilities"`
			} `json:"surface"`
			User struct {
				Locale string `json:"locale"`
				UserID string `json:"userId"`
			} `json:"user"`
		} `json:"data"`
		Source  string `json:"source"`
		Version string `json:"version"`
	} `json:"originalRequest"`
	Result struct {
		Action           string `json:"action"`
		ActionIncomplete bool   `json:"actionIncomplete"`
		Contexts         []struct {
			Lifespan   int    `json:"lifespan"`
			Name       string `json:"name"`
			Parameters struct {
				Genre         string `json:"genre"`
				GenreOriginal string `json:"genre.original"`
			} `json:"parameters"`
		} `json:"contexts"`
		Fulfillment struct {
			Messages []struct {
				ID     string `json:"id"`
				Speech string `json:"speech"`
				Type   int    `json:"type"`
			} `json:"messages"`
			Speech string `json:"speech"`
		} `json:"fulfillment"`
		Metadata struct {
			IntentID          string `json:"intentId"`
			IntentName        string `json:"intentName"`
			MatchedParameters []struct {
				DataType string `json:"dataType"`
				IsList   bool   `json:"isList"`
				Name     string `json:"name"`
				Value    string `json:"value"`
			} `json:"matchedParameters"`
			NluResponseTime           int    `json:"nluResponseTime"`
			WebhookForSlotFillingUsed string `json:"webhookForSlotFillingUsed"`
			WebhookUsed               string `json:"webhookUsed"`
		} `json:"metadata"`
		Parameters struct {
			Genre string `json:"genre"`
		} `json:"parameters"`
		ResolvedQuery string  `json:"resolvedQuery"`
		Score         float64 `json:"score"`
		Source        string  `json:"source"`
		Speech        string  `json:"speech"`
	} `json:"result"`
	SessionID string `json:"sessionId"`
	Status    struct {
		Code      int    `json:"code"`
		ErrorType string `json:"errorType"`
	} `json:"status"`
	Timestamp string `json:"timestamp"`
}

type Item struct {
	SimpleResponse Response `json:"simpleResponse"`
}

type Response struct {
	TextToSpeech string `json:"textToSpeech"`
	DisplayText  string `json:"displayText"`
}

// type WebhookResponse struct {
// 	Data struct {
// 		Google struct {
// 			ExpectUserResponse bool          `json:"expectUserResponse"`
// 			IsSsml             bool          `json:"isSsml"`
// 			NoInputPrompts     []interface{} `json:"noInputPrompts"`
// 			RichResponse       struct {
// 				Items       []Item `json:"items"`
// 				Suggestions []struct {
// 					Title string `json:"title"`
// 				} `json:"suggestions"`
// 			} `json:"richResponse"`
// 			SystemIntent struct {
// 				Intent string `json:"intent"`
// 				Data   struct {
// 					Type       string `json:"@type"`
// 					ListSelect struct {
// 						Items []struct {
// 							OptionInfo struct {
// 								Key      string   `json:"key"`
// 								Synonyms []string `json:"synonyms"`
// 							} `json:"optionInfo"`
// 							Title string `json:"title"`
// 						} `json:"items"`
// 					} `json:"listSelect"`
// 				} `json:"data"`
// 			} `json:"systemIntent"`
// 		} `json:"google"`
// 	} `json:"data"`
// }

type Speech struct {
	Speech string `json:"speech"`
}

var siteList []Site

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	decoder := json.NewDecoder(r.Body)
	d := DialogFlowRequest{}
	err := decoder.Decode(&d)

	//Parse Error
	if err != nil {
		w.WriteHeader(500)
		s := fmt.Sprintf("エラーが発生しました。%v", err)
		res := Speech{Speech: s}
		json.NewEncoder(w).Encode(&res)
		pp.Print(err)
		return
	}

	w.WriteHeader(200)

	//Which is the fastest?
	if d.Result.Action == "fastest" {
		fastest := getFastest()
		res := Speech{Speech: fmt.Sprintf("１番早いのは%vで%v分です", fastest.Name, *fastest.Time)}
		pp.Print(res)
		json.NewEncoder(w).Encode(&res)
		return
	}

	//Fetch waiting time by genre
	genre := d.Result.Parameters.Genre
	res := getResultByGenre(genre)
	pp.Println(res)
	json.NewEncoder(w).Encode(&res)

}

func getResultByGenre(genre string) (res Speech) {

	if genre == "" {
		res = Speech{Speech: "もつ鍋はなんふん待ち？　のように聞いてください。"}
		return res
	}

	var result int
	var err error
	for _, s := range siteList {
		if s.Name == genre {
			result, err = checkDeliveryTime(s.URL)
		}
	}
	if err == nil && result == 0 {
		res = Speech{Speech: fmt.Sprintf("%vは登録されていません。", genre)}
		return res
	}

	if err != nil {
		res = Speech{Speech: fmt.Sprintf("%vです。", err.Error())}
	} else {
		res = Speech{Speech: fmt.Sprintf("%vは%v分待ちです。", genre, result)}
	}
	return res
}

func getFastest() *Site {
	wg := &sync.WaitGroup{}
	for i := range siteList {
		wg.Add(1)
		go func(site *Site) {
			defer wg.Done()
			tempT, err := checkDeliveryTime(site.URL)
			if err == nil {
				site.Time = &tempT
			}
		}(&siteList[i])
	}
	wg.Wait()

	var fastest *Site
	for i := range siteList {
		s := &siteList[i]
		if s.Time == nil {
			continue
		}
		if fastest == nil || *s.Time < *fastest.Time {
			fastest = s
		}
	}
	return fastest
}

func checkDeliveryTime(url string) (int, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return 0, err
	}
	body := doc.Find("#topCont03 > article > h4 > span").Text()
	body = strings.TrimRight(body, "分")
	r, err := strconv.Atoi(body)
	if body == "ネット受付時間外" || body == "ネット受付休止中" {
		err = fmt.Errorf(body)
	}
	return r, err
}

func init() {
	siteList = []Site{
		Site{Name: "もつ鍋", URL: "https://demae-can.com/shop/menu/3008468/13102010002"},
		Site{Name: "インドカレー", URL: "https://demae-can.com/shop/menu/3014170/13102010002"},
		Site{Name: "中華", URL: "https://demae-can.com/shop/menu/1005888/13102010002"},
		Site{Name: "ココイチ", URL: "https://demae-can.com/shop/menu/1000609/13102010002"},
		Site{Name: "韓国料理", URL: "https://demae-can.com/shop/menu/3008770/13102010002"},
	}
}

func main() {

	port = flag.Int("p", 9090, "Port number")
	flag.Parse()

	ih := &IndexHandler{}
	http.Handle("/", ih)

	addr := fmt.Sprintf(":%d", *port)
	err := http.ListenAndServe(addr, nil)
	checkError(err)
	log.Println("Start Listening")
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
