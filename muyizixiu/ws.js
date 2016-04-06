var url="ws://"+IP+":81/"+location.search;
var ws=new WebSocket(url);
	ws.onopen = function(event) { 

  // 发送一个初始化消息
  //socket.send(""); 

  // 监听消息
  ws.onmessage = function(event) { 
    //console.log('Client received a message',event); 
    data=JSON.parse(event.data)
    //console.log(data)
    if(data.end){
    	//alert("your enemy has fleed away!");
	lock=true;
		return
    }
    if(data.error){
      console.log(data.error)
    	 window.location.href="http://"+IP+"/muyizixiu/ufo-war.html?gName=chess";
	 return
    }
    if(data.start){
      G.lock=false;
	    //alert("the battle has been lauched!");
	   //alert("开始了，点击就送!")
     ufo.move();
     setTimeout(function(){
		     enemy.x=G.nextData.enemy.x;
		     enemy.y=G.nextData.enemy.y;
		     console.log("enemy",enemy)
		     enemy.move();
		     },2000);
     setTimeout(G.freshFrame,500);
     return
    }
    if(data.rid){
      return
    }
    G.nextData.enemy=data.ufo;
    console.log("ufo",data.ufo);
  }
}
