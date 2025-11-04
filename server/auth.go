package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

func getToken(tokenStr string) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	user := &User{
		Username: claims["username"].(string),
		Color:    claims["color"].(string),
		UserID:   claims["id"].(string),
	}

	return user, nil
}

func setToken(user *User, w http.ResponseWriter) error {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"color":    user.Color,
		"id":       user.UserID,
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return errors.New("failed to sign key: " + err.Error())
	}

	cookie := &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		HttpOnly: true,
		Secure:   false, // make this secure later
		Path:     "/",
		MaxAge:   0,
	}

	http.SetCookie(w, cookie)

	return nil
}

func getUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	cookie, err := r.Cookie("token")
	var user *User
	if err != nil {
		user = &User{
			Color:    colors[rand.Intn(len(colors))],
			UserID:   randomString(10),
			Username: "guest_" + randomString(4),
		}

		err = setToken(user, w)
		if err != nil {
			return nil, err
		}
	} else {
		newUser, err := getToken(cookie.Value)
		if err != nil {
			return nil, err
		}
		user = newUser
	}

	// excellent
	// source of
	// vitamin c
	// same character length??

	usersMu.Lock()
	_, ok := users[user.UserID]
	if !ok {
		users[user.UserID] = user
	}
	usersMu.Unlock()

	return user, nil
}
