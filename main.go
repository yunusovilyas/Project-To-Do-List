package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type TaskStore struct {
	sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

var taskStore = NewTaskStore()

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "Добро пожаловать в API Списка Задач! Используйте /api/tasks для доступа к API.")
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		http.Error(w, "Неверное тело запроса", http.StatusBadRequest)
		return
	}

	taskStore.Lock()
	task.ID = taskStore.nextID
	taskStore.tasks[task.ID] = task
	taskStore.nextID++
	taskStore.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	taskStore.RLock()
	tasks := make([]Task, 0, len(taskStore.tasks))
	for _, task := range taskStore.tasks {
		tasks = append(tasks, task)
	}
	taskStore.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/tasks/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
		return
	}

	var updatedTask Task
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		http.Error(w, "Неверное тело запроса", http.StatusBadRequest)
		return
	}

	taskStore.Lock()
	if existingTask, exists := taskStore.tasks[id]; exists {
		updatedTask.ID = id
		if updatedTask.Title != "" {
			existingTask.Title = updatedTask.Title
		}
		existingTask.Completed = updatedTask.Completed
		taskStore.tasks[id] = existingTask
		taskStore.Unlock()
		json.NewEncoder(w).Encode(existingTask)
		return
	}
	taskStore.Unlock()

	http.Error(w, "Задача не найдена", http.StatusNotFound)
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/tasks/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
		return
	}

	taskStore.Lock()
	if _, exists := taskStore.tasks[id]; exists {
		delete(taskStore.tasks, id)
		taskStore.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	taskStore.Unlock()

	http.Error(w, "Задача не найдена", http.StatusNotFound)
}

func handleMarkTaskAsDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/tasks/") : len(r.URL.Path)-len("/done")]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
		return
	}

	taskStore.Lock()
	if task, exists := taskStore.tasks[id]; exists {
		task.Completed = true
		taskStore.tasks[id] = task
		taskStore.Unlock()
		json.NewEncoder(w).Encode(task)
		return
	}
	taskStore.Unlock()

	http.Error(w, "Задача не найдена", http.StatusNotFound)
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTasks(w, r)
		case http.MethodPost:
			handleCreateTask(w, r)
		default:
			http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if len(r.URL.Path) > len("/api/tasks/") && r.URL.Path[len(r.URL.Path)-len("/done"):] == "/done" {
				handleMarkTaskAsDone(w, r)
			} else {
				handleUpdateTask(w, r)
			}
		case http.MethodDelete:
			handleDeleteTask(w, r)
		default:
			http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
