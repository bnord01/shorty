package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

/* ********************************************** *
 * ************ SETUP THE TEST SUITE ************ *
 * ********************************************** */

// Struct for the test suite
type S struct {
	suite.Suite
	suite.SetupAllSuite
	suite.SetupTestSuite
	suite.TearDownAllSuite
	router *gin.Engine
}

// Setup the database and routes before the suite is executed
func (s *S) SetupSuite() {
	fmt.Println("In SetupSuite")
	default_db = "testing"
	default_collection = "test"
	err := Connect()
	if err != nil {
		s.Fail("Error connecting: %v", err)
	}
	s.router = setupRoutes()
}

// Before each test in the suite delete all documents
func (s *S) SetupTest() {
	coll.DeleteMany(UnboundContext(), bson.M{})
}

// After all tests are done cleanup
func (s *S) TearDownSuite() {
	// Drop the test database
	coll.Database().Drop(UnboundContext())
	// Disconnect
	coll.Database().Client().Disconnect(UnboundContext())
}

// Register the suite to be run by go test
func TestShortyTestSuite(t *testing.T) {
	suite.Run(t, new(S))
}

/* ********************************************** *
 * ************** INDIVIDUAL TESTS ************** *
 * ********************************************** */

/* TEST FOR CREATE */

func (s *S) TestCreateValid() {
	sl := exampleShortlink()

	c, _ := s.requestSL("POST", "/shortlinks", sl)

	s.Equal(201, c)
}

func (s *S) TestCreateDuplicate() {
	sl := exampleShortlink()

	s.requestSL("POST", "/shortlinks", sl)
	c, b := s.requestSL("POST", "/shortlinks", sl)

	s.Equal(http.StatusConflict, c)
	s.Equal(`{"error":"shortlink already exists"}`, b)
}

func (s *S) TestCreateInvalidShort() {
	sl := exampleShortlink()
	sl.ShortUrl = "asdf 77 / 324"

	c, _ := s.requestSL("POST", "/shortlinks", sl)

	s.Equal(http.StatusBadRequest, c)
}

func (s *S) TestCreateInvalidURL() {
	sl := exampleShortlink()
	sl.LongUrl = "example com"

	c, _ := s.requestSL("POST", "/shortlinks", sl)

	s.Equal(http.StatusBadRequest, c)
}

/* TESTS FOR GET */
func (s *S) TestGetValid() {
	sl := exampleShortlink()

	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")

	s.Equal(200, c)

	r := unmarshalShortlink(b)
	s.Equal(sl.Description, r.Description)
	s.Equal(sl.ShortUrl, r.ShortUrl)
	s.Equal(sl.LongUrl, r.LongUrl)
}

func (s *S) TestGetNotExisting() {
	c, b := s.request("GET", "/shortlinks/ex", "")
	s.Equal(404, c)
	s.Equal(`{"error":"shortlink not found"}`, b)
}

func (s *S) TestGetInvalidShort() {
	c, _ := s.request("GET", "/shortlinks/some.thing", "")
	s.Equal(400, c)

}

/* TESTS FOR UPDATE */

func (s *S) TestUpdateValid() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	oldshort := sl.ShortUrl
	sl.ShortUrl = "excom"
	c, b := s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)
	r := unmarshalShortlink(b)

	s.Equal(200, c, b)
	s.Equal(sl.Description, r.Description, b)
	s.Equal(sl.ShortUrl, r.ShortUrl, b)
	s.Equal(sl.LongUrl, r.LongUrl, b)
}

func (s *S) TestUpdateNotExist() {
	sl := exampleShortlink()
	c, _ := s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), sl)

	s.Equal(404, c)
}

func (s *S) TestUpdateDuplicate() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	oldshort := sl.ShortUrl

	sl.ShortUrl = "newshort"
	c, _ = s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)

	s.Equal(409, c)
	s.Equal(`{"error":"shortlink already exists"}`, b)
}

func (s *S) TestUpdateInvalidNewShort() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	oldshort := sl.ShortUrl
	sl.ShortUrl = "45+3/223?"
	c, b := s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)

	s.Equal(400, c)
	s.Equal(`{"error":"invalid short does not match ^[a-zA-Z0-9\\-_]+$"}`, b)
}

func (s *S) TestUpdateInvalidOldShort() {
	sl := exampleShortlink()
	c, b := s.requestSL("PUT", "/shortlinks/käse", sl)

	s.Equal(400, c)
	s.Equal(`{"error":"invalid short does not match ^[a-zA-Z0-9\\-_]+$"}`, b)
}

func (s *S) TestUpdateInvalidURL() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	oldshort := sl.ShortUrl
	sl.LongUrl = "some thing illegal"
	c, b := s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)

	s.Equal(400, c)
	s.Equal(`{"error":"invalid redirect url"}`, b)
}

/* TESTS FOR DELETE */

func (s *S) TestDeleteValid() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("DELETE", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")

	s.Equal(200, c)
	s.Equal(`{"deleted":1}`, b)
}

func (s *S) TestDeleteNotExisting() {
	c, b := s.request("DELETE", "/shortlinks/something", "")

	s.Equal(200, c)
	s.Equal(`{"deleted":0}`, b)
}

func (s *S) TestDeleteInvalid() {
	c, b := s.request("DELETE", "/shortlinks/käse", "")

	s.Equal(400, c)
	s.Equal(`{"error":"invalid short does not match ^[a-zA-Z0-9\\-_]+$"}`, b)
}

/* TESTS FOR GET ALL */

func (s *S) TestGetAllEmpty() {
	c, b := s.request("GET", "/shortlinks", "")

	s.Equal(200, c)
	s.Equal(`[]`, b)
}

