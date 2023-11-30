package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	// defines a user, used to interact with the database
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	created        time.Time
}

type UserModel struct {
	//	used to insert the user into the database
	DB *sql.DB
}

type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
}

func (m *UserModel) Insert(name, email, password string) error {
	// inserts a new user into the database

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created)
    VALUES(?, ?, ?, UTC_TIMESTAMP())`

	_, err = m.DB.Exec(stmt, name, email, string(hashedPassword))
	if err != nil {

		// check if the error is caused by the Email already existing
		// if so return a specific error message.
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}
	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	// checks for the existence of the relevant user using email and password,
	// returning their ID if they exist.
	var id int
	var hashedPassword []byte

	// Get id and password according to email from DB
	stmnt := "SELECT id, hashed_password FROM users WHERE email = ?"
	err := m.DB.QueryRow(stmnt, email).Scan(&id, &hashedPassword)
	if err != nil {
		return 0, err
	}

	// Check if password provided matches the hashed password stored in the DB
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// password matches email exists, return user ID
	return id, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	// checks if the user exists in the database given their ID.
	var exists bool

	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = ?)"

	err := m.DB.QueryRow(stmt, id).Scan(&exists)
	return exists, err
}
