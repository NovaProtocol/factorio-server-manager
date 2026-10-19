package api

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type contextKey string

// authenticatedUserKey carries the resolved user from AuthMiddleware to the
// handler, so a role check does not have to read the session and the database a
// second time.
const authenticatedUserKey contextKey = "authenticatedUser"

type User bootstrap.User

// sessionCookieName is the cookie this manager keeps its session in. It is
// deliberately not "authentication": cookies are scoped by host, not by port,
// so an older manager running on the same host would overwrite this session
// (and this manager would delete its cookie as unreadable). A distinct name
// lets several managers share a host without signing each other out.
const sessionCookieName = "factorio_multiworld_session"

type Auth struct {
	db *gorm.DB
}

var (
	sessionStore *sessions.CookieStore
	auth         Auth
)

func SetupAuth() {
	var err error

	config := bootstrap.GetConfig()

	cookieEncryptionKey, err := base64.StdEncoding.DecodeString(config.CookieEncryptionKey)
	if err != nil {
		log.Printf("Error decoding base64 cookie encryption key: %s", err)
		panic(err)
	}
	sessionStore = sessions.NewCookieStore(cookieEncryptionKey)
	sessionStore.Options = &sessions.Options{
		Path:   "/",
		Secure: config.Secure,
		// Stated rather than left to the browser default: a cross-site request
		// must not carry the session, and this is the only guard on top of the
		// routes that change server state.
		SameSite: http.SameSiteLaxMode,
	}

	auth.db, err = gorm.Open(sqlite.Open(config.SQLiteDatabaseFile), nil)
	if err != nil {
		log.Printf("Error opening sqlite or gorm database: %s", err)
		panic(err)
	}

	err = auth.db.AutoMigrate(&User{})
	if err != nil {
		log.Printf("Error AutoMigrating gorm database: %s", err)
		panic(err)
	}

	auth.reportDuplicateUsernames()

	// The admin credentials are configuration, not state: they are re-applied on
	// every start so the account cannot drift from what the operator configured.
	if err := auth.ensureConfiguredAdmin(config.Username, config.Password); err != nil {
		log.Printf("Error enforcing the configured admin credentials: %s", err)
		panic(err)
	}
}

// ensureConfiguredAdmin makes the configured credentials authoritative. It runs on
// every start, so the admin account cannot drift from the configuration: a
// password changed through the interface, or a database left behind by an earlier
// run, is put back to what the operator configured. Other users are untouched.
func (a *Auth) ensureConfiguredAdmin(username, password string) error {
	if username == "" || password == "" {
		return errors.New("the admin username and password must both be configured")
	}

	user, err := a.getUser(username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newUser := User{Username: username, Password: password, Role: "admin"}
		if err := a.addUser(newUser); err != nil {
			return err
		}

		log.Printf("Created the configured admin user %q", username)
		return nil
	}
	if err != nil {
		return err
	}

	if user.Role != "admin" {
		user.Role = "admin"
		if err := a.db.Save(&user).Error; err != nil {
			return err
		}

		log.Printf("Restored the admin role on user %q", username)
	}

	if err := a.checkPassword(username, password); err != nil {
		if err := a.changePassword(username, password); err != nil {
			return err
		}

		log.Printf("Reset the password of admin user %q to the configured one", username)
	}

	return nil
}

func (a *Auth) checkPassword(username, password string) error {
	var user User
	result := a.db.Where(&User{Username: username}).Take(&user)
	if result.Error != nil {
		log.Printf("Error reading user from database: %s", result.Error)
		return result.Error
	}

	decodedHashPw, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		log.Printf("Error decoding base64 password: %s", err)
		return err
	}

	err = bcrypt.CompareHashAndPassword(decodedHashPw, []byte(password))
	if err != nil {
		if err != bcrypt.ErrMismatchedHashAndPassword {
			log.Printf("Unexpected error comparing hash and pw: %s", err)
		}
		return err
	}

	// Password correct
	return nil
}

func (a *Auth) deleteUser(username string) error {
	adminUsers := []User{}
	adminQuery := a.db.Find(&User{}).Where(&User{Role: "admin"}).Find(&adminUsers)
	if adminQuery.Error != nil {
		log.Printf("Error retrieving admin user list from database: %s", adminQuery.Error)
		return adminQuery.Error
	}

	for _, user := range adminUsers {
		if user.Username == username {
			if adminQuery.RowsAffected == 1 {
				return errors.New("cannot delete single admin user")
			}
		}
	}

	result := a.db.Model(&User{}).Where(&User{Username: username}).Delete(&User{})
	if result.Error != nil {
		log.Printf("Error deleting user from database: %s", result.Error)
		return result.Error
	}
	return nil
}

