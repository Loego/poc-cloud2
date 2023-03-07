package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"text/template"
	"time"
	"path/filepath"
	"net/url"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/script/v1"
	"google.golang.org/api/option"
)

// Scopes: OAuth 2.0 scopes provide a way to limit the amount of access that is granted to an access token.
var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "https://hello-6omskd6wtq-uc.a.run.app/auth/google/callback/",
	ClientID:     "1099511269269-r56cpc4d50c2dqru4skald614ebea3dk.apps.googleusercontent.com",
	ClientSecret: "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",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

const oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

var email="NoUser"
var token oauth2.Token
var documentId string

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("----------------oauthGoogleLogin()")

	// Create oauthState cookie
	oauthState := generateStateOauthCookie(w)
	log.Println("----------------create oauthState cookie")
	/*
		AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
		validate that it matches the the state query parameter on your redirect callback.
	*/
	u := googleOauthConfig.AuthCodeURL(oauthState)
	log.Println("----------------create AuthCodeURL")
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Println("----------------oauthGoogleCallback()")
	// Read oauthState from Cookie
	// oauthState, _ := r.Cookie("oauthstate")
	log.Println("----------------Read oauthState from Cookie")
	log.Println("----------------%s", r.FormValue("state"))

	// if r.FormValue("state") != oauthState.Value {
	// 	log.Println("----------------invalid oauth google state")
	// 	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	// 	return
	// }

	data, err := getUserDataFromGoogle(r.FormValue("code"))
	log.Println("----------------getUserDataFromGoogle()")
	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// This time, you can see following response on website
	/* UserInfo: {
	   		"id": "******************",
				"email": "****************@gmail.com",
				"verified_email": true,
				"picture": "https://lh3.googleusercontent.com/a-/*********************************"
			}
	*/
	log.Printf("UserInfo: %s\n", data)
	
	var HomePageVars PageVariables
	err = json.Unmarshal(data, &HomePageVars)
	
	if err != nil {
		fmt.Println("Can;t unmarshal the byte array")
		return
	}
	email = HomePageVars.Email
	tpl.Execute(w, HomePageVars)
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)

}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(20 * time.Minute)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)

	return state
}

func getUserDataFromGoogle(code string) ([]byte, error) {
	// Use code to get token and get user info from Google.

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}
	response, err := http.Get(oauthGoogleUrlAPI + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %s", err.Error())
	}
	return contents, nil
}

type PageVariables struct {
	Email string
	Date  string
	Time  string
}

type TxData struct {
	Title   string
	Content string
	DocumentId string
}

type TokenData struct {
	AccessToken   string
	ExpiresIn 		int
	Scope 				string
	TokenType 		string
}

var tpl = template.Must(template.ParseFiles("index.html"))

func signinHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("signin handler")

	log.Printf("get fetch: %v", r.Body)
	var tokenData TokenData
	err := json.NewDecoder(r.Body).Decode(&tokenData)

	if err != nil {
		panic(err)
	}
	token.AccessToken = tokenData.AccessToken
	token.TokenType = tokenData.TokenType

	log.Printf("get fetch: %+v", tokenData)
	log.Printf("AccessToken: %s", token.AccessToken)
	log.Printf("TokenType: %s", token.TokenType)
	log.Printf("RefreshToken: %s", token.RefreshToken)

	// TODO: get userinfo.email and return
	w.Write([]byte("<h1>Hello World!</h1>"))
}

func signoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("signin handler")

	log.Printf("get fetch: %v", r.Body)
	err := json.NewDecoder(r.Body).Decode(&token)

	if err != nil {
		panic(err)
	}
	log.Printf("AccessToken: %s", token.AccessToken)
	log.Printf("TokenType: %s", token.TokenType)
	log.Printf("RefreshToken: %s", token.RefreshToken)

	// TODO: get userinfo.email and return
	// w.Write([]byte("<h1>Hello World!</h1>"))
	tpl.Execute(w, nil)
}

func docHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("doc handler")
	now := time.Now()              // find the time right now
	HomePageVars := PageVariables{ //store the date and time in a struct
		Email: "NoUser",
		Date:  now.Format("02-01-2006"),
		Time:  now.Format("15:04:05"),
	}
	log.Printf("get fetch: %+v", r.Body)
	var t TxData
	err := json.NewDecoder(r.Body).Decode(&t)

	if err != nil {
		panic(err)
	}
	log.Printf("Title: %s", t.Title)
	log.Printf("Content: %s", t.Content)
	log.Printf("DocumentId: %s", t.DocumentId)
	documentId = t.DocumentId

	// w.Write([]byte("<h1>Hello World!</h1>"))
	tpl.Execute(w, HomePageVars)
}

func pdfHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("pdf handler")
	// now := time.Now() // find the time right now
	// HomePageVars := PageVariables{ //store the date and time in a struct
	// 	Date: now.Format("02-01-2006"),
	// 	Time: now.Format("15:04:05"),
	// }
	// ctx := context.Background()

	// // process the credential file
	// credential, err := ioutil.ReadFile("credentials.json")
	// if err != nil {
	// 	log.Fatalf("Unable to read client secret file: %v", err)
	// 	w.Write([]byte("{'res':'Unable to read client secret file'}"))
	// }

	// In order for POST upload attachment to work
	// You need to authorize the Gmail API v1 scope
	// at https://developers.google.com/oauthplayground/
	// otherwise you will get Authorization error in the API JSON reply

	// Use DriveScope for this example. Because of we want to Manage the files in
	// Google Drive.

	// See the rest at https://godoc.org/google.golang.org/api/drive/v3#pkg-constants

	// config, err := google.ConfigFromJSON(credential, drive.DriveScope)
	// if err != nil {
	// 	// log.Fatalf("Unable to parse client secret file to config: %v", err)
	// 	w.Write([]byte("{'res':'Unable to initiate new Drive client'}"))
	// }

	// client := config.Client(ctx, &token)

	// // initiate a new Google Drive service
	// driveClientService, err := drive.New(client)
	// if err != nil {
	// 	// log.Fatalf("Unable to initiate new Drive client: %v", err)
	// 	w.Write([]byte("{'res':'Unable to initiate new Drive client'}"))
	// }

	// mimeType := "application/pdf"
	// filename := "sample.pdf"

	// res, err := driveClientService.Files.Export(documentId, mimeType).Download()
	// if err != nil {
	// 	// log.Fatalf("Error: %v", err)
	// 	w.Write([]byte("{'res':'Error: res, err := driveClientService.Files.Export(documentId, mimeType).Download()'}"))
	// }
	// fmt.Printf("File.Export Result: %v", res)
	// file, err := os.Create(filename)
	// if err != nil {
	// 	// log.Fatalf("Error: %v", err)
	// 	w.Write([]byte("{'res':'Error: file, err := os.Create(filename)'}"))
	// }
	// defer file.Close()
	// _, err = io.Copy(file, res.Body)

	w.Write([]byte(email))
	// tpl.Execute(w, HomePageVars)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()              // find the time right now
	HomePageVars := PageVariables{ //store the date and time in a struct
		Email: "NoUser",
		Date:  now.Format("02-01-2006"),
		Time:  now.Format("15:04:05"),
	}
	// w.Write([]byte("<h1>Hello World!</h1>"))

	tpl.Execute(w, HomePageVars)
}


// NOTE : we don't want to visit CSRF URL to get the authorization code
// and paste into the terminal each time we want to send an email
// therefore we will retrieve a token for our client, save the token into a file
// you will be prompted to visit a link in your browser for authorization code only ONCE
// and subsequent execution of the program will not prompt you for authorization code again
// until the token expires.

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
					log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
					tok = getTokenFromWeb(config)
					saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
					"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
					log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
					log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
					return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
					url.QueryEscape("google-drive-golang.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
					return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
					log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}


// savePdf uses a file path to create a file and store the
// PDF.
func savePdf() {
	// usr, err := user.Current()
	// if err != nil {
	// 	return
	// }
	// tokenCacheDir := filepath.Join(usr.HomeDir, ".sample")
	// os.MkdirAll(tokenCacheDir, 0700)
	// filepathname := filepath.Join(tokenCacheDir,
	// 	url.QueryEscape("google-drive-golang.pdf"))
	// fmt.Printf("Saving credential file to: %s\n", filepathname)
	// f, err := os.Create(filepathname)
	// if err != nil {
	// 	log.Fatalf("Unable to cache oauth token: %v", err)
	// }
	// defer f.Close()
	// json.NewEncoder(f).Encode(token)
	// now := time.Now() // find the time right now
	// HomePageVars := PageVariables{ //store the date and time in a struct
	// 	Date: now.Format("02-01-2006"),
	// 	Time: now.Format("15:04:05"),
	// }
	ctx := context.Background()

	// process the credential file
	credential, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		// w.Write([]byte("{'res':'Unable to read client secret file'}"))
	}

	// In order for POST upload attachment to work
	// You need to authorize the Gmail API v1 scope
	// at https://developers.google.com/oauthplayground/
	// otherwise you will get Authorization error in the API JSON reply

	// Use DriveScope for this example. Because of we want to Manage the files in
	// Google Drive.

	// See the rest at https://godoc.org/google.golang.org/api/drive/v3#pkg-constants

	config, err := google.ConfigFromJSON(credential, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		// w.Write([]byte("{'res':'Unable to initiate new Drive client'}"))
	}

	client := config.Client(ctx, &token)

	srv, err := script.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Unable to retrieve Script client: %v", err)
	}

	req := script.CreateProjectRequest{Title: "My Script"}
	createRes, err := srv.Projects.Create(&req).Do()
	if err != nil {
			// The API encountered a problem.
			log.Fatalf("The API returned an error: %v", err)
	}
	content := &script.Content{
			ScriptId: createRes.ScriptId,
			Files: []*script.File{{
							Name:   "hello",
							Type:   "SERVER_JS",
							Source: "function helloWorld() {\n  console.log('Hello, world!');}",
			}, {
							Name: "appsscript",
							Type: "JSON",
							Source: "{\"timeZone\":\"America/New_York\",\"exceptionLogging\":" +
											"\"CLOUD\"}",
			}},
	}
	updateContentRes, err := srv.Projects.UpdateContent(createRes.ScriptId,
			content).Do()
	if err != nil {
			// The API encountered a problem.
			log.Fatalf("The API returned an error: %v", err)
	}
	log.Printf("https://script.google.com/d/%v/edit", updateContentRes.ScriptId)
	// w.Write([]byte(email))
	// tpl.Execute(w, HomePageVars)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fs := http.FileServer(http.Dir("assets"))
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// Root
	mux.HandleFunc("/", indexHandler)

	// OauthGoogle
	mux.HandleFunc("/auth/google/login", oauthGoogleLogin)
	mux.HandleFunc("/auth/google/callback/", oauthGoogleCallback)
	mux.HandleFunc("/signin", signinHandler)
	mux.HandleFunc("/signout", signoutHandler)

	// FileGoogle
	mux.HandleFunc("/doc", docHandler)
	mux.HandleFunc("/pdf", pdfHandler)

	log.Printf("Starting HTTP Server. Listening at %q", port)
	if err := http.ListenAndServe(":"+port, mux); err != http.ErrServerClosed {
		log.Printf("%v", err)
	} else {
		log.Println("Server closed!")
	}
}
