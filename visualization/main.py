import json
import traceback

import networkx as nx
from dotenv import load_dotenv
from fastapi import FastAPI
from google.protobuf.json_format import MessageToJson
from pyvis.network import Network
from starlette.responses import HTMLResponse

from DockerAPI import DockerAPI
from KademliaClient import KademliaClient
from custom_html import show_edges_on_click_js, menu_html, terminal_html

load_dotenv()

app = FastAPI()

kademlia_client = KademliaClient()
node_container_info = {}

@app.get("/ping")
async def ping(id: str):
    for node_id in node_container_info.keys():
        if id == node_id:
            result = await kademlia_client.PING(node_container_info[node_id])
            return json.loads(MessageToJson(result))
    return None

@app.get("/ids")
async def get_ids():
    return list(node_container_info.keys())

@app.get("/node")
async def find_node(source_id: str, target_id: str):
    for node_id in node_container_info.keys():
        if source_id == node_id:
            result = await kademlia_client.FIND_NODE(node_container_info[node_id], target_id)
            return json.loads(MessageToJson(result))
    return None

@app.get("/value")
async def find_value(source_id: str, value: str):
    for node_id in node_container_info.keys():
        if source_id == node_id:
            result = await kademlia_client.FIND_VALUE(node_container_info[node_id], value)
            return json.loads(MessageToJson(result))
    return None

@app.post("/store")
async def store(source_id: str, key: str, value: int):
    for node_id in node_container_info.keys():
        if source_id == node_id:
            result = await kademlia_client.STORE(node_container_info[node_id], key, value)
            return json.loads(MessageToJson(result))
    return None

@app.get("/node-data")
async def node_data(id: str):
    for node_id in node_container_info.keys():
        if id == node_id:
            result = await kademlia_client.DATA_DUMP(node_container_info[node_id])
            return json.loads(MessageToJson(result))
    return None

@app.get("/node-routing-table")
async def node_routing_table(id: str):
    for node_id in node_container_info.keys():
        if id == node_id:
            result = await kademlia_client.ROUTING_TABLE_DUMP(node_container_info[node_id])
            return json.loads(MessageToJson(result))
    return None

@app.get("/", response_class=HTMLResponse)
def main_page():
    return """
    <html>
    <head>
    <title>Kademlia view</title>
    </head>
    <script>
        function refreshIframe(id) {
            const iframe = document.getElementById(id);
            iframe.src = iframe.src; 
        }
        
        setInterval(refreshIframe, 2000, 'data-frame')
    </script>
    <style>
        .container {
            display: grid;
            grid-template-columns: 0.70fr 0.30fr;
            grid-template-rows: 1fr;
            grid-column-gap: 0px;
            grid-row-gap: 0px;
        }
            
        .graph { grid-area: 1 / 1 / 2 / 2; }
        .data-overview { grid-area: 1 / 2 / 2 / 3; }
    </style>
    <body>
         <div class="container">
            <div class="graph"> 
                <button onclick="refreshIframe('graph-frame')" style="
                  margin-bottom:6px;
                  background:#222;
                  color:#f5f5f5;
                  border:1px solid #444;
                  padding:4px 10px;
                  font-family:monospace;
                  cursor:pointer;
                ">
                  refresh graph
                </button>
                
                <iframe
                  id="graph-frame"
                  src="/graph"
                  style="
                    width:100%;
                    height:95vh;
                    border:1px solid #333;
                    border-radius:4px;
                    background:#0d0d0d;
                  ">
                </iframe>
            </div>
            <div class="data-overview">
                <iframe
                  id="data-frame"
                  src="/data-overview"
                  style="
                    width:100%;
                    height:98vh;
                    border:1px solid #333;
                    border-radius:4px;
                    background: white;
                    margin-left: 5px;
                  ">
                </iframe>
            </div>
        </div> 
    </body>
    </html>
    """

