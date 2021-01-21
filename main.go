package main

import (
	"fmt"
	"html/template"
	"net/http"
	"reflect"

	japp "github.com/go-msvc/japp/msg"
	jcli "github.com/go-msvc/jcli"
	jclihttp "github.com/go-msvc/jcli/http"
	"github.com/go-msvc/logger"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
)

func main() {
	appAddr := "http://localhost:12345/app"
	web := newWeb(appAddr,
		"./tmpl/message.tmpl",
		"./tmpl/prompt.tmpl",
		"./tmpl/choice.tmpl")
	web.Debugf("Starting...")
	webAddr := "localhost:8080"
	http.ListenAndServe(webAddr, web)
}

type web struct {
	http.Handler
	logger.ILogger
	cli             jcli.IClient
	cookieStore     sessions.Store
	cookieName      string
	messageTemplate *template.Template
	promptTemplate  *template.Template
	choiceTemplate  *template.Template
}

func newWeb(appAddr string, mt, pt, ct string) *web {
	r := pat.New()
	cli, _ := jclihttp.New(appAddr)

	msgTmpl, err := template.ParseFiles(mt)
	if err != nil {
		panic(err)
	}
	promptTmpl, err := template.ParseFiles(pt)
	if err != nil {
		panic(err)
	}
	choiceTmpl, err := template.ParseFiles(ct)
	if err != nil {
		panic(err)
	}

	cookieEncryptionKey := []byte("thai3eim9Nahth0jaifu3ea9yooch7ti") //32-bytes long (AES-256)
	web := &web{
		ILogger:         logger.Top().NewLogger("jweb").WithStream(logger.Terminal(logger.LogLevelDebug)),
		Handler:         r,
		cli:             cli,
		cookieStore:     sessions.NewCookieStore(cookieEncryptionKey),
		cookieName:      "japp-cookie",
		messageTemplate: msgTmpl,
		promptTemplate:  promptTmpl,
		choiceTemplate:  choiceTmpl,
	}
	r.Post("/input/{step}", web.input)
	r.Get("/choose/{step}/{option}", web.choose)
	r.Get("/favicon.ico", web.unknown)
	r.Get("/", web.start)
	return web
}

func (web *web) unknown(httpRes http.ResponseWriter, httpReq *http.Request) {
	http.Error(httpRes, "", http.StatusNotFound)
}

func (web *web) start(httpRes http.ResponseWriter, httpReq *http.Request) {
	web.Debugf("HTTP %s %s", httpReq.Method, httpReq.URL.Path)
	cookie, _ := web.cookieStore.Get(httpReq, web.cookieName)
	cookieChanged := false

	web.Debugf("Retrieved cookie values:")
	for n, v := range cookie.Values {
		web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
	}

	clientID, _ := cookie.Values["client-id"].(string)
	startReq := japp.StartRequest{
		ClientID: clientID, //may be "" then will get unique client id in response
	}

	cliRes, err := web.cli.Call("start", startReq, reflect.TypeOf(japp.StartResponse{}))
	if err != nil {
		//todo: error page formatting...
		http.Error(httpRes, fmt.Sprintf("failed to start session: %+v", err), http.StatusUnauthorized)
		return
	}

	startRes := cliRes.(japp.StartResponse)
	cookie.Values["client-id"] = startRes.ClientID
	cookie.Values["session-id"] = startRes.SessionID
	cookieChanged = true

	if cookieChanged {
		web.Debugf("Saving cookie values:")
		for n, v := range cookie.Values {
			web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
		}
		cookie.Save(httpReq, httpRes) //send updated cookie values to client in response
	}

	//display app start page content
	httpRes.Header().Set("Content-Type", "text/html")
	if err := web.renderHTML(httpRes, startRes.Content.StepID, startRes.Content); err != nil {
		http.Error(httpRes, fmt.Sprintf("<h1>Error</h1><p>Failed to render page: %+v</p>", err), http.StatusInternalServerError)
		return
	}
} //web.start()

