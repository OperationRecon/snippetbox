package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"snippetbox.opre.net/internal/models"
	"snippetbox.opre.net/internal/validator"
)

// Create a struct that holds the form data and possible errors
type snippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

type userSignupFrom struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type accountPasswordUpdateForm struct {
	OldPassword         string `form:"old password"`
	NewPassword         string `form:"new password"`
	ConfirmPassword     string `form:"confirm password"`
	validator.Validator `form:"-"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {

	// get most recent snippets
	recentSnippets, err := app.snippets.Latest()

	if err != nil {
		app.serverError(w, err)
		return
	}

	if err != nil {
		app.serverError(w, err)
		return
	}

	// create new template
	data := app.newTemplateData(r)

	// add template files
	data.Snippets = recentSnippets

	// Pass in the templateData when executing the template.
	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	// render template
	app.render(w, http.StatusOK, "about.tmpl.html", app.newTemplateData(r))
}

// snippetView handler function
func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	// get parameters from request context
	parameters := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(parameters.ByName("id"))

	// check for invalid id input
	if err != nil || id < 1 {
		app.notFoundError(w)
		return
	}

	snippet, err := app.snippets.Get(id)

	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFoundError(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Snippet = snippet
	app.render(w, http.StatusOK, "view.tmpl.html", data)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	// open snippet creating form
	data := app.newTemplateData(r)
	data.Form = snippetCreateForm{
		Expires: 365,
	}
	app.render(w, http.StatusOK, "create.tmpl.html", data)
}

// function to post created snippets
func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {

	// Insert snippet into DB

	// create data form
	var createFrom snippetCreateForm

	// use the decoder to pass the value from the request into the form
	err := app.decodePostForm(r, &createFrom)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}

	// check for form validity
	createFrom.CheckField(validator.NotBlank(createFrom.Title), "title", "This field cannot be blank")
	createFrom.CheckField(validator.MaxChars(createFrom.Title, 100), "title", "This field cannot be more than 100 characters long")
	createFrom.CheckField(validator.NotBlank(createFrom.Content), "content", "This field cannot be blank")
	createFrom.CheckField(validator.PermittedValue(createFrom.Expires, 1, 7, 365), "expires", "This field must equal 1, 7 or 365")

	// Validation erros, re-render form
	if !(createFrom.Valid()) {
		data := app.newTemplateData(r)
		data.Form = createFrom
		app.render(w, http.StatusUnprocessableEntity, "create.tmpl.html", data)
		return
	}

	// all clear? insert snippet into DB
	id, err := app.snippets.Insert(createFrom.Title, createFrom.Content, createFrom.Expires)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if err != nil {
		app.serverError(w, err)
		return
	}

	// Use the Put() method to add a string value ("Snippet successfully
	// created!") and the corresponding key ("flash") to the session data.
	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	// Redirect user to viewing newly-created snippet
	// build URL
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	// send out the sign up form template
	data := app.newTemplateData(r)
	data.Form = userSignupFrom{}
	app.render(w, http.StatusOK, "signup.tmpl.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	// Validates the from posted to create a new user and adds them to the database

	// initalize form
	var form userSignupFrom

	// Decode and place form data into form
	err := app.decodePostForm(r, &form)

	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// validate form data
	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	if !form.Valid() {
		// invalid user input, return to signup form and re-enter data, fill
		// pre-existing fields except for the password
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
		return
	}

	// Insert user into DB
	err = app.users.Insert(form.Name, form.Email, form.Password)

	if err != nil {
		// check if the Error is caused by a dublicate Email,
		//  if so, Re-render template with given data
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)

		} else {
			app.serverError(w, err)

		}
		return
	}
	// all good? add flash message to notify of success
	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")
	// and redirect to login page
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.tmpl.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	// initalize form
	var form userLoginForm

	// Decode and place form data into form
	err := app.decodePostForm(r, &form)

	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// validate form data
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		return
	}

	// Check whether the credentials are valid. If they're not, add a generic
	// non-field error message and re-display the login page.
	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Add the ID of the current user to the session, so that they are now
	// 'logged in'.

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	path := app.sessionManager.PopString(r.Context(), "redirectPathAfterLogin")

	// Check what path the user tried to access before logging in,
	//  Redirect them to that path

	if path != "" {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}
	// Redirect the user to the create snippet page.
	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	// Use the RenewToken() method on the current session to change the session
	// ID again.
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Remove the authenticatedUserID from the session data so that the user is
	// 'logged out'.
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	// Add a flash message to the session to confirm to the user that they've been
	// logged out.
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	// Redirect the user to the application home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// View the logged in user's account
func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	user, err := app.users.Get(id)
	app.infoLog.Print(user)
	if err != nil {
		//  Check if the Error is not finding the user
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}
		app.serverError(w, err)
		return
	}
	app.infoLog.Print(user)
	// Wrtie user data into template
	data := app.newTemplateData(r)
	data.User = user

	app.render(w, http.StatusOK, "account.tmpl.html", data)
}

// Display From for creating a new password
func (app *application) accountPasswordUpdate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = accountPasswordUpdateForm{}

	app.render(w, http.StatusOK, "password.tmpl.html", data)
}

// Validates New password and updates the DB with it
func (app *application) accountPasswordUpdatePost(w http.ResponseWriter, r *http.Request) {
	var form accountPasswordUpdateForm

	// Decode and place form data into form
	err := app.decodePostForm(r, &form)

	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate Form
	values := []string{
		form.OldPassword,
		form.NewPassword,
		form.ConfirmPassword,
	}
	keys := []string{
		"oldPassword",
		"newPassword",
		"confirmPassword",
	}

	for i := 0; i < len(values); i++ {
		form.Validator.CheckField(validator.NotBlank(values[i]), keys[i], "This field cannot be blank")
		form.Validator.CheckField(validator.MinChars(values[i], 8), keys[i], "this field must be 8 Characters long")
	}
	form.Validator.CheckField(values[0] != values[1], keys[1], "New password can't match old one")
	form.Validator.CheckField(values[1] == values[2], keys[2], "Confirmation must match")

	if !form.Valid() {
		// invalid form re-render page
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "password.tmpl.html", data)
		return
	}

	// try to upadte DB
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	err = app.users.PasswordUpdate(id, form.OldPassword, form.NewPassword)

	// Make sure the  currnt password is the correct one
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddFieldError(keys[0], "Incorrect password")
			// invalid form re-render page
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "password.tmpl.html", data)
			return
		}

		app.serverError(w, err)
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	// all good? add flash message to notify of success
	app.sessionManager.Put(r.Context(), "flash", "Password Changed Succesfully")
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	// Redirect user to their account page
	http.Redirect(w, r, "/account/view", http.StatusSeeOther)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
