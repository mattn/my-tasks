package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "./statik"
	"github.com/go-gorp/gorp"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
)

type Data struct {
	Items []Task `json:"items"`
}

type Task struct {
	ID        int64  `json:"id" db:"id,primarykey,autoincrement"`
	Body      string `json:"body" db:"body"`
	Done      bool   `json:"done" db:"done"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
}

func (task *Task) PreInsert(s gorp.SqlExecutor) error {
	task.CreatedAt = time.Now().UnixNano()
	task.UpdatedAt = task.CreatedAt
	return nil
}

func (task *Task) PreUpdate(s gorp.SqlExecutor) error {
	task.UpdatedAt = time.Now().UnixNano()
	return nil
}

func (task *Task) PreDelete(s gorp.SqlExecutor) error {
	println(task.ID)
	_, err := s.Exec("delete from tasks where id = $1", task.ID)
	return err
}

func (task *Task) SetAttributes(c echo.Context) error {
	var err error
	if id := c.Param("id"); id != "" {
		task.ID, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return err
		}
	}
	err = c.Request().ParseForm()
	if err != nil {
		return err
	}
	params := c.Request().Form
	if value, ok := params["body"]; ok {
		task.Body = value[0]
	}
	if value, ok := params["done"]; ok {
		task.Done, _ = strconv.ParseBool(value[0])
	}
	return nil
}

func main() {
	fs, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", os.Getenv("postgres_uri"))
	if err != nil {
		log.Fatal(err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	dbmap.AddTableWithName(Task{}, "tasks").SetKeys(true, "id")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/*", echo.WrapHandler(http.FileServer(fs)))
	e.GET("/tasks", func(c echo.Context) error {
		var tasks []Task
		_, err := dbmap.Select(&tasks, "select * from tasks order by created_at desc")
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, Data{
			Items: tasks,
		})
	})
	e.POST("/tasks", func(c echo.Context) error {
		var task Task
		err := task.SetAttributes(c)
		if err != nil {
			return err
		}
		err = dbmap.Insert(&task)
		if err != nil {
			return err
		}
		fmt.Println(task)
		return c.JSON(http.StatusOK, task)
	})
	e.PUT("/tasks/:id", func(c echo.Context) error {
		var task Task
		err := task.SetAttributes(c)
		if err != nil {
			return err
		}
		_, err = dbmap.UpdateColumns(func(col *gorp.ColumnMap) bool {
			return col.ColumnName == "done"
		}, &task)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, task)
	})
	e.DELETE("/tasks/:id", func(c echo.Context) error {
		var task Task
		err := task.SetAttributes(c)
		if err != nil {
			return err
		}
		_, err = dbmap.Delete(&task)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, task)
	})
	e.Logger.Fatal(e.Start(":3000"))
}
