package client

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/server"
)

func TestInMemoryTaskManager_CreateTask(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if taskID == "" {
		t.Fatalf("CreateTask returned empty taskID")
	}
}

func TestInMemoryTaskManager_GetTask(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	task, err := tm.GetTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.ID != taskID {
		t.Fatalf("GetTask returned wrong taskID")
	}
}

func TestInMemoryTaskManager_UpdateTask(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	err = tm.UpdateTask(context.Background(), taskID, server.TaskStatusInProgress, nil, nil)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}
	task, err := tm.GetTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status.State != server.TaskStatusInProgress {
		t.Fatalf("UpdateTask did not update status")
	}
}

func TestInMemoryTaskManager_DeleteTask(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	err = tm.DeleteTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	_, err = tm.GetTask(context.Background(), taskID)
	if err == nil {
		t.Fatalf("GetTask succeeded after DeleteTask")
	}
}

func TestInMemoryTaskManager_ListTasks(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	taskID1, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	taskID2, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	tasks, err := tm.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("ListTasks returned wrong number of tasks")
	}

	found1, found2 := false, false
	for _, task := range tasks {
		if task.ID == taskID1 {
			found1 = true
		}
		if task.ID == taskID2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Fatalf("ListTasks did not return expected tasks")
	}
}

func TestInMemoryTaskManager_ThreadSafety(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	var wg sync.WaitGroup
	numRoutines := 100
	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func(i int) {
			defer wg.Done()
			taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
			if err != nil {
				t.Errorf("CreateTask failed: %v", err)
				return
			}
			err = tm.UpdateTask(context.Background(), taskID, server.TaskStatusInProgress, nil, nil)
			if err != nil {
				t.Errorf("UpdateTask failed: %v", err)
				return
			}
			_, err = tm.GetTask(context.Background(), taskID)
			if err != nil {
				t.Errorf("GetTask failed: %v", err)
				return
			}
		}(i)
	}
	wg.Wait()
}

func TestInMemoryTaskManager_TaskExpiry(t *testing.T) {
	tm := server.NewInMemoryTaskManager(nil)
	tm.SetTaskExpiry(1 * time.Second)
	taskID, err := tm.CreateTask(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = tm.GetTask(context.Background(), taskID)
	if err == nil {
		t.Fatalf("Expected task to be expired, but GetTask succeeded")
	}
}
