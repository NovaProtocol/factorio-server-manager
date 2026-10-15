package api

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// A second account under one name is refused rather than added: the model wants a
// unique username, but a database that already holds duplicates keeps that index
// from being created, so the check lives in addUser as well.
func TestAddUserRefusesADuplicateName(t *testing.T) {
	auth := testAuth(t)

	require.NoError(t, auth.addUser(User{Username: "viewer", Password: "first", Role: "user"}))

	err := auth.addUser(User{Username: "viewer", Password: "second", Role: "user"})
	assert.ErrorIs(t, err, ErrUserExists)

	var count int64
	require.NoError(t, auth.db.Model(&User{}).Where(&User{Username: "viewer"}).Count(&count).Error)
	assert.EqualValues(t, 1, count, "the duplicate must not have been written")
}

func TestAddUserStillAcceptsDistinctNames(t *testing.T) {
	auth := testAuth(t)

	require.NoError(t, auth.addUser(User{Username: "viewer", Password: "pw", Role: "user"}))
	require.NoError(t, auth.addUser(User{Username: "operator", Password: "pw", Role: "user"}))

	var count int64
	require.NoError(t, auth.db.Model(&User{}).Count(&count).Error)
	assert.EqualValues(t, 2, count)
}

// testAuth builds an auth on its own in-memory database, so a test never touches
// the manager's real user list.
func testAuth(t *testing.T) *Auth {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), nil)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))

	return &Auth{db: db}
}
