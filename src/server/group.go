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
	newList := clusterMap[clusterName]
	list := make([]Server, len(newList)-1)
	for i:= range newList {
		if newList[i] != myServer {
			list = append(list, newList[i])
		}
	}
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