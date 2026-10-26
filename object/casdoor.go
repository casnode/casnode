// Copyright 2026 The Casnode Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/casnode/casnode/conf"
)

type CasdoorConfig struct {
	ServerUrl        string `json:"serverUrl"`
	ClientId         string `json:"clientId"`
	OrganizationName string `json:"organizationName"`
	AppName          string `json:"appName"`
}

// CasdoorAccount is who Casdoor says signed in.
type CasdoorAccount struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
	Email       string `json:"email"`
	IsAdmin     bool   `json:"isAdmin"`
}

var casdoorHttpClient = &http.Client{Timeout: 15 * time.Second}

func GetCasdoorConfig() *CasdoorConfig {
	return &CasdoorConfig{
		ServerUrl:        strings.TrimSuffix(conf.GetConfigString("casdoorEndpoint"), "/"),
		ClientId:         conf.GetConfigString("clientId"),
		OrganizationName: conf.GetConfigString("casdoorOrganization"),
		AppName:          conf.GetConfigString("casdoorApplication"),
	}
}

// GetAccountByOAuthCode exchanges the code from /callback for an access token, then asks Casdoor who it belongs to.
// The answer comes from Casdoor itself, so it holds even when the application's signing key is a public one.
func GetAccountByOAuthCode(code string) (*CasdoorAccount, error) {
	config := GetCasdoorConfig()
	if config.ServerUrl == "" {
		return nil, fmt.Errorf("casdoorEndpoint is not set")
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {config.ClientId},
		"client_secret": {conf.GetConfigString("clientSecret")},
		"code":          {code},
	}
	resp, err := casdoorHttpClient.PostForm(config.ServerUrl+"/api/login/oauth/access_token", form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, fmt.Errorf("casdoor returned %d: %s", resp.StatusCode, string(body))
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("casdoor: %s %s", token.Error, token.ErrorDescription)
	}

	req, err := http.NewRequest(http.MethodGet, config.ServerUrl+"/api/get-account", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err = casdoorHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res struct {
		Status string          `json:"status"`
		Msg    string          `json:"msg"`
		Data   *CasdoorAccount `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("casdoor returned %d: %s", resp.StatusCode, string(body))
	}
	if res.Status != "ok" || res.Data == nil || res.Data.Name == "" {
		return nil, fmt.Errorf("casdoor: %s", res.Msg)
	}
	if config.OrganizationName != "" && res.Data.Owner != config.OrganizationName {
		return nil, fmt.Errorf("user %s/%s does not belong to organization %s", res.Data.Owner, res.Data.Name, config.OrganizationName)
	}
	if res.Data.DisplayName == "" {
		res.Data.DisplayName = res.Data.Name
	}
	return res.Data, nil
}
