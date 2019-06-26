//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mainflux

const (
	// Text represents text payload format.
	Text = "text"

	// Binary represents binary payload format.
	Binary = "binary"
)

const (
	// SenMLJSON represents SenML in JSON format content type.
	SenMLJSON = "application/senml+json"

	// SenMLCBOR represents SenML in CBOR format content type.
	SenMLCBOR = "application/senml+cbor"
)

// Types contains mapping between Mainflux supported content types
// and formats.
var Types = map[string]string{
	SenMLJSON: Text,
	SenMLCBOR: Binary,
}
