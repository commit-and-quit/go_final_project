package main

import (
	"fmt"
	"net/http"

	"github.com/commit-and-quit/yango-todo/db"
	httpHandler "github.com/commit-and-quit/yango-todo/http"
	"github.com/commit-and-quit/yango-todo/misc"
	_ "modernc.org/sqlite"
)

func main() {

	port := misc.GetPort()

	db.CreateDB()

	webDir := "./web/"
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc(`/api/nextdate`, httpHandler.ApiNextDate)
	http.HandleFunc(`/api/task`, httpHandler.ApiTask)
	http.HandleFunc(`/api/tasks`, httpHandler.ApiTasks)
	http.HandleFunc(`/api/task/done`, httpHandler.ApiTaskDone)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic(err)
	}
}
