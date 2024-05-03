package db

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/commit-and-quit/yango-todo/tests"
)

type Task struct {
	Id      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func GetDBFile() string {
	envFile := os.Getenv("TODO_DBFILE")
	if len(envFile) > 0 {
		return envFile
	}
	_, DBFile := filepath.Split(tests.DBFile)
	return DBFile
}

func Connect() (*sql.DB, error) {
	DBFile := GetDBFile()
	db, err := sql.Open("sqlite", DBFile)
	if err != nil {
		log.Print(err)
		return db, err
	}
	return db, nil
}

func CreateDB() bool {
	DBFile := GetDBFile()
	log.Print("started CreateDB")
	appPath, err := os.Executable()
	log.Printf("DBFile in db: %v", DBFile)
	if err != nil {
		log.Print(err)
	}
	DBFile = filepath.Join(filepath.Dir(appPath), DBFile)
	log.Print(DBFile)
	_, err = os.Stat(DBFile)

	var install bool
	if err != nil {
		install = true
	}

	if install {
		log.Printf("install is true, file %v", DBFile)
		db, err := Connect()
		if err != nil {
			log.Print(err)
			return false
		}
		defer db.Close()
		_, err = db.Exec("CREATE TABLE `scheduler` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `date` TEXT NOT NULL, `title` TEXT NOT NULL, `comment` TEXT NULL, `repeat` TEXT(128) NULL)")

		if err != nil {
			log.Print(err)
			return false
		}
		log.Print("created")
	} else {
		log.Print("install is false")
	}
	return true
}

func AddTask(task Task) (int, error) {

	db, err := Connect()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	res, err := db.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)",
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat))
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil

}

func GetTasks(search string, limit int) ([]Task, error) {

	res := make([]Task, 0)

	db, err := Connect()
	if err != nil {
		return res, err
	}
	defer db.Close()
	searchSet := false
	if search != "" {
		searchSet = true
	}

	searchIsDate := false
	searchIsDate, _ = regexp.MatchString("[0-9]{2}\\.[0-9]{2}\\.[0-9]+", search)
	if searchIsDate {
		dateAsTime, err := time.Parse("02.01.2006", search)
		if err != nil {
			return res, err
		}
		search = dateAsTime.Format("20060102")
	} else {
		search = "%" + search + "%"
	}
	query := "SELECT id, date, title, comment, repeat FROM scheduler LIMIT :limit "
	if searchSet && searchIsDate {
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = :search ORDER BY date LIMIT :limit "
	} else if searchSet {
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE :search or comment LIKE :search ORDER BY date LIMIT :limit "
	}
	rows, err := db.Query(query, sql.Named("search", search), sql.Named("limit", limit))
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		task := Task{}
		err = rows.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return res, err
		}
		res = append(res, task)
	}
	return res, nil
}

func GetTask(id string) (Task, error) {

	var task Task

	db, err := Connect()
	if err != nil {
		return Task{}, err
	}
	defer db.Close()
	row := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = :id", sql.Named("id", id))
	err = row.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	if err == errors.New("sql: no rows in result set") {
		return Task{}, errors.New("задача не найдена")
	}
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func UpdateTask(task Task) error {

	db, err := Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = GetTask(task.Id)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE scheduler SET date = :date, title = :title,  comment = :comment, repeat = :repeat WHERE id = :id",
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat),
		sql.Named("id", task.Id))
	return err

}

func DeleteTask(id string) error {

	db, err := Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = GetTask(id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM scheduler WHERE id = :id",
		sql.Named("id", id))
	return err

}
