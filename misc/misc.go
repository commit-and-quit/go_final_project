package misc

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"sort"
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

func RepeatParser(repeat string) ([][]int, error) {
	var res [][]int

	makeSlice := func(s string) ([]int, error) {
		var res []int
		valueSplited := strings.Split(s, ",")
		for _, v := range valueSplited {
			vAsInt, err := strconv.Atoi(v)
			if err != nil {
				return make([]int, 0), err
			}
			res = append(res, vAsInt)
		}

		sort.Ints(res)
		return res, nil
	}

	if len(strings.Split(repeat[2:], " ")) == 1 {
		values, err := makeSlice(repeat[2:])
		if err != nil {
			return make([][]int, 0), err
		}
		res = append(res, values)
		return res, nil
	}
	valueSplited := strings.Split(repeat[2:], " ")
	days, err := makeSlice(valueSplited[0])
	if err != nil {
		return make([][]int, 0), err
	}
	sort.Ints(days)
	months, err := makeSlice(valueSplited[1])
	if err != nil {
		return make([][]int, 0), err
	}
	sort.Ints(months)
	res = append(res, days)
	res = append(res, months)
	return res, nil
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	//m 25,26,7
	repeatRegEx, _ := regexp.Compile("d [0-9]+|y|w [1-7,]+|m [-,0-9]+( [,0-9]+)?")

	taskDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	repeatIsValid := repeatRegEx.MatchString(repeat)
	if !repeatIsValid {
		return "", errors.New("некорректный формат повторения")
	}

	targetAction := repeat[:1]

	newDate := taskDate
	switch targetAction {
	case "d":
		values, err := RepeatParser(repeat)
		repeatDays := values[0][0]
		if len(values) > 1 || repeatDays > 400 || err != nil {
			return "", errors.New("incorrect format")
		}
		if taskDate.Format("20060102") >= now.Format("20060102") {
			newDate = newDate.AddDate(0, 0, repeatDays)
		} else {
			for now.Format("20060102") > newDate.Format("20060102") {
				newDate = newDate.AddDate(0, 0, repeatDays)
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
	case "w":
		dayInRepeat := func(t time.Time, values []int) bool {
			_, res := slices.BinarySearch(values, int(t.Weekday()))
			return res
		}

		values, err := RepeatParser(repeat)
		if err != nil {
			return "", errors.New("incorrect format")
		}
		weekDays := values[0]
		r, _ := regexp.Compile("[0-6]")
		for i, v := range weekDays {
			if v == 7 {
				weekDays[i] = 0
				v = 0
			}
			valuetIsValid := r.MatchString(strconv.Itoa(v))
			if !valuetIsValid {
				return "", errors.New("incorrect format")
			}
		}
		newDate = now
		for {
			newDate = newDate.AddDate(0, 0, 1)
			if dayInRepeat(newDate, weekDays) {
				break
			}
		}
	case "m":
		values, err := RepeatParser(repeat)
		if err != nil {
			return "", errors.New("incorrect format")
		}
		monthDays := values[0]
		months := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		if len(values) == 2 {
			months = values[1]
		}
		for _, monthDay := range monthDays {
			if monthDay < -2 || monthDay > 31 || monthDay == 0 {
				return "", errors.New("incorrect format")
			}
		}
		for _, month := range months {
			if month < 1 || month > 12 {
				return "", errors.New("incorrect format")
			}
		}
		daysInMonth := func(m time.Month, year int) int {
			return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
		}

		invertDays := func(daysInMonthCount int, monthDays []int) []int {
			for i, monthDay := range monthDays {
				if monthDay < 0 {
					monthDays[i] = monthDay + daysInMonthCount + 1
				}
			}
			sort.Ints(monthDays)
			return monthDays
		}

		isMonthInRepeat := func(t time.Time, values []int) bool {
			_, res := slices.BinarySearch(values, int(t.Month()))
			return res
		}

		isDayInRepeat := func(t time.Time, monthDays []int) bool {
			_, res := slices.BinarySearch(monthDays, int(t.Day()))
			return res
		}

		if now.Format("20060102") > taskDate.Format("20060102") {
			newDate = now
		} else {
			newDate = taskDate
		}
		for {
			newDate = newDate.AddDate(0, 0, 1)
			daysInMonthCount := daysInMonth(newDate.Month(), newDate.Year())
			monthDays = invertDays(daysInMonthCount, monthDays)
			if isMonthInRepeat(newDate, months) && isDayInRepeat(newDate, monthDays) {
				break
			}
		}

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
	task.Date = taskDateForDB
	return true, nil
}
