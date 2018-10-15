//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (sdk mfSDK) CreateUser(user User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	url := createURL(sdk.url, sdk.usersPrefix, "users")

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}

func (sdk mfSDK) CreateToken(user User) (string, error) {
	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	url := createURL(sdk.url, sdk.usersPrefix, "tokens")

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%d", resp.StatusCode)
	}

	var t tokenRes
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}

	return t.Token, nil
}
