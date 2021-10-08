var conn;
async function initWS(){
  if(conn) return
  if (window["WebSocket"]) {
    conn = new WebSocket("ws://" + document.location.host + "/api")
    conn.onclose = function (evt) {
      appendLog(createLog("<b>Connection closed.</b>"))
      conn = null;
    }
    conn.onmessage = onMessage;
  } else console.log("<b>Your browser does not support WebSockets.</b>")
  console.log("initWS")
}
function sendData(data){
  conn.send(JSON.stringify(data))
}