package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/util"

	"github.com/open-policy-agent/opa/metrics"
)

type result struct {
	Total int64
}

func run(i int, ch chan<- result) {
	client := &http.Client{}
	for {
		func() {
			t0 := time.Now()
			resp, err := client.Get("http://bob:password@192.168.64.41:30949/people")
			if err != nil {
				panic(err)
			}

			defer util.Close(resp)
			if resp.StatusCode != 200 {
				panic(err)
			}
			var r result
			r.Total = int64(time.Since(t0))
			ch <- r
		}()
	}
}

func printHeader(keys []string) {
	for i := range keys {
		fmt.Printf("%-14s ", keys[i])
	}
	fmt.Print("\n")
	for i := range keys {
		fmt.Printf("%-14s ", strings.Repeat("-", len(keys[i])))
	}
	fmt.Print("\n")
}

func printRow(keys []string, row map[string]interface{}) {
	for _, k := range keys {
		fmt.Printf("%-14v ", row[k])
	}
	fmt.Print("\n")
}

func main() {

	monitor := make(chan result)

	metricKeys := []string{
		"rps",
		"cli(min)",
		"cli(mean)",
		"cli(median)",
		"cli(75%)",
		"cli(90%)",
		"cli(99%)",
		"cli(99.9%)",
		"cli(99.99%)",
	}

	printHeader(metricKeys)

	go func() {
		delay := time.Second * 10
		ticker := time.NewTicker(delay)
		var n int64
		m := metrics.New()
		tLast := time.Now()
		for {
			select {
			case <-ticker.C:

				now := time.Now()
				dt := int64(now.Sub(tLast))
				rps := int64((float64(n) / float64(dt)) * 1e9)

				row := map[string]interface{}{
					"rps": rps,
				}

				hists := []string{"cli"}

				for _, h := range hists {
					hist := m.Histogram(h).Value().(map[string]interface{})

					keys := []string{"min", "mean", "median", "75%", "90%", "99%", "99.9%", "99.99%"}
					for i := range keys {
						switch x := hist[keys[i]].(type) {
						case int64:
							row[fmt.Sprintf("%v(%v)", h, keys[i])] = time.Duration(x)
						case float64:
							row[fmt.Sprintf("%v(%v)", h, keys[i])] = time.Duration(x)
						default:
							panic("bad type")
						}
					}
				}

				printRow(metricKeys, row)

				tLast = now
				n = 0
				m = metrics.New()

			case r := <-monitor:
				m.Histogram("cli").Update(r.Total)
				n++
			}
		}
	}()

	for i := 0; i < 1; i++ {
		go run(i, monitor)
	}

	eof := make(chan struct{})
	<-eof
}
