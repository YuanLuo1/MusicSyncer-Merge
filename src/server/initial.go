package main

import (
	"fmt"
    "io"
    "os"
    "encoding/csv"
)

func readMusicConfig(){
	file, err:= os.Open("./initMusic.csv")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	defer file.Close()
	
	reader := csv.NewReader(file)	
	for {
		record, err := reader.Read()
		if err == io.EOF {
	    	break
		} else if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		//TODO: initial music list
		groupName := record[0]
		musicFile := record[1]
		if hasGroups[groupName] {
			mList := new(MusicList)
			mList.NewInstance()
			mList.name = groupName
			mList.add(musicFile)
		}
		
    }
}

func readGroupConfig(){
	hasGroups = make(map[string]bool)
	groupMap = make(map[string]string)
	file, err:= os.Open("./initGroups.csv")
	if err != nil {
		fmt.Println("[init] Error: ", err)
		return
	}
	defer file.Close()
	
	reader := csv.NewReader(file)	
	for {
		record, err := reader.Read()
		if err == io.EOF {
	    	break
		} else if err != nil {
			fmt.Println("[init] Error: ", err)
			return
		}
		/*var newGroup Group
		newGroup.name = record[0]
		newGroup.serverList = make(map[string]bool)
		newGroup.addServer(record[1])
		groups = append(groups, newGroup)*/
		
		if(record[1] == myServer.cluster){
			hasGroups[record[0]] = true
		}
		groupMap[record[0]] = record[1]
    }
	fmt.Println("[init-group] Group Map: ", groupMap)
	fmt.Println("[init-group] Has Groups: ", hasGroups)
}

func readServerConfig(){
	clusterMap = make(map[string][]string)
	file, err:= os.Open("./initServers.csv")
	if err != nil {
		fmt.Println("[init] Error: ", err)
		return
	}
	defer file.Close()
	
	reader := csv.NewReader(file)	
	for {
		record, err := reader.Read()
		if err == io.EOF {
	    	break
		} else if err != nil {
			fmt.Println("[init] Error: ", err)
			return
		}
		newServer :=  Server{record[1], record[2], record[3], record[4], record[5]} //ip, comm, http, heartbeat, cluster
		servers = append(servers, newServer) 
		
		clusterMap[record[5]] = append(clusterMap[record[5]], newServer.combineAddr("comm"))
		
    }
	fmt.Println("[init-server] Cluster Map: ", clusterMap)
	fmt.Println("[init-server] Servers: ", servers)	
}

func InitialHeartBeat(){
	//TODO: make sure how to do heartbeat
    fmt.Println("[init-heartbeat] heartbeat at port", myServer.heartbeat_port)
    // argument : (myIP, other servers)
    //var hbServers []string
    hbServers := make([]string, len(servers)-1)
    for i:= range servers {
    	if servers[i] != myServer {
    		hbServers = append(hbServers, servers[i].combineAddr("heartbeat"))
    	}
    }
    fmt.Println("[init-heartbeat]",hbServers)
    
    heartBeatTracker.newInstance(myServer.combineAddr("heartbeat"), hbServers)

}