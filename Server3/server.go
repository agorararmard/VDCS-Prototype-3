package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"os"
	"./vdcs"
)

var gm_pendingEval = make(map[string]vdcs.GarbledMessage)
var in_pendingEval = make(map[string]vdcs.GarbledMessage)

var pendingRepo = make(map[string]bool)

var mutexE = sync.RWMutex{}

var wg = sync.WaitGroup{}

func main() {

	server()
}

func server() {
	initServer()
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/get", getHandler)
	port := ":" + strconv.Itoa(vdcs.MyOwnInfo.PartyInfo.Port)
	print(port)
	print(vdcs.MyOwnInfo.PartyInfo.PublicKey)
	http.ListenAndServe(port, nil)
}

func initServer() {
	//set whatever to the directory
	port, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		log.Fatal("Error reading commandline arguments", err)
	}
	vdcs.SetDirectoryInfo([]byte("127.0.0.1"), int(port))

	//register now
	ServerRegister(300, 2.4)
}

func ServerRegister(numberOfGates int, feePerGate float64) {

	vdcs.SetMyInfo()
	regMsg := vdcs.RegisterationMessage{
		Type: "Server",
		Server: vdcs.ServerInfo{
			PartyInfo: vdcs.MyOwnInfo.PartyInfo,
			ServerCapabilities: vdcs.ServerCapabilities{
				NumberOfGates: numberOfGates,
				FeePerGate:    feePerGate,
			},
		},
	}
	fmt.Println(regMsg)
	for !vdcs.SendToDirectory(regMsg, vdcs.DirctoryInfo.IP, vdcs.DirctoryInfo.Port) {
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I'm solving the token right now!")
	if r.Method == "GET" {
		var x vdcs.Token
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		ret := vdcs.SolveToken(x)
		vdcs.MyToken = ret
		responseJSON, err := json.Marshal(ret)
		if err != nil {
			fmt.Fprintf(w, "error %s", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		//getting the array of messages
		var x vdcs.MessageArray
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}

		//Decryption
		sk := vdcs.RSAPrivateKeyFromBytes(vdcs.MyOwnInfo.PrivateKey)
		k, err := vdcs.RSAPrivateDecrypt(sk, x.Keys[0])
		if err != nil {
			log.Fatal("Error Decrypting the key", err)
		}
		//saving it so I won't have to decrypt it again in each thread
		x.Array[0] = vdcs.DecryptMessageAES(k, x.Array[0])

		//checking the type
		reqType := x.Array[0].Type

		if reqType == "Garble" {
			//the garbling thread
			go garbleLogic(x)

		} else if reqType == "Rerand" {
			//the rerand thread
			go rerandLogic(x)
		} else if reqType == "SEval" {
			//the eval thread
			go evalLogic(x.Array[0], reqType)
		} else if reqType == "CEval" {
			//the thread for the client requesting the result
			go evalLogic(x.Array[0], reqType)
		}

	}
}

//ServerRegister registers a client to directory of service
//should work fine after gouhar fix the issue of naming

func garbleLogic(arr vdcs.MessageArray) {

	//access the first message which is aleardy decrypted in the post handler
	x := arr.Array[0]

	//create the circuit message to garble it
	circM := vdcs.CircuitMessage{
		Circuit: vdcs.Circuit{
			InputGates:  x.Circuit.InputGates,
			MiddleGates: x.Circuit.MiddleGates,
			OutputGates: x.Circuit.OutputGates,
		},
		ComID: vdcs.ComID{
			CID: x.CID,
		},
		Randomness: vdcs.Randomness{
			Rin:       x.Rin,
			Rout:      x.Rout,
			Rgc:       x.Rgc,
			LblLength: x.LblLength,
		},
	}

	//garbling
	gm := vdcs.Garble(circM)

	//the message to be sent
	mess := vdcs.Message{
		GarbledMessage: gm,
		ComID: vdcs.ComID{
			CID: x.CID,
		},
		NextServer: arr.Array[0].NextServer,
	}

	//appending the new message
	arr.Array = append(arr.Array, mess)

	//removing the first one
	arr.Array = append(arr.Array[:0], arr.Array[1:]...)
	//setting the request type for the next one
	if len(arr.Array) > 1 {
		arr.Array[len(arr.Array)-1].Type = "Rerand"
	} else {
		arr.Array[len(arr.Array)-1].Type = "SEval"
	}

	//encrypting the message by generating a new key first then using it
	k := vdcs.RandomSymmKeyGen()
	arr.Array[len(arr.Array)-1] = vdcs.EncryptMessageAES(k, arr.Array[len(arr.Array)-1])

	//encreypting the key used in previous line
	pk := vdcs.RSAPublicKeyFromBytes(mess.NextServer.PublicKey)
	key, err := vdcs.RSAPublicEncrypt(pk, k)
	if err != nil {
		log.Fatal("Error in decrypting", err)
	}
	//appending the new key
	arr.Keys = append(arr.Keys, key)
	//removing the first one
	arr.Keys = append(arr.Keys[:0], arr.Keys[1:]...)

	//send to the next server
	vdcs.SendToServer(arr, mess.NextServer.IP, mess.NextServer.Port)
}

func rerandLogic(arr vdcs.MessageArray) {

	//the first message already decrypted

	//the info for the next server
	// variable-array consistency potential problem
	next := arr.Array[0].NextServer

	//get the last & first Message
	x0 := arr.Array[0]
	x1 := arr.Array[len(arr.Array)-1]

	//decrypting x1
	sk := vdcs.RSAPrivateKeyFromBytes(vdcs.MyOwnInfo.PrivateKey)
	k, err := vdcs.RSAPrivateDecrypt(sk, arr.Keys[len(arr.Keys)-1])
	if err != nil {
		log.Fatal("Error Decrypting the key", err)
	}
	x1 = vdcs.DecryptMessageAES(k, x1)

	// getting the garble message from x1 and the nextserver from x0
	mess := vdcs.Message{
		//from x1
		GarbledMessage: vdcs.GarbledMessage{
			InputWires: x1.GarbledMessage.InputWires,
			GarbledCircuit: vdcs.GarbledCircuit{
				InputGates:  x1.GarbledCircuit.InputGates,
				MiddleGates: x1.GarbledCircuit.MiddleGates,
				OutputGates: x1.GarbledCircuit.OutputGates,
				ComID: vdcs.ComID{
					CID: x1.ComID.CID,
				},
			},
			OutputWires: x1.GarbledMessage.OutputWires,
		},

		//from x0
		NextServer: x0.NextServer,

		//from x0
		Randomness: vdcs.Randomness{
			Rin:       x0.Rin,
			Rout:      x0.Rout,
			Rgc:       x0.Rgc,
			LblLength: x0.LblLength,
		},

		ComID: vdcs.ComID{
			CID: x0.CID,
		},
	}

	/*
		call rerand function on it assume it returns the same (mess) variable
		NOT YET DONE
	*/

	//removing the first one
	arr.Array = append(arr.Array[:0], arr.Array[1:]...)
	//remove the last one
	arr.Array = arr.Array[:len(arr.Array)-1]
	//appending the new message
	arr.Array = append(arr.Array, mess)
	//setting the type of the new message
	if len(arr.Array) > 1 {
		arr.Array[len(arr.Array)-1].Type = "Rerand"
	} else {
		arr.Array[len(arr.Array)-1].Type = "SEval"
	}

	//encrypting the message by generating a new key first then using it
	kn := vdcs.RandomSymmKeyGen()
	arr.Array[len(arr.Array)-1] = vdcs.EncryptMessageAES(kn, arr.Array[len(arr.Array)-1])

	//encreypting the key used in previous line using public key
	pk := vdcs.RSAPublicKeyFromBytes(mess.NextServer.PublicKey)
	key, err := vdcs.RSAPublicEncrypt(pk, kn)
	if err != nil {
		log.Fatal("Error in decrypting", err)
	}

	//removing the first one
	arr.Keys = append(arr.Keys[:0], arr.Keys[1:]...)
	//remove the last one
	arr.Keys = arr.Keys[:len(arr.Keys)-1]
	//appending the new message
	arr.Keys = append(arr.Keys, key)

	//send it to the next server.... (from the first message)
	vdcs.SendToServer(arr, next.IP, next.Port)
}

func evalLogic(mess vdcs.Message, reqType string) {

	//the first one is already decrypted
	gm := vdcs.GarbledMessage{

		InputWires: mess.InputWires,

		OutputWires: mess.OutputWires,

		GarbledCircuit: vdcs.GarbledCircuit{

			InputGates: mess.GarbledCircuit.InputGates,

			OutputGates: mess.GarbledCircuit.OutputGates,

			MiddleGates: mess.GarbledCircuit.MiddleGates,

			ComID: vdcs.ComID{
				CID: mess.ComID.CID,
			},
		},
	}

	//check whether this ComID have any pending wires OR Circuts
	mutexE.Lock()
	if _, ok := pendingRepo[gm.CID]; ok {

		if reqType == "SEval" {

			evalGm := vdcs.GarbledMessage{

				InputWires: in_pendingEval[gm.CID].InputWires,

				OutputWires: gm.OutputWires,

				GarbledCircuit: vdcs.GarbledCircuit{

					InputGates: gm.GarbledCircuit.InputGates,

					OutputGates: gm.GarbledCircuit.OutputGates,

					MiddleGates: gm.GarbledCircuit.MiddleGates,

					ComID: vdcs.ComID{
						CID: gm.CID,
					},
				},
			}

			//remove the pending from the map
			delete(pendingRepo, gm.CID)
			delete(in_pendingEval, gm.CID)
			mutexE.Unlock()

			//send them
			vdcs.MyResult = vdcs.Evaluate(evalGm)
			vdcs.SendToClient(vdcs.MyResult, mess.NextServer.IP, mess.NextServer.Port)

			//if the client send the input wires
		} else {

			evalGm := vdcs.GarbledMessage{
				InputWires:  gm.InputWires,
				OutputWires: gm_pendingEval[gm.CID].OutputWires,
				GarbledCircuit: vdcs.GarbledCircuit{
					InputGates:  gm_pendingEval[gm.CID].InputGates,
					OutputGates: gm_pendingEval[gm.CID].OutputGates,
					MiddleGates: gm_pendingEval[gm.CID].MiddleGates,
					ComID: vdcs.ComID{
						CID: gm_pendingEval[gm.CID].CID,
					},
				},
			}

			delete(pendingRepo, gm.CID)
			delete(gm_pendingEval, gm.CID)
			mutexE.Unlock()

			//send them
			vdcs.MyResult = vdcs.Evaluate(evalGm)
			vdcs.SendToClient(vdcs.MyResult, mess.NextServer.IP, mess.NextServer.Port)

		}

	} else {
		// cid potential problem
		pendingRepo[gm.CID] = true

		if reqType == "SEval" {
			gm_pendingEval[gm.CID] = gm
		} else {
			in_pendingEval[gm.CID] = gm
		}
		mutexE.Unlock()
	}
}
