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

	// ✅ Initial count should be 0
	assert.Equal(t, 0, c.value)

	// ✅ Simulate incrementing by 5 from localhost:8000
	c.Increment("localhost:8000", 1, 5)
	assert.Equal(t, 5, c.value)

	// ✅ Duplicate request from same node should not increment
	c.Increment("localhost:8000", 1, 5)
	assert.Equal(t, 5, c.value)

	// ✅ Another increment of 3 from a different node
	c.Increment("localhost:8001", 2, 3)
	assert.Equal(t, 8, c.value) // ✅ 5 + 3 = 8

	// ✅ Increment from a new node with a different request ID
	c.Increment("localhost:8002", 3, 2)
	assert.Equal(t, 10, c.value) // ✅ 8 + 2 = 10
}

func TestHandleIncrement(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)

	// ✅ Simulate an HTTP request to increment by 4
	reqBody := `{"node_id":"localhost:8000", "increment_count": 4}`
	req := httptest.NewRequest("POST", "/increment", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c.HandleIncrement(rec, req)

	// ✅ Ensure the request was processed successfully
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"success": true}`, rec.Body.String())

	// ✅ Ensure counter is incremented by 4
	assert.Equal(t, 4, c.value)
}

func TestHandleGetCount(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)
	c.value = 15

	req := httptest.NewRequest("GET", "/count", nil)
	rec := httptest.NewRecorder()

	c.HandleGetCount(rec, req)

	expectedResponse := `{"count":15}`
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, expectedResponse, rec.Body.String())
}

func TestSetCount(t *testing.T) {
	sd := discovery.NewServiceDiscovery("localhost:8000", []string{})
	c := NewCounter(sd)

	// ✅ Setting count to a higher value should update
	c.SetCount(20)
	assert.Equal(t, 20, c.value)

	// ✅ Setting count to a lower value should NOT update
	c.SetCount(10)
	assert.Equal(t, 20, c.value) // ✅ Remains 20
}
