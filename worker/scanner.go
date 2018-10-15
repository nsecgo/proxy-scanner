package worker

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

func Scanner(parameter []string) {
	scannerTaskch <- struct{}{}
	defer func() { <-scannerTaskch }()
	scannerTaskKey := time.Now().Format("2006-01-02 15:04") + " : masscan " + strings.Join(parameter, " ")
	cmd := exec.Command("masscan", parameter...)
	// rate:  0.00-kpps, 100.00% done, waiting 0-secs, found=0
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	// Discovered open port 80/tcp on 100.64.243.156
	// Banner on port 80/tcp on 100.64.243.156: [http] HTTP/1.0 403 Forbidden\x0d\x0aConnection: close\x0d\x0aContent-Type: text/html\x0d\x0a\x0d
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var scannerStatus = []interface{}{"", uint(0)}
	ScannerTaskStat.Store(scannerTaskKey, scannerStatus)
	defer ScannerTaskStat.Delete(scannerTaskKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 防止stderr reader不退出
	go func() {
		stderrReader := bufio.NewReader(stderr)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				s, err := stderrReader.ReadString('%')
				scannerStatus[0] = s[strings.LastIndex(s, " ")+1:]
				if scannerStatus[0] == "100.00%" || err != nil {
					return
				}
			}
		}
	}()

	stdoutScanner := bufio.NewScanner(stdout)
	for stdoutScanner.Scan() {
		if line := stdoutScanner.Text(); strings.HasPrefix(line, "Discovered") {
			if s := addrReg.FindStringSubmatch(line); len(s) == 3 {
				waitCheckch <- s[2] + ":" + s[1]
				// 计数器加一
				atomic.AddUint32(&WaitCheckCount, 1)
				if i, ok := scannerStatus[1].(uint); ok {
					scannerStatus[1] = i + 1
				}
			}
		}
	}

	if err := stdoutScanner.Err(); err != nil {
		log.Fatal(err) // check stdout Scanner's err
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
