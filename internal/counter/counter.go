package counter

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/viralkansarav/distributed-counter/internal/discovery"
	"github.com/viralkansarav/distributed-counter/pkg/backoff"
)

type Counter struct {
	mu      sync.Mutex
	value   int
	history map[string]map[int]bool
	sd      *discovery.ServiceDiscovery
}

func NewCounter(sd *discovery.ServiceDiscovery) *Counter {
	return &Counter{
		history: make(map[string]map[int]bool),
		sd:      sd,
	}
}

// Set the latest count when syncing from a peer
func (c *Counter) SetCount(latestValue int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if latestValue > c.value {
		c.value = latestValue
		log.Printf("Counter synchronized to latest value: %d", c.value)
	}
}

// Process increments only once per request
func (c *Counter) Increment(nodeID string, requestID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.history[nodeID]; !exists {
		c.history[nodeID] = make(map[int]bool)
	}

	if c.history[nodeID][requestID] {
		log.Printf("Request %d from %s already processed, skipping.", requestID, nodeID)
		return
	}

	c.history[nodeID][requestID] = true
	c.value++
	log.Printf("Counter incremented by node %s, new value: %d", nodeID, c.value)
}

// Handle incoming increment requests
func (c *Counter) HandleIncrement(w http.ResponseWriter, r *http.Request) {
	var data struct {
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	requestID := rand.Intn(1000000)
	c.Increment(data.NodeID, requestID)

	// Propagate increment only if this is the original request
	if data.NodeID == c.sd.ID {
		go c.PropagateIncrement(data.NodeID)
	}

	w.WriteHeader(http.StatusOK)
}

// Propagate increments with exponential backoff
func (c *Counter) PropagateIncrement(originNode string) {
	peers := c.sd.GetPeers()
	log.Printf("Propagating increment to peers: %+v", peers)

	for _, peer := range peers {
		if peer == originNode {
			continue
		}

		go func(p string) {
			err := backoff.RetryWithBackoff(func() error {
				data := map[string]string{"node_id": c.sd.ID}
				body, _ := json.Marshal(data)

				resp, err := http.Post("http://"+p+"/increment", "application/json", bytes.NewBuffer(body))
				if err != nil {
					log.Printf("Failed to propagate to %s: %v", p, err)
					return err // Retry on failure
				}
				defer resp.Body.Close()
				return nil // Success, no more retries
			})

			if err != nil {
				log.Printf("Failed to propagate increment to %s after retries", p)
			}
		}(peer)
	}
}

// Get the current counter value
func (c *Counter) HandleGetCount(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Printf("Returning count: %d", c.value)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": c.value})
}
