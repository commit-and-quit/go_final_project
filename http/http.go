package httpHandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/commit-and-quit/yango-todo/auth"
	"github.com/commit-and-quit/yango-todo/db"
	"github.com/commit-and-quit/yango-todo/misc"
)

func ValidateId(id string) error {
	if id == "" {
		return errors.New("не указан идентификатор")
	}
	_, err := strconv.Atoi(id)
	if err != nil {
		return errors.New("не идентификатор")
	}
	return nil
}

func ErrToBytes(e error) []byte {
	return []byte(fmt.Sprintf("{\"error\" : \"%s\"}", e.Error()))
}

func ApiNextDate(w http.ResponseWriter, req *http.Request) {
	errString := "incorrect request"
	now := req.URL.Query().Get("now")
	date := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")
	if now == "" || date == "" || repeat == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errString))
		return
	}

	tNow, err := time.Parse("20060102", now)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errString))
		return
	}

	res, err := misc.NextDate(tNow, date, repeat)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprint(err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(res))
}

func ApiTasks(w http.ResponseWriter, req *http.Request) {

	var res []db.Task
	searchLimit := 10
	search := req.URL.Query().Get("search")
	res, err := db.GetTasks(search, searchLimit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(ErrToBytes(err))
	}

	resp, err := json.Marshal(map[string][]db.Task{"tasks": res})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func ApiTaskGet(req *http.Request) (httpStatus int, resp []byte) {
	id := req.URL.Query().Get("id")

	res, err := db.GetTask(id)
	if err != nil {
		return http.StatusOK, ErrToBytes(err)
	}
	resp, err = json.Marshal(res)
	if err != nil {
		return http.StatusBadGateway, ErrToBytes(err)
	}
	return http.StatusOK, resp
}

func ApiTaskPostAndPut(req *http.Request) (httpStatus int, resp []byte) {
	var task db.Task
	var buf bytes.Buffer

	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		return http.StatusBadRequest, ErrToBytes(err)
	}

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		return http.StatusBadRequest, ErrToBytes(err)
	}

	if task.Title == "" {
		return http.StatusOK, ErrToBytes(errors.New("не указан заголовок задачи"))
	}

	ok, err := misc.CalcDateForDB(&task)
	if !ok {
		return http.StatusOK, ErrToBytes(err)
	}

	switch req.Method {
	case http.MethodPost:
		id, err := db.AddTask(task)
		var k, v string
		if err != nil {
			k = "error"
			v = err.Error()
		} else {
			k = "id"
			v = strconv.Itoa(id)
		}
		resp, err = json.Marshal(map[string]string{k: v})
		if err != nil {
			return http.StatusBadRequest, ErrToBytes(err)
		}
		return http.StatusCreated, resp
	case http.MethodPut:
		err := ValidateId(task.Id)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}
		err = db.UpdateTask(task)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}
		return http.StatusOK, []byte("{}")
	}
	return http.StatusForbidden, []byte("{}")
}

func ApiTaskDelete(req *http.Request) (httpStatus int, resp []byte) {
	id := req.URL.Query().Get("id")
	err := ValidateId(id)
	if err != nil {
		return http.StatusOK, ErrToBytes(err)
	}
	err = db.DeleteTask(id)
	if err != nil {
		return http.StatusOK, ErrToBytes(err)
	}
	return http.StatusOK, []byte("{}")
}

func ApiTask(w http.ResponseWriter, req *http.Request) {

	var httpCode int
	var resp []byte

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	switch req.Method {
	case http.MethodGet:
		httpCode, resp = ApiTaskGet(req)
	case http.MethodPost:
		httpCode, resp = ApiTaskPostAndPut(req)
	case http.MethodPut:
		httpCode, resp = ApiTaskPostAndPut(req)
	case http.MethodDelete:
		httpCode, resp = ApiTaskDelete(req)
	default:
		httpCode = http.StatusBadRequest
		resp = ErrToBytes(errors.New("метод не поддерживается"))

	}
	w.WriteHeader(httpCode)
	w.Write(resp)
}

func ApiTaskDone(w http.ResponseWriter, req *http.Request) {

	getResp := func(req *http.Request) (httpCode int, resp []byte) {

		id := req.URL.Query().Get("id")
		err := ValidateId(id)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}

		task, err := db.GetTask(id)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}

		if task.Repeat == "" {
			err = db.DeleteTask(id)
			if err != nil {
				return http.StatusOK, ErrToBytes(err)
			}
			return http.StatusOK, []byte("{}")

		}

		newDate, err := misc.NextDate(time.Now(), task.Date, task.Repeat)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}
		task.Date = newDate
		err = db.UpdateTask(task)
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}
		return http.StatusOK, []byte("{}")
	}

	httpCode, resp := getResp(req)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpCode)
	w.Write(resp)
}

func ApiSignIn(w http.ResponseWriter, req *http.Request) {
	getResp := func(req *http.Request) (httpCode int, resp []byte) {
		var userInput map[string]string
		var buf bytes.Buffer
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			return http.StatusBadRequest, ErrToBytes(err)
		}
		if err = json.Unmarshal(buf.Bytes(), &userInput); err != nil {
			return http.StatusBadRequest, ErrToBytes(err)
		}
		pass, err := auth.GetPass()
		if err != nil {
			return http.StatusOK, []byte("")
		}
		if pass != userInput["password"] {
			return http.StatusOK, ErrToBytes(errors.New("неверный пароль"))
		}
		signedToken, err := auth.GetSignedToken()
		if err != nil {
			return http.StatusOK, ErrToBytes(err)
		}
		return http.StatusOK, []byte(fmt.Sprintf("{\"token\" : \"%s\"}", signedToken))
	}

	httpCode, resp := getResp(req)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpCode)
	w.Write(resp)

}

func CheckAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pass := os.Getenv("TODO_PASSWORD")
		if len(pass) > 0 {
			var jwt string
			cookie, err := r.Cookie("token")
			if err == nil {
				jwt = cookie.Value
			}
			valid := auth.VerifyUser(jwt)
			if !valid {
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	})
}
