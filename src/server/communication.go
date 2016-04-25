package main

import(
	"encoding/gob"
    "fmt"
    "net"
)

//type Message struct{
//	Dst string
//	Src string
//	Kind string
//	Data string	
//}

func sendOneMsg(dest string, src string, kind string, data string) {
	//fmt.Println("start client");
	msg := &Message{dest, src, kind, data, ListContent{}, ElectionMsg{}}
    conn, err := net.Dial("tcp", msg.Dest)
    if err != nil {
        fmt.Println("Connection error: ", err)
    }
    encoder := gob.NewEncoder(conn)
    encoder.Encode(msg)
   
    /*connbuf := bufio.NewReader(conn)
	for{
    	str, err := connbuf.ReadString('\n')
    	if len(str)>0{
        	fmt.Println(str)
    	}
    	if err!= nil {
        	break
    	}
	}*/
    conn.Close()
   // fmt.Println(groups);
}

func whichCluster(ip string) string {
	for i := range servers {
		if servers[i].combineAddr("comm") == ip {
			return servers[i].cluster
		}
	}
	return ""
}

func handleCreateGroup(msg *Message) {
	groupName := msg.RemMemName
	if !isGroupNameExist(groupName){
		clusterName := whichCluster(msg.Src)
		if clusterName == myServer.cluster {
			hasGroups[msg.RemMemName] = true
			mList := new(MusicList)
			mList.NewInstance()
			mList.name = groupName
			musicList = append(musicList, *mList)
		} 
		groupMap[groupName] = clusterName   	
	}
	fmt.Println("[MSG-HandlerCreateGroup] group map", groupMap)
	fmt.Println("[MSG-HandlerCreateGroup] has groups", hasGroups);
	fmt.Println("[MSG-HandlerCreateGroup] music list", musicList);
}

func handleJoinGroup(msg *Message) {
	/*for i:= range groups {
		if groups[i].name == msg.Data {
				groups[i].addServer(msg.Src)
		}
	}*/
}
func requestHandler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	msg := &Message{}
	dec.Decode(msg)
	
	fmt.Printf("[MSG-rec] Received: %+v\n", msg);
	switch msg.Type {
		case "create_group": 
			handleCreateGroup(msg)
		case "join_group":
			handleJoinGroup(msg)
		//case "remove_server": removeServer(msg)
		//case "group_list": groupList(msg)
	}
}

func listeningMsg() {
	fmt.Println("[init] communication at port", myServer.comm_port)
	socket, err := net.Listen("tcp", myServer.combineAddr("comm"))
  	if err != nil { 
  		fmt.Println("tcp listen error") 
  	} 
  	for {
    	conn, err := socket.Accept()
    	if err != nil { 
    		fmt.Println("connection error") 
    	}
    	go requestHandler(conn)
  	}
}

func multicastServers(data string, kind string) {
	//serverList := clusterMap[myServer.cluster]
	//fmt.Println("multicast",clusterMap)
	for i := range servers{ 
		if servers[i] != myServer{
			//fmt.Println("[debug]multicast",servers[i].comm_port)
			sendOneMsg(servers[i].combineAddr("comm"), myServer.combineAddr("comm"), kind, data)
		}
	}
}

