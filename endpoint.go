package simple

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fnproject/fn/api/server"
)

// SimpleEndpoint is used for logging in. Returns a JWT token if successful.
type SimpleEndpoint struct {
	simple *SimpleAuth
}

func (e *SimpleEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("SIMPLEENDPOINT SERVEHTTP")
	ctx := r.Context()
	// parse JSON input containing username and password
	var login Login
	err := json.NewDecoder(r.Body).Decode(&login)
	if err != nil {
		server.HandleErrorResponse(ctx, w, err)
		return
	}
	var masterToken string
	ahSplit := strings.Split(r.Header.Get("Authorization"), " ")


	if len(ahSplit) == 2 {
		masterToken = ahSplit[1]
	}
	user, created, err := e.authenticate(ctx, &login, masterToken)
	if err != nil {
		server.WriteError(ctx, w, http.StatusUnauthorized, err)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":      time.Now().Unix(),
		"user_id":  user.ID,
		"username": user.Username,
	})
	tokenString, err := token.SignedString([]byte(os.Getenv(EnvSecret)))
	if err != nil {
		server.HandleErrorResponse(ctx, w, err)
		return
	}
	var msg string
	if created {
		msg = "New user created"
	} else {
		msg = "Thanks for coming back!"
	}
	json.NewEncoder(w).Encode(LoginResponse{Token: tokenString, Msg: msg})
}

func (e *SimpleEndpoint) authenticate(ctx context.Context, login *Login, masterToken string) (*User, bool, error) {
	user, err := e.simple.findUser(ctx, login.Username, login.Password)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		// check pass
		err := CheckPasswordHash(user.PassHash, login.Password)
		if err != nil {
			return nil, false, err
		}
		return user, false, nil
	}
	// Since this is dumb, we'll just automatically create a user and return a token if user doesn't already exist and the master token is corrent.
	if masterToken != os.Getenv(EnvMasterToken) {
		fmt.Errorf("Master token mismatch.")
		return nil, false, fmt.Errorf("Authentication Failed")
	}
	user, err = e.simple.createUser(ctx, login.Username, login.Password)
	if err != nil {
		return nil, false, err
	}
	return user, true, nil
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Msg   string `json:"msg"`
}
