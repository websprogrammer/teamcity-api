package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/gcfg.v1"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var redisClient *redis.Client

var config = struct {
	Config struct {
		Username string
		Password string
	}
}{}

var upgrader websocket.Upgrader

const (
	// Proxy value to use in API requests. Put your proxy if needed.
	proxyStr = "http://proxy:80"

	// TeamCity API URL. Put your url here.
	buildURL = "http://teamcity.local:8111/app/rest/builds"

	// TeamCity log file URL.
	logURL = "http://teamcity.local:8111/httpAuth/downloadBuildLog.html?buildId=%s&archived=true"
)

func init() {
	// Create a new client for Redis Server.
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set by default
		DB:       0,  // use default DB
	})

	// Initialize parameters for WebSocket connection.
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Read config from .ini file.
	if err := gcfg.ReadFileInto(&config, "conf.ini"); err != nil {
		panic(err)
	}
}


func handler(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return conn, nil
}



// Download file from API
func downloadFile(filepath string, fileUrl string) error {
	response := fetchData(fileUrl, false)
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	return err
}

// Fetch response from API.
func fetchData(requestURL string, useHeader bool) *http.Response {
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}

	//adding the proxy settings to the Transport object
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	//adding the Transport object to the http Client
	client := &http.Client{
		Transport: transport,
	}

	request, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Println(err)
	}

	request.SetBasicAuth(config.Config.Username, config.Config.Password)
	if useHeader {
		request.Header.Set("Accept", "application/json")
	}

	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}

	return response
}

// Custom html/template renderer for Echo framework.
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document.
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Fetch data from API and set it to Redis.
// We update the build information every 10 seconds.
func dataWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		response := fetchData(buildURL, true)
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Error receiving data from API. %s", err)
			return
		}

		if err := redisClient.Set("Prod", data, 0).Err(); err != nil {
			log.Fatalf("Error setting data to Redis. %s", err)
			return
		}
	}
}

func main() {
	go dataWorker()

	// Create a new Pool of WebSocket connections.
	pool := NewPool()
	go pool.Start()

	// Create an instance of Echo web framework.
	e := echo.New()

	// Log the information about each HTTP request.
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Implement CORS specification.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET,
			echo.PUT,
			echo.POST,
			echo.DELETE},
	}))

	// Serve static files.
	e.Static("/static", "assets")

	// Template Rendering.
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("*.html")),
	}
	e.Renderer = renderer

	// Handle home page.
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "template.html", []interface{}{})
	})

	// Handle WebSocket connection.
	e.GET("/get_info", func(c echo.Context) error {
		conn, err := handler(c.Response(), c.Request())
		if err != nil {
			log.Fatalf("WebSocket connection error. %s", err)
			return err
		}

		// Create a new Websocket client.
		client := &Client{
			Conn: conn,
			Pool: pool,
		}

		pool.Register <- client
		client.read()

		return nil
	})

	// Log file uploader.
	e.GET("/build_log/:build_id", func(c echo.Context) error {
		buildId := c.Param("build_id")

		filename := fmt.Sprintf("build_log_%s.zip", buildId)
		filepath := "files/" + filename

		fileUrl := fmt.Sprintf(logURL, buildId)

		if err := downloadFile(filepath, fileUrl); err != nil {
			log.Fatalf("Error downloading log file. %s", err)
		}

		return c.Inline(filepath, filename)
	})

	e.Logger.Fatal(e.Start("0.0.0.0:7777"))
}
