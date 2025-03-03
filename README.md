# Distributed Counter System  

A **distributed counter system** built in **Go** using **peer discovery**, **eventual consistency**, and **exponential backoff retries** for reliable communication.

---

## **🚀 Features**

✅ **Service Discovery:** Nodes dynamically register and detect each other.  
✅ **Eventual Consistency:** Each node maintains a counter that syncs across peers.  
✅ **Automatic Recovery:** Failed nodes sync data when they come back online.  
✅ **Exponential Backoff:** Retries failed requests with increasing delays.  
---

## **📌 How It Works**

### **1️⃣ Node Registration & Discovery**

- Each node **registers itself** with an ID (e.g., `localhost:8000`).  
- Nodes **send heartbeats** (`/ping`) to detect failures.  
- Nodes **notify peers** when a new node joins.  

### **2️⃣ Counter Propagation**

- When a node **increments** its counter, it **propagates** the update to peers.  
- If a peer is **offline**, the update is retried using **exponential backoff**.  

### **3️⃣ Handling Failures**

- Nodes **retry failed requests** with increasing delays (`500ms → 1s → 2s → 4s`).  
- When a failed node **comes back online**, it **syncs the latest counter value**.  

---

## **📌 Running the System**

### **1️⃣ Start the First Node**

```sh
export NODE_ID="localhost:8000"
export PEERS=""
go run cmd/node/main.go
```

### Start Additional Nodes

```sh
export NODE_ID="localhost:8001"
export PEERS="localhost:8000"
go run cmd/node/main.go
```

```sh
export NODE_ID="localhost:8002"
export PEERS="localhost:8000,localhost:8001"
go run cmd/node/main.go
```
### 3️⃣ Join a New Node to the Cluster

```sh
export NODE_ID="localhost:8003"
export PEERS=""
go run cmd/node/main.go
```

### API Endpoints

🔹 Get List of Peers
```sh
curl -X GET http://localhost:8000/peers
```

#### Response

```sh
["localhost:8001", "localhost:8002", "localhost:8003"]
```
## How the System Handles Failures

### If a Node Goes Down
The node is removed from the peer list after multiple failed heartbeats.
When the node comes back online, it fetches the latest counter value.
### If a Peer is Down During an Increment
The update is retried using exponential backoff.
If the peer comes back online, it syncs missed updates.
### If a New Node Joins Late
The new node fetches the latest counter value.
It is added to the peer list and notified to all other peers.




