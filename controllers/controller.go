package controllers

import (
	"encoding/json"
	"github.com/nsecgo/proxy-scanner/models"
	"github.com/nsecgo/proxy-scanner/worker"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var statusPage, addPage *template.Template

func init() {
	var err error
	statusPage, err = template.ParseFiles("views/status.html")
	if err != nil {
		log.Fatal(err)
	}
	addPage, err = template.ParseFiles("views/addtoscan.html")
	if err != nil {
		log.Fatal(err)
	}
}

type status struct {
	ScannerTaskStat map[string][]interface{}
	WaitCheckCount  uint32
}

func Header(w http.ResponseWriter, r *http.Request) {
	conn, bufrw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	var proxyMode models.HttpMode
	forward := r.Header.Get("X-Forwarded-For")
	via := r.Header.Get("Via")
	if len(forward) == 0 && len(via) == 0 {
		proxyMode = models.EliteHttp
	} else if strings.Contains(forward, worker.ServerAddr) || strings.Contains(via, worker.ServerAddr) {
		proxyMode = models.TransparentHttp
	} else if strings.Contains(forward, r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]) {
		proxyMode = models.AnonymousHttp
	} else {
		proxyMode = models.DistortingHttp
	}
	j, _ := json.Marshal(r.Header)
	// begin#ProxyMode#end
	mode := strconv.Itoa(int(proxyMode))
	bufrw.Write(append([]byte("HTTP/1.0 200 OK\r\n"+
		"Warning: begin#"+mode+"#end\r\n"+
		"Set-Cookie: m=begin#"+mode+"#end\r\n\r\n"), j...))
	bufrw.Flush()
}
func Status(w http.ResponseWriter, r *http.Request) {

	var s = make(map[string][]interface{})
	worker.ScannerTaskStat.Range(func(key, value interface{}) bool {
		s[key.(string)] = value.([]interface{})
		return true
	})
	err := statusPage.Execute(w, status{ScannerTaskStat: s, WaitCheckCount: worker.WaitCheckCount})
	if err != nil {
		log.Println(err)
	}
}
func AddToScan(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		go worker.Scanner([]string{r.PostFormValue("ips"), "-p", r.PostFormValue("ports"), "--rate", "5000"})
		http.Redirect(w, r, "/status", 302)
		return
	}

	addPage.Execute(w, nil)
}
func Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Coming soon"))
}
