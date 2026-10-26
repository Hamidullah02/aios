package webui

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// stubSubmitter is a TaskSubmitter that records what it was
// asked to do and does nothing else.
//
// It exists so the server's task endpoint can be tested without
// a planner, a model provider, or a tool bus. The point under
// test is the HTTP contract: what the operator sends, what the
// server answers, and what the browser is told about the
// server's capability.
type stubSubmitter struct {
	goals []string
	ids   []string
	ready bool
	err   error
	next  int
}

func (s *stubSubmitter) Submit(goal string) (string, error) {
	s.goals = append(s.goals, goal)
	if s.err != nil {
		return "", s.err
	}
	s.next++
	id := "task-" + string(rune('a'+s.next-1))
	s.ids = append(s.ids, id)
	return id, nil
}

func (s *stubSubmitter) CanSubmit() bool { return s.ready }

// --- the endpoint ---

// TestPostTask_AcceptsAPlainSentence is the core contract: the
// operator posts one sentence and gets a task back.
//
// Before this endpoint the browser could only post a message
// between agents. Nothing in the product consumed that message,
// so there was no way at all to give the product something to
// do.
func TestPostTask_AcceptsAPlainSentence(t *testing.T) {
	s, _ := newTestServer(t)
	sub := &stubSubmitter{ready: true}
	s.hub.SetSubmitter(sub)

	w := do(t, s, "POST", "/api/tasks", `{"goal":"make my shop show up on Google"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/tasks = %d: %s", w.Code, w.Body.String())
	}
	var res struct {
		ID   string `json:"id"`
		Goal string `json:"goal"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.ID == "" {
		t.Error("the response carried no task id")
	}
	if res.Goal != "make my shop show up on Google" {
		t.Errorf("goal = %q, want the operator's own words echoed back", res.Goal)
	}
	if len(sub.goals) != 1 || sub.goals[0] != "make my shop show up on Google" {
		t.Errorf("the submitter saw %v, want the goal verbatim", sub.goals)
	}
}

// TestPostTask_EmptyGoalIsRejected pins that a blank request is
// refused rather than sent to the planner.
//
// An empty goal cannot be planned, and the planner's failure for
// an empty goal is a confusing "plan generation failed". The
// error has to say what is actually wrong.
func TestPostTask_EmptyGoalIsRejected(t *testing.T) {
	s, _ := newTestServer(t)
	sub := &stubSubmitter{ready: true}
	s.hub.SetSubmitter(sub)

	for _, body := range []string{
		`{"goal":""}`,
		`{"goal":"   "}`,
		`{}`,
	} {
		w := do(t, s, "POST", "/api/tasks", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("POST %s = %d, want 400", body, w.Code)
		}
	}
	if len(sub.goals) != 0 {
		t.Errorf("the submitter saw %v, want nothing", sub.goals)
	}
}

// TestPostTask_RecordsTheGoalInTheConversation pins that a goal
// typed into the composer joins the thread the planner reads.
//
// The composer is the front door, and it is the only place a
// non-coder types. If the goal never reached the conversation, a
// second request ("now do the same for q4") would have nothing
// to refer back to, and the "a plan made from the conversation"
// promise would hold for the message box but not for the box
// people actually use.
func TestPostTask_RecordsTheGoalInTheConversation(t *testing.T) {
	s, hub := newTestServer(t)
	hub.SetSubmitter(&stubSubmitter{ready: true})

	w := do(t, s, "POST", "/api/tasks", `{"goal":"the numbers are in reports/q3.csv"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/tasks = %d: %s", w.Code, w.Body.String())
	}
	msgs := hub.Messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	if msgs[0].From != OperatorSender {
		t.Errorf("message from = %q, want %q", msgs[0].From, OperatorSender)
	}
	if msgs[0].Body != "the numbers are in reports/q3.csv" {
		t.Errorf("message body = %q, want the operator's own words", msgs[0].Body)
	}
}

// TestPostTask_ServerWithoutASubmitterSaysSo pins that a
// read-only server does not accept a goal and drop it.
//
// The old behaviour was the worst kind: the browser POSTed, the
// server said nothing, and the operator was left staring at a
// page that looked idle. The error names the actual limitation.
func TestPostTask_ServerWithoutASubmitterSaysSo(t *testing.T) {
	s, _ := newTestServer(t)
	w := do(t, s, "POST", "/api/tasks", `{"goal":"do a thing"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/tasks = %d, want 503", w.Code)
	}
	var res struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Error, "cannot start tasks") {
		t.Errorf("error = %q, want it to say the server cannot start tasks", res.Error)
	}
}

// TestPostTask_SubmitterThatIsNotReadyIsRefused pins the
// difference between "no submitter" and "a submitter that cannot
// take work right now". Both are refused; both say so.
func TestPostTask_SubmitterThatIsNotReadyIsRefused(t *testing.T) {
	s, _ := newTestServer(t)
	s.hub.SetSubmitter(&stubSubmitter{ready: false})
	w := do(t, s, "POST", "/api/tasks", `{"goal":"do a thing"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/tasks = %d, want 503", w.Code)
	}
}

// TestPostTask_InvalidJSONIsRejected pins that a malformed body
// is a 400 and not a 500.
func TestPostTask_InvalidJSONIsRejected(t *testing.T) {
	s, _ := newTestServer(t)
	s.hub.SetSubmitter(&stubSubmitter{ready: true})
	w := do(t, s, "POST", "/api/tasks", `{not json`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/tasks = %d, want 400", w.Code)
	}
}

// --- the capability flag ---

// TestState_CarriesCanSubmitTasks pins that the browser learns
// whether this server can start work.
//
// The UI needs this before the operator types anything: a
// disabled Start button with a reason is a far better answer
// than a button that fails after they have written their
// request.
func TestState_CarriesCanSubmitTasks(t *testing.T) {
	s, _ := newTestServer(t)

	w := do(t, s, "GET", "/api/state", "")
	if !strings.Contains(w.Body.String(), `"can_submit_tasks":false`) {
		t.Errorf("a server with no submitter must report can_submit_tasks:false, got %s", w.Body.String())
	}

	s.hub.SetSubmitter(&stubSubmitter{ready: true})
	w = do(t, s, "GET", "/api/state", "")
	if !strings.Contains(w.Body.String(), `"can_submit_tasks":true`) {
		t.Errorf("a server with a submitter must report can_submit_tasks:true, got %s", w.Body.String())
	}
}

// TestState_CarriesTasks pins that the task list is part of the
// snapshot, so a browser that connects mid-run sees the work
// already in flight.
func TestState_CarriesTasks(t *testing.T) {
	s, _ := newTestServer(t)
	s.hub.TaskStore().SetStatus("task-1", TaskRunning, "working on it")

	w := do(t, s, "GET", "/api/state", "")
	var snap Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(snap.Tasks))
	}
	if snap.Tasks[0].ID != "task-1" || snap.Tasks[0].Status != TaskRunning {
		t.Errorf("task = %+v", snap.Tasks[0])
	}
}

// TestGetTasks_ListsThem pins the list endpoint.
func TestGetTasks_ListsThem(t *testing.T) {
	s, _ := newTestServer(t)
	s.hub.TaskStore().SetStatus("task-1", TaskDone, "done")

	w := do(t, s, "GET", "/api/tasks", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks = %d", w.Code)
	}
	var res struct {
		Tasks     []Task `json:"tasks"`
		CanSubmit bool   `json:"can_submit"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Tasks) != 1 || res.Tasks[0].ID != "task-1" {
		t.Errorf("tasks = %+v", res.Tasks)
	}
}

// --- the operator's message starts a plan ---

// TestPostMessage_FromTheOperatorStartsATask is the promise the
// whole product rests on: the one endpoint a client already
// knows how to call turns a sentence into work.
//
// Before this, `POST /api/messages` was a log. The message was
// accepted, written to the communication panel, and nothing
// consumed it -- so a non-coder could type into the browser and
// the system would sit there doing nothing.
func TestPostMessage_FromTheOperatorStartsATask(t *testing.T) {
	s, hub := newTestServer(t)
	sub := &stubSubmitter{ready: true}
	hub.SetSubmitter(sub)

	w := do(t, s, "POST", "/api/messages",
		`{"from":"operator","body":"make my shop show up on Google"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST message = %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Message Message `json:"message"`
		TaskID  string  `json:"task_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.TaskID == "" {
		t.Error("the response carried no task id")
	}
	if len(sub.goals) != 1 || sub.goals[0] != "make my shop show up on Google" {
		t.Errorf("the submitter saw %v, want the operator's words", sub.goals)
	}
	// The message is still in the log: the operator's words
	// belong in the conversation the planner reads.
	msgs := hub.Messages()
	if len(msgs) != 1 || msgs[0].From != OperatorSender {
		t.Errorf("messages = %+v, want one from the operator", msgs)
	}
}

// TestPostMessage_FromAnAgentDoesNotStartATask pins the other
// side of the line. An agent's message is a note between
// agents: nothing consumes those, so starting a plan from one
// would turn the agent chatter into a queue of accidental runs.
func TestPostMessage_FromAnAgentDoesNotStartATask(t *testing.T) {
	s, hub := newTestServer(t)
	sub := &stubSubmitter{ready: true}
	hub.SetSubmitter(sub)

	w := do(t, s, "POST", "/api/messages", `{"from":"t1","body":"take it"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST message = %d: %s", w.Code, w.Body.String())
	}
	if len(sub.goals) != 0 {
		t.Errorf("an agent's message started work: %v", sub.goals)
	}
}

// TestPostMessage_OperatorMessageOnAServerThatCannotWork pins
// the honest refusal. A server with no planner accepts the
// message and says so, rather than taking the request and
// leaving the operator waiting for work that will never start.
func TestPostMessage_OperatorMessageOnAServerThatCannotWork(t *testing.T) {
	s, _ := newTestServer(t)

	w := do(t, s, "POST", "/api/messages", `{"from":"operator","body":"do a thing"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST message = %d, want 503", w.Code)
	}
	var res struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Error, "cannot start tasks") {
		t.Errorf("error = %q, want it to say the server cannot start tasks", res.Error)
	}
}

// TestPostMessage_OperatorMessageThatIsRefusedIsReported pins
// that a refusal reaches the operator. The submitter refuses a
// second run while one is in flight, and that refusal has to
// arrive as an answer rather than as a success.
func TestPostMessage_OperatorMessageThatIsRefusedIsReported(t *testing.T) {
	s, hub := newTestServer(t)
	hub.SetSubmitter(&stubSubmitter{ready: true, err: errors.New("a task is already running; wait for it to finish")})

	w := do(t, s, "POST", "/api/messages", `{"from":"operator","body":"do a thing"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST message = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "already running") {
		t.Errorf("body = %s, want the refusal in words", w.Body.String())
	}
}

// --- the task store ---

// TestTaskStore_EchoesTheGoal pins the promise the UI makes: the
// operator's own words appear on the card.
//
// An operator who asked for something and came back to the page
// has to be able to tell whether the job on screen is the job
// they asked for.
func TestTaskStore_EchoesTheGoal(t *testing.T) {
	store := NewTaskStore()
	store.Upsert("task-1", func(t *Task) {
		t.Goal = "make my shop show up on Google"
		t.Status = TaskPlanning
	})
	tasks := store.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks))
	}
	if tasks[0].Goal != "make my shop show up on Google" {
		t.Errorf("goal = %q", tasks[0].Goal)
	}
	if tasks[0].StartedAt.IsZero() {
		t.Error("a task must record when it was submitted")
	}
}

// TestTaskStore_TerminalStatusStampsFinishedAt pins that a
// finished task says when it finished, so the UI can show how
// long it took without tracking it itself.
func TestTaskStore_TerminalStatusStampsFinishedAt(t *testing.T) {
	store := NewTaskStore()
	store.SetStatus("task-1", TaskRunning, "")
	store.SetStatus("task-1", TaskDone, "done")
	tasks := store.Tasks()
	if tasks[0].FinishedAt.IsZero() {
		t.Error("a done task must have a finish time")
	}
}

// TestTaskStore_RunningDoesNotStampFinishedAt pins the other
// half: work in flight has no end time yet.
func TestTaskStore_RunningDoesNotStampFinishedAt(t *testing.T) {
	store := NewTaskStore()
	store.SetStatus("task-1", TaskRunning, "working")
	tasks := store.Tasks()
	if !tasks[0].FinishedAt.IsZero() {
		t.Error("a running task must not have a finish time")
	}
}

// TestTaskStore_TasksAreOldestFirst pins the ordering the UI
// relies on: the history reads top to bottom.
func TestTaskStore_TasksAreOldestFirst(t *testing.T) {
	store := NewTaskStore()
	store.Upsert("task-1", func(t *Task) { t.Goal = "first" })
	time.Sleep(2 * time.Millisecond)
	store.Upsert("task-2", func(t *Task) { t.Goal = "second" })

	tasks := store.Tasks()
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	if tasks[0].ID != "task-1" || tasks[1].ID != "task-2" {
		t.Errorf("order = %s, %s; want task-1 then task-2", tasks[0].ID, tasks[1].ID)
	}
}

// TestTaskStore_SubscribeReceivesUpdates pins that the live feed
// carries task changes, so an open browser sees a task move
// from planning to running without reloading.
func TestTaskStore_SubscribeReceivesUpdates(t *testing.T) {
	store := NewTaskStore()
	ch, cancel := store.subscribe()
	defer cancel()

	store.SetStatus("task-1", TaskRunning, "working")

	select {
	case ev := <-ch:
		if ev.Kind != "task" {
			t.Fatalf("event kind = %q, want task", ev.Kind)
		}
		var task Task
		if err := json.Unmarshal(ev.Payload, &task); err != nil {
			t.Fatal(err)
		}
		if task.ID != "task-1" || task.Status != TaskRunning {
			t.Errorf("task = %+v", task)
		}
	case <-time.After(time.Second):
		t.Fatal("no task event was published")
	}
}

// --- the panel ---

// TestLayout_TasksPanelIsKnown pins that the tasks panel is a
// first-class panel, not an orphan template.
//
// A template with no panel entry renders nowhere: the operator
// would submit a task and never see the card.
func TestLayout_TasksPanelIsKnown(t *testing.T) {
	if !KnownPanel(PanelTasks) {
		t.Fatal("tasks panel is not registered in AllPanels")
	}
	found := false
	for _, p := range DefaultLayout().Panels {
		if p.ID == PanelTasks {
			found = true
		}
	}
	if !found {
		t.Error("the tasks panel is missing from the default layout")
	}
}
