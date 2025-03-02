package counter

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viralkansarav/distributed-counter/internal/discovery"
)

func TestIncrementCounter(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)

	// Initial count should be 0
	assert.Equal(t, 0, c.value)

	// Simulate an increment from localhost:8000
	c.Increment("localhost:8000", 1)
	assert.Equal(t, 1, c.value)

	// Simulate a duplicate increment (should not increase count)
	c.Increment("localhost:8000", 1)
	assert.Equal(t, 1, c.value)

	// Simulate another increment from a different node
	c.Increment("localhost:8001", 2)
	assert.Equal(t, 2, c.value)
}

func TestHandleIncrement(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)

	reqBody := `{"node_id":"localhost:8000"}`
	req := httptest.NewRequest("POST", "/increment", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c.HandleIncrement(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, c.value) // Ensure counter incremented
}

func TestHandleGetCount(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)
	c.value = 5

	req := httptest.NewRequest("GET", "/count", nil)
	rec := httptest.NewRecorder()

	c.HandleGetCount(rec, req)

	expectedResponse := `{"count":5}`
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, expectedResponse, rec.Body.String())
}

func TestSetCount(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)

	// Setting count higher should update
	c.SetCount(10)
	assert.Equal(t, 10, c.value)

	// Setting count lower should NOT update
	c.SetCount(5)
	assert.Equal(t, 10, c.value)
}
