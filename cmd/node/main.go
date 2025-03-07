package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/viralkansarav/distributed-counter/internal/counter"
	"github.com/viralkansarav/distributed-counter/internal/discovery"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("NODE_ID is required")
	}

	peersEnv := os.Getenv("PEERS")
	var initialPeers []string
	if peersEnv != "" {
		initialPeers = strings.Split(peersEnv, ",")
	}

	sd := discovery.NewServiceDiscovery(nodeID, initialPeers)
	c := counter.NewCounter(sd)

	// Fetch latest count from a peer if available
	if len(initialPeers) > 0 {
		go syncCounterOnStartup(c, initialPeers)
	}

	// Automatically join known peers and update stored peers
	if len(initialPeers) > 0 {
		go autoJoinCluster(sd, nodeID, initialPeers)
	}

	// Start background processes
	go func() {
		for {
			sd.Heartbeat()
			time.Sleep(5 * time.Second)
		}
	}()

	//  Register API endpoints
	http.HandleFunc("/join", sd.HandleJoin)
	http.HandleFunc("/ping", sd.HandlePing)
	http.HandleFunc("/increment", c.HandleIncrement)
	http.HandleFunc("/count", c.HandleGetCount)
	http.HandleFunc("/peers", sd.HandlePeers)

	log.Printf("Node %s is running...", nodeID)
	log.Fatal(http.ListenAndServe(nodeID, nil))
}

// Fetch the latest counter value from peers when a node restarts
func syncCounterOnStartup(c *counter.Counter, peers []string) {
	for _, peer := range peers {
		resp, err := http.Get("http://" + peer + "/count")
		if err != nil {
			log.Printf("Failed to fetch count from %s: %v", peer, err)
			continue
		}
		defer resp.Body.Close()

		var result map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			latestCount := result["count"]
			log.Printf("Synced latest count from %s: %d", peer, latestCount)
			c.SetCount(latestCount) // Set the latest count
			break
		}
	}
}

// Ensure a node registers itself and updates its peer list with backoff
func autoJoinCluster(sd *discovery.ServiceDiscovery, nodeID string, knownPeers []string) {
	for _, peer := range knownPeers {
		log.Printf("Attempting to join cluster via %s...", peer)
		peer = strings.TrimSpace(peer)

		data := map[string]string{"id": nodeID}
		body, _ := json.Marshal(data)

		retries := 5                        //  Max retries
		baseDelay := 500 * time.Millisecond //  Initial delay

		for i := 0; i < retries; i++ {
			resp, err := http.Post("http://"+peer+"/join", "application/json", bytes.NewBuffer(body))
			if err == nil {
				defer resp.Body.Close()

				// Read and store the peer list from the response
				var peerList []string
				if err := json.NewDecoder(resp.Body).Decode(&peerList); err == nil {
					log.Printf("Received updated peer list from %s: %+v", peer, peerList)
					for _, newPeer := range peerList {
						sd.RegisterPeer(newPeer)
					}
					return
				}
			}

			//  Apply exponential backoff
			delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
			delay += time.Duration(rand.Intn(100)) * time.Millisecond
			log.Printf("Failed to join via %s. Retrying in %v...", peer, delay)
			time.Sleep(delay)
		}

		log.Printf("Failed to join cluster via %s after retries", peer)
	}
}
