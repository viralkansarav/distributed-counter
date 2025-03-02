package discovery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterAndGetPeers(t *testing.T) {
	sd := NewServiceDiscovery("localhost:8000", []string{})

	// Initially, no peers
	assert.Empty(t, sd.GetPeers())

	// Register a new peer
	sd.RegisterPeer("localhost:8001")
	assert.Equal(t, []string{"localhost:8001"}, sd.GetPeers())

	// Register another peer
	sd.RegisterPeer("localhost:8002")
	assert.ElementsMatch(t, []string{"localhost:8001", "localhost:8002"}, sd.GetPeers())
}

func TestHandleJoin(t *testing.T) {
	sd := NewServiceDiscovery("localhost:8000", []string{})

	reqBody := `{"id":"localhost:8001"}`
	req := httptest.NewRequest("POST", "/join", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	sd.HandleJoin(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var peers []string
	err := json.Unmarshal(rec.Body.Bytes(), &peers)
	assert.NoError(t, err)
	assert.Contains(t, peers, "localhost:8001")
}

func TestHandlePeers(t *testing.T) {
	sd := NewServiceDiscovery("localhost:8000", []string{"localhost:8001", "localhost:8002"})

	req := httptest.NewRequest("GET", "/peers", nil)
	rec := httptest.NewRecorder()

	sd.HandlePeers(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var peers []string
	err := json.Unmarshal(rec.Body.Bytes(), &peers)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"localhost:8001", "localhost:8002"}, peers)
}
