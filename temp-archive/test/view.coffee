console.log("view.coffee loaded")

class Graph
  constructor: () ->
    @data = []
    @element = document.createElement("div")
    console.log("New graph")

  record: (value) ->
    time = (new Date()).getTime() / 1000.0;
    @data.append({ x: time, y: value })
    if !@rickshaw
      @rickshaw = new Rickshaw.Graph({
        element: @element,
        width: 700,
        height: 100,
        series: @data
      })
      document.body.appendChild(@element)
    @rickshaw.render
# class Graph

class GraphList
  constructor: () -> 
    @graphs = {}

  record: (identity, value) ->
    @graphs[identity] ||= new Graph
    console.log(identity)
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
      socket.close()
      retry = () => @connect()
      setTimeout(retry, 1000)

    socket.onmessage = (event) =>
      obj = JSON.parse(event.data)
      @callback(obj)

graphlist = new GraphList()
callback = (event) =>
  console.log(event)
  metrics = event["@fields"]
  for key, value of metrics
    [root, name, metric] = key.split(".")
    continue if root != "age" && metric != "mean"
    console.log(name + ": " + value)
    graphlist.record(name, value)

socket = new LogStashSocket("ws://" + document.location.hostname + ":3232/",
                            callback)
