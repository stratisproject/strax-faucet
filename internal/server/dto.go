package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/chainflag/eth-faucet/internal/chain"
)

type claimRequest struct {
	Address string `json:"address"`
}

type loginRequest struct {
	Code string `json:"code"`
}

type claimResponse struct {
	Message string `json:"msg"`
}

type loginResponse struct {
	Message string `json:"msg"`
}

type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type isPendingResponse struct {
	Pending bool `json:"pending"`
}

type infoResponse struct {
	Account         string `json:"account"`
	Network         string `json:"network"`
	Payout          string `json:"payout"`
	Symbol          string `json:"symbol"`
	HcaptchaSiteKey string `json:"hcaptcha_sitekey,omitempty"`
	RemoteAddr      string `json:"remote_addr,omitempty"`
	Forward         string `json:"forward,omitempty"`
	RealIP          string `json:"real_ip,omitempty"`
	DiscorClientId  string `json:"discord_client_id"`
}

type authResponse struct {
	Token string `json:"token"`
}

type malformedRequest struct {
	status  int
	message string
}

func (mr *malformedRequest) Error() string {
	return mr.message
}

func decodeJSONBody(r *http.Request, dst interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	defer r.Body.Close()
	if err != nil {
		return &malformedRequest{status: http.StatusBadRequest, message: "Unable to read request body"}
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			return &malformedRequest{status: http.StatusBadRequest, message: msg}
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, message: msg}
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, message: msg}
		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, message: msg}
		default:
			return err
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

func readAddress(r *http.Request) (string, error) {
	var claimReq claimRequest
	if err := decodeJSONBody(r, &claimReq); err != nil {
		return "", err
	}
	if !chain.IsValidAddress(claimReq.Address, true) {
		return "", &malformedRequest{status: http.StatusBadRequest, message: "invalid address"}
	}

	return claimReq.Address, nil
}

func readCode(r *http.Request) (string, error) {
	var loginReq loginRequest
	if err := decodeJSONBody(r, &loginReq); err != nil {
		return "", err
	}

	return loginReq.Code, nil
}

func readToken(r *http.Request) (string, error) {
	var discordRes discordTokenResponse
	if err := decodeJSONBody(r, &discordRes); err != nil {
		return "", err
	}

	return discordRes.AccessToken, nil
}

func renderJSON(w http.ResponseWriter, v interface{}, code int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}
