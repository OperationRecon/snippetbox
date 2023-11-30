package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"snippetbox.opre.net/internal/models"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

// adding an application struct to hold app-wide dependencies
type application struct {
	errLog         *log.Logger
	infoLog        *log.Logger
	snippets       models.SnippetModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	users          models.UserModelInterface
	debugMode      bool
}

func main() {

	// create loggers
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errLog := log.New(os.Stderr, "Error\t", log.Lshortfile|log.Ldate|log.Ltime)

	// specify address from cmd input
	address := flag.String("address", ":4000", "The address used to host the server")

	// define MySQL DSN from cmd input
	dsn := flag.String("dsn", "web:M4N@/snippetbox?parseTime=true", "MySQL data source name.")

	debugMode := flag.Bool("debug", false, "Start the server in debug mode.")

	flag.Parse()

	// open MySQL database
	db, err := openDB(*dsn)

	if err != nil {
		errLog.Fatal(err)
	}
	defer db.Close()

	// create a template cache for html pages
	templateCache, err := newTemplateCache()

	if err != nil {
		errLog.Fatal(err)
	}

	// initalize a session manager, that uses the database to
	// store session data and keep each session up for 12hrs
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	// used to switch to https
	sessionManager.Cookie.Secure = true

	// Initialize a decoder instance...
	formDecoder := form.NewDecoder()

	// create backend app
	app := &application{
		// create new loggers for info and errors
		infoLog:        infoLog,
		errLog:         errLog,
		snippets:       &models.SnippetModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
		users:          &models.UserModel{DB: db},
		debugMode:      *debugMode,
	}

	// Initialize a tls.Config struct to hold the non-default TLS settings we
	// want the server to use.
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	// start HTTP server using a struct
	server := &http.Server{
		Addr:         *address,
		Handler:      app.routes(),
		ErrorLog:     errLog,
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	defer server.Close()

	infoLog.Printf("Starting server on %s", *address)

	err = server.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	// opens mySQL database connection, makes sure connection is alive
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
