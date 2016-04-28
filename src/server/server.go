package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
	"bufio"
	"net/rpc"
)

var (
	dir          string
	servers      []Server //all servers and port
	masterServer map[string]Server

	myServer         Server           //this server info
	heartBeatTracker = new(HeartBeat) //heart beat related

	musicList  []MusicList         //local groups music list
	hasGroups  map[string]bool     //local groups map
	clusterMap map[string][]Server //key:cluster name, value:cluster's server list
	//clusterMap map[string][]string
	groupMap map[string]string //key: groupName, value: cluster name

	/* New */
	master      Server
	multicaster Mulitcaster
	
	/* Map lock */
	mapLock *sync.Mutex = new(sync.Mutex)
	
	Directory string
)

type Music struct {
	GroupName string
	FilesMap  map[string]string //key: music name, value: music path
}

type Group struct {
	GroupMap map[string]string
}

type Server struct {
	ip             string
	name           string
	comm_port      string
	http_port      string
	heartbeat_port string
	cluster        string
	heartbeatFreq  int
	FilePort       string
	backup_port    string
}

func (s *Server) combineAddr(port string) string {
	switch port {
	case "comm":
		return s.ip + ":" + s.comm_port
	case "http":
		return s.ip + ":" + s.http_port
	case "heartbeat":
		return s.ip + ":" + s.heartbeat_port
	case "File":
		return s.ip + ":" + s.FilePort
	case "backup":
		return s.ip + ":" + s.backup_port
	}
	return ""
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {		
		t, _ := template.ParseFiles("UI/create.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		groupName := r.PostFormValue("groupname")
		fmt.Println("[debug]", groupName)
		if !isGroupNameExist(groupName) {

			createNewGroupLocal(groupName, myServer.cluster) //local
			multicastServers(groupName, "create_group")
			// Send a request to every server to request create new server
			if myServer == master {
				fmt.Println("I'm a master and multicasting a  update to every slave")
				go multicaster.UpdateList(ListContent{groupName, "create", -1, ""})
			} else {
				fmt.Println("I'm a slave and sending a request update to master")
				go multicaster.RequestUpdateList(ListContent{groupName, "create", -1, ""})
			}	
			
			http.Redirect(w, r, "http://"+myServer.combineAddr("http")+"/join.html?"+groupName, http.StatusFound)
		} else {
			w.Write([]byte("Create Group failed, please try another groupname or check servers alive"))
		}
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Println(r.RequestURI)
		if strings.Contains(r.RequestURI,"?") {
			groupName := strings.Split(r.RequestURI,"?")[1]
			fmt.Println("group name", groupName)
			
			mList := getMusicList(groupName)
			
			if mList != nil {
				fmt.Println(mList)
				data := Music{GroupName: groupName}
				data.FilesMap = make(map[string]string)
				for key, _ := range mList.fileList {
					data.FilesMap[key] = Directory + key
				}
				t, _ := template.ParseFiles("UI/upload.html")
				t.Execute(w, data)				
			} else {
				redirectToCorrectServer(groupName, w, r) 
			}
		} else {
			fmt.Println("[debug]get request handler without groupname")		
			data := Group{GroupMap: groupMap}
			t, _ := template.ParseFiles("UI/join.html")
			t.Execute(w, data)
		}
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func redirectToCorrectServer(groupName string, w http.ResponseWriter, r *http.Request) {
	fmt.Println(groupMap[groupName])
	serverList := clusterMap[groupMap[groupName]]
	fmt.Println(serverList)
	for i:= range serverList {
		conn, err := rpc.Dial("tcp", serverList[i].combineAddr("comm"))
		if err != nil {
			continue
		} else {
			fmt.Println("[Debug]redirect", serverList[i].combineAddr("http")+"/join.html?"+groupName)
			conn.Close()
			http.Redirect(w, r, "http://"+serverList[i].combineAddr("http")+"/join.html?"+groupName, http.StatusFound)
			return
		}
	}
	fmt.Fprintf(w, "One of clusters all fail, cannot do redirect")
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		hasher := md5.New()
		io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
		token := fmt.Sprintf("%x", hasher.Sum(nil))
		t, _ := template.ParseFiles("UI/upload.html")
		t.Execute(w, token)
	} else if r.Method == "POST" {
		fmt.Println("Upload Post")
		r.ParseMultipartForm(32 << 20)
		if r.FormValue("type") == "addfile" {
			file, handler, err := r.FormFile("uploadfile")
			groupName := strings.TrimSpace(r.FormValue("groupname"))
			if err != nil {
				http.Redirect(w, r, "/upload.html", http.StatusFound)
				fmt.Println(err)
				return
			}
			defer file.Close()
			fmt.Println("[upload] file name: ", handler.Filename)
			fmt.Println("[upload] group name: ", groupName)

			mList := getMusicList(groupName)

			mList.add(handler.Filename)
			fmt.Println("MList: ", mList)
			//mList.Add(handler.Filename, getServerListByClusterName(myServer.cluster))

			f, err := os.OpenFile(Directory + handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				fmt.Println(err)
				return
			}
			io.Copy(f, file)
			f.Close()

			afterReceiveFile(handler.Filename, mList, file, "add")
			
			http.Redirect(w, r, "http://"+myServer.combineAddr("http")+"/join.html?"+groupName, http.StatusFound)
		} else if r.FormValue("type") == "deletefile" {
			//TODO: delete file
			groupName := strings.TrimSpace(r.FormValue("groupname"))
			deleteMusic := r.FormValue("music")
			fmt.Println("[Debug]", deleteMusic)
			
			mList := getMusicList(groupName)
			mList.delete(deleteMusic) //TODO: only delete local
			
			fmt.Println("MList: ", mList)
			
			afterReceiveFile(deleteMusic, mList, nil, "delete")
			http.Redirect(w, r, "http://"+myServer.combineAddr("http")+"/join.html?"+groupName, http.StatusFound)
			
		}

	}
}

func afterReceiveFile(fileName string, mList *MusicList, file multipart.File, tp string) {
	// If Master, Simply broadcast to everyone
	if myServer == master {
		multicaster.UpdateList(ListContent{mList.name, tp, -1, fileName})
		// TODO: file sharding and send file to others
	} else {
		// Slave will request update list to master, master will handle this request
		// and therefore broadcast to everyone
		multicaster.RequestUpdateList(ListContent{mList.name, tp, -1, fileName})
		// mList.Add(handler.Filename, getServerListByClusterName(myServer.cluster))

	}
	
	// Transfer file
	if tp == "add" {
		fileTransfer(fileName, mList, file)
	}
	fmt.Println("Upload success")
}

func fileTransfer(fileName string, mList *MusicList, file multipart.File) {
	// File Sharding, send to different servers
	// candidates := mList.selectServer(fileName, getServerListByClusterName(myServer.cluster))
	candidates := clusterMap[myServer.cluster]
//	fileLock := new(sync.Mutex)
	for i := range candidates {
//		fileLock.Lock()
//		tmpFile := file
//		fileLock.Unlock()
		if candidates[i].combineAddr("File") != myServer.combineAddr("File") {
			go clientSendFile(fileName, candidates[i].combineAddr("File"))
		}
//		} else {
//			// Save file to local directory if you're also one of the candidate
//			if checkFileExist(fileName) {
//				fmt.Println("File already exists")
//				continue
//			}
//			f, err := os.OpenFile("./" + myServer.name + "/"+fileName, os.O_WRONLY|os.O_CREATE, 0666)
//			if err != nil {
//				fmt.Println(err)
//				return
//			}
//			io.Copy(f, tmpFile)
//			f.Close()
//		}
	}
}

// Delivers message from multicaster's message chan
func DeliverMessage(listName string) {
	msgChan := multicaster.GetMsgChans(listName)
	for {
		listcontent := <-msgChan
		fmt.Println("recv action type ", listcontent.Type)
		switch listcontent.Type {
		case "add":
			// Transfer file
			// fileTransfer(listcontent.File , mList, file)
			mList := getMusicList(listcontent.ListName)
			mList.add(listcontent.File)
		case "delete":
			mList := getMusicList(listcontent.ListName)
			mList.delete(listcontent.File)
		case "create":
			createNewGroupLocal(listcontent.ListName, myServer.cluster) //local
		case "update":
		}
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) { //clear
	if r.Method == "GET" {
		t, _ := template.ParseFiles("UI/index.html")
		t.Execute(w, nil)
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("UI/about.html")
		t.Execute(w, nil)	
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("UI/contact.html")
		t.Execute(w, nil)	
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func startHTTP() {
	fmt.Println("[init] HTTP at port", myServer.http_port)

	http.Handle("/css/", http.FileServer(http.Dir("UI")))
	http.Handle("/js/", http.FileServer(http.Dir("UI")))
	http.Handle("/images/", http.FileServer(http.Dir("UI")))
	http.Handle("/fonts/", http.FileServer(http.Dir("UI")))
	http.Handle("/music/", http.FileServer(http.Dir("UI")))
	http.Handle("/test/",http.FileServer(http.Dir(".")))

	http.HandleFunc("/index.html", homeHandler)
	http.HandleFunc("/about.html", aboutHandler)
	http.HandleFunc("/contact.html", contactHandler)
	
	http.HandleFunc("/create.html", createHandler)
	http.HandleFunc("/join.html", joinHandler)
	http.HandleFunc("/upload.html", fileHandler)
	
	http.ListenAndServe(":"+myServer.http_port, nil)

}

func getDeadServer() {
	//fmt.Println("Get dead servers from heartbeat manager's deadchannel")
	deadServerChannel := heartBeatTracker.GetDeadChannel()

	for {
		dead := <-deadServerChannel
		fmt.Println("[Heart Beat] dead: ", dead)
		// If I'm the master, then I must detect some slave died
		// Told every slaves to update their server list
		if master == myServer {
			memToRemove := myServer
			fmt.Println("[default memToRemove]", memToRemove)
			for i := range clusterMap[myServer.cluster] {
				fmt.Println("clusterMap[myServer.cluster]", clusterMap[myServer.cluster][i].combineAddr("heartbeat"))
				if clusterMap[myServer.cluster][i].combineAddr("heartbeat") == dead {
					memToRemove = clusterMap[myServer.cluster][i]
					break
				}
			}
			fmt.Println("[current list] ", clusterMap[myServer.cluster])
			if master == memToRemove {
				fmt.Println("Can not found dead server within the list", dead)
				return
			}
			// Tell other slaves to remove this slave from their list
			multicaster.RemoveMemberGlobal(memToRemove.combineAddr("comm"))
			//remove dead server from map
			rmDeadServer(memToRemove)

		} else {
			// If I'm the client which detects the master is dead
			// Become a candidate and raise election
			// raise an election
			if dead != master.combineAddr("heartbeat") {
				continue
			}
			multicaster.SendElectionMsg(master.combineAddr("comm"))
			fmt.Println("[current List i have in slaves] ", getServerListByClusterName(myServer.cluster))
			if len(getServerListByClusterName(myServer.cluster)) == 1 {
				fmt.Println("I'm the only one survive in the cluster :(")
				rmDeadServer(master)
				UpdateMaster(myServer.name)
			} else {
				// Wait for others to vote for you
				select {
				case newMaster := <-multicaster.masterChan:
					rmDeadServer(master)
					UpdateMaster(newMaster)
				case <-time.After(time.Millisecond * 1500):
					multicaster.numVotes = 0
	   				multicaster.voted = false
					fmt.Println("time out in being a new master")
				}
			}
		}

	}
}

func rmDeadServer(memToRemove Server) {
	mapLock.Lock()
	list := clusterMap[memToRemove.cluster]
	for i := range list {
		if list[i] == memToRemove {
			list = append(list[:i], list[i+1:]...)
			clusterMap[memToRemove.cluster] = list
			break
		}
	}
	
	for i:= range servers {
		if servers[i] == memToRemove {
			servers = append(servers[:i], servers[i+1:]...)
			break
		}
	}
	mapLock.Unlock()
}

// TODO: update the list in heartbeat and server.go
func UpdateMaster(new_master string) {


    multicaster.numVotes = 0
    multicaster.voted = false
    if myServer.name == new_master {
        master = myServer
        tmpList := make([]Server, 0)
        cServerList := getServerListByClusterName(myServer.cluster)
        for i := range cServerList {
            if cServerList[i] != master {
                tmpList = append(tmpList, cServerList[i])
            }
        }
        fmt.Println("UpdateAliveList in master, now track: ", tmpList)
        heartBeatTracker.updateAliveList(tmpList)
    } else {
        tmpmaster := myServer
        for i := range servers {
            if servers[i].name == new_master {
                tmpmaster = servers[i]
                break
            }
        }

        if tmpmaster == myServer {
            fmt.Println("False finding new master in my list ")
            return
        }
        master = tmpmaster
        tmpList := make([]Server, 0)
        tmpList = append(tmpList, master)
        fmt.Println("The new master is", master.name)
        fmt.Println("New tracking list", tmpList)
        heartBeatTracker.updateAliveList(tmpList)
    }
}

func GetElecMsg() {
	for {
		eMsg := <-multicaster.elecChan
		fmt.Println("I'm", myServer.name)
		fmt.Println("[GetElecMsg] msg.type:", eMsg.Type)
		fmt.Println("[GetElecMsg] msg.NewMaster: ", eMsg.NewMaster)
		
		switch eMsg.Type {
		case "candidate":
			test, err  := net.Dial("tcp", master.combineAddr("heartbeat"))
			if err == nil {
				fmt.Println("Master is still alive")
				test.Close()
				break
			}
			if master != myServer {
				go multicaster.SendVoteMessage(eMsg)
			}
		case "announce":
			rmDeadServer(master)
			UpdateMaster(eMsg.NewMaster)
			fmt.Println("Somebody else is the new master!")
		case "vote":
			multicaster.numVotes += 1
			fmt.Println("vote requries: ", len(clusterMap[myServer.cluster])-2, "vote I have:", multicaster.numVotes)
			if multicaster.numVotes >= (len(clusterMap[myServer.cluster])-2) {
				fmt.Println("I'm now the new master")
				// Delivers message to itself
				multicaster.masterChan <- eMsg.NewMaster
				multicaster.RemoveMemberGlobal(master.name)
				multicaster.SendNewMasterMsg()
			}
		case "novote":
		}
	}
}

/*
 * File Transfer as Client, either REQUEST when client select particular file we didnt have,
 * or SEND file when client add file, add file by using sharding to select best server
 */

func clientRequestFile(fileName string, addr string) {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Unable to connect server")
		return
	}
	fmt.Println("Connected to server ....")

	// Dircetory -- where file saved
	// directory := "./test/"
	directory := Directory
	// send action
	conn.Write([]byte("get\n"))
	// send request file name
	conn.Write([]byte(fileName + "\n"))
	// fmt.Fprintf(conn, fileName)

	msg, _ := bufio.NewReader(conn).ReadString('\n')
	// if server doesn't have that file || client isn't in the group
	if strings.Compare(msg, "success\n") != 0 {
		fmt.Println("<ERROR> ", msg)
		return
	}

	var receivedBytes int64
	// reader := bufio.NewReader(conn)
	f, err := os.Create(directory + fileName)
	defer f.Close()
	if err != nil {
		fmt.Println("Error creating file")
	}
	receivedBytes, err = io.Copy(f, conn)
	conn.Close()
	if err != nil {
		panic("Transmission error")
	}

	fmt.Printf("Finished transferring file. Received: %d \n", receivedBytes)
}

func clientSendFile(fileName string, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Unable to connect server")
		return
	}
	fmt.Println("Connected to server ....")

	// Send action
	conn.Write([]byte("upload\n"))
	// Send file name
	conn.Write([]byte(fileName + "\n"))
	msg, _ := bufio.NewReader(conn).ReadString('\n')
	// if already exists
	if strings.Compare(msg, "success\n") != 0 {
		fmt.Println("msg: ", msg)
		fmt.Println("File already exists in server")
		return
	}

	var n int64
	file, err := os.Open(strings.TrimSpace(Directory + fileName))
	if err != nil {
		fmt.Println("[clientSendFile] Not able to open file")
	    log.Fatal(err)
	}
	defer file.Close()
	n, err = io.Copy(conn, file)
	conn.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n, "bytes sent")
}

/* File Transfer port Listener */
func FileListener() {

	fmt.Println("Launching File Listener Port")
	listen, err := net.Listen("tcp", ":"+myServer.FilePort)
	if err != nil {
		fmt.Println("<Error> Can not listen too port!")
		return
	}

	for {
		conn, err := listen.Accept()
		conn.(*net.TCPConn).SetNoDelay(true)
		if err != nil {
			fmt.Println("<Error> Error when connecting to client!")
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Println("Start handling connection")
	reader := bufio.NewReader(conn)
	request, _ := reader.ReadString('\n')
	request = strings.Trim(request, "\n")
	switch request {
	case "upload":
		fmt.Println("upload file")
		serverRecvUploadFile(conn, reader)
		return
	case "get":
		fmt.Println("user tries to retrieve file")
		serverSendFile(conn, reader)
		return
	}
	fmt.Println("action not valid....")
	conn.Close()
}

func serverRecvUploadFile(conn net.Conn, reader *bufio.Reader) {
	// Dirctory
	directory := Directory

	fileName, _ := reader.ReadString('\n')
	fileName = strings.Trim(fileName, "\n")
	fmt.Println("Filename: ", fileName)
	// Check if file already exists
	if checkFileExist(fileName) {
		fmt.Println("file already exists\n")
		fmt.Fprintf(conn, "File already exists\n")
		return
	}
	fmt.Println("file not exists")
	// send file success
	fmt.Fprintf(conn, "success\n")

	// Wait to read file
	var receivedBytes int64
	// reader := bufio.NewReader(conn)
	f, err := os.Create(directory + fileName)
	defer f.Close()
	if err != nil {
		fmt.Println("Error creating file")
	}
	receivedBytes, err = io.Copy(f, conn)
	fmt.Println("recvUploadFile succeess!")
	if err != nil {
		panic("Transmission error")
	}
	fmt.Printf("Finished transferring file. Received: %d \n", receivedBytes)
	conn.Close()
}

func serverSendFile(conn net.Conn, reader *bufio.Reader) {
	directory := Directory
	fileName, _ := reader.ReadString('\n')
	fileName = strings.Trim(fileName, "\n")
	fmt.Println("fileName: ", fileName)
	// we don't have that file
	if !checkFileExist(fileName) {
		fmt.Fprintf(conn, "No such file\n")
		return
	}
	fmt.Fprintf(conn, "success\n")
	var n int64
	file, err := os.Open(strings.TrimSpace(directory + fileName))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	n, err = io.Copy(conn, file)
	conn.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n, "bytes sent")
}

/* MAIN FUNCTION RUNNING THE SERVER */

func main() {
	readServerConfig()

	//select server's configuration
	fmt.Print("[init] Enter a number(0-5) set up this server: ")
	var i int
	fmt.Scan(&i)
	myServer = servers[i]
	master = masterServer[myServer.cluster]
	
	// Create a folder for server to use
	err := os.Mkdir("test",0711)
 	if err != nil {
    	fmt.Println("Error creating directory")
    	fmt.Println(err)
 	}

	Directory = "./test/"
	
	readGroupConfig()
	readMusicConfig()

	InitialHeartBeat(master)
	go getDeadServer()
	go GetElecMsg()
	go FileListener()
	go listeningMsg()
	
	startHTTP()
}
