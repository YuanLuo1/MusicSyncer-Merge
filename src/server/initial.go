package main

import (
	"fmt"
    "io"
    "os"
    "encoding/csv"
)

func readMusicConfig(){ //clear
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
			mList := getMusicList(groupName)
			mList.add(musicFile)
		}
    }
	fmt.Println("[init] music list: ", musicList)
}

func readGroupConfig(){ //clear
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
		groupName := record[0] 
		clusterName := record[1]
		if(clusterName == myServer.cluster){
			hasGroups[groupName] = true
			mList := new(MusicList)
			mList.NewInstance()
			mList.name = groupName
			musicList = append(musicList, *mList)
		}
		groupMap[groupName] = clusterName
    }
	fmt.Println("[init-group] Group Map: ", groupMap)
	fmt.Println("[init-group] Has Groups: ", hasGroups)
}

func readServerConfig(){ //clear
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
		newServer :=  Server{record[1], record[0],record[2], record[3], record[4], record[5]} //ip, comm, http, heartbeat, cluster
		servers = append(servers, newServer) 
		
		clusterMap[record[5]] = append(clusterMap[record[5]], newServer.combineAddr("comm"))
		
    }
	fmt.Println("[init-server] Cluster Map: ", clusterMap)
	fmt.Println("[init-server] Servers: ", servers)	
}

func InitialHeartBeat(master Server){
    fmt.Println("[init-heartbeat] heartbeat at port", myServer.heartbeat_port)
    // If this server is the master, track all the slaves
    if myServer.name == master.name{
    	hbServers := make([]string, len(servers)-1)
    	for i:= range servers {
	    	if servers[i] != myServer {
	    		hbServers = append(hbServers, servers[i].combineAddr("heartbeat"))
	    	}
	    }
    }
    // Slaves: only keep track of the master
    else {
    	hbServers := make([]string, 1)
    	hbServers = append(hbServers, master.combineAddr("heartbeat"))
    }
    fmt.Println("[init-heartbeat]",hbServers)
    
    heartBeatTracker.newInstance(myServer.combineAddr("heartbeat"), hbServers)

    // Also InitialMulticaster
    multicaster.Initiallized(myServer, servers)

}