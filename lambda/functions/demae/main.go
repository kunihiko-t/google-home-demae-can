package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/apex/go-apex"
	"strconv"
	"strings"
	"sync"
)

type Speech struct {
	Speech string `json:"speech"`
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

type Site struct {
	Name string
	URL  string
	Time *int
}

var siteList []Site

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
	if result == 0 {
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
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {

		var res Speech
		var d DialogFlowRequest
		if err := json.Unmarshal(event, &d); err != nil {
			s := fmt.Sprintf("エラーが発生しました。%v", err)
			res = Speech{Speech: s}
			return res, err
		}

		//Which is the fastest?
		if d.Result.Action == "fastest" {
			fastest := getFastest()
			res := Speech{Speech: fmt.Sprintf("１番早いのは%vで%v分です", fastest.Name, *fastest.Time)}
			return res, nil
		}

		//Fetch waiting time by genre
		genre := d.Result.Parameters.Genre
		res = getResultByGenre(genre)
		return res, nil
	})
}
