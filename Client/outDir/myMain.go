package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"./vdcs"
)

func main() {
	port, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		log.Fatal("Error reading commandline arguments", err)
	}
	vdcs.SetDirectoryInfo([]byte("127.0.0.1"), int(port))

	vdcs.ClientRegister()
	fmt.Println("Registration Complete!")
	go vdcs.ClientHTTP()
	fmt.Println("My IP: ", string(vdcs.MyOwnInfo.IP))
	fmt.Println("My Port: ", vdcs.MyOwnInfo.Port)

	time.Sleep(5 * time.Second)
	_myEqual_string_1_string_1Ch1 := make(chan vdcs.ChannelContainer)
	go vdcs.Comm("myEqual_string_1_string_1", 1, 3, 1, _myEqual_string_1_string_1Ch1)
	time.Sleep(5 * time.Second)
	_myEqual_string_1_string_1Ch0 := make(chan vdcs.ChannelContainer)
	go vdcs.Comm("myEqual_string_1_string_1", 0, 3, 1, _myEqual_string_1_string_1Ch0)

	var i string = "a"
	var j string = "a"
	//VDCS
	if eval0(i, j, 0, _myEqual_string_1_string_1Ch0) == true {
		fmt.Println("i == j")
	} else {
		fmt.Println("i != j")
	}

	var z string = "b"
	//VDCS
	if eval1(i, z, 1, _myEqual_string_1_string_1Ch1) == true {
		fmt.Println("i == z")
	} else {
		fmt.Println("i != z")
	}
}

func eval0(i string, j string, cID int64, chVDCSEvalCircRes <-chan vdcs.ChannelContainer) bool {
	_inWire0 := []byte(i)

	_inWire1 := []byte(j)

	//generate input wires for given inputs
	k := <-chVDCSEvalCircRes
	myInWires := make([]vdcs.Wire, len(_inWire0)*8*2)
	for idxByte := 0; idxByte < len(_inWire0); idxByte++ {
		for idxBit := 0; idxBit < 8; idxBit++ {
			contA := (_inWire0[idxByte] >> idxBit) & 1
			myInWires[(idxBit+idxByte*8)*2] = k.InputWires[(idxBit+idxByte*8)*4+int(contA)]
			contB := (_inWire1[idxByte] >> idxBit) & 1
			myInWires[(idxBit+idxByte*8)*2+1] = k.InputWires[(idxBit+idxByte*8)*4+2+int(contB)]
		}
	}
	/*myInWires := make([]vdcs.Wire, 6)
	  for idxBit := 0; idxBit < 3; idxBit++ {
	  contA := (_inWire0[0] >> idxBit) & 1
	  myInWires[(idxBit)*2] = k.InputWires[(idxBit)*4+int(contA)]
	  contB := (_inWire1[0] >> idxBit) & 1
	  myInWires[(idxBit)*2+1] = k.InputWires[(idxBit)*4+2+int(contB)]
	  }*/
	message := vdcs.Message{
		Type:       []byte("CEval"),
		ComID:      vdcs.ComID{CID: []byte(strconv.FormatInt(cID, 10))},
		InputWires: myInWires,
		NextServer: vdcs.MyOwnInfo.PartyInfo,
	}
	key := vdcs.RandomSymmKeyGen()
	messageEnc := vdcs.EncryptMessageAES(key, message)
	nkey, err := vdcs.RSAPublicEncrypt(vdcs.RSAPublicKeyFromBytes(k.PublicKey), key)
	if err != nil {
		panic("Invalid PublicKey")
	}
	mTmp := make([]vdcs.Message, 1)
	mTmp[0] = messageEnc
	kTmp := make([][]byte, 1)
	kTmp[0] = nkey
	msgArr := vdcs.MessageArray{
		Array: mTmp,
		Keys:  kTmp,
	}
	for ok := vdcs.SendToServer(msgArr, k.IP, k.Port); !ok; {
	}
	var res vdcs.ResEval
	vdcs.ReadyMutex.RLock()
	for vdcs.ReadyFlag {
		vdcs.ReadyMutex.RUnlock()
		vdcs.ReadyMutex.RLock()
	}
	res = vdcs.MyResult
	vdcs.ReadyMutex.RUnlock()
	//validate and decode res
	if bytes.Compare(res.Res[0], k.OutputWires[0].WireLabel) == 0 {
		return false
	} else if bytes.Compare(res.Res[0], k.OutputWires[1].WireLabel) == 0 {
		return true
	} else {
		panic("The server cheated me while evaluating")
	}
}
func eval1(i string, z string, cID int64, chVDCSEvalCircRes <-chan vdcs.ChannelContainer) bool {
	_inWire0 := []byte(i)

	_inWire1 := []byte(z)

	//generate input wires for given inputs
	k := <-chVDCSEvalCircRes
	myInWires := make([]vdcs.Wire, len(_inWire0)*8*2)
	for idxByte := 0; idxByte < len(_inWire0); idxByte++ {
		for idxBit := 0; idxBit < 8; idxBit++ {
			contA := (_inWire0[idxByte] >> idxBit) & 1
			myInWires[(idxBit+idxByte*8)*2] = k.InputWires[(idxBit+idxByte*8)*4+int(contA)]
			contB := (_inWire1[idxByte] >> idxBit) & 1
			myInWires[(idxBit+idxByte*8)*2+1] = k.InputWires[(idxBit+idxByte*8)*4+2+int(contB)]
		}
	}
	/*myInWires := make([]vdcs.Wire, 6)
	  for idxBit := 0; idxBit < 3; idxBit++ {
	  contA := (_inWire0[0] >> idxBit) & 1
	  myInWires[(idxBit)*2] = k.InputWires[(idxBit)*4+int(contA)]
	  contB := (_inWire1[0] >> idxBit) & 1
	  myInWires[(idxBit)*2+1] = k.InputWires[(idxBit)*4+2+int(contB)]
	  }*/
	message := vdcs.Message{
		Type:       []byte("CEval"),
		ComID:      vdcs.ComID{CID: []byte(strconv.FormatInt(cID, 10))},
		InputWires: myInWires,
		NextServer: vdcs.MyOwnInfo.PartyInfo,
	}
	key := vdcs.RandomSymmKeyGen()
	messageEnc := vdcs.EncryptMessageAES(key, message)
	nkey, err := vdcs.RSAPublicEncrypt(vdcs.RSAPublicKeyFromBytes(k.PublicKey), key)
	if err != nil {
		panic("Invalid PublicKey")
	}
	mTmp := make([]vdcs.Message, 1)
	mTmp[0] = messageEnc
	kTmp := make([][]byte, 1)
	kTmp[0] = nkey
	msgArr := vdcs.MessageArray{
		Array: mTmp,
		Keys:  kTmp,
	}
	for ok := vdcs.SendToServer(msgArr, k.IP, k.Port); !ok; {
	}
	var res vdcs.ResEval
	vdcs.ReadyMutex.RLock()
	for vdcs.ReadyFlag {
		vdcs.ReadyMutex.RUnlock()
		vdcs.ReadyMutex.RLock()
	}
	res = vdcs.MyResult
	vdcs.ReadyMutex.RUnlock()
	//validate and decode res
	if bytes.Compare(res.Res[0], k.OutputWires[0].WireLabel) == 0 {
		return false
	} else if bytes.Compare(res.Res[0], k.OutputWires[1].WireLabel) == 0 {
		return true
	} else {
		panic("The server cheated me while evaluating")
	}
}