func (s *S) TestGetAllOne() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", "/shortlinks", "")
	r := unmarshalShortlinkArray(b)

	s.Equal(200, c)

	s.Equal(sl.Description, r[0].Description, b)
	s.Equal(sl.ShortUrl, r[0].ShortUrl, b)
	s.Equal(sl.LongUrl, r[0].LongUrl, b)
}

/* TESTS FOR CHECK */

func (s *S) TestCheckNotExisting() {
	c, b := s.request("GET", "/check/something", "")

	s.Equal(200, c)
	s.Equal(`{"free":true}`, b)
}
func (s *S) TestCheckExistent() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", fmt.Sprintf("/check/%s", sl.ShortUrl), "")

	s.Equal(200, c)
	s.Equal(`{"free":false}`, b)
}

func (s *S) TestCheckInvalid() {
	c, b := s.request("GET", "/check/käse", "")

	s.Equal(400, c)
	s.Equal(`{"error":"invalid short does not match ^[a-zA-Z0-9\\-_]+$"}`, b)
}

/* TEST FOR REDIRECT */

func (s *S) TestRedirectNotExisting() {
	c, b := s.request("GET", "/go/somewhere", "")

	s.Equal(404, c)
	s.Equal(`{"error":"no redirect for somewhere"}`, b)
}

func (s *S) TestRedirectExisting() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", fmt.Sprintf("/go/%s", sl.ShortUrl), "")

	s.Equal(307, c)
	s.Equal(fmt.Sprintf("<a href=\"%s\">Temporary Redirect</a>.\n\n", sl.LongUrl), b)
}

func (s *S) TestRedirectInvalid() {
	c, b := s.request("GET", "/go/käse", "")

	s.Equal(400, c)
	s.Equal(`{"error":"invalid short does not match ^[a-zA-Z0-9\\-_]+$"}`, b)
}

/* ********************************************** *
 * *************** BEHAVIOR TESTS *************** *
 * ********************************************** */

// Check that redirects increment the access_count and that it is preserved by update
func (s *S) TestRedirectCount() {
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r := unmarshalShortlink(b)
	s.Equal(0, r.AccessCount)

	c, b = s.request("GET", fmt.Sprintf("/go/%s", sl.ShortUrl), "")
	s.Equal(307, c)
	s.Equal(fmt.Sprintf("<a href=\"%s\">Temporary Redirect</a>.\n\n", sl.LongUrl), b)

	c, b = s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r = unmarshalShortlink(b)
	s.Equal(1, r.AccessCount)

	s.request("GET", fmt.Sprintf("/go/%s", sl.ShortUrl), "")
	c, b = s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r = unmarshalShortlink(b)
	s.Equal(2, r.AccessCount)

	oldshort := sl.ShortUrl
	sl.ShortUrl = "newurl"
	c, b = s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)
	s.Equal(200, c)
	r = unmarshalShortlink(b)

	c, b = s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r = unmarshalShortlink(b)
	s.Equal(2, r.AccessCount)
}

// Check the created and updated times
func (s *S) TestUpdateTimes() {
	start := now()
	sl := exampleShortlink()
	c, _ := s.requestSL("POST", "/shortlinks", sl)
	s.Equal(201, c)

	c, b := s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r := unmarshalShortlink(b)

	created := r.CreatedAt
	s.True(before(start, created))
	s.True(before(created, now()))

	updated := r.UpdatedAt
	s.True(before(start, updated))
	s.True(before(updated, now()))

	start_update := now()
	oldshort := sl.ShortUrl
	sl.ShortUrl = "newurl"
	c, b = s.requestSL("PUT", fmt.Sprintf("/shortlinks/%s", oldshort), sl)
	s.Equal(200, c)
	r = unmarshalShortlink(b)

	s.Equal(created, r.CreatedAt)

	updated = r.UpdatedAt
	s.True(before(start_update, updated))
	s.True(before(updated, now()))

	c, b = s.request("GET", fmt.Sprintf("/shortlinks/%s", sl.ShortUrl), "")
	s.Equal(200, c)
	r2 := unmarshalShortlink(b)
	s.Equal(r, r2)
}

/* ********************************************** *
 * ************** HELPER FUNCTIONS ************** *
 * ********************************************** */

// Send a request with the given method/url/body and return the status code and body
func (s *S) requestSL(method string, url string, body ShortlinkUpdate) (int, string) {
	b, err := json.Marshal(body)
	if err != nil {
		s.Fail("Failed creating request", err)
	}
	return s.request(method, url, string(b))
}

// Send a request with the given method/url/body and return the status code and body
func (s *S) request(method string, url string, body string) (int, string) {
	bodyreader := strings.NewReader(body)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, bodyreader)
	if err != nil {
		s.Fail("Failed creating request", err)
	}
	s.router.ServeHTTP(resp, req)

	return resp.Code, resp.Body.String()
}

// Unmarshal a string to a Shortlink
func unmarshalShortlink(body string) Shortlink {
	sl := Shortlink{}
	json.Unmarshal([]byte(body), &sl)
	return sl
}

// Unmarshal a string to a Shortlink array
func unmarshalShortlinkArray(body string) []Shortlink {
	sl := []Shortlink{}
	json.Unmarshal([]byte(body), &sl)
	return sl
}

// Shortlink "ex" pointing to http://example.com
func exampleShortlink() ShortlinkUpdate {
	sl := ShortlinkUpdate{}
	sl.Description = "Example item"
	sl.LongUrl = "http://example.com"
	sl.ShortUrl = "ex"
	return sl
}

// Returns the current time in millisecond precission
func now() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

// Returns true if t1 less than t2 + 10ms
func before(t1 time.Time, t2 time.Time) bool {
	return t1.Before(t2.Add(time.Millisecond * 10))
}
