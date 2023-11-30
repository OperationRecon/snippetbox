package models

import (
	"database/sql"
	"errors"
	"time"
)

type SnippetModelInterface interface {
	Insert(title string, content string, expires int) (int, error)
	Get(id int) (*Snippet, error)
	Latest() ([]*Snippet, error)
}

// Type that holds data of individual snippets
type Snippet struct {
	ID      int
	Title   string
	Content string
	Created time.Time
	Expires time.Time
}

// Model used to access snippet DB
type SnippetModel struct {
	DB *sql.DB
}

// adding new snippet to DB returns its ID and possible error
func (model *SnippetModel) Insert(title string, content string, expiry int) (int, error) {
	statement := `INSERT INTO snippets (title, content, created, expires) 
	VALUES (?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY))`
	result, err := model.DB.Exec(statement, title, content, expiry)

	if err != nil {
		return 0, err
	}

	// get resulting id
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil

}

// get specfic snippet by id
func (model *SnippetModel) Get(ID int) (*Snippet, error) {
	statement := `SELECT title, content, created, expires FROM snippets 
				WHERE expires > UTC_TIMESTAMP() AND id = ?`

	row := model.DB.QueryRow(statement, ID)

	// parse values into vraibles to place into a snippet object
	snippet := &Snippet{
		ID: ID,
	}
	err := row.Scan(&snippet.Title, &snippet.Content, &snippet.Created, &snippet.Expires)

	if err != nil {
		// check for the no rows error specifically
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return snippet, nil
}

// get most recent snippets
func (model *SnippetModel) Latest() ([]*Snippet, error) {
	statement := `SELECT id, title, content, created, expires FROM snippets 
	WHERE expires > UTC_TIMESTAMP() ORDER BY id DESC LIMIT 10`

	rows, err := model.DB.Query(statement)

	if err != nil {
		return nil, err
	}

	// create place to hold snippets
	snippets := []*Snippet{}

	defer rows.Close()

	// iterate over rows
	for rows.Next() {
		// create place to hold an idvidual snippet
		snippet := &Snippet{}

		err := rows.Scan(&snippet.ID, &snippet.Title, &snippet.Content,
			&snippet.Created, &snippet.Expires)

		if err != nil {
			return nil, err
		}

		snippets = append(snippets, snippet)
	}

	// make sure iteration went without a hitch
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// all good? return snippets
	return snippets, nil
}
