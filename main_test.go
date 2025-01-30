package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of sql.DB
type MockDB struct {
	mock.Mock
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		username string
		phone    string
		want     bool
	}{
		{"Valid phone number", "1234567890", true},
		{"Valid phone with dashes", "123-456-7890", true},
		{"Valid phone with spaces", "123 456 7890", true},
		{"Invalid phone - too short", "123456789", false},
		{"Invalid phone - too long", "12345678901", false},
		{"Invalid phone - contains letters", "123abc4567", false},
		{"Empty phone", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			got := validatePhone(tt.phone)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		inputUser      User
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Valid user creation",
			inputUser: User{
				Username: "testuser",
				Phone:    "1234567890",
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "Invalid phone number",
			inputUser: User{
				Username: "testuser",
				Phone:    "123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Invalid user name",
			inputUser: User{
				Username: "abc",
				Phone:    "9898767654",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock db connection with debug logging
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			// Create a new gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create request with JSON body
			jsonValue, _ := json.Marshal(tt.inputUser)
			req := httptest.NewRequest("POST", "/create", bytes.NewBuffer(jsonValue))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			if !tt.expectedError {
				// Expect transaction begin
				mock.ExpectBegin()

				// Expect user insert with any account_id (since it's random)
				mock.ExpectQuery("INSERT INTO users (username, phone, account_id, balance) VALUES ($1, $2, $3, 0) RETURNING id").
					WithArgs(tt.inputUser.Username, tt.inputUser.Phone, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

				// Expect account insert
				mock.ExpectExec("INSERT INTO accounts (user_id, balance) VALUES ($1, 0)").
					WithArgs("1").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Expect transaction commit
				mock.ExpectCommit()
			}

			// Call the function
			createUser(c, db)

			// Check for unmet expectations
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled mock expectations: %v", err)
			}

			// Log the response body for debugging
			t.Logf("Response Status: %d", w.Code)
			t.Logf("Response Body: %s", w.Body.String())

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			} else {
				var response User
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.inputUser.Username, response.Username)
				assert.Equal(t, tt.inputUser.Phone, response.Phone)
				assert.NotEmpty(t, response.AccountID)
			}
		})
	}
}

func TestGetUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(mock sqlmock.Sqlmock)
		expectedStatus int
		expectedUsers  []User
	}{
		{
			name: "Successfully retrieve users",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "phone", "account_id", "balance"}).
					AddRow("1", "user1", "1234567890", "acc1", 100.0).
					AddRow("2", "user2", "0987654321", "acc2", 200.0)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedUsers: []User{
				{ID: "1", Username: "user1", Phone: "1234567890", AccountID: "acc1", Balance: 100.0},
				{ID: "2", Username: "user2", Phone: "0987654321", AccountID: "acc2", Balance: 200.0},
			},
		},
		{
			name: "No users found",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "phone", "account_id", "balance"})
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectedStatus: http.StatusNotFound,
			expectedUsers:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)

			getUsers(c, db)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedUsers != nil {
				var response []User
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUsers, response)
			}
		})
	}
}
