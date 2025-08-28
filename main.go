// main.go - HTMGO initialization with task management optimizations
package main

import (
	"fmt"
	"github.com/maddalax/htmgo/framework/h"
	"log"
	"net/http"
	"time"
)

func main() {
	// Initialize htmgo with task management optimizations
	app := h.NewApp(
		h.AppOpts{
			LiveReload: true,   // Development live reload
			Port:       "8080", // Server port
		},
	)

	// Register page routes automatically based on file paths
	// app.RegisterPages("web/pages")      // Full page components
	// app.RegisterPartials("web/partials") // HTMX partial components for real-time updates

	// Custom middleware for task management
	// app.Use(CSRFMiddleware)
	// app.Use(AuthMiddleware)
	// app.Use(RealTimeMiddleware)

	fmt.Println("Task Management Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", app))
}
