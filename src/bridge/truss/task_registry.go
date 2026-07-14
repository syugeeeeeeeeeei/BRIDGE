package truss

import "sync"

type TaskState string

const (
	TaskRunnable  TaskState = "runnable"
	TaskRunning   TaskState = "running"
	TaskPaused    TaskState = "paused"
	TaskCompleted TaskState = "completed"
	TaskCancelled TaskState = "cancelled"
)

type TaskRecord struct {
	ID, ParentID string
	Epoch        uint64
	State        TaskState
}
type TaskRegistry struct {
	mu    sync.RWMutex
	tasks map[string]TaskRecord
}

func NewTaskRegistry() *TaskRegistry     { return &TaskRegistry{tasks: map[string]TaskRecord{}} }
func (r *TaskRegistry) Put(t TaskRecord) { r.mu.Lock(); defer r.mu.Unlock(); r.tasks[t.ID] = t }
func (r *TaskRegistry) Transition(id string, to TaskState) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return false
	}
	t.State = to
	r.tasks[id] = t
	return true
}
func (r *TaskRegistry) Get(id string) (TaskRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	return t, ok
}
