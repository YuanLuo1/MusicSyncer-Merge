package main

import (
	"fmt"
    "io"
    "os"
    "encoding/csv"
    "strconv"
    //"net/http"
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
	clusterMap = make(map[string][]Server)
	masterServer = make(map[string]Server)
	//file, err:= os.Open("https://s3.amazonaws.com/ds-rujia/initServers.csv")
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
			fmt.Println("[init] Error", err)
			return
		}
		
		//                     ip,     serverName,   comm,     http,    heartbeat, cluster, HB Freq, FilePort,  backup Port
		newServer :=  Server{record[1], record[0],record[2], record[3], record[4], record[5], -1, record[8], record[9]} 
		i, err := strconv.Atoi(record[7])
		if err != nil {
			fmt.Println("error in parsing heart beat freq to integer")
			continue
		}
		newServer.heartbeatFreq = i
		servers = append(servers, newServer) 
		
		clusterMap[record[5]] = append(clusterMap[record[5]], newServer)
		
		if record[6] == "Y" {
			masterServer[record[5]] = newServer
		}
		
    }
	fmt.Println("[init-server] Cluster Map: ", clusterMap)
	fmt.Println("[init-server] Servers: ", servers)
	fmt.Println("[init-server] master servers: ", masterServer)
}

func InitialHeartBeat(master Server){
    fmt.Println("[init-heartbeat] heartbeat at port", myServer.heartbeat_port)
    // If this server is the master, track all the slaves
    var hbServers []Server
    if myServer.name == master.name{
    	hbServers = make([]Server, len(clusterMap[myServer.cluster])-1)
    	x := 0
    	for i:= range clusterMap[myServer.cluster] {
	    	if clusterMap[myServer.cluster][i] != myServer {
	    		hbServers[x] = clusterMap[myServer.cluster][i]
	    		x ++
	    	}
	    }
    } else {
	    // Slaves: only keep track of the master
    
    	hbServers = make([]Server, 1)
    	hbServers[0] = master
    	
    }
    fmt.Println("[init-heartbeat]",hbServers)
    
    heartBeatTracker.newInstance(myServer, hbServers)
    // Also InitialMulticaster
    multicaster.Initiallized(myServer, clusterMap[myServer.cluster])

}