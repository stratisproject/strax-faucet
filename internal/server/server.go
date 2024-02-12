package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/chainflag/eth-faucet/internal/chain"
	"github.com/chainflag/eth-faucet/web"
	"gopkg.in/fsnotify.v1"
)

type Server struct {
	chain.TxBuilder
	cfg *Config
}

func NewServer(builder chain.TxBuilder, cfg *Config) *Server {
	return &Server{
		TxBuilder: builder,
		cfg:       cfg,
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/", http.FileServer(web.Dist()))
	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
	hcaptcha := NewCaptcha(s.cfg.hcaptchaSiteKey, s.cfg.hcaptchaSecret)
	auth := NewAuth("")
	router.Handle("/api/login", s.handleLogin())
	router.Handle("/api/check", negroni.New(auth, negroni.Wrap(s.handleLoginCheck())))
	router.Handle("/api/claim", negroni.New(limiter, hcaptcha, auth, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/info", s.handleInfo())

	return router
}

func (s *Server) Run() {
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())

	// creates a new file watcher for App_offline.htm
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("ERROR", err)
	}
	defer watcher.Close()

	// watch for App_offline.htm and exit the program if present
	// This allows continuous deployment on App Service as the .exe will not be
	// terminated otherwise
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if strings.HasSuffix(event.Name, "app_offline.htm") {
					fmt.Println("Exiting due to app_offline.htm being present")
					os.Exit(0)
				}
			}
		}
	}()

	log.Infof("Starting http server %d", s.cfg.httpPort)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(s.cfg.httpPort), n))
}

func (s *Server) handleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		// The error always be nil since it has already been handled in limiter
		address, _ := readAddress(r)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		txHash, err := s.Transfer(ctx, address, chain.EtherToWei(int64(s.cfg.payout)))
		if err != nil {
			log.WithError(err).Error("Failed to send transaction")
			renderJSON(w, claimResponse{Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": address,
		}).Info("Transaction sent successfully")
		resp := claimResponse{Message: fmt.Sprintf("Txhash: %s", txHash)}
		renderJSON(w, resp, http.StatusOK)
	}
}

func (s *Server) handleInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}
		renderJSON(w, infoResponse{
			Account:         s.Sender().String(),
			Network:         s.cfg.network,
			Symbol:          s.cfg.symbol,
			Payout:          strconv.Itoa(s.cfg.payout),
			HcaptchaSiteKey: s.cfg.hcaptchaSiteKey,
			RemoteAddr:      r.RemoteAddr,
			Forward:         r.Header.Get("X-Forwarded-For"),
			RealIP:          r.Header.Get("X-Real-IP"),
		}, http.StatusOK)
	}
}

func (s *Server) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		// The error always be nil since it has already been handled in limiter
		code, _ := readCode(r)

		token, err := exchangeCodeForToken(code, s.cfg.discordClientId, s.cfg.discordClientSecret, s.cfg.discordRedirectUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		expirationTime := time.Now().Add(24 * time.Hour) // Expires in 24 hours
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token.AccessToken, // Replace with the actual token value
			Expires:  expirationTime,
			HttpOnly: true, // This makes the cookie inaccessible to JavaScript
		})

		// You can send back a simple response
		//w.Write([]byte("User logged in"))

		renderJSON(w, authResponse{
			Token: token.AccessToken,
		}, http.StatusOK)
	}
}

func (s *Server) handleLoginCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			// Handle the case where the cookie is not present or invalid
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Additional checks can be performed here, such as validating the token
		log.Info(cookie.Value)

		// If everything is okay
		w.WriteHeader(http.StatusOK)
	}
}

func exchangeCodeForToken(code, discordClientId, discordClientSecret, discordRedirectUrl string) (*discordTokenResponse, error) {
	log.Info(discordClientId, discordClientSecret, discordRedirectUrl, code)
	// Prepare the request data
	data := url.Values{}
	data.Set("client_id", discordClientId)
	data.Set("client_secret", discordClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", discordRedirectUrl)
	data.Set("scope", "identify")

	// Make the request
	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Info(resp.Body)

	// Decode the response
	var tokenResp discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Error(err)
		return nil, err
	}

	log.WithFields(log.Fields{
		"token":             tokenResp.AccessToken,
		"erorr":             tokenResp.Error,
		"error_description": tokenResp.ErrorDesc,
	}).Info("Response")

	return &tokenResp, nil
}
