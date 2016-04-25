package main

import (
    "fmt"
    "strings"
    "html/template"
    "io"
    "time"
    "crypto/md5"
    "strconv"
    "os"
    "net"
    "bufio"
    "net/http"
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
)

type Music struct {
	GroupName string
	FilesMap  map[string]string
	//Link []string
}

type Server struct {
	ip             string
	name           string
	comm_port      string
	http_port      string
	heartbeat_port string
	cluster        string
	heartbeatFreq  int
    FilePort 	   string
}

func (s Server) combineAddr(port string) string {
	switch port {
		case "comm": return s.ip + ":" + s.comm_port
		case "http": return s.ip + ":" + s.http_port
		case "heartbeat": return s.ip + ":" + s.heartbeat_port
        case "File": return s.ip + ":" + s.File_port
	}
	return ""
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	
	if r.Method == "GET" {
		fmt.Println("[debug] get")
		
		t, _ := template.ParseFiles("UI/create.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		groupName := strings.TrimSpace(r.PostFormValue("groupname"))
		fmt.Println("[debug]", groupName)
		if !isGroupNameExist(groupName) {
			createNewGroupLocal(groupName, myServer.cluster) //local
			//multicastServers(groupName, "create_group") //check group type

			data := Music{GroupName: groupName}
			data.FilesMap = make(map[string]string)
			//data.FilesMap["test"] = "music/music.mp3"
			//data.FilesMap["test2"] = "music/music.mp3"
			//fmt.Println("[DDDDDDDDDDDDDDD]", data)
			//Files: []string{"test", "test1", "test2"}
			//Link: []string{"music/music.mp3","music/music.mp3","music/music.mp3"}
			t, _ := template.ParseFiles("UI/upload.html")
			t.Execute(w, data)
			//http.Redirect(w, r, "/upload.html/" + groupName, http.StatusFound)
			//multicastServers(groupName, "create_group") //check group type
		} else {
			w.Write([]byte("Create Group failed, please try another groupname or check servers alive"))
		}
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		//TODO: get music list file and render to join.html
		t, _ := template.ParseFiles("UI/join.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		
		groupName := strings.TrimSpace(r.PostFormValue("groupname"))
		if isGroupHere(groupName) {
			w.Write([]byte("you are in the group: " + groupName))

			//TODO: go to listen music page with group name
			t, _ := template.ParseFiles("UI/upload.html")
			t.Execute(w, nil)
		} else {
			redirectToCorrectServer(groupName, w, r) //didn't check
			w.Write([]byte("join successful"))
		}
	} else {
		fmt.Fprintf(w, "Error Method")
	}
}

func redirectToCorrectServer(groupName string, w http.ResponseWriter, r *http.Request) {
	serverList := clusterMap[groupMap[groupName]]
	http.Redirect(w, r, serverList[0].combineAddr("http")+"/upload.html", http.StatusFound)
}

func addfileHandler(w http.ResponseWriter, r *http.Request) {
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
			// defer file.Close()
			fmt.Println("[upload] file name: ", handler.Filename)
			fmt.Println("[upload] group name: ", groupName)

			//TODO: check
			mList := getMusicList(groupName)

			// mList.add(handler.Filename)
			// fmt.Println("MList: ", mList)
			// mList.Add(handler.Filename, getServerListByClusterName(myServer.cluster))

			// If Master, Simply broadcast to everyone
	        if myServer == master {
	            multicaster.UpdateList(ListContent{mList.name, "add", -1, handler.Filename})
	            // TODO: file sharding and send file to others
	        } else {
	            // Slave will request update list to master, master will handle this request
	            // and therefore broadcast to everyone
	            multicaster.RequestUpdateList(ListContent{mList.name, "add", -1, handler.FileName})
	            // mList.Add(handler.Filename, getServerListByClusterName(myServer.cluster))

	        }
	        // File Sharding, send to different servers
	        candidates := mList.selectServer(handler.FileName, getServerListByClusterName(myServer.cluster))
	        for i := range candidates {
	            if candidates.combineAddr("File") != myServer.combineAddr("File"){
	                clientSendFile(file, handler.FileName, candidates.combineAddr("File"))
	            } else {
	                // Save file to local directory if you're also one of the candidate
	                if checkFileExist(handler.Filename){
	                	continue
	                }
	                f, err := os.OpenFile("./test/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	                if err != nil {
	                    fmt.Println(err)
	                    return
	                }
	                io.Copy(f, file)
	                f.close()
	            }
	        }

			fmt.Println("Upload success")

			data := Music{GroupName: groupName}
			data.FilesMap = make(map[string]string)
			for key, _ := range mList.fileList {
				data.FilesMap[key] = "music/music.mp3"
			}
			t, _ := template.ParseFiles("UI/upload.html")
			t.Execute(w, data)

		} else if r.FormValue("type") == "deletefile" {

			//TODO: delete file
		}

	}
}

