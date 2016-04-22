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

func getServerListByGroupname(groupName string) []string {
	return clusterMap[groupMap[groupName]]
}

func getServerListByClustername(clusterName string) []string {
	return clusterMap[clusterName]
}
