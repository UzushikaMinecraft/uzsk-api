package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/uzushikaminecraft/api/config"
	"github.com/uzushikaminecraft/api/structs"
)

func Callback(state string, code string) (*structs.JWTCallback, error) {
	// read `state` parameter to validate OAuth request
	if state != config.Conf.Credentials.State {
		return nil, errors.New(
			fmt.Sprintf("state string does not match: %v", state),
		)
	}

	// read `code` parameter to get a token
	if code == "" {
		return nil, errors.New("required parameter code is not provided")
	}

	// OAuth exchange phase
	cxt := context.Background()

	token, err := oauthConf.Exchange(
		cxt, code,
	)
	if err != nil {
		return nil, errors.New("failed to exchange token")
	}
	if token == nil {
		return nil, errors.New("failed to contact with Discord")
	}

	// retrieve user information from Discord
	url := "https://discordapp.com/api/users/@me"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.New("error occured while making request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token.AccessToken))

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("error occured while executing request")
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil,
			errors.New(
				fmt.Sprintf(
					"Discord returned status code %v: %v",
					resp.StatusCode, string(b),
				),
			)
	}

	var user structs.DiscordUser
	if err := json.Unmarshal(b, &user); err != nil {
		return nil,
			errors.New(
				fmt.Sprintf(
					"failed to parse Discord's JSON: %v",
					err,
				),
			)
	}

	// generate JWT token
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtAccessToken, err := jwtToken.SignedString([]byte(config.Conf.Credentials.JWTSecret))
	if err != nil {
		return nil,
			errors.New(
				fmt.Sprintf(
					"error occured while generating JWT token: %v",
					err,
				),
			)
	}

	var jwtCallback = &structs.JWTCallback{
		Claims:      claims,
		AccessToken: jwtAccessToken,
	}

	return jwtCallback, nil
}
