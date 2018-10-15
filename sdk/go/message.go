//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (sdk *mfSDK) SendMessage(chanID, msg, token string) error {
	endpoint := fmt.Sprintf("channels/%s/messages", chanID)
	url := createURL(sdk.url, sdk.httpAdapterPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}

func (sdk *mfSDK) SetContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return errors.New("Unknown Content Type")
	}

	sdk.msgContentType = ct

	return nil
}
