package main

/*
 * Original thought: 
 */

import (
	"fmt"
	"os"
	"net"
	"sync"
	"time"
)

const (
	DEAD_DETECT = 3
)

type HeartBeat struct {
	host Server
	track_server []string
	track_server_addr []*net.UDPAddr
	listenSock *net.UDPConn
	timeStamps map[string]time.Time
	deadChannel chan string
	lock *sync.Mutex
	
	serverHBFreq []int
}

func checkErr(err error){
	if err != nil {
		fmt.Println("<Error> ", err)
		os.Exit(0)
	}
}

func (this *HeartBeat) newInstance(host Server, connect_servers []Server){
	this.host = host
	// Set up listen socket
	addr, err := net.ResolveUDPAddr("udp", this.host.combineAddr("heartbeat"))
	checkErr(err)
	this.listenSock, err = net.ListenUDP("udp", addr)
	checkErr(err)

	// Initiallize the arguments
	this.lock = new(sync.Mutex)
	this.deadChannel = make(chan string)
	this.updateAliveList(connect_servers)
	
	go this.recvAliveMsg()
	if this.host != master {
		// Slave send hb to master to trigger connection
		go this.sendAliveMsg()
	}
}

func (this *HeartBeat) updateAliveList(connect_servers []Server){
	this.lock.Lock()
	this.serverHBFreq = make([]int, len(connect_servers))
	this.track_server = make([]string, len(connect_servers))
	this.track_server_addr = make([]*net.UDPAddr, len(connect_servers))
	// slave sends the heartbeat based on its own rythm
	if len(connect_servers) != 0 && this.host.combineAddr("heartbeat") != master.combineAddr("heartbeat") {
		addr, err := net.ResolveUDPAddr("udp", connect_servers[0].combineAddr("heartbeat"))
		checkErr(err)
		this.track_server[0] = this.connect_servers[0].combineAddr("heartbeat")
		this.track_server_addr[0] = addr
		this.serverHBFreq[0] = this.host.heartbeatFreq
	} else {
		// Master keep track of slaves heartbeat based on their frequency
		for idx, server := range connect_servers{
			addr, err := net.ResolveUDPAddr("udp", server.combineAddr("heartbeat"))
			checkErr(err)
			this.track_server[idx] = server.combineAddr("heartbeat")
			this.track_server_addr[idx] = addr
			this.serverHBFreq[idx] = server.heartbeatFreq
		}
	}
	this.timeStamps = make(map[string]time.Time)
	this.lock.Unlock()
}

func (this *HeartBeat) recvAliveMsg(){
	for{

		buffer := make([]byte, 64)
		numBytes, _, err := this.listenSock.ReadFromUDP(buffer)
		checkErr(err)
		recvServerName := string(buffer[:numBytes])

		// Master check whether this is the first message, start sending heartbeat if first 
		if this.host == master {
			if _, ok := this.timeStamps[recvServerName]; !ok {
				fmt.Println("I'm master and this is the first time I talk to ", recvServerName)
				idx := -1
				for i := range this.track_server {
					if this.track_server[i] == recvServerName {
						idx = i
						break
					}
				}
				if idx == -1 {
					fmt.Println("Error finding corresponding slave")
					return
				}
				go this.startTicker(this.serverHBFreq[idx], this.track_server[idx], this.track_server_addr[idx])
			}
		}

		// Update the timer
		this.lock.Lock()
		this.timeStamps[recvServerName] = time.Now()
		this.lock.Unlock()
	}
}

func (this *HeartBeat) startTicker(freq int, connServer string, connServerAddr *net.UDPAddr){
	go func() {
		ticker := time.NewTicker(time.Millisecond * time.Duration(freq))
		for _ = range ticker.C {
			this.lock.Lock()
			// Send message to corresponding slave/master
			this.listenSock.WriteToUDP([]byte(this.host.combineAddr("heartbeat")), connServerAddr)
			
			// To check whether the track servers are still alive
			server := connServer
			latestTime := this.timeStamps[server]
			
			if time.Now().After(latestTime.Add(time.Millisecond * DEAD_DETECT * time.Duration(freq))) {
				fmt.Println("Found a dead server", server)
				delete(this.timeStamps, server)
				// Delete the tracking servers
				for i := range this.track_server {
					if this.track_server[i] == server {
						this.track_server = append(this.track_server[:i], this.track_server[i+1:]...)
						this.track_server_addr = append(this.track_server_addr[:i], this.track_server_addr[i+1:]...)
						break
					}
				}
				this.deadChannel <- server
			}
			this.lock.Unlock()
		}
	}()
}

func (this *HeartBeat) sendAliveMsg(){
	for i:= 0; i< len(this.track_server); i++ {
		go this.startTicker(this.serverHBFreq[i], this.track_server[i], this.track_server_addr[i])
	}
}

func (this *HeartBeat) GetDeadChannel() chan string{
	return this.deadChannel
}

// Test
/*func main() {
	master_addr := "127.0.0.1:10001"
	slave1_addr := "127.0.0.1:10002"
	slave2_addr := "127.0.0.1:10003"
		
	master_addrs := []string{master_addr}
	slave_addrs := []string{slave1_addr, slave2_addr}
	master_heartbeat := new(HeartBeat)
	master_heartbeat.newInstance(master_addr, slave_addrs)
	slave1_heartbeat := new(HeartBeat)
	slave1_heartbeat.newInstance(slave1_addr, master_addrs)
	// remove slave2_heartbeat to test dead function
	// slave2_heartbeat := new(Heartbeat)
	// slave2_heartbeat.Initialize(slave2_addr, master_addrs, master_addrs)

	// need receive at least one notification (packet) to start detection
	time.Sleep(time.Second * 3)
	master_udpaddr, err := net.ResolveUDPAddr("udp", master_addr)
	checkErr(err)
	slave2_udpaddr, err := net.ResolveUDPAddr("udp", slave2_addr)
	checkErr(err)
	socket2, err := net.ListenUDP("udp", slave2_udpaddr)
	checkErr(err)
	socket2.WriteToUDP([]byte(slave2_addr), master_udpaddr)
	
	deadChan := master_heartbeat.GetDeadChannel()
	for {
		dead := <-deadChan
		fmt.Println(dead)
		updated_slave_addrs := []string{slave1_addr}
		master_heartbeat.updateAliveList(updated_slave_addrs)
	}
}*/
