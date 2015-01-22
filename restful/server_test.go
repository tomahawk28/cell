package restful

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type testData struct {
	subject      string
	url          string
	method       string
	argument     map[string]string
	expected     string
	expectedCode int
}

var (
	rtr           = BuildCellAdvisorRestfulAPI("", 4, "10.82.26.12", time.Second*10)
	testDataArray = []testData{
		testData{
			subject:      "touch : Missing y value",
			url:          "/api/scpi/touch",
			method:       "POST",
			argument:     map[string]string{"x": "10"},
			expected:     "value missing",
			expectedCode: http.StatusBadRequest,
		},
		testData{
			subject:      "touch : X, Y Exist",
			url:          "/api/scpi/touch",
			method:       "POST",
			argument:     map[string]string{"x": "10", "y": "20"},
			expected:     "OK",
			expectedCode: http.StatusOK,
		},
		testData{
			subject:      "keyp : value given",
			url:          "/api/scpi/keyp",
			method:       "POST",
			argument:     map[string]string{"value": "MODE"},
			expected:     "OK",
			expectedCode: http.StatusOK,
		},
		testData{
			subject:      "keyp : value not given",
			url:          "/api/scpi/keyp",
			method:       "POST",
			argument:     map[string]string{},
			expected:     "keyp value missing",
			expectedCode: http.StatusBadRequest,
		},
		testData{
			subject:      "refresh_screen : ",
			url:          "/api/screen/refresh_screen",
			method:       "POST",
			argument:     map[string]string{},
			expected:     "OK",
			expectedCode: http.StatusOK,
		},
		testData{
			subject:      "screen : ",
			url:          "/api/screen/screen",
			method:       "GET",
			argument:     map[string]string{},
			expected:     "JFIF",
			expectedCode: http.StatusOK,
		},
		testData{
			subject:      "unknown command : ",
			url:          "/api/scpi/heyoman",
			method:       "POST",
			argument:     map[string]string{},
			expected:     "unknown",
			expectedCode: http.StatusBadRequest,
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
		if b := w.Body.String(); !strings.Contains(b, testcase.expected) {
			t.Logf("inner testcases : %s", testcase.subject)
			t.Fatalf("body = %s, want %s", b, testcase.expected)
		}
		if w.Code != testcase.expectedCode {
			t.Logf("inner testcases : %s", testcase.subject)
			t.Fatalf("code = %d, want %d", w.Code, testcase.expectedCode)
		}
	}

}

func BenchmarkProcessingRESTfulRequest(b *testing.B) {
	sampletestcase := testDataArray[2]
	v := createQuery(sampletestcase.argument)
	var r *http.Request
	for i := 0; i < b.N; i++ {
		r, _ = http.NewRequest(sampletestcase.method, sampletestcase.url, strings.NewReader(v.Encode()))
		r.Form = v
		w := httptest.NewRecorder()
		rtr.ServeHTTP(w, r)
		if content := w.Body.String(); !strings.Contains(content, sampletestcase.expected) {
			b.Logf("inner testcases : %s", sampletestcase.subject)
			b.Fatalf("body = %s, want %s", content, sampletestcase.expected)
		}
		if w.Code != sampletestcase.expectedCode {
			b.Logf("inner testcases : %s", sampletestcase.subject)
			b.Fatalf("code = %d, want %d", w.Code, sampletestcase.expectedCode)
		}
	}
}
