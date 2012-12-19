console.log("view.coffee loaded")

class Graph
  constructor: () ->
    @data = []
    @element = document.createElement("div")
    @rickshaw = new Rickshaw.Graph({
      element: @element,
      width: 700,
      height: 100,
      series: @data
    })
    document.body.appendChild(@element)

  record: (value) ->
    time = (new Date()).getTime() / 1000.0;
    @data.append({ x: time, y: value })
    @rickshaw.render
# class Graph

class GraphList
  constructor: () -> 
    @graphs = {}

  record: (identity, value) ->
    @graphs[identity] ||= new Graph
    @graphs[identity].record(value)
# GraphList
    
class LogStashSocket
  constructor: (@url, @callback) -> 
    @connect()

  connect: () ->
    console.log("Connecting to " + @url)
    socket = new WebSocket(@url)

    socket.onopen = (event) => console.log("Connected!")
    socket.onerror = (event) =>
      console.log("websocket error: " + event)
      retry = () => @connect()
      setTimeout(retry, 1000)
      socket.close()

    socket.onmessage = (event) =>
      obj = JSON.parse(event.data)
      @callback(obj);

graphlist = new GraphList()
socket = new LogStashSocket("ws://demo.logstash.net:3232/", (event) =>
  metrics = event["@fields"]
  for key, value in metrics
    [root, name, metric] = key.split(".")
    continue if root != "age" && metric != "mean"
    graphlist.record(name, value)
