// main.go - Simple Easy Tasks with basic HTTP server
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	PriorityMedium = "medium"
	StatusTodo     = "todo"
)

// writeResponse writes a response and handles errors
func writeResponse(w http.ResponseWriter, data []byte) {
	if _, err := w.Write(data); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Task represents a task in our system
type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// In-memory storage for demo purposes
var tasks = []Task{
	{
		ID:          1,
		Title:       "Design user interface",
		Description: "Create wireframes and mockups",
		Status:      StatusTodo,
		Priority:    PriorityMedium,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	},
	{
		ID:          2,
		Title:       "Set up database",
		Description: "Configure PostgreSQL",
		Status:      StatusTodo,
		Priority:    "high",
		CreatedAt:   time.Now().Add(-12 * time.Hour),
		UpdatedAt:   time.Now().Add(-12 * time.Hour),
	},
	{
		ID:          3,
		Title:       "Implement authentication",
		Description: "Add login/logout functionality",
		Status:      "inprogress",
		Priority:    "high",
		CreatedAt:   time.Now().Add(-6 * time.Hour),
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
	},
	{
		ID:          4,
		Title:       "Project setup",
		Description: "Initialize Go project with HTMGO",
		Status:      "done",
		Priority:    PriorityMedium,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-36 * time.Hour),
	},
}

var nextTaskID = 5

