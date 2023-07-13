
package main

import (
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strconv"
)

var schema = `
CREATE TABLE IF NOT EXISTS todo(
	id SERIAL primary key,
	title VARCHAR(50) not null,
	complete boolean not null
);
`

var db *sqlx.DB = nil

type Todo struct {
	ID       int    `json:"id" db:"id"`
	Title    string `json:"title" db:"title"`
	Complete bool   `json:"complete" db:"complete"`
}

type TodoList struct {
	Todos []Todo `json:"todos"`
}

func getTodos(c *gin.Context) {
	todos := []Todo{}
	if err := db.Select(&todos, "SELECT id,title,complete FROM todo ORDER BY id ASC;"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var todos_json TodoList
	todos_json.Todos = todos

	c.JSON(http.StatusOK, todos_json)
}
func createTodo(c *gin.Context) {
	var newTodo Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	newTitle, newComplete := newTodo.Title, newTodo.Complete
	var newID int

	err2 := tx.QueryRow("INSERT INTO todo (title,complete) VALUES ($1, $2) RETURNING id", newTitle, newComplete).Scan(&newID)
	if err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	msg := fmt.Sprintf("Added todo with id = %v", newID)
	c.JSON(http.StatusOK, gin.H{"message": msg})
}
func deleteTodo(c *gin.Context) {
	input := c.Param("id")

	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if input == "all" {
		if _, err := tx.Exec("TRUNCATE TABLE todo;"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "Deleted everything"})
		return
	}

	todoID, err := strconv.Atoi(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err1 := tx.Exec("DELETE FROM todo WHERE id = $1;", todoID)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err1.Error()})
		return
	}

	tx.Commit()

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todo not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Todo deleted"})
}
func updateTodoStatus(c *gin.Context) {
	todoID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	result, err1 := tx.Exec("UPDATE todo SET complete = CASE WHEN complete = true THEN false ELSE true END WHERE id = $1;", todoID)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err1.Error()})
		return
	}

	tx.Commit()

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todo not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}
func modifyTodo(c *gin.Context) {
	var newTodo Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ID, newTitle, newComplete := newTodo.ID, newTodo.Title, newTodo.Complete
	result, err1 := tx.Exec("UPDATE todo SET Title = $2, Complete = $3 WHERE id = $1;", ID, newTitle, newComplete)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err1.Error()})
		return
	}

	tx.Commit()

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todo not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Todo modified"})
}
func main() {
	fmt.Println("Start")

	var err error
	//db, err = sqlx.Connect("postgres", "user=postgres dbname=postgres password=mysecretpassword sslmode=disable port=5432")
	db, err = sqlx.Connect("postgres", "host=8888:8080 port=5432 user=postgres dbname=postgres password=mysecretpassword sslmode=disable")
	for err != nil {
		log.Println("Failed! Reconnecting in 3 secs......")
		log.Println(err)
		time.Sleep(3*time.Second)
		db, err = sqlx.Connect("postgres", "host=host.docker.internal port=5432 user=postgres dbname=postgres password=mysecretpassword sslmode=disable")
		//log.Fatalln(err)
	}

	db.MustExec(schema)

	r := gin.Default()

	r.GET("/todos", getTodos)
	r.POST("/todos", createTodo)
	r.PUT("/todos/:id", updateTodoStatus)
	r.DELETE("/todos/:id", deleteTodo)
	r.PATCH("/todos", modifyTodo)

	r.Run(":8080")
}