@app.get("/data-overview", response_class=HTMLResponse)
async def data_overview():
    try:
        docker_api = DockerAPI()
        containers = docker_api.list_kademlia_containers()
        replicas = {}
        values = {}
        nodes = {}
        versions = {}

        for container in containers:
            container_data = await kademlia_client.DATA_DUMP(container)
            container_details = await kademlia_client.PING(container)
            for key, value in container_data.pairs.items():
                replicas[key] = replicas.get(key, 0) + 1
                leaser = "*" if container_details.id in container_data.leasers.get(key, []) else ""
                nodes[key] = nodes.get(key, []) + [container[0][14:-2] + leaser]
                versions[key] = versions.get(key, []) + [str(container_data.versions.get(key, []))]
                values[key] = values.get(key, []) + [str(container_data.pairs.get(key, []))]

        rows = []
        for key in sorted(replicas.keys()):
            rows.append(
                f"<tr><td>{str(key)[:6]}...</td><td>{','.join(values[key])}</td><<td>{replicas[key]}</td><td>{','.join(nodes[key])}</td><td>{','.join(versions[key])}</td></tr>"
            )

        table_html = f"""
            <div style="border:1px solid #444; padding:6px; text-align:center; font-family:monospace; font-size:13px;">Global Data View</div>
            <table style="
                width:100%;
                border-collapse: collapse;
                font-family: monospace;
                font-size: 13px;
            ">
              <thead>
                <tr>
                  <th style="border:1px solid #444; padding:6px; text-align:left;">data</th>
                  <th style="border:1px solid #444; padding:6px; text-align:left;">values</th>
                  <th style="border:1px solid #444; padding:6px; text-align:left;">replicas</th>
                  <th style="border:1px solid #444; padding:6px; text-align:left;">nodes</th>
                  <th style="border:1px solid #444; padding:6px; text-align:left;">versions</th>
                </tr>
              </thead>
              <tbody>
                {''.join(rows)}
              </tbody>
            </table>
            """

        return f"""
        <!DOCTYPE html>
        <html>
        <head>
          <meta charset="utf-8">
          <title>Data Overview</title>
        </head>
        <body style="margin:0; padding:10px; background:#0d0d0d; color:#f5f5f5;">
          {table_html}
        </body>
        </html>
        """
    except:
        traceback.print_exc()
        return "Loading..."

@app.get("/graph", response_class=HTMLResponse)
async def graph():
    docker_api = DockerAPI()
    containers = docker_api.list_kademlia_containers()

    G = nx.DiGraph()
    edges = []

    for container in containers:
        info = await kademlia_client.PING(container)
        node_container_info[info.id] = container
        G.add_node(info.id, label=f"Node {info.id[:6]}...\n{container[0]}")
        routing_table = await kademlia_client.ROUTING_TABLE_DUMP(container)
        for key, value in routing_table.pairs.items():
            if len(value.nodes) != 0:
                for node in value.nodes:
                    if info.id != node.id:
                        edges.append((info.id, node.id))

    G.add_edges_from(edges)

    nodes_sorted = sorted(G.nodes(), key=lambda x: int(x, 16))

    G_sorted = nx.DiGraph()
    G_sorted.add_nodes_from(nodes_sorted)
    G_sorted.add_edges_from(G_sorted.edges())

    pos = nx.circular_layout(G_sorted, scale=300)

    net = Network(height="600px", width="100%", directed=True)
    net.toggle_physics(False)

    for node, (x, y) in pos.items():
        net.add_node(
            node,
            label=G.nodes[node].get("label", "FAULTY"),
            x=float(x),
            y=float(y),
            fixed=True
        )

    edge_id = 0
    edge_ids = {}
    for u, v in G.edges():
        net.add_edge(u, v, id=edge_id, hidden=True)
        edge_ids[edge_id] = (u, v)
        edge_id += 1

    net.toggle_physics(True)

    html = net.generate_html()

    html = html.replace("</body>", show_edges_on_click_js + "\n</body>")
    html = html.replace("</body>", menu_html + "\n</body>")
    html = html.replace("</body>", terminal_html + "\n</body>")

    return html