// Delivers message from multicaster's message chan
func DeliverMessage(listName string) {
    msgChan := multicaster.GetMsgChans(listName)
    for {
        listcontent := <- msgChan
        switch listcontent.Type {
        case "add":
        	mList := getMusicList(listcontent.ListName)
			mList.add(listcontent.File)

        case "delete":
        	mList := getMusicList(listcontent.ListName)
			mList.add(listcontent.File)
       
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

func groupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		//groupName := strings.Split(r.URL.Path[1:],"/")[1]
		/*data := Music{Content: groupName}
		fmt.Println(groupName)
		t, _ := template.ParseFiles("UI/group.html")
		t.Execute(w, data)*/

	}
}

func startHTTP() {
	fmt.Println("[init] HTTP at port", myServer.http_port)

	http.Handle("/css/", http.FileServer(http.Dir("UI")))
	http.Handle("/js/", http.FileServer(http.Dir("UI")))
	http.Handle("/images/", http.FileServer(http.Dir("UI")))
	http.Handle("/fonts/", http.FileServer(http.Dir("UI")))
	http.Handle("/music/", http.FileServer(http.Dir("UI")))

	http.HandleFunc("/index.html", homeHandler)
	http.HandleFunc("/create.html", createHandler)
	http.HandleFunc("/join.html", joinHandler)
	http.HandleFunc("/upload.html", addfileHandler)
	http.HandleFunc("/upload.html/", groupHandler)
	//http.HandleFunc("/leave", leaveHandler)

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
			for i := range servers {
				if clusterMap[myServer.cluster][i].combineAddr("heartbeat") == dead {
					memToRemove = servers[i]
					break
				}
			}
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
			multicaster.SendElectionMsg(master.combineAddr("comm"))
			// Wait for others to vote for you
			select {
			case newMaster := <-multicaster.masterChan:
				rmDeadServer(master)
				UpdateMaster(newMaster)
			case <-time.After(time.Millisecond * 500):
				fmt.Println("time out in getting a new master")
			}
		}

	}
}

