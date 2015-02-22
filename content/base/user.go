package base

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"

	"github.com/urandom/readeef/content/info"
)

type User struct {
	ArticleSorting
	Error

	info info.User
}

func (u User) String() string {
	if u.info.Email == "" {
		return u.info.FirstName + " " + u.info.LastName
	} else {
		return u.info.FirstName + " " + u.info.LastName + " <" + u.info.Email + ">"
	}
}

func (u *User) Set(info info.User) {
	if u.Err() != nil {
		return
	}

	var err error

	if len(info.ProfileJSON) == 0 {
		info.ProfileJSON, err = json.Marshal(info.ProfileData)
	} else {
		if len(info.ProfileJSON) != 0 {
			if err = json.Unmarshal(info.ProfileJSON, &info.ProfileData); err != nil {
				u.SetErr(err)
				return
			}
		}
		if info.ProfileData == nil {
			info.ProfileData = make(map[string]interface{})
		}
	}

	u.SetErr(err)
	u.info = info
}

func (u User) Info() info.User {
	return u.info
}

func (u User) Validate() error {
	if u.info.Login == "" {
		return ValidationError{errors.New("Invalid user login")}
	}
	if u.info.Email != "" {
		if _, err := mail.ParseAddress(u.String()); err != nil {
			return ValidationError{err}
		}
	}

	return nil
}

func (u *User) Password(password string, secret []byte) {
	if u.Err() != nil {
		return
	}

	h := md5.Sum([]byte(fmt.Sprintf("%s:%s", u.info.Login, password)))

	u.info.MD5API = h[:]

	c := 30
	salt := make([]byte, c)
	if _, err := rand.Read(salt); err != nil {
		u.SetErr(err)
		return
	}

	u.info.Salt = salt

	u.info.HashType = "sha1"
	u.info.Hash = u.generateHash(password, secret)
}

func (u User) Authenticate(password string, secret []byte) bool {
	return bytes.Equal(u.info.Hash, u.generateHash(password, secret))
}

func (u User) generateHash(password string, secret []byte) []byte {
	hash := sha1.Sum(append(secret, append(u.info.Salt, []byte(password)...)...))

	return hash[:]
}
