package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var Db *sql.DB

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
	appPath, err := os.Executable()
	if err != nil {
		panic("Can't exec os.Executable()")
	}
	DBFile := "scheduler.db"
	return filepath.Join(filepath.Dir(appPath), DBFile)
}

func Init() (*sql.DB, error) {
	DBFile := GetDBFile()
	db, err := sql.Open("sqlite", DBFile)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS`scheduler` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `date` TEXT NOT NULL, `title` TEXT NOT NULL, `comment` TEXT NULL, `repeat` TEXT(128) NULL)")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AddTask(task Task) (int, error) {

	res, err := Db.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)",
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
	rows, err := Db.Query(query, sql.Named("search", search), sql.Named("limit", limit))
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
	if err = rows.Err(); err != nil {
		return res, err
	}
	return res, nil
}

func GetTask(id string) (Task, error) {

	var task Task

	row := Db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = :id", sql.Named("id", id))
	err := row.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errors.New("задача не найдена")
		}
		return Task{}, err

	}
	return task, nil
}

func UpdateTask(task Task) error {

	_, err := GetTask(task.Id)
	if err != nil {
		return err
	}
	_, err = Db.Exec("UPDATE scheduler SET date = :date, title = :title,  comment = :comment, repeat = :repeat WHERE id = :id",
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat),
		sql.Named("id", task.Id))
	return err

}

func DeleteTask(id string) error {

	_, err := Db.Exec("DELETE FROM scheduler WHERE id = :id",
		sql.Named("id", id))
	return err

}
