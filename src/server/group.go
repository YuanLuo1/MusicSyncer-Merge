package main
import(
	"strings"
	"fmt"
)
type Group struct{
	name string
	serverList map[string]bool	
}

func (this Group) addServer(ip string) {
	this.serverList[ip] = true
}

func (this Group) delServer(ip string) {
	delete(this.serverList, ip)
}


func (this Group) setName(name string) {
	this.name = name
}


type GroupMusic struct{
	name string
	musicList []string
	//music MusicList
}

func (this GroupMusic) addMusic(music string) {
	this.musicList = append(this.musicList, music)
}

func (this GroupMusic) delMusic(music string) {
	for i:= range this.musicList {
		if this.musicList[i] == music {
			//f.musicList = append(f.musicList[:i], f.musicList[i+1:])
		}
	}
}

func isGroupNameExist(groupName string) bool{
	for i := range groups {
		if strings.Compare(groups[i].name, groupName) == 0 {
			fmt.Println("debug")
			return true
		}
	}
	return false
}

func isGroupHere(groupName string) bool {
	if hasGroups[groupName]{
		return true
	} else {
		return false
	}
}