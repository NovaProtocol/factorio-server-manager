package factorio

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type Credentials struct {
	Username string `json:"username"`
	Userkey  string `json:"userkey"`
}

func (credentials *Credentials) Save() error {
	var err error
	config := bootstrap.GetConfig()
	credentialsJson, err := json.Marshal(credentials)
	if err != nil {
		log.Printf("error mashalling the credentials: %s", err)
		return err
	}

	// 0600: the file holds the mod portal token in clear, and only the manager's
	// own account has any business reading it. WriteFile alone would keep the
	// permissions of an existing file, so the mode is applied either way.
	err = ioutil.WriteFile(config.FactorioCredentialsFile, credentialsJson, 0600)
	if err == nil {
		err = os.Chmod(config.FactorioCredentialsFile, 0600)
	}
	if err != nil {
		log.Printf("error on saving the credentials. %s", err)
		return err
	}

	return nil
}

func (credentials *Credentials) Load() (bool, error) {
	var err error
	config := bootstrap.GetConfig()
	if _, err := os.Stat(config.FactorioCredentialsFile); os.IsNotExist(err) {
		return false, nil
	}

	fileBytes, err := ioutil.ReadFile(config.FactorioCredentialsFile)
	if err != nil {
		credentials.Del()
		log.Printf("error reading CredentialsFile: %s", err)
		return false, err
	}

	err = json.Unmarshal(fileBytes, credentials)
	if err != nil {
		credentials.Del()
		log.Printf("error on unmarshal credentials_file: %s", err)
		return false, err
	}

	if credentials.Userkey != "" && credentials.Username != "" {
		return true, nil
	} else {
		credentials.Del()
		return false, errors.New("incredients incomplete")
	}
}

func (credentials *Credentials) Del() error {
	var err error
	config := bootstrap.GetConfig()
	err = os.Remove(config.FactorioCredentialsFile)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("error delete the credentialfile: %s", err)
		return err
	}

	return nil
}

// Bookmarks lists the mods the account has bookmarked. It is what a stored
// token actually gives access to, so it doubles as the check that the
// credentials are valid: the mod portal answers a bad username or token with
// 403 and {"message":"Invalid username or token"}.
func (credentials *Credentials) Bookmarks() ([]string, error, int) {
	bookmarksUrl := "https://mods.factorio.com/api/bookmarks?username=" + url.QueryEscape(credentials.Username) +
		"&token=" + url.QueryEscape(credentials.Userkey)

	req, err := http.NewRequest(http.MethodGet, bookmarksUrl, nil)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		// The portal answers with {"message":"..."} on a rejected token; report
		// just that message rather than the raw body.
		var portalError struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &portalError); err == nil && portalError.Message != "" {
			return nil, errors.New(portalError.Message), resp.StatusCode
		}

		return nil, errors.New(string(body)), resp.StatusCode
	}

	var bookmarks []string
	if err := json.Unmarshal(body, &bookmarks); err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return bookmarks, nil, resp.StatusCode
}
