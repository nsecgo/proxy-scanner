package worker

import (
	"reflect"
	"regexp"
	"sync"
)

var addrReg *regexp.Regexp

var spiderWait sync.WaitGroup
var exportWait sync.WaitGroup

type spider struct {
	ips     sync.Map
	ports   sync.Map
	history sync.Map
	count   uint32
}

var (
	scannerTaskch   = make(chan struct{}, 3)
	ScannerTaskStat sync.Map
)
var (
	inspectorTaskch = make(chan struct{}, 100)
	waitCheckch     = make(chan string, 30000)
	WaitCheckCount  uint32
)
var (
	ServerAddr string
	ServerPort string
)

func Run(crawl bool) {
	addrReg, _ = regexp.Compile(`open port ([0-9]+)/tcp on ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	if crawl {
		spiderV := reflect.ValueOf(&spider{})
		for i := 0; i < spiderV.NumMethod(); i++ {
			go spiderV.Method(i).Call(nil)
		}
	}
	go inspector()
	go regularCheck()
}