func main() {
	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Page routes
	http.HandleFunc("/", indexHandler)

	// API routes
	http.HandleFunc("/api/tasks", tasksHandler)
	http.HandleFunc("/api/tasks/create", createTaskHandler)
	http.HandleFunc("/api/tasks/search", searchTasksHandler)
	http.HandleFunc("/api/tasks/validate", validateTaskHandler)

	// HTMX partial routes
	http.HandleFunc("/partials/task-form", taskFormHandler)
	http.HandleFunc("/partials/task-card", taskCardHandler)
	http.HandleFunc("/partials/task-list", taskListHandler)

	// Progressive enhancement fallback routes
	http.HandleFunc("/tasks/create", fallbackCreateTaskHandler)
	http.HandleFunc("/tasks", fallbackTasksHandler)

	fmt.Println("Task Management Server starting on :8081")
	fmt.Println("Visit: http://localhost:8081")

	server := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Simple Easy Tasks</title>
    <meta name="csrf-token" content="demo-csrf-token">
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="/static/css/htmx-styles.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" defer></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.2/gsap.min.js" defer></script>
    <script src="/static/js/htmx-config.js" defer></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto p-6">
        <header class="mb-8 flex justify-between items-center">
            <div>
                <h1 class="text-3xl font-bold text-gray-800 mb-2">Simple Easy Tasks</h1>
                <p class="text-gray-600">Current time: {{.Time}}</p>
            </div>
            <div class="space-x-4">
                <button 
                    class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors duration-200"
                    hx-get="/partials/task-form"
                    hx-target="#modal-container"
                    hx-swap="innerHTML"
                    onclick="this.href='/tasks/create'"
                    aria-label="Add new task">
                    + Add Task
                </button>
                <noscript>
                    <a href="/tasks/create" 
                       class="inline-block bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg 
                              transition-colors duration-200">
                        + Add Task
                    </a>
                </noscript>
                <div class="relative inline-block">
                    <input type="text" 
                           name="q"
                           placeholder="Search tasks..." 
                           class="pl-10 pr-4 py-2 border border-gray-300 rounded-lg 
                                  focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                           hx-get="/api/tasks/search"
                           hx-trigger="keyup changed delay:300ms"
                           hx-target="#task-board"
                           hx-indicator="#search-loading">
                    <div class="absolute left-3 top-2.5 text-gray-400">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
                        </svg>
                    </div>
                    <div id="search-loading" class="htmx-indicator absolute right-3 top-2.5">
                        <div class="spinner w-4 h-4"></div>
                    </div>
                </div>
            </div>
        </header>
        
        <div id="task-board" 
             hx-get="/partials/task-list" 
             hx-trigger="load"
             hx-swap="innerHTML"
             class="grid grid-cols-1 md:grid-cols-3 gap-6"
             role="application"
             aria-label="Task management board">
            <!-- Loading state -->
            <div class="col-span-3 text-center py-8">
                <div class="spinner mx-auto mb-4"></div>
                <p class="text-gray-600">Loading tasks...</p>
            </div>
        </div>
        
        <footer class="mt-8 text-center text-sm text-gray-500">
            <p>Simple Easy Tasks - Task Management System</p>
        </footer>
    </div>
    
    <!-- Modal Container -->
    <div id="modal-container"></div>
    
    <!-- Notification Container -->
    <div id="notification-container" class="fixed top-4 right-4 z-50"></div>
</body>
</html>`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Time string
	}{
		Time: time.Now().Format("2006-01-02 15:04:05"),
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// tasksHandler handles API requests for tasks
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": tasks,
			"count": len(tasks),
		}); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createTaskHandler handles task creation
func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	priority := r.FormValue("priority")
	status := r.FormValue("status")

	// Validation
	if title == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(w, []byte(`<div class="error-message text-red-600 text-sm">Title is required</div>`))
		return
	}

	if priority == "" {
		priority = PriorityMedium
	}
	if status == "" {
		status = StatusTodo
	}

	// Create new task
	newTask := Task{
		ID:          nextTaskID,
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextTaskID++

	// Add to tasks slice
	tasks = append(tasks, newTask)

	// Return the new task card
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger", "taskCreated")

	taskCardHTML := generateTaskCardHTML(newTask)
	writeResponse(w, []byte(taskCardHTML))
}

// searchTasksHandler handles task search
func searchTasksHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.FormValue("q")))
	if query == "" {
		query = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	}

	var filteredTasks []Task
	if query == "" {
		filteredTasks = tasks
	} else {
		for _, task := range tasks {
			if strings.Contains(strings.ToLower(task.Title), query) ||
				strings.Contains(strings.ToLower(task.Description), query) {
				filteredTasks = append(filteredTasks, task)
			}
		}
	}

	w.Header().Set("Content-Type", "text/html")
	taskListHTML := generateTaskListHTML(filteredTasks)
	writeResponse(w, []byte(taskListHTML))
}

// validateTaskHandler handles real-time validation
func validateTaskHandler(w http.ResponseWriter, r *http.Request) {
	field := r.URL.Query().Get("field")
	value := strings.TrimSpace(r.FormValue(field))

	w.Header().Set("Content-Type", "text/html")

	switch field {
	case "title":
		switch {
		case value == "":
			w.WriteHeader(http.StatusBadRequest)
			writeResponse(w, []byte(`<div class="field-error text-red-600 text-sm mt-1">Title is required</div>`))
		case len(value) < 3:
			w.WriteHeader(http.StatusBadRequest)
			writeResponse(w, []byte(`<div class="field-error text-red-600 text-sm mt-1">Title must be at least 3 characters</div>`))
		default:
			writeResponse(w, []byte(`<div class="field-success text-green-600 text-sm mt-1">✓ Title looks good</div>`))
		}
	default:
		writeResponse(w, []byte(""))
	}
}

// taskFormHandler returns the task creation form modal
func taskFormHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	formHTML := `
<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 modal-backdrop">
    <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4 modal-content">
        <div class="flex justify-between items-center mb-4">
            <h3 class="text-lg font-medium text-gray-900">Create New Task</h3>
            <button class="text-gray-400 hover:text-gray-600" 
                    onclick="document.getElementById('modal-container').innerHTML = ''">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
            </button>
        </div>
        
        <form hx-post="/api/tasks/create" 
              hx-target="#task-board" 
              hx-swap="innerHTML"
              hx-on::after-request="if(event.detail.successful) { 
                        document.getElementById('modal-container').innerHTML = ''; 
                        htmx.trigger('#task-board', 'load'); 
                    }"
              class="space-y-4">
              
            <div>
                <label for="title" class="block text-sm font-medium text-gray-700 mb-1">
                    Title *
                </label>
                <input type="text" 
                       id="title" 
                       name="title" 
                       required
                       class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                       hx-post="/api/tasks/validate?field=title"
                       hx-trigger="blur"
                       hx-target="next .validation-message">
                <div class="validation-message"></div>
            </div>
            
            <div>
                <label for="description" class="block text-sm font-medium text-gray-700 mb-1">
                    Description
                </label>
                <textarea id="description" 
                          name="description" 
                          rows="3"
                          class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500"></textarea>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
                <div>
                    <label for="priority" class="block text-sm font-medium text-gray-700 mb-1">
                        Priority
                    </label>
                    <select id="priority" 
                            name="priority"
                            class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
                        <option value="low">Low</option>
                        <option value="medium" selected>Medium</option>
                        <option value="high">High</option>
                        <option value="critical">Critical</option>
                    </select>
                </div>
                
                <div>
                    <label for="status" class="block text-sm font-medium text-gray-700 mb-1">
                        Status
                    </label>
                    <select id="status" 
                            name="status"
                            class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
                        <option value="todo" selected>To Do</option>
                        <option value="inprogress">In Progress</option>
                        <option value="done">Done</option>
                    </select>
                </div>
            </div>
            
            <div class="flex justify-end space-x-3 pt-4">
                <button type="button" 
                        class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 
                               hover:bg-gray-200 rounded-md transition-colors"
                        onclick="document.getElementById('modal-container').innerHTML = ''">
                    Cancel
                </button>
                <button type="submit"
                        class="px-4 py-2 text-sm font-medium text-white bg-blue-600 
                               hover:bg-blue-700 rounded-md transition-colors">
                    Create Task
                </button>
            </div>
        </form>
    </div>
