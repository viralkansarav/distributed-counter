package discovery

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/viralkansarav/distributed-counter/pkg/backoff"
)

// ServiceDiscovery manages peer discovery and health checks
type ServiceDiscovery struct {
	mu        sync.Mutex
	ID        string
	peers     map[string]bool
	seenPeers map[string]bool //  Tracks already propagated peers
}

func NewServiceDiscovery(id string, initialPeers []string) *ServiceDiscovery {
	sd := &ServiceDiscovery{
		ID:        id,
		peers:     make(map[string]bool),
		seenPeers: make(map[string]bool),
	}
	for _, peer := range initialPeers {
		sd.peers[peer] = true
	}
	return sd
}

func (sd *ServiceDiscovery) RegisterPeer(peer string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if !sd.peers[peer] {
		sd.peers[peer] = true
		log.Printf("Registered new peer: %s", peer)
	}
}

// Handle a node joining the cluster
func (sd *ServiceDiscovery) HandleJoin(w http.ResponseWriter, r *http.Request) {
	log.Println("Received /join request")

	var newNode struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&newNode); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sd.RegisterPeer(newNode.ID)

	// Delay propagation to avoid race conditions
	time.AfterFunc(500*time.Millisecond, func() {
		sd.PropagateNewPeer(newNode.ID)
	})

	//  Send back the full updated peer list
	peers := sd.GetPeers()
	log.Printf("New peer registered: %s, sending updated peer list: %+v", newNode.ID, peers)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

// Propagate new peer only if it has not already been processed
func (sd *ServiceDiscovery) PropagateNewPeer(newPeer string) {
	sd.mu.Lock()
	if sd.seenPeers[newPeer] {
		sd.mu.Unlock()
		return
	}
	sd.seenPeers[newPeer] = true
	peers := make([]string, 0, len(sd.peers))
	for peer := range sd.peers {
		peers = append(peers, peer)
	}
	sd.mu.Unlock()

	for _, peer := range peers {
		if peer == newPeer {
			continue
		}

		go func(p string) {
			err := backoff.RetryWithBackoff(func() error {
				data := map[string]string{"id": newPeer}
				body, _ := json.Marshal(data)

				resp, err := http.Post("http://"+p+"/join", "application/json", bytes.NewBuffer(body))
				if err != nil {
					log.Printf("Failed to inform %s about new peer %s: %v", p, newPeer, err)
					return err // Retry on failure
				}
				defer resp.Body.Close()
				return nil // Success, no more retries
			})

			if err != nil {
				log.Printf("Failed to inform %s about new peer %s after retries", p, newPeer)
			}
		}(peer)
	}
}

func (sd *ServiceDiscovery) GetPeers() []string {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	peers := make([]string, 0, len(sd.peers))
	for peer := range sd.peers {
		peers = append(peers, peer)
	}
	return peers
}

func (sd *ServiceDiscovery) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (sd *ServiceDiscovery) Heartbeat() {
	for {
		sd.mu.Lock()
		knownPeers := make([]string, 0, len(sd.peers))
		for peer := range sd.peers {
			knownPeers = append(knownPeers, peer)
		}
		sd.mu.Unlock()

		for _, peer := range knownPeers {
			go func(p string) {
				resp, err := http.Get("http://" + p + "/ping")
				if err != nil || resp.StatusCode != http.StatusOK {
					log.Printf("Peer %s is down, removing from list", p)
					sd.mu.Lock()
					delete(sd.peers, p)
					sd.mu.Unlock()
				}
			}(peer)
		}
	}
}

func (sd *ServiceDiscovery) HandlePeers(w http.ResponseWriter, r *http.Request) {
	peers := sd.GetPeers()
	log.Printf("Returning peer list: %+v", peers)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}
