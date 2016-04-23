package main 

import (
	"fmt"
	"net"
	"time"
	"log"
	"net/rpc"
)

/*
 * Message passing structure
 */

type ElectionMsg struct{
	// Vote, Already vote or candidate, announce
	Type string
	NewMaster string
}

type ListConent struct {
	ListName string
	Type string
	Pos string
	File string
}  

type Message struct {
	Dest string
	Src string
	Type string
	remMemName string
	ListInfo ListConent
	ElectionInfo ElectionMsg
}

type RPCRecver struct {
	rcvMedia *Mulitcaster
	ackChans map[string]chan string
}

type Mulitcaster struct {
	members map[string]string
	myInfo Server
	myId string
	msgChans map[string]chan ListConent 	// key: MusicList name, value: new message
	ackChans map[string]chan string 	// Key: MusicList name, value: Ack messages
	elecChan chan ElectionMsg
	sender RPCRecver
}

/*
 * Below are how we handle the rpc for multicasting
 */

func (this *Mulitcaster) Initiallized(server Server, members []Server){
	this.members = make(map[string]string)
	for s := range members {
		// TODO: add server name as key and server ip + comm_port as server
		this.members[s.name] = s.ip + ":" + s.comm_port
	}
	this.myInfo = server
	this.sender = RPCRecver{this, make(map[string]chan string)}

	this.ackChans = make(map[string]chan string)
	this.msgChans = make(map[string]chan Message)
	this.elecChan = make(chan ElectionMsg, 128)
	go this.lisenter(server)
}

/*
 * Start the listener which runs the multicaster 
 */
