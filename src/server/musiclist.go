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
	"math/rand"
)

const (
	DIRECTORY = "./test/"
)

type MusicList struct {
	name string
	orderList map[int]string // Key: Position, Value: FileName
	fileList map[string]bool // Key: FileName value: Position
	numFiles int
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
	this.numFiles = 0
}

func hashCode(fileName string) int{
	result := 0
	for i, r := range fileName {
		result += i * int(r)
	}
	return result
}

func (this *MusicList) selectServer(fileName string) []Server{
	/* Global value for servers in server.go */ 
	hcode := hashCode(fileName) % len(servers)
	numServers := int(math.Sqrt(float64(hcode)))
	fservers := make([]Server, numServers)
	i := 0
	for {
		if i == numServers {
			break
		}
		fservers[i] = servers[hcode]
		i += 1
		hcode += 1
		if hcode == len(servers){
			hcode = 0
		}
	}
	return fservers
}

func (this *MusicList) add(fileName string) {
	this.fileList[fileName] = true
	this.numFiles++
}

func (this *MusicList) Add(fileName string, hosts []string){
	this.lock.Lock()
	this.orderList[this.numFiles] = fileName
	this.fileList[fileName] = true
	this.numFiles++

	// TODO: reach agreement among group servers (hosts)

	// if file exists, don't need to request file from other servers
	if checkFileExist(fileName){
		return
	}
	// shuffle the hosts
	fservers := this.selectServer(fileName)
	dest := make([]Server, len(fservers))
	perm := rand.Perm(len(fservers))
	// dest := make([]string, len(hosts))
	// perm := rand.Perm(len(hosts))
	for i, v := range perm {
		dest[v] = fservers[i]
	}
	// Request file from other servers
	for _, addr := range dest{
		if this.request(fileName, addr.combineAddr("comm")){
			fmt.Println("music List: ", this.orderList)
			this.lock.Unlock()
			return
		}
	}
	fmt.Println("No servers contain this file")
	this.numFiles--
	delete(this.orderList, this.numFiles)
	delete(this.fileList, fileName)
	this.lock.Unlock()
}

// func (this *MusicList) Upload(fileName string){

// }

func (this *MusicList) Update(fileName string, position int){
	
}

func (this *MusicList) Delete(fileName string){
	
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

func getMusicList(groupName string) MusicList{
	var list MusicList
	for i:= range musicList {
		if musicList[i].name == groupName {
			list = musicList[i]
		}
	}
	return list
}


/*func main(){
	mList := new(MusicList)
	mList.NewInstance()
	serverList := []string{"127.0.0.1:9999"}
	mList.Add("belief.mp3", serverList)
}*/