func (web web) input(httpRes http.ResponseWriter, httpReq *http.Request) {
	//expect POST /input/<step>
	stepID := httpReq.URL.Query().Get(":step")
	httpReq.ParseForm()
	input := httpReq.Form.Get("input")
	web.Debugf("HTTP %s %s (step=%s, input=%s)", httpReq.Method, httpReq.URL.Path, stepID, input)

	cookie, _ := web.cookieStore.Get(httpReq, web.cookieName)
	cookieChanged := false
	web.Debugf("Retrieved cookie values:")
	for n, v := range cookie.Values {
		web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
	}

	sessionID, _ := cookie.Values["session-id"].(string)

	contReq := japp.ContinueRequest{
		SessionID: sessionID,
		StepID:    stepID,
		Data: map[string]interface{}{
			"input": input,
		},
	}
	cliRes, err := web.cli.Call("cont", contReq, reflect.TypeOf(japp.ContinueResponse{}))
	if err != nil {
		//todo: error page formatting template and back link
		http.Error(httpRes, fmt.Sprintf("failed to continue session: %+v", err), http.StatusUnauthorized)
		return
	}

	contRes := cliRes.(japp.ContinueResponse)

	if cookieChanged {
		web.Debugf("Saving cookie values:")
		for n, v := range cookie.Values {
			web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
		}
		cookie.Save(httpReq, httpRes) //send updated cookie values to client in response
	}

	//display app start page content
	httpRes.Header().Set("Content-Type", "text/html")
	if err := web.renderHTML(httpRes, contRes.Content.StepID, contRes.Content); err != nil {
		http.Error(httpRes, fmt.Sprintf("<h1>Error</h1><p>Failed to render page: %+v</p>", err), http.StatusInternalServerError)
		return
	}
} //web.input()

func (web web) choose(httpRes http.ResponseWriter, httpReq *http.Request) {
	//expect GET /select/<step>/<option>
	stepID := httpReq.URL.Query().Get(":step")
	optionID := httpReq.URL.Query().Get(":option")
	web.Debugf("HTTP %s %s (step=%s,option=%s)", httpReq.Method, httpReq.URL.Path, stepID, optionID)

	cookie, _ := web.cookieStore.Get(httpReq, web.cookieName)
	cookieChanged := false

	web.Debugf("Retrieved cookie values:")
	for n, v := range cookie.Values {
		web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
	}

	sessionID, _ := cookie.Values["session-id"].(string)
	contReq := japp.ContinueRequest{
		SessionID: sessionID,
		StepID:    stepID,
		Data: map[string]interface{}{
			"input": optionID,
		},
	}
	cliRes, err := web.cli.Call("cont", contReq, reflect.TypeOf(japp.ContinueResponse{}))
	if err != nil {
		//todo: error page formatting template and back link
		http.Error(httpRes, fmt.Sprintf("failed to continue session: %+v", err), http.StatusUnauthorized)
		return
	}

	contRes := cliRes.(japp.ContinueResponse)

	if cookieChanged {
		web.Debugf("Saving cookie values:")
		for n, v := range cookie.Values {
			web.Debugf("  Cookie(%s).[%s]=(%T)%+v", web.cookieName, n, v, v)
		}
		cookie.Save(httpReq, httpRes) //send updated cookie values to client in response
	}

	//display app start page content
	httpRes.Header().Set("Content-Type", "text/html")
	if err := web.renderHTML(httpRes, contRes.Content.StepID, contRes.Content); err != nil {
		http.Error(httpRes, fmt.Sprintf("<h1>Error</h1><p>Failed to render page: %+v</p>", err), http.StatusInternalServerError)
		return
	}
} //web.choose()

func (web web) renderHTML(httpRes http.ResponseWriter, stepID string, c japp.Content) error {
	if m := c.Message; m != nil {
		return web.messageTemplate.Execute(httpRes, map[string]interface{}{
			"step_id": stepID,
			"text":    m.Text,
		})
	}
	if p := c.Prompt; p != nil {
		return web.promptTemplate.Execute(httpRes, map[string]interface{}{
			"step_id": stepID,
			"text":    p.Text,
		})
	}
	if c := c.Choice; c != nil {
		data := map[string]interface{}{}
		data["step_id"] = stepID
		data["header"] = c.Header
		options := map[string]interface{}{}
		for _, o := range c.Options {
			options[o.ID] = map[string]interface{}{
				"step_id": stepID, //repeated to use inside template range iterator
				"id":      o.ID,
				"text":    o.Text,
			}
		}
		data["options"] = options
		return web.choiceTemplate.Execute(httpRes, data)
	}
	return nil
} //web.renderHTML()
