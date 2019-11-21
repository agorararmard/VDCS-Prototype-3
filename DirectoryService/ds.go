package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/scanner"
	"time"

	"./vdcs"
)

// string here is the string conversion of the byte array of Token.TokenGen
var servers = make(map[string]vdcs.ServerInfo)

var clients map[string]vdcs.ClientInfo

var wg = sync.WaitGroup{}
var id = 0
var mypI vdcs.PartyInfo

//not testedddddddddddddddddddddddddddddddddddddddddddd
func main() {
	var sk []byte
	mypI, sk = vdcs.GetPartyInfo()
	fmt.Println(mypI, " ", sk)
	//ReadDS-> if available: read it, else create it

	createEmptyDS()
	wg.Add(1)
	go server()
	wg.Wait()
}
func server() {
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/get", getHandler)

	fmt.Println("I'm ready to listen on " + ":" + strconv.Itoa(mypI.Port))
	http.ListenAndServe(":"+strconv.Itoa(mypI.Port), nil)
}

//GetServersForACycle
func getServers(lines []string, NumberOfGates int, NumberOfServers int, feePerGate float64) vdcs.CycleMessage {
	counter := 0
	var s vdcs.CycleMessage
	for i := 1; i < len(lines) && counter < NumberOfServers; i += 7 {
		linesi3, err := strconv.ParseInt(lines[i+3], 10, 64)
		linesi4, err := strconv.ParseFloat(lines[i+4], 64)
		if err != nil {
			panic("Cannot Convert this!")
		}
		if lines[i-1] == "Server" && (int(linesi3) >= NumberOfGates) && (linesi4 <= feePerGate) {
			s.ServersCycle[counter].IP = []byte(lines[i])
			s.ServersCycle[counter].Port, _ = strconv.Atoi(lines[i+1])
			s.ServersCycle[counter].PublicKey = []byte(lines[i+2])
			counter++
		}
	}
	return s
}

//PostHandler
func postHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I was invoked!")
	fmt.Println(r.Method)
	if r.Method == "POST" {
		var x vdcs.RegisterationMessage
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		go writeToDS(x, &id)

	}
}

//GetHandler
func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "Get" {
		var x vdcs.CycleRequestMessage
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading", err)
		}
		err = json.Unmarshal(jsn, &x)
		if err != nil {
			log.Fatal("bad decode", err)
		}
		value := getServers(readFromDS(), x.NumberOfGates, x.NumberOfServers, x.FeePerGate)

		responseJSON, err := json.Marshal(value)
		if err != nil {
			fmt.Fprintf(w, "error %s", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	}
}

