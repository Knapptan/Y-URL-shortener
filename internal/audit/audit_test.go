package audit

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObserver struct {
	events []Event
}

func (o *testObserver) Notify(event Event) {
	o.events = append(o.events, event)
}

func TestPublisher(t *testing.T) {
	publisher := NewPublisher()
	first := &testObserver{}
	second := &testObserver{}
	publisher.Register(first)
	publisher.Register(nil)
	publisher.Register(second)

	event := NewEvent(ActionShorten, "user-1", "https://ya.ru")
	publisher.Publish(event)

	assert.Equal(t, []Event{event}, first.events)
	assert.Equal(t, []Event{event}, second.events)
}

func TestPublisherNil(t *testing.T) {
	var publisher *Publisher

	assert.NotPanics(t, func() {
		publisher.Publish(NewEvent(ActionShorten, "", "https://ya.ru"))
	})
}

func TestNewFileObserverError(t *testing.T) {
	_, err := NewFileObserver(filepath.Join(t.TempDir(), "missing", "audit.log"))
	assert.Error(t, err)
}

func TestFileObserver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	require.NoError(t, os.WriteFile(path, []byte("existing\n"), 0644))

	observer, err := NewFileObserver(path)
	require.NoError(t, err)

	first := NewEvent(ActionShorten, "user-1", "https://ya.ru")
	second := NewEvent(ActionFollow, "", "https://google.com")
	observer.Notify(first)
	observer.Notify(second)
	require.NoError(t, observer.Close())

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	require.NoError(t, scanner.Err())
	require.Len(t, lines, 3)
	assert.Equal(t, "existing", lines[0])

	var events []Event
	for _, line := range lines[1:] {
		var event Event
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		events = append(events, event)
	}
	assert.Equal(t, []Event{first, second}, events)
}

func TestFileObserverAddsNewlineAfterExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0644))

	observer, err := NewFileObserver(path)
	require.NoError(t, err)

	event := NewEvent(ActionShorten, "user-1", "https://ya.ru")
	observer.Notify(event)
	require.NoError(t, observer.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "existing", lines[0])

	var got Event
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &got))
	assert.Equal(t, event, got)
}

func TestPublisherNotifiesAllReceivers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	fileObserver, err := NewFileObserver(path)
	require.NoError(t, err)
	defer fileObserver.Close()

	var httpEvent Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&httpEvent))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewPublisher()
	publisher.Register(fileObserver)
	publisher.Register(NewHTTPObserver(server.URL))

	event := NewEvent(ActionFollow, "user-1", "https://ya.ru")
	publisher.Publish(event)

	assert.Equal(t, event, httpEvent)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var fileEvent Event
	require.NoError(t, json.Unmarshal(content, &fileEvent))
	assert.Equal(t, event, fileEvent)
}

func TestFileObserverWriteError(t *testing.T) {
	observer, err := NewFileObserver(filepath.Join(t.TempDir(), "audit.log"))
	require.NoError(t, err)
	require.NoError(t, observer.Close())

	assert.NotPanics(t, func() {
		observer.Notify(NewEvent(ActionShorten, "user-1", "https://ya.ru"))
	})
}

func TestHTTPObserver(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)
	event := NewEvent(ActionFollow, "user-1", "https://ya.ru")
	observer.Notify(event)

	assert.Equal(t, event, received)
}

func TestHTTPObserverNon2xx(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)
	assert.NotPanics(t, func() {
		observer.Notify(NewEvent(ActionShorten, "", "https://ya.ru"))
	})
	assert.True(t, called)
}

func TestHTTPObserverPostError(t *testing.T) {
	observer := NewHTTPObserver("://invalid-url")

	assert.NotPanics(t, func() {
		observer.Notify(NewEvent(ActionShorten, "", "https://ya.ru"))
	})
}
