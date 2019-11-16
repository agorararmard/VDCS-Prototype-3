package main

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"time"
)

//test the directory service
func main() {
	id := 0
	var k ServerInfo
	k.IP = []byte("192.168.1.1")
	k.Port = 3030
	k.PublicKey = []byte("abcsjsjsa")
	k.NumberOfGates = 321
	k.FeePerGate = 256.55
	writeServer(k, &id)
	writeServer(k, &id)
	var k2 ClientInfo
	k2.IP = []byte("192.168.1.1")
	k2.Port = 3030
	k2.PublicKey = []byte("abcsjsjsa")
	writeClient(k2)
	writeClient(k2)
	k2.IP = []byte("192.168.3.1")
	k2.Port = 3030
	k2.PublicKey = []byte("abjsjsa")
	writeClient(k2)

}

//read Directory Service word by word
func read() []string {
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
func validNewServer(lines []string, k ServerInfo) bool {
	for i := 1; i < len(lines); i += 7 {
		if lines[i-1] == "Server" {
			if lines[i] == string(k.IP) {
				return false
			} else if lines[i+2] == string(k.PublicKey) {
				return false
			}
		}
	}
	return true

}

//check if a registered server sends his correct information
func validRegisteredServer(lines []string, k ServerInfo, t Token) bool {
	for i := 1; i < len(lines); i += 7 {
		if lines[i-1] == "Server" && lines[i] == string(k.IP) && lines[i+2] == string(k.PublicKey) && lines[i+5] == string(t.TokenGen) {
			return true
		}

	}
	return false

}

//write new server to the directory service
func writeServer(k ServerInfo, id *int) {
	if validNewServer(read(), k) {
		token := string(k.IP) + strconv.FormatInt(int64(k.Port), 10) + string(k.PublicKey) + strconv.FormatInt(int64(k.NumberOfGates), 10) + strconv.FormatFloat(k.FeePerGate, 'f', 6, 64)
		token = strconv.FormatInt(int64(*id), 10) + generateToken(token)
		*id++
		f, err := os.OpenFile("DirectoryService.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write([]byte("Server " + string(k.IP) + " " + strconv.FormatInt(int64(k.Port), 10) + " " + string(k.PublicKey) + " " + strconv.FormatInt(int64(k.NumberOfGates), 10) + " " + strconv.FormatFloat(k.FeePerGate, 'f', 6, 64) + " " + string(token) + "\n")); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		var t Token
		t.TokenGen = []byte(token)
		//send token to the server
		//GetFromClient(t, k.IP, k.Port)
		println("New Server been registered")

	} else {
		println("Server Has already been registered")
	}
}

//check if new Client is valid or has already been registered
func validNewClient(lines []string, k ClientInfo) bool {
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

//check if a registered Client sends his correct information
func validRegisteredClient(lines []string, k ClientInfo) bool {
	for i := 1; i < len(lines); i += 7 {
		if lines[i-1] == "Client" && lines[i] == string(k.IP) && lines[i+2] == string(k.PublicKey) {
			return true
		}

	}
	return false

}

//write new Client to the directory service
func writeClient(k ClientInfo) {
	if validNewClient(read(), k) {
		f, err := os.OpenFile("DirectoryService.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write([]byte("Client " + string(k.IP) + " " + strconv.FormatInt(int64(k.Port), 10) + " " + string(k.PublicKey) + " NULL NULL NULL\n")); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		println("New Client been registered")

	} else {
		println("Client Has already been registered")
	}
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