//read Directory Service word by word
func readFromDS() []string {
	file, err := os.Open("DirectoryService.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanWords)

	var words []string

	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return words
}

//check if new Server is valid or has already been registered
func validNewServer(lines []string, k vdcs.ServerInfo) bool {
	for i := 1; i < len(lines); i += 7 {
		if lines[i-1] == "Server" {
			if lines[i+2] == string(k.PublicKey) {
				return false
			}
		}
	}
	return true

}

//check if a registered User sends his correct information
func validRegisteredUser(lines []string, k vdcs.RegisterationMessage, t vdcs.Token) bool {
	//case Server
	if k.Type == "Server" {
		for i := 1; i < len(lines); i += 7 {
			if lines[i-1] == "Server" && lines[i] == string(k.Server.IP) && lines[i+2] == string(k.Server.PublicKey) && lines[i+5] == string(t.TokenGen) {
				return true
			}

		}
		return false
	}
	if k.Type == "Client" {
		//case Client
		for i := 1; i < len(lines); i += 7 {
			if lines[i-1] == "Client" && lines[i] == string(k.Server.IP) && lines[i+2] == string(k.Server.PublicKey) && lines[i+5] == string(t.TokenGen) {
				return true
			}

		}
		return false

	}
	return false
}

//CreateToken Create a token challenge
func CreateToken(token vdcs.Token, publickey []byte) vdcs.Token {
	fmt.Println(publickey)
	pk := vdcs.RSAPublicKeyFromBytes(publickey)
	fmt.Println("Here is what i'm gonna encrypt isA: " + string(token.TokenGen))
	ans, err := vdcs.RSAPublicEncrypt(pk, token.TokenGen)
	if err != nil {
		panic("Cannot encrypt Token!")
	}
	return vdcs.Token{TokenGen: ans}
}

//Write to Directory Service
func writeToDS(k vdcs.RegisterationMessage, id *int) {
	//write new server to the directory service
	if k.Type == "Server" {
		if validNewServer(readFromDS(), k.Server) {
			//		token := string(k.Server.IP) + strconv.FormatInt(int64(k.Server.Port), 10) + string(k.Server.PublicKey) + strconv.FormatInt(int64(k.Server.NumberOfGates), 10) + strconv.FormatFloat(k.Server.FeePerGate, 'f', 6, 64)
			//		token = strconv.FormatInt(int64(*id), 10) + generateToken(token)
			token := "Here is The Directory Of Service" + strconv.Itoa(rand.Int())
			fmt.Println("Here is your token: " + token)
			var t vdcs.Token
			t.TokenGen = []byte(token)
			fmt.Println("Here is your token as byte array: " + string(t.TokenGen))

			t1 := CreateToken(t, k.Server.PublicKey)
			var success bool = false
			for !success {
				fmt.Println(k.Server.IP)
				fmt.Println(k.Server.Port)
				t2, success := vdcs.GetFromServer(t1, k.Server.IP, k.Server.Port)
				//fmt.Println("My Token: ", string(t.TokenGen))
				//fmt.Println("His Token: ", string(t2.TokenGen))
				//fmt.Println("success: ", success)
				//fmt.Println(bytes.Compare(t2.TokenGen, t.TokenGen) == 0)
				if bytes.Compare(t2.TokenGen, t.TokenGen) == 0 && success {
					*id++
					f, err := os.OpenFile("DirectoryService.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatal(err)
					}
					if _, err := f.Write([]byte("Server " + string(k.Server.IP) + " " + strconv.FormatInt(int64(k.Server.Port), 10) + " " + string(k.Server.PublicKey) + " " + strconv.FormatInt(int64(k.Server.NumberOfGates), 10) + " " + strconv.FormatFloat(k.Server.FeePerGate, 'f', 6, 64) + " " + string(token) + "\n")); err != nil {
						log.Fatal(err)
					}
					if err := f.Close(); err != nil {
						log.Fatal(err)
					}
					break

				}
				println("New Server been registered")
			}

		} else {
			println("Server Has already been registered")
		}
	} else if k.Type == "Client" {
		if validNewClient(readFromDS(), k.Server) {
			//	token := string(k.Server.IP) + strconv.FormatInt(int64(k.Server.Port), 10) + string(k.Server.PublicKey) + strconv.FormatInt(int64(k.Server.NumberOfGates), 10) + strconv.FormatFloat(k.Server.FeePerGate, 'f', 6, 64)
			//	token = strconv.FormatInt(int64(*id), 10) + generateToken(token)
			token := "Here is The Directory Of Service" + strconv.Itoa(rand.Int())

			var t vdcs.Token
			t.TokenGen = []byte(token)
			t1 := CreateToken(t, k.Server.PublicKey)
			var success bool = false
			for !success {
				t2, success := vdcs.GetFromClient(t1, k.Server.IP, k.Server.Port)
				fmt.Println("My Token: ", string(t.TokenGen))
				fmt.Println("His Token: ", string(t2.TokenGen))
				fmt.Println("success: ", success)
				fmt.Println(bytes.Compare(t2.TokenGen, t.TokenGen) == 0)
				if bytes.Compare(t2.TokenGen, t.TokenGen) == 0 && success {
					*id++
					f, err := os.OpenFile("DirectoryService.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatal(err)
					}
					if _, err := f.Write([]byte("Client " + string(k.Server.IP) + " " + strconv.FormatInt(int64(k.Server.Port), 10) + " " + string(k.Server.PublicKey) + " " + strconv.FormatInt(int64(k.Server.NumberOfGates), 10) + " " + strconv.FormatFloat(k.Server.FeePerGate, 'f', 6, 64) + " " + string(token) + "\n")); err != nil {
						log.Fatal(err)
					}
					if err := f.Close(); err != nil {
						log.Fatal(err)
					}
					break
				}
				println("New Client been registered")
			}

		} else {
			println("Client Has already been registered")
		}
	}
}

//check if new Client is valid or has already been registered
func validNewClient(lines []string, k vdcs.ServerInfo) bool {
	for i := 1; i < len(lines); i += 7 {
		if lines[i-1] == "Client" {
			if lines[i] == string(k.IP) {
				return false
			} else if lines[i+2] == string(k.PublicKey) {
				return false
			}
		}
	}
	return true

}

//break string to array of characters
func breakToCharSlice(str string) []string {

	tokens := []rune(str)

	var result []string

	for _, char := range tokens {
		result = append(result, scanner.TokenString(char))
	}

	return result
}

//shuffle array of characters
func shuffle(src []string) []string {
	final := make([]string, len(src))
	rand.Seed(time.Now().UTC().UnixNano())
	perm := rand.Perm(len(src))

	for i, v := range perm {
		final[v] = src[i]
	}
	return final
}

//generate token based on the server information and shuffling this information
func generateToken(str string) string {
	str = strings.Join(shuffle(breakToCharSlice(str)), "")
	return strings.Replace(str, "\"", "", -1)
}

//create Empty DirectorySevice
func createEmptyDS() {

	var file, err = os.Create("DirectoryService.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fmt.Println("File Created Successfully", "DirectoryService.txt")
}