</div>`

	writeResponse(w, []byte(formHTML))
}

// taskCardHandler returns a single task card
func taskCardHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	for _, task := range tasks {
		if task.ID == id {
			w.Header().Set("Content-Type", "text/html")
			taskCardHTML := generateTaskCardHTML(task)
			writeResponse(w, []byte(taskCardHTML))
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

// taskListHandler returns the complete task list organized by status
func taskListHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	taskListHTML := generateTaskListHTML(tasks)
	writeResponse(w, []byte(taskListHTML))
}

// Helper functions
func generateTaskCardHTML(task Task) string {
	priorityClass := getPriorityClass(task.Priority)

	return fmt.Sprintf(`
<div class="bg-gray-50 p-3 rounded-lg border border-gray-200 task-card %s" 
     data-task-id="%d"
     role="listitem"
     tabindex="0"
     aria-describedby="task-%d-description"
     aria-label="Task: %s, Priority: %s">
    <div class="flex justify-between items-start mb-2">
        <h3 class="text-sm font-medium text-gray-900">%s</h3>
        <span class="px-2 py-1 text-xs rounded-full %s" aria-label="%s priority">%s</span>
    </div>
    <p id="task-%d-description" class="text-sm text-gray-600">%s</p>
    <div class="mt-2 text-xs text-gray-500">
        <time datetime="%s">Created: %s</time>
    </div>
