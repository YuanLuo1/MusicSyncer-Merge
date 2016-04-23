package main
import(
	//"strings"
	//"fmt"
)
/*type Group struct{
	name string
	serverList map[string]bool	
}

func (g Group) addServer(ip string) {
	g.serverList[ip] = true
}

func (g Group) delServer(ip string) {
	delete(g.serverList, ip)
}


func (g Group) setName(name string) {
	g.name = name
}

func (g Group) getServerList() []string{
	keys := make([]string, 0, len(g.serverList))
	for k:= range g.serverList{
		keys = append(keys, k)
	}
	return keys
}*/

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

/*func getServerListByGroupname(groupName string) []string {
	return clusterMap[groupMap[groupName]]
}*/

func getServerListByClustername(clusterName string) []string {
	newList := clusterMap[clusterName]
	list := make([]string, len(newList)-1)
	for i:= range newList {
		if newList[i] != myServer.combineAddr("comm"){
			list = append(list, newList[i])
		}
	}
	return list
}
