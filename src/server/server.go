package main

import (
	"net/http"
    "fmt"
    "strings"
    "html/template"
    "io"
    "time"
    "crypto/md5"
    "strconv"
    "os"
)

type Server struct {
    ip string
    comm_port string
    http_port string
    heartbeat_port string
}

func (s Server) combineAddr(port string) string{
	switch port {
		case "comm": return s.ip + ":" + s.comm_port
		case "http": return s.ip + ":" + s.http_port
		case "heartbeat": return s.ip + ":" + s.heartbeat_port
	}
	return ""
}

var (
    groups []Group
    localGroups []GroupMusic
    hasGroups map[string]bool
    dir string
    servers []Server
    myServer Server
    heartBeatTracker = new(HeartBeat)
)



func createHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(dir)
    if r.Method == "GET" {
    	t, _ := template.ParseFiles("UI/create.html")
    	t.Execute(w,nil)
    } else if r.Method == "POST" {
    	r.ParseForm()
    	groupName := strings.TrimSpace(r.PostFormValue("groupname"))
    	if !isGroupNameExist(groupName) {  		
    		multicastServers(groupName, "create_group")
    		var newGroup Group
			newGroup.name = groupName
			newGroup.serverList = make(map[string]bool)
			newGroup.addServer(myServer.ip + ":" + myServer.comm_port)
			groups = append(groups, newGroup)
			hasGroups[groupName] = true
			fmt.Println(groups)
			fmt.Println(hasGroups)
			//t, _ := template.ParseFiles("UI/create.html")
    		//t.Execute(w,nil)
			http.Redirect(w, r, "http://127.0.0.1:8282/upload.html", 301)
    	} else {
    		w.Write([]byte("group name exist, please try another"))
    	}
    } else {
    	fmt.Fprintf(w, "Error Method")
    }
    
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
    	t, _ := template.ParseFiles("UI/join.html")
    	t.Execute(w,nil)
    } else if r.Method == "POST"{
    	r.ParseForm()
    	groupName := strings.TrimSpace(r.PostFormValue("groupname"))
    	if isGroupHere(groupName) {
    		w.Write([]byte("you are in the group: " + groupName))
    	} else {
    		multicastServers(groupName, "join_group")
    		hasGroups[groupName] = true
    		w.Write([]byte("join successful"))    		
    	}
    	
    }
}

func addfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
        hasher := md5.New()
        io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
        token := fmt.Sprintf("%x", hasher.Sum(nil))
        t, _ := template.ParseFiles("UI/upload.html")
        t.Execute(w, token)
    } else {
        r.ParseMultipartForm(32 << 20)
        file, handler, err := r.FormFile("uploadfile")
        groupName := strings.TrimSpace(r.PostFormValue("groupname"))
        if err != nil {
        	http.Redirect(w, r, "localhost:8282/upload.html", 301)
            fmt.Println(err)
            return
        }
        defer file.Close()
        //fmt.Fprintf(w, "%v", handler.Header)
        fmt.Println(handler.Filename)
        f, err := os.OpenFile("./test/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
        for i := range localGroups {
        	if localGroups[i].name == groupName {
        		localGroups[i].addMusic(handler.Filename)
        	}
        }
        if err != nil {
            fmt.Println(err)
            return
        }
        defer f.Close()
        
        io.Copy(f, file)
        
        http.Redirect(w, r, "http://" + myServer.combineAddr("http") + "/upload.html", 301)
        //replicaFiles()
        //t, _ := template.ParseFiles("UI/upload.html")
    	//t.Execute(w,nil)
    }
}


func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
    	t, _ := template.ParseFiles("UI/index.html")
    	t.Execute(w, nil)
    } else {
    	fmt.Fprintf(w, "Error Method")
    }
}

func startHTTP() {
	fmt.Println("[HTTP] server started")
  	
  	http.Handle("/css/", http.FileServer(http.Dir("UI")))
    http.Handle("/js/", http.FileServer(http.Dir("UI")))
    http.Handle("/images/", http.FileServer(http.Dir("UI")))
    http.Handle("/fonts/", http.FileServer(http.Dir("UI")))
    
	http.HandleFunc("/index.html", homeHandler)
	http.HandleFunc("/create.html", createHandler)
    http.HandleFunc("/join.html", joinHandler)
    http.HandleFunc("/upload.html", addfileHandler)
    //http.HandleFunc("/leave", leaveHandler)
    
    http.ListenAndServe(":" + myServer.http_port, nil)
   
}

func getDeadServer(){
    fmt.Println("Get dead servers from heartbeat manager's deadchannel")
    deadServerChannel := heartBeatTracker.GetDeadChannel()
    // TODO: Do something for the dead servers
    
    for {
		dead := <-deadServerChannel
		//if dead {
		fmt.Println("dead: ", dead)
		//}
    }
}

func main() {
	readServerConfig() 
	
	//select server's configuration
    fmt.Print("Enter a number(0-3) set up this server: ")
    var i int
    fmt.Scan(&i)
    myServer = servers[i]

	readGroupConfig()
	readMusicConfig()
	
	InitialHeartBeat()
	go getDeadServer()

	go listeningMsg(myServer.ip, myServer.comm_port)
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