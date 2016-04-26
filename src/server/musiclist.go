package main

import (
	"fmt"
	"os"
	"net"
	"sync"
 	"bufio"
	"strings"
	"io"
	"math"
)

const (
	DIRECTORY = "./test/"
)

type MusicList struct {
	name string
	orderList map[int]string // Key: Position, Value: FileName
	fileList map[string]bool // Key: FileName value: Position
	NumFiles int
	lock *sync.Mutex
}

func checkFileExist(fileName string) bool{
	fileName = "./test/" + fileName
	if _, err := os.Stat(fileName); err == nil{
		return true
	}
	return false
}

func (this *MusicList) NewInstance(){
	this.name = ""
	this.lock = new(sync.Mutex)
	this.orderList = make(map[int]string)
	this.fileList = make(map[string]bool)
	this.NumFiles = 0
}

func hashCode(fileName string) int{
	result := 0
	for i, r := range fileName {
		result += i * int(r)
	}
	return result
}

func (this *MusicList) selectServer(fileName string, candidates []Server) []Server{
	/* Global value for servers in server.go */ 
	hcode := hashCode(fileName) % len(candidates)
	numServers := int(math.Sqrt(float64(hcode)))
	fservers := make([]Server, numServers)
	i := 0
	for {
		if i == numServers {
			break
		}
		fservers[i] = candidates[hcode]
		i += 1
		hcode += 1
		if hcode == len(candidates){
			hcode = 0
		}
	}
	return fservers
}

func (this *MusicList) add(fileName string) {
	this.lock.Lock()
	this.fileList[fileName] = true
	//this.orderList[this.NumFiles] = fileName
	this.NumFiles=this.NumFiles + 1
	//fmt.Println()
	this.lock.Unlock()
}

// func (this *MusicList) Upload(fileName string){

// }

func (this *MusicList) Update(fileName string, position int){
	
}

func (this *MusicList) Delete(fileName string){
	
}

func (this *MusicList) delete(fileName string){
	this.lock.Lock()
	delete(this.fileList, fileName)
	this.NumFiles=this.NumFiles - 1
	this.lock.Unlock()
}

func (this *MusicList) request(fileName string, addr string) bool{
	// Trying to get file from addr
	conn, err  := net.Dial("tcp", addr)
	if err != nil {
		conn.Close()
		fmt.Println("Unable to connect server")
		return false
	}
	fmt.Println("Request file from servers ....")

	// TODO: Wrap it into a message
	conn.Write([]byte("get\n"))
	// send request file name
	conn.Write([]byte(fileName + "\n"))
	// fmt.Fprintf(conn, fileName)
	
	msg, _ := bufio.NewReader(conn).ReadString('\n')
	// if server doesn't have that file || client isn't in the group
	if strings.Compare(msg, "success\n") != 0{
		fmt.Println("<ERROR> ", msg)
		conn.Close()
		return false
	}

	var receivedBytes int64
	// reader := bufio.NewReader(conn)
	f, err := os.Create(DIRECTORY + fileName)
	defer f.Close()
	if err != nil {
		conn.Close()
	    fmt.Println("Error creating file")
	    return false
	}
	receivedBytes, err = io.Copy(f, conn)
	conn.Close()
	if err != nil {
	    panic("Transmission error")
	    return false
	}

	fmt.Printf("Finished transferring file. Received: %d \n", receivedBytes)
	return true
}

func getMusicList(groupName string) *MusicList{
	//groupName = "pop"
	fmt.Println("getMusicList(): groupName/ ", groupName)
	for i:= range musicList {
		if musicList[i].name == groupName {
			return &musicList[i]
		}
	}
	return nil
}


/*func main(){
	mList := new(MusicList)
	mList.NewInstance()
	serverList := []string{"127.0.0.1:9999"}
	mList.Add("belief.mp3", serverList)
}*/
