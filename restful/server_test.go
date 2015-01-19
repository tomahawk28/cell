package restful

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type testData struct {
	subject  string
	url      string
	method   string
	argument map[string]string
	expected string
}

var (
	rtr           = BuildCellAdvisorRestfulAPI(4, "10.82.26.12", time.Second*10)
	testDataArray = []testData{
		testData{
			subject:  "touch : Missing y value",
			url:      "/api/scpi/touch",
			method:   "POST",
			argument: map[string]string{"x": "10"},
			expected: "value missing",
		},
		testData{
			subject:  "touch : X, Y Exist",
			url:      "/api/scpi/touch",
			method:   "POST",
			argument: map[string]string{"x": "10", "y": "20"},
			expected: "OK",
		},
		testData{
			subject:  "keyp : value given",
			url:      "/api/scpi/keyp",
			method:   "POST",
			argument: map[string]string{"value": "MODE"},
			expected: "OK",
		},
		testData{
			subject:  "keyp : value not given",
			url:      "/api/scpi/keyp",
			method:   "POST",
			argument: map[string]string{},
			expected: "keyp value missing",
		},
		testData{
			subject:  "refresh_screen : ",
			url:      "/api/screen/refresh_screen",
			method:   "POST",
			argument: map[string]string{},
			expected: "OK",
		},
		testData{
			subject:  "screen : ",
			url:      "/api/screen/screen",
			method:   "GET",
			argument: map[string]string{},
			expected: "JFIF",
		},
		testData{
			subject:  "unknown command : ",
			url:      "/api/scpi/heyoman",
			method:   "POST",
			argument: map[string]string{},
			expected: "unknown",
		},
	}
)

func createQuery(argument map[string]string) url.Values {
	u := url.Values{}
	for k, v := range argument {
		u.Set(k, v)
	}
	return u
}
func TestSCPIArgument(t *testing.T) {

	var v url.Values
	var r *http.Request
	var err error
	for _, testcase := range testDataArray {
		v = createQuery(testcase.argument)
		if testcase.method == "POST" {
			r, err = http.NewRequest(testcase.method, testcase.url, strings.NewReader(v.Encode()))
			r.Form = v
		} else {
			r, err = http.NewRequest(testcase.method, testcase.url, nil)
		}
		if err != nil {
			t.Log(err.Error())
			return
		}

		w := httptest.NewRecorder()
		rtr.ServeHTTP(w, r)
		log.Println(w.Code)
		if b := w.Body.String(); !strings.Contains(b, testcase.expected) {
			t.Logf("inner testcases : %s", testcase.subject)
			log.Println(w.Code)
			t.Fatalf("body = %s, want %s", b, testcase.expected)
		}
	}

}
