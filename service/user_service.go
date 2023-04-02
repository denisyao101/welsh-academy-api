package service

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/denisyao1/welsh-academy-api/exception"
	"github.com/denisyao1/welsh-academy-api/model"
	"github.com/denisyao1/welsh-academy-api/repository"
	"github.com/denisyao1/welsh-academy-api/schema"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	ValidateUserCreation(userSchema schema.CreateUserSchema) []error
	validateCredentials(loginSchema schema.LoginSchema) (model.User, error)
	CreateUser(userSchema schema.CreateUserSchema) (model.User, error)
	CreateAccessToken(loginSchema schema.LoginSchema) (string, error)
	UpdatePaswword(userID int, newPwd schema.PasswordSchema) error
	GetInfos(userID int) (model.User, error)
	CreateDefaultAdmin() error
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s userService) ValidateUserCreation(userSchema schema.CreateUserSchema) []error {
	var newErrValidation = exception.NewValidationError
	var errs []error

	if userSchema.Username == "" {
		errs = append(errs, newErrValidation("username", "username is required"))
	}

	// Username must be at least 3 characters long
	if len([]rune(userSchema.Username)) < 3 {
		errs = append(errs, newErrValidation("username", "username must be at least 3 characters long"))
	}

	if userSchema.Password == "" {
		errs = append(errs, newErrValidation("password", "password is required"))
	}

	// Password must be at least 4 characters long
	if len([]rune(userSchema.Password)) < 4 {
		errs = append(errs, newErrValidation("password", "password must be at least 4 characters long"))
	}

	return errs
}

func (s userService) CreateUser(userSchema schema.CreateUserSchema) (model.User, error) {

	user := model.User{Username: userSchema.Username, IsAdmin: userSchema.IsAdmin}

	// Check if username is already used in DB
	ok, checkErr := s.repo.IsNotCreated(user)

	if checkErr != nil {
		return user, checkErr
	}

	if !ok {
		return user, exception.ErrDuplicateKey
	}

	// hash password
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(userSchema.Password), 10)
	if hashErr != nil {
		return user, hashErr
	}

	user.Password = string(hash)

	err := s.repo.Create(&user)

	return user, err
}

func (s userService) validateCredentials(loginSchema schema.LoginSchema) (model.User, error) {
	var user model.User

	if loginSchema.Username == "" || loginSchema.Pasword == "" {
		return user, exception.ErrInvalidCredentials
	}

	// username must be at least 3 characters long; password at least 4 characters long
	if len([]rune(loginSchema.Username)) < 3 || len([]rune(loginSchema.Pasword)) < 4 {
		return user, exception.ErrInvalidCredentials
	}

	user.Username = loginSchema.Username
	err := s.repo.GetByUsername(&user)
	if err != nil {
		return user, err
	}

	if user.ID == 0 {
		return user, exception.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginSchema.Pasword))

	if err != nil {
		return user, exception.ErrInvalidCredentials
	}

	return user, nil
}

func (s userService) CreateAccessToken(loginSchema schema.LoginSchema) (string, error) {
	// validate user credentials
	user, err := s.validateCredentials(loginSchema)
	if err != nil {
		return "", err
	}

	role := model.RoleUser
	if user.IsAdmin {
		role = model.RoleAdmin
	}

	// Create the Claims
	claims := jwt.MapClaims{
		"ID":   strconv.Itoa(int(user.ID)),
		"role": role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	encodedToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return encodedToken, nil
}

func (s userService) UpdatePaswword(userID int, newPwdSchema schema.PasswordSchema) error {
	//check if password is valid
	if newPwdSchema.Password == "" || len([]rune(newPwdSchema.Password)) < 4 {
		return exception.ErrInvalidPassword
	}

	// retrieve user from database
	var user model.User
	user.ID = userID
	err := s.repo.GetByID(&user)
	if err != nil {
		return err
	}

	if user.ID == 0 {
		return exception.ErrRecordNotFound
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPwdSchema.Password))
	if err == nil {
		return exception.ErrPasswordSame
	}

	hash, hashErr := bcrypt.GenerateFromPassword([]byte(newPwdSchema.Password), 10)
	if hashErr != nil {
		return hashErr
	}

	user.Password = string(hash)

	return s.repo.UpdatePassword(&user)
}

func (s userService) GetInfos(userID int) (model.User, error) {
	var user model.User
	user.ID = userID
	err := s.repo.GetByID(&user)
	return user, err
}

func (s userService) CreateDefaultAdmin() error {
	var admin model.User
	admin.Username = "admin"
	admin.IsAdmin = true

	// check if admin is already created
	err := s.repo.GetByUsername(&admin)

	if err != nil && !errors.Is(err, exception.ErrRecordNotFound) {
		// Unexpected error occured exist
		log.Fatalln("Unexpected error : ", err.Error())
		log.Fatal("Failed to create default admin user")
	}

	if admin.ID != 0 {
		log.Println("Default admin alredy exists")
		return nil
	}

	// create hash for admin password password
	password := "admin"
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), 10)
	if hashErr != nil {
		log.Fatalln("Unexpected error : ", err.Error())
		log.Fatal("Failed to create default admin user")
	}

	admin.Password = string(hash)
	err = s.repo.Create(&admin)
	if err != nil {
		log.Fatalln("Unexpected error : ", err.Error())
		log.Fatal("Failed to create default admin user")
	}

	log.Println("default admin user created successfully")
	return nil
}
