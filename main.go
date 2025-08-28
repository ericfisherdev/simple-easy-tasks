// main.go - Simple Easy Tasks with basic HTTP server
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

func main() {
	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Main page route
	http.HandleFunc("/", indexHandler)

	fmt.Println("Task Management Server starting on :8080")
	fmt.Println("Visit: http://localhost:8080")

	server := &http.Server{
		Addr:         ":8080",
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
    <link rel="stylesheet" href="/static/css/tailwind.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" defer></script>
    <script src="/static/js/app.js" defer></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto p-6">
        <header class="mb-8">
            <h1 class="text-3xl font-bold text-gray-800 mb-2">Simple Easy Tasks</h1>
            <p class="text-gray-600">Current time: {{.Time}}</p>
        </header>
        
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
            <!-- To Do Column -->
            <div class="bg-white rounded-lg shadow-md">
                <div class="p-4 border-b border-t-4 border-t-blue-500 bg-blue-50">
                    <h2 class="text-xl font-semibold text-blue-700">To Do</h2>
                    <span class="text-sm text-gray-600">(2)</span>
                </div>
                <div class="p-4 space-y-3">
                    <div class="bg-gray-50 p-3 rounded-lg border border-gray-200 task-card">
                        <div class="flex justify-between items-start mb-2">
                            <h3 class="text-sm font-medium text-gray-900">Design user interface</h3>
                            <span class="px-2 py-1 text-xs rounded-full bg-yellow-100 text-yellow-800">medium</span>
                        </div>
                        <p class="text-sm text-gray-600">Create wireframes and mockups</p>
                    </div>
                    <div class="bg-gray-50 p-3 rounded-lg border border-gray-200 task-card">
                        <div class="flex justify-between items-start mb-2">
                            <h3 class="text-sm font-medium text-gray-900">Set up database</h3>
                            <span class="px-2 py-1 text-xs rounded-full bg-red-100 text-red-800">high</span>
                        </div>
                        <p class="text-sm text-gray-600">Configure PostgreSQL</p>
                    </div>
                </div>
            </div>
            
            <!-- In Progress Column -->
            <div class="bg-white rounded-lg shadow-md">
                <div class="p-4 border-b border-t-4 border-t-yellow-500 bg-yellow-50">
                    <h2 class="text-xl font-semibold text-yellow-700">In Progress</h2>
                    <span class="text-sm text-gray-600">(1)</span>
                </div>
                <div class="p-4 space-y-3">
                    <div class="bg-gray-50 p-3 rounded-lg border border-gray-200 task-card">
                        <div class="flex justify-between items-start mb-2">
                            <h3 class="text-sm font-medium text-gray-900">Implement authentication</h3>
                            <span class="px-2 py-1 text-xs rounded-full bg-red-100 text-red-800">high</span>
                        </div>
                        <p class="text-sm text-gray-600">Add login/logout functionality</p>
                    </div>
                </div>
            </div>
            
            <!-- Done Column -->
            <div class="bg-white rounded-lg shadow-md">
                <div class="p-4 border-b border-t-4 border-t-green-500 bg-green-50">
                    <h2 class="text-xl font-semibold text-green-700">Done</h2>
                    <span class="text-sm text-gray-600">(1)</span>
                </div>
                <div class="p-4 space-y-3">
                    <div class="bg-gray-50 p-3 rounded-lg border border-gray-200 task-card opacity-75">
                        <div class="flex justify-between items-start mb-2">
                            <h3 class="text-sm font-medium text-gray-900">Project setup</h3>
                            <span class="px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-800">medium</span>
                        </div>
                        <p class="text-sm text-gray-600">Initialize Go project with HTMGO</p>
                    </div>
                </div>
            </div>
        </div>
        
        <footer class="mt-8 text-center text-sm text-gray-500">
            <p>Simple Easy Tasks - Task Management System</p>
        </footer>
    </div>
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