func rmDeadServer(memToRemove Server) {
	list := clusterMap[memToRemove.cluster]
	for i := range list {
		if list[i] == memToRemove {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
}

// TODO: update the list in heartbeat and server.go
func UpdateMaster(new_master string) {

    multicaster.numVotes = 0
    multicaster.voted = false
    if myServer.name == new_master {
        master = myServer
        tmpList := make([]Server, 1)
        cServerList := getServerListByClusterName(myServer.cluster)
        for i := range cServerList {
            if cServerList[i] != master {
                tmpList = append(tmpList, cServerList[i])
            }
        }
        fmt.Println("UpdateAliveList in master, now track: ", tmpList)
        heartbeat.updateAliveList(tmpList)
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
        tmpList := make([]Server, 1)
        tmpList = append(tmpList, master)
        heartbeat.updateAliveList(tmpList)
    }
}

func GetElecMsg() {
	for {
		eMsg := <-multicaster.elecChan
		switch eMsg.Type {
		case "candidate":
			go multicaster.SendVoteMessage(eMsg)
		case "announce":
			rmDeadServer(master)
			UpdateMaster(eMsg.NewMaster)
			fmt.Println("Somebody else is the new master!")
		case "vote":
			multicaster.numVotes += 1
			if multicaster.numVotes == int((len(servers)-1)/2) {
				// Delivers message to itself
				multicaster.masterChan <- eMsg.NewMaster
				multicaster.RemoveMemberGlobal(master.name)
				multicaster.SendNewMasterMsg()
				UpdateMaster(eMsg.NewMaster)
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

    conn, err  := net.Dial("tcp", addr)
    if err != nil {
        log.Fatal(err)
        fmt.Println("Unable to connect server")
        return
    }
    fmt.Println("Connected to server ....")

    // Dircetory -- where file saved
    directory := "./test/"

    // send action
    conn.Write([]byte("get\n"))
    // send request file name
    conn.Write([]byte(fileName + "\n"))
    // fmt.Fprintf(conn, fileName)
    
    msg, _ := bufio.NewReader(conn).ReadString('\n')
    // if server doesn't have that file || client isn't in the group
    if strings.Compare(msg, "success\n") != 0{
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

 func clientSendFile(sf multipart.File, fileName string, addr string) {

    conn, err  := net.Dial("tcp", addr)
    if err != nil {
        log.Fatal(err)
        fmt.Println("Unable to connect server")
        return
    }
    fmt.Println("Connected to server ....")

    // Send action
    connection.Write([]byte("upload\n"))
    // Send file name
    connection.Write([]byte(fileName+"\n"))
    msg, _ := bufio.NewReader(connection).ReadString('\n')
    // if already exists
    if strings.Compare(msg, "success\n") != 0{
        fmt.Println("msg: ", msg)
        fmt.Println("File already exists in server")
        return
    }

    var n int64
    n, err = io.Copy(connection, sf)
    connection.Close()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(n, "bytes sent")
 }

/* File Transfer port Listener */

func FileListener() {
    var port string = "9999"

    fmt.Println("Launching File Listener Port")
    listen, err := net.Listen("tcp", ":"+myServer.FilePort)
    if err != nil{
        fmt.Println("<Error> Can not listen too port!")
        return
    }

    for{
        conn, err := listen.Accept()
        conn.(*net.TCPConn).SetNoDelay(true)
        if err != nil{
            fmt.Println("<Error> Error when connecting to client!")
            continue
        }
        go handleConnection(conn)
    }
}

/*func checkFileExist(fileName string) bool{
    fileName = "./test/" + fileName
    if _, err := os.Stat(fileName); err == nil{
        return true
    }
    return false
}*/

func handleConnection(conn net.Conn){
    fmt.Println("Start handling connection")
    reader := bufio.NewReader(conn)
    request, _ := reader.ReadString('\n')
    request = strings.Trim(request, "\n")
    switch request{
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

func serverRecvUploadFile(conn net.Conn, reader *bufio.Reader){

    // Dirctory
    directory := "./test/"

    fileName, _ := reader.ReadString('\n')
    fileName = strings.Trim(fileName, "\n")
    fmt.Println("Filename: ", fileName)
    // Check if file already exists
    if checkFileExist(fileName){
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

func serverSendFile(conn net.Conn, reader *bufio.Reader){
    directory := "./test/"
    fileName, _ := reader.ReadString('\n')
    fileName = strings.Trim(fileName, "\n")
    fmt.Println("fileName: ", fileName)
    // we don't have that file
    if !checkFileExist(fileName){
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
	fmt.Print("[init] Enter a number(0-3) set up this server: ")
	var i int
	fmt.Scan(&i)
	myServer = servers[i]
	master = masterServer[myServer.cluster]

	readGroupConfig()
	readMusicConfig()
	
	// InitialHeartBeat(master)
	// go getDeadServer()
    // go GetElecMsg()
    // go FileListener()
	go listeningMsg()
	startHTTP()
}

/*func leaveHandler(w http.ResponseWriter, r *http.Request) {
    //fmt.Fprintln(w, "<h1>%s!</h1>", r.URL.Path[1:])
    r.ParseForm()
    if r.Method == "GET" {
    	fmt.Fprintf(w, "Error Method")
    } else {
    	ip := strings.TrimSpace(r.PostFormValue("clientip"))
    	groupid := strings.TrimSpace(r.PostFormValue("groupid"))
    	comMainGroup(groupid, "remove_server")
    	//leaveGroup(ip, groupid)
    }
}*/
