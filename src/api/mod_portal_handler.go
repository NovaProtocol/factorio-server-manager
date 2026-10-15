package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func ModPortalListModsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var statusCode int
	resp, err, statusCode = factorio.ModPortalList()
	w.WriteHeader(statusCode)
	if err != nil {
		resp = fmt.Sprintf("Error in listing mods from mod portal: %s\nresponse: %+v", err, resp)
		log.Println(resp)
		return
	}
}

// ModPortalModInfoHandler returns JSON response with the mod details
func ModPortalModInfoHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	modId := vars["mod"]

	var statusCode int
	resp, err, statusCode = factorio.ModPortalModDetails(modId)

	if err != nil {
		resp = fmt.Sprintf("Error in getting mod details from mod portal: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
}

func ModPortalInstallHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	// Get Data out of the request
	var data struct {
		DownloadURL string `json:"downloadUrl"`
		Filename    string `json:"fileName"`
		ModName     string `json:"modName"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	mods, resp, err := CreateNewMods(w)
	if err != nil {
		return
	}

	err = mods.DownloadMod(data.DownloadURL, data.Filename, data.ModName)
	if err != nil {
		resp = fmt.Sprintf("Error downloading a mod: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = mods.ListInstalledMods()
}

// CredentialsStatus is what the credentials page renders: whether an account is
// stored, whose it is, the account data the stored credentials reach, and a
// message when they are present but the mod portal rejected them.
type CredentialsStatus struct {
	LoggedIn  bool     `json:"loggedIn"`
	Username  string   `json:"username"`
	Bookmarks []string `json:"bookmarks"`
	Error     string   `json:"error"`
}

func credentialsStatus(credentials factorio.Credentials, bookmarks []string, statusError string) CredentialsStatus {
	if bookmarks == nil {
		bookmarks = []string{}
	}

	return CredentialsStatus{
		LoggedIn:  credentials.Username != "" && credentials.Userkey != "",
		Username:  credentials.Username,
		Bookmarks: bookmarks,
		Error:     statusError,
	}
}

func ModPortalLoginHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	err, statusCode := factorio.FactorioLogin(data.Username, data.Password)
	if err != nil {
		resp = fmt.Sprintf("Error trying to login into Factorio: %s", err)
		log.Println(resp)
		w.WriteHeader(statusCode)
		return
	}

	// FactorioLogin stores the account. Report the same shape the status
	// endpoint does, so the page can render the result without a second call.
	credentials := factorio.Credentials{Username: data.Username, Userkey: ""}
	if _, err := credentials.Load(); err != nil {
		log.Printf("Error loading the stored credentials after login: %s", err)
	}

	bookmarks, bookmarksErr, _ := credentials.Bookmarks()
	if bookmarksErr != nil {
		resp = credentialsStatus(credentials, nil, bookmarksErr.Error())
		return
	}

	resp = credentialsStatus(credentials, bookmarks, "")
}

// ModPortalTokenLoginHandler stores a profile token directly, without trading a
// password for one. The token is the one shown on factorio.com/profile and is
// the same value the mod download uses as its `token` query parameter, so a
// token accepted here is immediately usable for gated downloads.
func ModPortalTokenLoginHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	if data.Username == "" || data.Token == "" {
		resp = "a username and a token are both required"
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	credentials := factorio.Credentials{Username: data.Username, Userkey: data.Token}

	// Verify before storing, so an invalid token is never written to disk.
	bookmarks, err, statusCode := credentials.Bookmarks()
	if err != nil {
		resp = fmt.Sprintf("The mod portal rejected the token: %s", err)
		log.Println(resp)
		w.WriteHeader(statusCode)
		return
	}

	if err := credentials.Save(); err != nil {
		resp = fmt.Sprintf("Error saving the Factorio credentials: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("Stored Factorio mod portal token for user %q", data.Username)
	resp = credentialsStatus(credentials, bookmarks, "")
}

func ModPortalLoginStatusHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var credentials factorio.Credentials
	loggedIn, err := credentials.Load()
	if err != nil {
		resp = credentialsStatus(credentials, nil, err.Error())
		return
	}

	if !loggedIn {
		resp = credentialsStatus(credentials, nil, "")
		return
	}

	bookmarks, err, _ := credentials.Bookmarks()
	if err != nil {
		resp = credentialsStatus(credentials, nil, err.Error())
		return
	}

	resp = credentialsStatus(credentials, bookmarks, "")
}

func ModPortalLogoutHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	var credentials factorio.Credentials
	err = credentials.Del()

	if err != nil {
		resp = fmt.Sprintf("Error on logging out of factorio: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = false
}

func ModPortalInstallMultipleHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data []struct {
		Name    string           `json:"name"`
		Version factorio.Version `json:"version"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	modList, resp, err := CreateNewMods(w)
	if err != nil {
		return
	}

	for _, datum := range data {
		// skip base mod because it is already included in factorio
		if datum.Name == "base" {
			continue
		}
		details, err, statusCode := factorio.ModPortalModDetails(datum.Name)
		if err != nil || statusCode != http.StatusOK {
			resp = fmt.Sprintf("Error in getting mod details from mod portal: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//find correct mod-version
		var found = false
		for _, release := range details.Releases {
			if release.Version.Equals(datum.Version) {
				found = true

				err := modList.DownloadMod(release.DownloadURL, release.FileName, details.Name)
				if err != nil {
					resp = fmt.Sprintf("Error downloading mod {%s}, error: %s", details.Name, err)
					log.Println(resp)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				break
			}
		}
		if !found {
			log.Printf("Error downloading mod {%s}, error: %s", details.Name, "version not found")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	resp = modList.ListInstalledMods()
}
