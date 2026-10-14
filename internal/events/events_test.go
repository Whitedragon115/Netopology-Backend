package events_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func nextEvent(t *testing.T, scanner *bufio.Scanner) events.Event {
	t.Helper()
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var e events.Event
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &e); err != nil {
				t.Fatal(err)
			}
			return e
		}
	}
	t.Fatalf("stream ended: %v", scanner.Err())
	return events.Event{}
}

func TestStreamVisibilityRedactionAndResume(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	var ws struct{ Items []struct{ ID uuid.UUID } }
	owner.Do(http.MethodGet, "/workspaces", nil).JSON(t, &ws)
	root := "/workspaces/" + ws.Items[0].ID.String()
	var tab struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/canvases", map[string]any{"name": "Map"}).Expect(t, 201).JSON(t, &tab)
	viewer := api.ClientAs("viewer")
	editor := api.ClientAs("editor")
	var me struct{ ID uuid.UUID }
	editor.Do(http.MethodGet, "/me", nil).JSON(t, &me)
	owner.Do(http.MethodPut, root+"/members/"+me.ID.String()+"/canvas-grants", map[string]any{"canvasIds": []uuid.UUID{tab.ID}}).Expect(t, 200)
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Second)
	defer cancel()
	vr := viewer.Stream(ctx, root+"/events?canvasId="+tab.ID.String())
	defer vr.Body.Close()
	er := editor.Stream(ctx, root+"/events")
	defer er.Body.Close()
	if vr.StatusCode != http.StatusOK || vr.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("response %v", vr)
	}
	vs := bufio.NewScanner(vr.Body)
	if !vs.Scan() || vs.Text() != ": ok" || !vs.Scan() || vs.Text() != "" {
		t.Fatalf("missing immediate opening comment: %q, %v", vs.Text(), vs.Err())
	}
	// The editor must skip this unplaced subject before receiving the visible one.
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Invisible", "category": "network"}).Expect(t, 201)
	var node struct{ ID uuid.UUID }
	owner.Do(http.MethodPost, root+"/nodes", map[string]any{"name": "Visible", "category": "network", "placeOn": map[string]any{"canvasId": tab.ID, "x": 0, "y": 0}, "properties": []map[string]any{{"key": "secret", "label": "Secret", "value": "DO-NOT-SEND", "sensitive": true}}}).Expect(t, 201).JSON(t, &node)
	// Canvas filtering excludes the workspace inventory event with no canvas relevance.
	event := nextEvent(t, vs)
	if data, ok := event.Data.(map[string]any); !ok || data["name"] != "Invisible" {
		t.Fatalf("workspace-wide event %+v", event)
	}
	event = nextEvent(t, vs)
	raw, _ := json.Marshal(event.Data)
	if strings.Contains(string(raw), "DO-NOT-SEND") {
		t.Fatal("sensitive value in SSE")
	}
	es := bufio.NewScanner(er.Body)
	ee := nextEvent(t, es)
	data := ee.Data.(map[string]any)
	if data["id"] != node.ID.String() {
		t.Fatalf("editor got invisible event %+v", ee)
	}
	_ = vr.Body.Close()
	owner.Do(http.MethodPatch, root+"/nodes/"+node.ID.String(), map[string]any{"name": "Updated"}).Expect(t, 200)
	resumed := viewer.Stream(ctx, root+"/events", "Last-Event-ID: "+event.ID.String())
	defer resumed.Body.Close()
	replay := nextEvent(t, bufio.NewScanner(resumed.Body))
	if replay.Type != "node.updated" {
		t.Fatalf("resume %+v", replay)
	}
	api.CreateUser("outside", "admin").Do(http.MethodGet, root+"/events", nil).Expect(t, 404)
	editor.Do(http.MethodGet, root+"/events?canvasId="+uuid.NewString(), nil).Expect(t, 404)
}