func (this *Mulitcaster) lisenter(server Server){
	rpc.Register(&this.recver)
	l, e := net.Listen("tcp", ":"+server.comm_port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	for{
		c, e := l.Accept()
		if e != nil {
			log.Fatal("client connect error: ", e)
		}
		go rpc.ServeConn(c)
	}
}

// All should return 'success' if communicate successfully
func (this *RPCRecver) Communicate (msg Message, reply *string) {
	switch msg.Type {
	case "remMem":
		// Only master can remove member
		fmt.Println("Remove member: ", msg.remMemName)
		this.rcvMedia.RemoveMemberLocal(msg.remMemName)
		*reply = "success"
	case "election":
		this.rcvMedia.elecChan <- msg.ElectionInfo
		*reply = "success"
	case "listMsg":
		*reply = "false"
		// Slaves recieves messages, send an ack back to the master
		msg := Message{msg.Src, this.rcvMedia.myInfo.ip+":"+this.rcvMedia.myInfo.comm_port, "ackRcv", "", msg.ListInfo, msg.ElectionInfo}
		go this.rcvMedia.SendMsg(msg)
		// wait for master to send a commit log back, if timeout discard the msg

		// New list 
		if msg.ListInfo.Type == "create" {
			this.rcvMedia.msgChans[msg.ListInfo.ListName] = make(chan ListConent)
			this.rcvMedia.ackChans[msg.ListInfo.ListName] = make (chan string)
			this.ackChans[msg.ListInfo.ListName] = make (chan string)
		}
		select {
			case <-this.ackChans[msg.ListInfo.ListName]:
				*reply = "success"
				// rcv commit message from master, delivers
				this.rcvMedia.msgChans[msg.ListInfo.ListName] <- msg.ListInfo
				fmt.Println("Deliver msg success")
			case <- time.After(time.Second * 1):
				if msg.ListInfo.Type == "create" {
					delete(this.rcvMedia.msgChans, msg.ListInfo.ListName)
					delete(this.rcvMedia.ackChans, msg.ListInfo.ListName)
					delete(this.ackChans, msg.ListInfo.ListName)
				}
				fmt.Println("time out")
		}
	// TODO: when a client upload file to the server, shard and replica and sending to corresponding servers
	case "uploadFile":
		*reply = "success"
	case "ackRcv":
		// Master will rcv this type of msg if slave recv a file and send back an ack
		this.rcvMedia.ackChans[msg.ListInfo.ListName] <- "ack"
	case "commit":
		this.ackChans[msg.ListInfo.ListName] <- "ack"
	default:
		fmt.Println("Message type not correct: ", msg.Type)
		*reply = ""
	}
}

/* Multicast a update List message to inform update to all the members,
	Only master can use this function */
func (this *Mulitcaster) UpdateList(content ListConent) bool {
	msg := Message{"", this.myInfo.ip+":"+this.myInfo.comm_port, "listMsg", "", content, ElectionMsg{}}

	// New list, Create new channel to rcv messages
	if msg.ListInfo.Type == "create" {
		this.msgChans[content.ListName] = make(chan ListConent)
		this.ackChans[content.ListName] = make (chan string)
		this.sender.ackChans[content.ListName] = make (chan string)
	}

	for key := range this.members {
		if key == this.myInfo.name {
			continue
		}
		msg.Dest = this.members[key]
		go this.SendMsg(msg)
	}

	// Request message from 
	numVote := int(len(this.members)/2)
	numRcv := 0
	for i:=0; i<numVote-1; i++ {
		select {
			case <- this.ackChans[content.ListName]:
				numRcv += 1
				fmt.Println("rcv Ack ", numRcv)
			case <- time.After(time.Second * 0.6):
				fmt.Println("Not enough vote :(")
				if content.Type == "create" {
					delete(this.msgChans, msg.ListInfo.ListName)
					delete(this.ackChans, msg.ListInfo.ListName)
					delete(this.sender.ackChans, msg.ListInfo.ListName)
				}
				return false
		}
	}
	// Rcv a majority of votes, multicast commit message to those slaves
	msg.Type = "commit"
	for key := range this.members {
		if key == this.myInfo.name {
			continue
		}
		msg.Dest = this.members[key]
		go this.SendMsg(msg)
	}
	// Delivers to itself
	this.msgChans[content.ListName] <- msg.ListInfo
	return true
}

func (this *Mulitcaster) GetElecChan() chan ElectionMsg {
	return this.elecChan
}

func (this *Mulitcaster) GetMsgChans(server string) chan Message {
	return this.msgChans[server]
} 

func (this *Mulitcaster) SendMsg(msg Message) string{
	c, err := rpc.Dial("tcp", msg.Dest)
	defer c.Close()
	if err != nil {
		log.Fatal(err)
		return ""
	}
	var result string
	c.Call("PasserRPC.Communicate", message, &result)
	return result
}

func (this *Mulitcaster) RemoveMemberLocal(memberName string){
	delete(this.members, memberName)
}

// Return true if success, else return false. Only for master
func (this *Mulitcaster) RemoveMemberGlobal(memberName string) bool{
	this.RemoveMemberLocal(memberName)
	msg := Message{"", this.myInfo.ip+":"+this.myInfo.comm_port, "remMem", memberName, ListConent{}, ElectionMsg{}}
	for key := range this.members{
		if key == this.myInfo.name {
			continue
		}
		msg.Dest = this.members[key]
		if this.SendMsg(msg) != "success" {
			fmt.Println("Error removing file to client: ", key)
			return false
		}
	}
	return true
}

/* send  election message, ask group memeber to vote for me */
func (this *Mulitcaster) SendElectionMsg(oldMaster string) bool{
	for key := range this.members{
		if key == this.myInfo.name || key == oldMaster{
			continue
		}
		msg := Message{this.members[key], this.myInfo.ip+":"+this.myInfo.comm_port, "election", "", ListConent{}, ElectionMsg{"candidate", this.myInfo.name}}
		if this.SendMsg(msg) != "success" {
			fmt.Println("Error Asking client to vote for me: ", key)
			return false
		}
	}
	return true
}

/* send announce message to told everyone I'm the new master */
func (this *Mulitcaster) SendNewMasterMsg() {
	for key := range this.members {
		if key == this.myInfo.name {
			continue
		}
		msg := Message{this.members[key], this.myInfo.ip+":"+this.myInfo.comm_port, "election", "", ListConent{}, ElectionMsg{"announce", this.myInfo.name}}
		go this.SendMsg(msg)
	}
}