</div>`,
		priorityClass,
		task.ID,
		task.ID,
		task.Title,
		task.Priority,
		task.Title,
		getPriorityBadgeClass(task.Priority),
		task.Priority,
		task.Priority,
		task.ID,
		task.Description,
		task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		task.CreatedAt.Format("Jan 2, 15:04"))
}

func generateTaskListHTML(taskList []Task) string {
	backlogTasks := []Task{}
	todoTasks := []Task{}
	developingTasks := []Task{}
	reviewTasks := []Task{}
	completeTasks := []Task{}

	for _, task := range taskList {
		switch task.Status {
		case "backlog":
			backlogTasks = append(backlogTasks, task)
		case "todo":
			todoTasks = append(todoTasks, task)
		case "developing", "inprogress": // Support both old and new status names
			developingTasks = append(developingTasks, task)
		case "review":
			reviewTasks = append(reviewTasks, task)
		case "complete", "done": // Support both old and new status names
			completeTasks = append(completeTasks, task)
		}
	}

	return fmt.Sprintf(`
<!-- Backlog Column -->
<div class="bg-white rounded-lg shadow-md" role="region" aria-labelledby="backlog-header">
    <div class="p-4 border-b border-t-4 border-t-gray-500 bg-gray-50">
        <h2 id="backlog-header" class="text-xl font-semibold text-gray-700">Backlog</h2>
        <span class="text-sm text-gray-600" aria-label="%d tasks">(%d)</span>
    </div>
    <div class="p-4 space-y-3" role="list" aria-labelledby="backlog-header">
        %s
    </div>
</div>

<!-- To Do Column -->
<div class="bg-white rounded-lg shadow-md" role="region" aria-labelledby="todo-header">
    <div class="p-4 border-b border-t-4 border-t-blue-500 bg-blue-50">
        <h2 id="todo-header" class="text-xl font-semibold text-blue-700">To Do</h2>
        <span class="text-sm text-gray-600" aria-label="%d tasks">(%d)</span>
    </div>
    <div class="p-4 space-y-3" role="list" aria-labelledby="todo-header">
        %s
    </div>
</div>

<!-- In Progress Column -->
<div class="bg-white rounded-lg shadow-md" role="region" aria-labelledby="developing-header">
    <div class="p-4 border-b border-t-4 border-t-yellow-500 bg-yellow-50">
        <h2 id="developing-header" class="text-xl font-semibold text-yellow-700">In Progress</h2>
        <span class="text-sm text-gray-600" aria-label="%d tasks">(%d)</span>
    </div>
    <div class="p-4 space-y-3" role="list" aria-labelledby="developing-header">
        %s
    </div>
</div>

<!-- Review Column -->
<div class="bg-white rounded-lg shadow-md" role="region" aria-labelledby="review-header">
    <div class="p-4 border-b border-t-4 border-t-purple-500 bg-purple-50">
        <h2 id="review-header" class="text-xl font-semibold text-purple-700">Review</h2>
        <span class="text-sm text-gray-600" aria-label="%d tasks">(%d)</span>
    </div>
    <div class="p-4 space-y-3" role="list" aria-labelledby="review-header">
        %s
    </div>
</div>

<!-- Done Column -->
<div class="bg-white rounded-lg shadow-md" role="region" aria-labelledby="complete-header">
    <div class="p-4 border-b border-t-4 border-t-green-500 bg-green-50">
        <h2 id="complete-header" class="text-xl font-semibold text-green-700">Done</h2>
        <span class="text-sm text-gray-600" aria-label="%d tasks">(%d)</span>
    </div>
    <div class="p-4 space-y-3" role="list" aria-labelledby="complete-header">
        %s
    </div>
