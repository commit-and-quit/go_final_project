package misc

import (
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/commit-and-quit/yango-todo/db"
	"github.com/commit-and-quit/yango-todo/tests"
)

func GetPort() int {
	port, err := strconv.Atoi(os.Getenv("TODO_PORT"))
	if err != nil {
		port = tests.Port
	}
	return port
}

func RepeatParser(repeat string) ([]int, error) {

	valueSplited := strings.Split(repeat[2:], ",")
	var values []int
	for _, v := range valueSplited {
		vAsInt, err := strconv.Atoi(v)
		if err != nil {
			return make([]int, 0), err
		}
		values = append(values, vAsInt)
	}

	return values, nil

}

func NextDate(now time.Time, date string, repeat string) (string, error) {

	repeatRegEx := "d [0-9]+|y|[m,w] [-,0-9]+"

	taskDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	repeatIsValid, _ := regexp.MatchString(repeatRegEx, repeat)
	if !repeatIsValid {
		return "", errors.New("некорректный формат повторения")
	}

	targetAction := repeat[:1]

	newDate := taskDate
	switch targetAction {
	case "d":
		values, err := RepeatParser(repeat)
		if len(values) > 1 || values[0] > 400 || err != nil {
			return "", errors.New("incorrect format")
		}
		if taskDate.Format("20060102") >= now.Format("20060102") {
			newDate = newDate.AddDate(0, 0, values[0])
		} else {
			for now.Format("20060102") > newDate.Format("20060102") {
				newDate = newDate.AddDate(0, 0, values[0])
			}
		}
	case "y":
		if taskDate.Format("20060102") >= now.Format("20060102") {
			newDate = newDate.AddDate(1, 0, 0)
		} else {
			if taskDate.Format("0102") > now.Format("0102") {
				newDate = time.Date(now.Year(), taskDate.Month(), taskDate.Day(), 0, 0, 0, 0, time.UTC)
			} else {
				newDate = time.Date(now.Year()+1, taskDate.Month(), taskDate.Day(), 0, 0, 0, 0, time.UTC)
			}
		}
		// Дописать кейсы для m и w
	}
	return newDate.Format("20060102"), nil
}

func CalcDateForDB(task *db.Task) (bool, error) {

	var taskDateForDB string

	now := time.Now()
	if task.Date == "" {
		taskDateForDB = now.Format("20060102")
	} else {
		_, err := time.Parse("20060102", task.Date)
		if err != nil {
			return false, errors.New("некорректный формат даты")
		}
		tmp, _ := time.Parse("20060102", task.Date)
		taskDateForDB = tmp.Format("20060102")
		if now.Format("20060102") > taskDateForDB {
			if task.Repeat == "" {
				taskDateForDB = now.Format("20060102")
			} else {
				taskDateForDB, err = NextDate(now, taskDateForDB, task.Repeat)
				if err != nil {
					return false, err
				}
			}
		}
	}
	log.Printf("Task: %v Calculated date: %v", task, taskDateForDB)
	task.Date = taskDateForDB
	return true, nil
}
