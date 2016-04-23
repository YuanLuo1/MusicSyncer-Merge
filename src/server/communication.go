package main

import(
	"encoding/gob"
    "fmt"
    "net"
)

type Message struct{
	Dst string
	Src string
	Kind string
	Data string	
}

func sendOneMsg(dest string, src string, kind string, data string) {
	//fmt.Println("start client");
	msg := &Message{dest, src, kind, data}
    conn, err := net.Dial("tcp", msg.Dst)
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

func createGroup(msg *Message) {
	if !isGroupNameExist(msg.Data){
		groupMap[msg.Data] = myServer.cluster 
    	hasGroups[msg.Data] = true
	}
}

func requestHandler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	msg := &Message{}
	dec.Decode(msg)
	
	fmt.Printf("[MSG-rec] Received: %+v\n", msg);
	switch msg.Kind {
		case "create_group": 
			createGroup(msg)
			//fmt.Println(groups)
		/*case "join_group": 
			for i:= range groups {
				if groups[i].name == msg.Data {
					groups[i].addServer(msg.Src)
				}
			}
			fmt.Println(groups)*/
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
	serverList := clusterMap[myServer.cluster]
	fmt.Println("multicast",clusterMap)
	for i := range serverList{ 
		if serverList[i] != myServer.combineAddr("comm") {
			sendOneMsg(serverList[i], myServer.combineAddr("comm"), kind, data)
		}
	}
}

