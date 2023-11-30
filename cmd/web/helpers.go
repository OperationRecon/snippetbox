package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/justinas/nosurf"
)

// Help send out server error messages
func (app *application) serverError(w http.ResponseWriter, err error) {
	// print error message to log
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errLog.Output(2, trace)

	// respond to request with an error 500
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

}

// send client error reply
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// Send E404
func (app *application) notFoundError(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	// Retrieve the appropriate template set from the cache based on the page
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}

	// create buffer to run template on and check for runtime errors
	buff := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buff, page, data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// no runtime error, render page.
	w.WriteHeader(status)

	// Execute the template set and write the response body.
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.serverError(w, err)
	}
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	// return template data to be used by app durinh current session
	return &templateData{
		CurrentYear: time.Now().Year(),
		// acts like a one-time fetch. If there is no matching key in the session
		// data this will return the empty string.
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
		CSRFToken:       nosurf.Token(r),
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	// Call ParseForm() on the request, in the same way that we did in our
	// createSnippetPost handler.
	err := r.ParseForm()
	if err != nil {
		return err
	}

	// Call Decode() on our decoder instance, passing the target destination as
	// the first parameter.
	err = app.formDecoder.Decode(&dst, r.PostForm)
	if err != nil {
		// If we try to use an invalid target destination, the Decode() method
		// will return an error with the type *form.InvalidDecoderError.We use
		// errors.As() to check for this and raise a panic rather than returning
		// the error.
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		// For all other errors, we return them as normal.
		return err
	}
	app.infoLog.Printf("%v", dst)
	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	// Checks if the user making the request is logged in or not
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}

	return isAuthenticated

}
