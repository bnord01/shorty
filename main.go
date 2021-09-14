package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/gin-gonic/gin"
)

/* ********************************************** *\
 * ****************** HANDLERS ****************** *
\* ********************************************** */

// Handler for GET /shortlinks
// Returns code 200 with [..shortlinks..] on success and
// code 500 with {error:msg} in case of an error.
func handleGetShortlinks(c *gin.Context) {
	var loadedShortlinks, err = GetAllShortlinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loadedShortlinks)
}

// Handler for GET /shortlinks/:short
// Returns code 200 with the requested shortlink as json on success,
// code 400 if the short is invalid, 404 if the shortlinks doesn't exist and
// code 500 in case of another error.
func handleGetShortlink(c *gin.Context) {
	short := c.Param("short")
	if invalidShort(short, c) {
		return
	}
	var loadedShortlink, err = GetShortlinkByShort(short)
	if err != nil {
		// Code 404 if not found
		if isNotFundError(err) {
			log.Printf("Failed finding shortlink: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "shortlink not found"})
			return
		}
		// Other error, code 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, loadedShortlink)
}

// Handler for POST /shortlinks
// Creates the shortlink provided as json and returns code 201 if successfull,
// code 400 if the shortlink is invalid,
// code 409 if it already exists and
// code 500 in case of another error.
func handleCreateShortlink(c *gin.Context) {
	var shortlink Shortlink
	if err := c.ShouldBindJSON(&shortlink); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if invalidShort(shortlink.ShortUrl, c) || invalidURL(shortlink.LongUrl, c) {
		return
	}

	err := Create(&shortlink)
	if err != nil {
		if isDuplicateError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "shortlink already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

// Handler for PUT /shortlinks/:short
// Updates the shortlink with the provided data.
// Returns code 200 with the updated shortlink as json on success,
// code 400 if the data is invalid,
// code 404 if the short link does not exist
// code 409 if there is already a shortlink with the updated short and
// code 500 in case of another error.
func handleUpdateShortlink(c *gin.Context) {
	short := c.Param("short")
	if invalidShort(short, c) {
		return
	}
	var shortlink ShortlinkUpdate
	if err := c.ShouldBindJSON(&shortlink); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if invalidShort(shortlink.ShortUrl, c) || invalidURL(shortlink.LongUrl, c) {
		return
	}

	savedShortlink, err := Update(short, &shortlink)
	if err != nil {
		if isDuplicateError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "shortlink already exists"})
			return
		}
		if isNotFundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "shortlink not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, savedShortlink)
}

// Handler for DELETE /shortlinks/:short
// Returns code 200 with {deleted:1} on success if the shortlink existed,
// code 200 with {deleted:0} if the provided shortlink did not exist,
// code 400 if the provided short is invalid and
// code 500 in case of another error.
func handleDeleteShortlink(c *gin.Context) {
	short := c.Param("short")
	if invalidShort(short, c) {
		return
	}

	num_deleted, err := Delete(short)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": num_deleted})
}

// Handler for GET /go/:short
// Returns code 307 (TemporaryRedirect) to the saved link on success,
// code 400 if the shortlink is invalid, 404 if it doesn't exist and
// code 500 in case of another error.
func handleRedirect(c *gin.Context) {
	short := c.Param("short")
	if invalidShort(short, c) {
		return
	}

	link, err := GetRedirect(short)
	if err != nil {
		if isNotFundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("no redirect for %s", short)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, link)
}

// Handler for GET /check/:short
// Returns code 200 with {free:true} if the shortlink does not exist,
// code 200 with {free:false} if the shortlink does exist or
// code 400 if the shortlink is invalid and
// code 500 in case of another error.
func handleCheck(c *gin.Context) {
	short := c.Param("short")
	if invalidShort(short, c) {
		return
	}

	free, err := IsFree(short)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"free": free})
}

/* ********************************************** *\
 * ***************** VALIDATORS ***************** *
\* ********************************************** */

// The Validators accept a gin Context to which they write StatusBadRequest if the input is invalid.

// invalidURL returns true if the provided string does not represent a valid URL
func invalidURL(input string, c *gin.Context) bool {
	u, err := url.ParseRequestURI(input)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Printf("Checked invald url: %v", input)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect url"})
		return true
	}
	return false
}

// invalidShort returns true if the provided string does not match ^[a-zA-Z0-9\-_]+$
func invalidShort(input string, c *gin.Context) bool {
	m, e := regexp.MatchString("^[a-zA-Z0-9\\-_]+$", input)
	if !m || e != nil {
		log.Printf("Checked invald short: %v", input)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid short does not match ^[a-zA-Z0-9\\-_]+$"})
		return true
	}
	return false
}

/* ********************************************** *\
 * **************** SETUP ROUTES **************** *
\* ********************************************** */

// Setup the gin router
func setupRoutes() *gin.Engine {
	router := gin.Default()

	// Optionally set CORS to allow all origins.
	// See https://github.com/gin-contrib/cors
	// router.Use(cors.Default())

	// Redirect service
	router.GET("/go/:short", handleRedirect)

	// CRUD operations
	router.GET("/shortlinks", handleGetShortlinks)
	router.GET("/shortlinks/:short", handleGetShortlink)
	router.PUT("/shortlinks/:short", handleUpdateShortlink)
	router.POST("/shortlinks", handleCreateShortlink)
	router.DELETE("/shortlinks/:short", handleDeleteShortlink)

	// Checking for free redirects
	router.GET("/check/:short", handleCheck)

	// Serve swagger-ui if ./swagger-dist exists.
	if _, err := os.Stat("swagger-dist"); !os.IsNotExist(err) {
		router.Static("/api", "./swagger-dist")
	}

	return router
}

/* ********************************************** *\
 * **************** MAIN FUNCTION *************** *
\* ********************************************** */

// Main function, connects to the MongoDB and sets up the router
func main() {
	// Connect to the MongoDB
	err := Connect()
	if err != nil {
		log.Fatal(err)
	}

	// Setup the routes
	router := setupRoutes()

	// listen and serve on port 8080 unless PORT is set
	router.Run()
}
