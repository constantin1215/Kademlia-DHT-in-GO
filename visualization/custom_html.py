show_edges_on_click_js = """
<script type="text/javascript">
network.on("selectNode", function (params) {
    var nodeId = params.nodes[0];

    edges.forEach(function(edge) {
        edges.update({ id: edge.id, hidden: true });
    });

    edges.forEach(function(edge) {
        if (edge.from === nodeId) {
            edges.update({ id: edge.id, hidden: false });
        }
    });
});

network.on("deselectNode", function () {
    edges.forEach(function(edge) {
        edges.update({ id: edge.id, hidden: true });
    });
});
</script>
"""

menu_html = """
<style>
#node-menu {
  position: absolute;
  display: none;
  background: white;
  border: 1px solid #ccc;
  padding: 8px;
  font-family: sans-serif;
  font-size: 13px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  z-index: 1000;
}
#node-menu button {
  display: block;
  width: 100%;
  margin: 4px 0;
}
</style>

<div id="node-menu">
  <strong id="menu-title"></strong><br/>

  <button onclick="trigger_ping()">PING</button>
  <button onclick="trigger_data_dump()">SHOW DATA</button>
  <button onclick="trigger_routing_table_dump()">SHOW ROUTING TABLE</button>

  <label for="target-node-select">Target node:</label><br/>
  <select id="target-node-select"></select>
  <button onclick="trigger_find_node()">FIND NODE</button>

  <button onclick="trigger_find_value()">FIND VALUE</button>
  <button onclick="trigger_store()">STORE</button>
</div>


<script type="text/javascript">

var selectedNodeID;

function append_to_terminal(label, data) {
    const term = document.getElementById("terminal");
    term.textContent = "";
    term.innerHTML += label + "\\n";
    term.innerHTML += "<pre>" + JSON.stringify(data, null, 2) + "</pre>";
}

async function trigger_ping() {
    const response = await fetch(`/ping?id=${selectedNodeID}`);
    const data = await response.json();
    append_to_terminal("PING " + selectedNodeID, data);
}

async function trigger_data_dump() {
    const response = await fetch(`/node-data?id=${selectedNodeID}`);
    const data = await response.json();
    append_to_terminal("DATA_DUMP " + selectedNodeID, data);
}

async function trigger_routing_table_dump() {
    const response = await fetch(`/node-routing-table?id=${selectedNodeID}`);
    const data = await response.json();
    append_to_terminal("ROUTING_TABLE_DUMP " + selectedNodeID, data);
}

async function trigger_find_node() {
    const select = document.getElementById("target-node-select");
    const targetId = select.value;

    if (!targetId) return;

    const response = await fetch(
        `/node?source_id=${selectedNodeID}&target_id=${targetId}`
    );

    const data = await response.json();

    append_to_terminal(
        `FIND_NODE from ${selectedNodeID} to ${targetId}`,
        data
    );
}

async function trigger_find_value() {
    const value = prompt("Value to search for:");
    if (!value) return;

    const response = await fetch(
        `/value?source_id=${selectedNodeID}&value=${value}`
    );
    const data = await response.json();
    append_to_terminal(
        `FIND_VALUE from ${selectedNodeID}`,
        data
    );
}

async function trigger_store() {
    const key = prompt("Key:");
    if (!key) return;

    const value = prompt("Value (int):");
    if (!value) return;

    const response = await fetch(
        `/store?source_id=${selectedNodeID}&key=${key}&value=${value}`,
        { method: "POST" }
    );
    const data = await response.json();
    append_to_terminal(
        `STORE from ${selectedNodeID}`,
        data
    );
}

async function populate_node_ids() {
    const response = await fetch("/ids");
    const ids = await response.json();

    const select = document.getElementById("target-node-select");
    select.innerHTML = "";

    ids.forEach(id => {
        const option = document.createElement("option");
        option.value = id;
        option.text = id;
        select.appendChild(option);
    });
}

populate_node_ids();

var menu = document.getElementById("node-menu");
var title = document.getElementById("menu-title");

network.on("selectNode", function (params) {
    var nodeId = params.nodes[0];
    selectedNodeID = nodeId;
    var node = nodes.get(nodeId);

    title.innerText = node.label;
    
    menu.style.position = "fixed";
    menu.style.top = "0%";
    menu.style.right = "0%";
    menu.style.display = "block";
    
    populate_node_ids();
});

network.on("deselectNode", function () {
    menu.style.display = "none";
});
</script>
"""

terminal_html = """
<div style="margin-bottom:6px;">
  <button onclick="clearTerminal()" style="
      background:#222;
      color:#f5f5f5;
      border:1px solid #444;
      padding:4px 10px;
      font-family:'Courier New', Courier, monospace;
      font-size:13px;
      cursor:pointer;
  ">
    clear
  </button>
</div>

<div id="terminal" style="
    background-color:#0d0d0d; 
    color:#f5f5f5; 
    font-family:'Courier New', Courier, monospace; 
    font-size:14px;
    line-height:1.4em;
    white-space:pre-line;
    padding:15px; 
    width:100%; 
    height:29vh;
    overflow-y:auto; 
    box-shadow:0 0 10px rgba(0,0,0,0.5); 
    border-radius:5px; 
    border:1px solid #333;
">
</div>

<script>
function clearTerminal() {
    document.getElementById("terminal").textContent = "";
}
</script>
"""

refresh_html = """
<script>
async function refreshGraph() {
    const res = await fetch("/");
    const html = await res.text();

    const container = document.body;

    container.innerHTML = "";
}
</script>
"""