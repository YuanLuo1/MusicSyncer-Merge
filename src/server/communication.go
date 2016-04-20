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
    fmt.Println(groups);
}

func createGroup(msg *Message) {
	if !isGroupNameExist(msg.Data){
		var newGroup Group
		newGroup.name = msg.Data
		newGroup.serverList = make(map[string]bool)
		newGroup.addServer(msg.Src)
		groups = append(groups,newGroup)
	}
}

func requestHandler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	msg := &Message{}
	dec.Decode(msg)
	
	fmt.Printf("Received:\n %+v\n", msg);
	switch msg.Kind {
		case "create_group": 
			createGroup(msg)
			fmt.Println(groups)
		case "join_group": 
			for i:= range groups {
				if groups[i].name == msg.Data {
					groups[i].addServer(msg.Src)
				}
			}
			fmt.Println(groups)
		//case "remove_server": removeServer(msg)
		//case "group_list": groupList(msg)
	}
}

func listeningMsg(myIp string, myPort string) {
	fmt.Println("listening messages at port", myPort)
	socket, err := net.Listen("tcp", myIp + ":" + myPort)
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
	for i := range servers{ 
		if servers[i] != myServer {
			sendOneMsg(servers[i].combineAddr("comm"), myServer.combineAddr("comm"), kind, data)
		}
	}
}

