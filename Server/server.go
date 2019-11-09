package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"./vdcs"
)

var pendingGarble = make(map[string]vdcs.CircuitMessage)
var completedGarble = make(map[string]vdcs.GarbledMessage)
var mutexG = sync.RWMutex{}

var pendingEval = make(map[string]vdcs.GarbledMessage)
var completedEval = make(map[string]vdcs.ResEval)
var mutexE = sync.RWMutex{}

var wg = sync.WaitGroup{}

func main() {
	wg.Add(2)
	go serverG()
	go serverE()
	wg.Wait()
}

func garbleCircuit(ID string) {

	mutexG.Lock()
	completedGarble[ID] = vdcs.Garble(pendingGarble[ID])
	//fmt.Println("\n\n\nHere is a completed Garble: ", completedGarble[ID], "\n\n\n")
	mutexG.Unlock()

	mutexG.Lock()
	delete(pendingGarble, ID)
	mutexG.Unlock()
}

func postHandlerG(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var x vdcs.CircuitMessage
		jsn, err := ioutil.ReadAll(r.Body)

		if err != nil {
			log.Fatal("Error reading", err)
		}

		err = json.Unmarshal(jsn, &x)

		if err != nil {
			log.Fatal("bad decode", err)
		}
		fmt.Println("CID:", x.CID)
		mutexG.Lock()
		pendingGarble[x.CID] = x
		mutexG.Unlock()
		fmt.Println(x)
		go garbleCircuit(x.CID)
	}
}

func getHandlerG(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var x vdcs.ComID
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		mutexG.RLock()
		for _, ok := pendingGarble[x.CID]; ok && (len(pendingGarble) != 0); {
			mutexG.RUnlock()
			time.Sleep(10 * time.Microsecond)
			mutexG.RLock()
			if _, oke := completedGarble[x.CID]; oke {
				break
			}
			fmt.Println("Trapped in Here!!")
		}
		mutexG.RUnlock()

		mutexG.RLock()
		value, ok := completedGarble[x.CID]
		mutexG.RUnlock()
		if ok {
			responseJSON, err := json.Marshal(value)
			if err != nil {
				fmt.Fprintf(w, "error %s", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseJSON)
			mutexG.Lock()
			delete(completedGarble, x.CID)
			mutexG.Unlock()
		}
	}
}

func evalCircuit(ID string) {

	mutexE.Lock()
	fmt.Println("Pending Eval before send: ", pendingEval[ID])
	completedEval[ID] = vdcs.Evaluate(pendingEval[ID])
	fmt.Println("Completed Eval before send: ", completedEval[ID])
	mutexE.Unlock()

	mutexE.Lock()
	delete(pendingEval, ID)
	mutexE.Unlock()
}

func postHandlerE(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var x vdcs.GarbledMessage
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		mutexE.Lock()
		pendingEval[x.CID] = x
		fmt.Println("Pending Evaluation: ", pendingEval[x.CID])
		mutexE.Unlock()
		go evalCircuit(x.CID)
	}
}

func getHandlerE(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var x vdcs.ComID
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		mutexE.RLock()
		for _, ok := pendingEval[x.CID]; ok && (len(pendingEval) != 0); {
			mutexE.RUnlock()
			time.Sleep(10 * time.Microsecond)
			mutexE.RLock()
			if _, oke := completedEval[x.CID]; oke {
				break
			}
			fmt.Println("Trapped In Here!")
		}
		mutexE.RUnlock()

		mutexE.RLock()
		value, ok := completedEval[x.CID]
		fmt.Println("Completed Execution: ", completedEval[x.CID])
		mutexE.RUnlock()

		if ok {
			responseJSON, err := json.Marshal(value)
			if err != nil {
				fmt.Fprintf(w, "error %s", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseJSON)
			mutexE.Lock()
			delete(completedEval, x.CID)
			mutexE.Unlock()
		}
	}
}

func serverG() {
	http.HandleFunc("/post", postHandlerG)
	http.HandleFunc("/get", getHandlerG)
	http.ListenAndServe(":8080", nil)
	wg.Done()
}

func serverE() {
	http.HandleFunc("/post", postHandlerE)
	http.HandleFunc("/get", getHandlerE)
	http.ListenAndServe(":8081", nil)
	wg.Done()
}
