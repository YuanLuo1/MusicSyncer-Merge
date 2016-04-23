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

var (
    //groups []Group //all servers metadata
    dir string
    servers []Server //all servers and port
    myServer Server //this server info
    heartBeatTracker = new(HeartBeat) //heart beat related
    
    musicList []MusicList //local groups music list
    hasGroups map[string]bool //local groups map
    clusterMap map[string][]string //key:cluster name, value:cluster's server list
    groupMap map[string]string //key: groupName, value: cluster name
)
type Content struct{
	Test string
}

type Server struct {
    ip string
	name string
    comm_port string
    http_port string
    heartbeat_port string
    cluster string
}

func (s Server) combineAddr(port string) string{
	switch port {
		case "comm": return s.ip + ":" + s.comm_port
		case "http": return s.ip + ":" + s.http_port
		case "heartbeat": return s.ip + ":" + s.heartbeat_port
	}
	return ""
}


func createHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(dir)
    if r.Method == "GET" {
    	t, _ := template.ParseFiles("UI/create.html")
    	t.Execute(w,nil)
    } else if r.Method == "POST" {
    	r.ParseForm()
    	groupName := strings.TrimSpace(r.PostFormValue("groupname"))
    	//fmt.Printf("create group debug")
    	if !isGroupNameExist(groupName) { 
    		createNewGroup(groupName)
    		multicastServers(groupName, "create_group")
			
			http.Redirect(w, r, "/upload.html", http.StatusFound)
    	} else {
    		w.Write([]byte("group name exist, please try another"))
    	}
    } else {
    	fmt.Fprintf(w, "Error Method")
    }
    
}
func createNewGroup(groupName string) {
	groupMap[groupName] = myServer.cluster 
    hasGroups[groupName] = true
    
    newList := new(MusicList)
    newList.NewInstance()
    newList.name = groupName
    musicList = append(musicList, *newList)
    fmt.Println("[Create] Group Map", groupMap)
	fmt.Println("[Create] Has Groups", hasGroups)
    //TODO: have to tell all the servers - new group created
    //MSG other servers in the cluster
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
    	t, _ := template.ParseFiles("UI/join.html")
    	t.Execute(w,nil)
    }  else {
    	fmt.Fprintf(w, "Error Method")
    }
    
    /*else if r.Method == "POST"{
    	r.ParseForm()
    	groupName := strings.TrimSpace(r.PostFormValue("groupname"))
    	if isGroupHere(groupName) {
    		w.Write([]byte("you are in the group: " + groupName))
    	} else {
    		multicastServers(groupName, "join_group")
    		hasGroups[groupName] = true
    		w.Write([]byte("join successful"))    		
    	}
    	
    }*/
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
        	http.Redirect(w, r, "/upload.html", http.StatusFound)
            fmt.Println(err)
            return
        }
        defer file.Close()
        //fmt.Fprintf(w, "%v", handler.Header)
        fmt.Println(handler.Filename)
        f, err := os.OpenFile("./test/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
        //getGroup(groupName).getServerList()
        list := getMusicList(groupName)
        list.Add(handler.Filename, getServerListByClustername(myServer.cluster))
       /* for i := range localGroups {
        	if localGroups[i].name == groupName {
        		localGroups[i].addMusic(handler.Filename)
        	}
        }*/
        
        if err != nil {
            fmt.Println(err)
            return
        }
        defer f.Close()
        
        io.Copy(f, file)
        
        http.Redirect(w, r, "/upload.html", 301)
        //replicaFiles()
        //t, _ := template.ParseFiles("UI/upload.html")
    	//t.Execute(w,nil)
    }
}


func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		c := Content{Test: "test!!!!!"}
    	t, _ := template.ParseFiles("UI/index.html")
    	t.Execute(w, c)
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
    
	http.HandleFunc("/index.html", homeHandler)
	http.HandleFunc("/create.html", createHandler)
    http.HandleFunc("/join.html", joinHandler)
    http.HandleFunc("/upload.html", addfileHandler)
    //http.HandleFunc("/leave", leaveHandler)
    
    http.ListenAndServe(":" + myServer.http_port, nil)
   
}

func getDeadServer(){
    //fmt.Println("Get dead servers from heartbeat manager's deadchannel")
    deadServerChannel := heartBeatTracker.GetDeadChannel()
    // TODO: Do something for the dead servers
    
    for {
		dead := <-deadServerChannel
		fmt.Println("[Heart Beat] dead: ", dead)
    }
}

func main() {
	readServerConfig() 
	
	//select server's configuration
    fmt.Print("[init] Enter a number(0-3) set up this server: ")
    var i int
    fmt.Scan(&i)
    myServer = servers[i]

	readGroupConfig()
	readMusicConfig()
	
	//InitialHeartBeat()
	//go getDeadServer()

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