func (a *Auth) hasUser(username string) (bool, error) {
	var count int64
	result := a.db.Model(&User{}).Where(&User{Username: username}).Count(&count)
	if result.Error != nil {
		log.Printf("Error checking if user exists in database: %s", result.Error)
		return false, result.Error
	}
	return count == 1, nil
}

func (a *Auth) getUser(username string) (User, error) {
	var user User
	result := a.db.Model(&User{}).Where(&User{Username: username}).Take(&user)
	if result.Error != nil {
		log.Printf("Error reading user from database: %s", result.Error)
		return User{}, result.Error
	}

	return user, nil
}

func (a *Auth) listUsers() ([]User, error) {
	var users []User
	result := a.db.Find(&users)
	if result.Error != nil {
		log.Printf("Error listing all users in database: %s", result.Error)
		return nil, result.Error
	}
	return users, nil
}

// reportDuplicateUsernames says out loud what the schema cannot enforce: once a
// database holds two accounts under one name the unique index cannot be created,
// and the operator deserves to know before wondering why a name is listed twice.
func (a *Auth) reportDuplicateUsernames() {
	var duplicates []struct {
		Username string
		Total    int64
	}

	if err := a.db.Model(&User{}).
		Select("username, count(*) as total").
		Group("username").
		Having("count(*) > 1").
		Scan(&duplicates).Error; err != nil {
		log.Printf("Could not check for duplicate usernames: %s", err)
		return
	}

	for _, duplicate := range duplicates {
		log.Printf("Warning: %d accounts are named %q; remove the extras to let the username index be created",
			duplicate.Total, duplicate.Username)
	}
}

// ErrUserExists is what a second account with the same name gets, instead of a
// silent duplicate row. The username column carries a unique index in the model,
// but a database that already holds duplicates cannot have it created, so the
// check has to be here too — a duplicate account is a confusing thing to hand an
// operator, whatever the schema says.
var ErrUserExists = errors.New("a user with this name already exists")

func (a *Auth) addUser(user User) error {
	var existing int64
	if result := a.db.Model(&User{}).Where(&User{Username: user.Username}).Count(&existing); result.Error != nil {
		log.Printf("Error checking for an existing user: %s", result.Error)
		return result.Error
	}
	if existing > 0 {
		return ErrUserExists
	}

	// encrypt password
	pwHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error generating bcrypt hash from password: %s", err)
		return err
	}

	user.Password = base64.StdEncoding.EncodeToString(pwHash)

	// add user to db
	result := a.db.Create(&user)
	if result.Error != nil {
		log.Printf("Error creating user in database: %s", result.Error)
		return result.Error
	}

	return nil
}

func (a *Auth) addUserWithHash(user User) error {
	// add user to db
	result := a.db.Create(&user)
	if result.Error != nil {
		log.Printf("Error creating user in database: %s", result.Error)
		return result.Error
	}

	return nil
}

func (a *Auth) changePassword(username, password string) error {
	var user User
	result := a.db.Model(&User{}).Where(&User{Username: username}).Take(&user)
	if result.Error != nil {
		log.Printf("Error reading user from database: %s", result.Error)
		return result.Error
	}

	hashPW, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error generatig bcrypt hash from new password: %s", err)
		return err
	}

	user.Password = base64.StdEncoding.EncodeToString(hashPW)

	result = a.db.Save(&user)
	if result.Error != nil {
		log.Printf("Error resaving user in database: %s", result.Error)
		return result.Error
	}

	return nil
}

// middleware function, that will be called for every request, that has to be authorized
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionCookieName)
		if err != nil {
			if session != nil {
				session.Options.MaxAge = -1
				err2 := session.Save(r, w)
				if err2 != nil {
					log.Printf("Error deleting cookie: %s", err2)
				}
			}
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		username, ok := session.Values["username"]
		if !ok {
			http.Error(w, "Could not read username from sessioncookie", http.StatusUnauthorized)
			return
		}

		// The user is loaded on every request rather than trusted from the
		// cookie, so a role change or a deleted account takes effect at once.
		user, err := auth.getUser(username.(string))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Unauthenticated request %s %s %s", r.Method, r.Host, r.RequestURI)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authenticatedUserKey, user)))
	})
}

// currentUser returns the user AuthMiddleware resolved for this request.
func currentUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(authenticatedUserKey).(User)
	return user, ok
}

// RequireAdmin gates a route on the admin role. It assumes AuthMiddleware has
// already run, so a request from a non-admin is answered with 403 rather than a
// login redirect.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok || user.Role != "admin" {
			http.Error(w, "forbidden: this action requires the admin role", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
