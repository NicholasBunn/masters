package authentication

import (
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type User struct {
	/* This struct describes the user, as their info will
	be stored in the DB */
	Username       string
	HashedPassword string
	Role           string
}

func CreateUser(username string, password string, role string) (*User, error) {
	// This function creates and returns a new user object

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &User{
		Username:       username,
		HashedPassword: string(hashedPassword),
		Role:           role,
	}

	return user, nil
}

func (user *User) CheckPassword(password string) bool {
	/* This function checks whether the password provided for the user is the same
	as the password stored for that user */
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))

	return err == nil
}
