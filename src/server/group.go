package main
import(
	"fmt"
)

func isGroupNameExist(groupName string) bool{
	if _,ok := groupMap[groupName]; ok {
		return true
	}
	return false
}

func isGroupHere(groupName string) bool {
	if _, ok := hasGroups[groupName]; ok{
		return true
	}
	return false
}

func getServerListByClusterName(clusterName string) []Server {
	mapLock.Lock()
	newList := clusterMap[clusterName]
	fmt.Println(newList)
	list := make([]Server, 0)
	for i:= range newList {
		if newList[i].name != myServer.name {
			list = append(list, newList[i])
		}
	}
	mapLock.Unlock()
	return list
}

func createNewGroupLocal(groupName string, clusterName string) {
	groupMap[groupName] = clusterName 
    hasGroups[groupName] = true
    
    newList := new(MusicList)
    newList.NewInstance()
    newList.name = groupName
    musicList = append(musicList, *newList)
    
    fmt.Println("[Create] Group Map", groupMap)
	fmt.Println("[Create] Has Groups", hasGroups)
	fmt.Println("[Create] Music List", musicList)
}