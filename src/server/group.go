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

func getServerListByClusterName(clusterName string) []string {
	newList := clusterMap[clusterName]
	list := make([]string, len(newList)-1)
	for i:= range newList {
		if newList[i] != myServer.combineAddr("comm"){
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