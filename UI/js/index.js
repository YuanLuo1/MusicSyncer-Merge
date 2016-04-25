//<script>
$(document).ready(function(){
  var playlist =[];
  {{range $key, $value := .FilesMap}}
  playlist.push({
    title:"{{$key}}",
    mp3:"{{$value}}",
    poster: "images/1.jpg"
  })
  {{end}}
  var cssSelector = {
    jPlayer: "#jquery_jplayer",
    cssSelectorAncestor: ".music-player"
  };
  
  var options = {
    swfPath: "Jplayer.swf",
    supplied: "ogv, m4v, oga, mp3"
  };
  
  var myPlaylist = new jPlayerPlaylist(cssSelector, playlist, options);
  
});

//</script>