</div>`,
		len(backlogTasks), len(backlogTasks), generateTaskCardsHTML(backlogTasks),
		len(todoTasks), len(todoTasks), generateTaskCardsHTML(todoTasks),
		len(developingTasks), len(developingTasks), generateTaskCardsHTML(developingTasks),
		len(reviewTasks), len(reviewTasks), generateTaskCardsHTML(reviewTasks),
		len(completeTasks), len(completeTasks), generateTaskCardsHTML(completeTasks))
}

func generateTaskCardsHTML(taskList []Task) string {
	if len(taskList) == 0 {
		return `<div class="text-gray-500 text-sm italic">No tasks</div>`
	}

	html := ""
	for _, task := range taskList {
		html += generateTaskCardHTML(task)
	}
	return html
}

func getPriorityClass(priority string) string {
	switch priority {
	case "low":
		return "priority-low"
	case "medium":
		return "priority-medium"
	case "high":
		return "priority-high"
	case "critical":
		return "priority-critical"
	default:
		return "priority-medium"
	}
}

func getPriorityBadgeClass(priority string) string {
	switch priority {
	case "low":
		return "bg-green-100 text-green-800"
	case "medium":
		return "bg-yellow-100 text-yellow-800"
	case "high":
		return "bg-orange-100 text-orange-800"
	case "critical":
		return "bg-red-100 text-red-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// Fallback handlers for progressive enhancement
func fallbackCreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Show standalone create task form
		w.Header().Set("Content-Type", "text/html")

		fallbackFormHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Create Task - Simple Easy Tasks</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="/static/css/htmx-styles.css">
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto p-6 max-w-md">
        <div class="bg-white rounded-lg shadow-md p-6">
            <div class="flex justify-between items-center mb-6">
                <h1 class="text-xl font-semibold text-gray-900">Create New Task</h1>
                <a href="/" class="text-blue-600 hover:text-blue-700">← Back to Dashboard</a>
            </div>
            
            <form action="/tasks/create" method="POST" class="space-y-4">
                <div>
                    <label for="title" class="block text-sm font-medium text-gray-700 mb-1">
                        Title *
                    </label>
                    <input type="text" 
                           id="title" 
                           name="title" 
                           required
                           class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
                </div>
                
                <div>
                    <label for="description" class="block text-sm font-medium text-gray-700 mb-1">
                        Description
                    </label>
                    <textarea id="description" 
                              name="description" 
                              rows="3"
                              class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500"></textarea>
                </div>
                
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label for="priority" class="block text-sm font-medium text-gray-700 mb-1">
                            Priority
                        </label>
                        <select id="priority" 
                                name="priority"
                                class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
                            <option value="low">Low</option>
                            <option value="medium" selected>Medium</option>
                            <option value="high">High</option>
                            <option value="critical">Critical</option>
                        </select>
                    </div>
                    
                    <div>
                        <label for="status" class="block text-sm font-medium text-gray-700 mb-1">
                            Status
                        </label>
                        <select id="status" 
                                name="status"
                                class="w-full px-3 py-2 border border-gray-300 rounded-md 
                              focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
                            <option value="todo" selected>To Do</option>
                            <option value="inprogress">In Progress</option>
                            <option value="done">Done</option>
                        </select>
                    </div>
                </div>
                
                <div class="flex justify-end space-x-3 pt-4">
                    <a href="/" 
                       class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 
                              hover:bg-gray-200 rounded-md transition-colors">
                        Cancel
                    </a>
                    <button type="submit"
                            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 
                               hover:bg-blue-700 rounded-md transition-colors">
                        Create Task
                    </button>
                </div>
            </form>
        </div>
    </div>
</body>
</html>`

		writeResponse(w, []byte(fallbackFormHTML))

	case http.MethodPost:
		// Handle form submission and redirect back to main page
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		description := strings.TrimSpace(r.FormValue("description"))
		priority := r.FormValue("priority")
		status := r.FormValue("status")

		// Validation
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		if priority == "" {
			priority = PriorityMedium
		}
		if status == "" {
			status = StatusTodo
		}

		// Create new task
		newTask := Task{
			ID:          nextTaskID,
			Title:       title,
			Description: description,
			Status:      status,
			Priority:    priority,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		nextTaskID++

		// Add to tasks slice
		tasks = append(tasks, newTask)

		// Redirect back to dashboard
		http.Redirect(w, r, "/?created=1", http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func fallbackTasksHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":   tasks,
		"count":   len(tasks),
		"message": "Fallback API endpoint",
	}); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
