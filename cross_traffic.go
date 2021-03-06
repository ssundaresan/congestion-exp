package main

import (
	"math/rand"
	"net/http"
	"sort"
	"io/ioutil"
	"time"
)

type CrossTrafficComponent struct {
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	NextEvent int64
}

type CrossTrafficComponentArr []CrossTrafficComponent

type CrossTrafficGenerator struct {
	Duration               int64
	Targets                []string
	CrossTrafficComponents CrossTrafficComponentArr
	Start                  int64
	End                    int64
	Done                   chan int64
	CounterStart					 int64
	CounterEnd  					 int64
	CounterBytes					 int64
}

func (ctg *CrossTrafficGenerator) NewCrossTrafficGenerator(duration int64, targets []string, ctc CrossTrafficComponentArr, done chan int64) {
	ctg.Targets = targets
	ctg.Duration = duration
	ctg.CrossTrafficComponents = ctc
	ctg.Done = done
}

func (ctg *CrossTrafficGenerator) Fetch(curr CrossTrafficComponent, fetchChan chan int64, eventCounter int64) {
	//fmt.Println(ctg.Target, curr.Name, curr.NextEvent, eventCounter)
	tIndex := rand.Int31n(int32(len(ctg.Targets)))
	target := ctg.Targets[tIndex] + "/" + curr.Name
	ctg.CounterStart++
	resp, err := http.Get(target)
	if err != nil {
		//fmt.Println("error", curr.Name, eventCounter)
		fetchChan <- eventCounter
		return
	}
	defer resp.Body.Close()
	body, _:= ioutil.ReadAll(resp.Body)
	ctg.CounterEnd++
	ctg.CounterBytes += int64(len(body))
	fetchChan <- eventCounter
}

func (ctg *CrossTrafficGenerator) GenerateNextEvent(rate float64) int64 {
	return int64((rand.ExpFloat64() / rate) * 1e9)
}

func (ctg *CrossTrafficGenerator) InitializeEventArr() {
	ctg.Start = time.Now().UTC().UnixNano()
	ctg.End = ctg.Start + ctg.Duration*1e9
	rand.Seed(ctg.Start)
	for i := range ctg.CrossTrafficComponents {
		if ctg.CrossTrafficComponents[i].NextEvent == 0 {
			ctg.CrossTrafficComponents[i].NextEvent = ctg.Start + ctg.GenerateNextEvent(ctg.CrossTrafficComponents[i].Rate)
		}
	}
	sort.Sort(ctg.CrossTrafficComponents)
}

func (ctg *CrossTrafficGenerator) Run() {
	fetchChan := make(chan int64, 1e6)
	var eventCounter int64 = 0
	//var eventDone int64

	ctg.InitializeEventArr()
	now := time.Now().UTC().UnixNano()

	for ctg.End > now {
		curr := ctg.CrossTrafficComponents[0]
		time.Sleep(time.Duration(curr.NextEvent-now) * time.Nanosecond)
		go ctg.Fetch(curr, fetchChan, eventCounter)
		eventCounter++
		ctg.CrossTrafficComponents[0].NextEvent = ctg.CrossTrafficComponents[0].NextEvent + ctg.GenerateNextEvent(ctg.CrossTrafficComponents[0].Rate)
		if ctg.CrossTrafficComponents[0].NextEvent > ctg.CrossTrafficComponents[1].NextEvent { //optimization assuming order of magnitude difference in arrival rates
			sort.Sort(ctg.CrossTrafficComponents)
		}
		now = time.Now().UTC().UnixNano()
	}
	//for eventDone < eventCounter {
	//	<-fetchChan
	//	eventDone++
	//}
	ctg.Done <- 1
}
