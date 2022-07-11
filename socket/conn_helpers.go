package socket

import (
	"encoding/binary"
	"errors"
	"strings"
)

type packaged_op []byte

const DEFAULT_SPLIT_CHAR = ":"

var DEFAULT_DELIM byte = DEFAULT_SPLIT_CHAR[0]

// Converts binary data to string parameters and payload
func byte_to_params_and_payload(b []byte, n_param byte, payload_exists bool) (param []string, payload []byte, err error) {

	param = make([]string, n_param)
	err = nil

	if n_param == 0 {
		if payload_exists {
			payload = b
		} else if len(b) > 1 || (len(b) == 1 && b[0] != DEFAULT_DELIM) {
			err = errors.New("unexpected payload")
		}
		return
	}

	var n byte = 0
	last_delim_i := 0
	for i := 0; i < len(b); i++ {
		if n == n_param-1 && !payload_exists {
			param[n] = string(b[last_delim_i:])
			return
		}
		if b[i] == DEFAULT_DELIM {
			param[n] = string(b[last_delim_i:i])
			last_delim_i = i + 1
			n++
			if n == n_param {
				if payload_exists {
					payload = b[last_delim_i:]
				} else {
					if i == len(b)-1 {
						return
					}
					err = errors.New("expected less parameters")
				}
				return
			}
		}
	}

	err = errors.New("expected more parameters")
	return
}

func package_op(op byte, params []string, payload []byte) packaged_op {

	b_param := []byte(strings.Join(params, DEFAULT_SPLIT_CHAR))

	if params != nil && payload != nil {
		b_param = append(b_param, DEFAULT_DELIM)
	}

	size := len(b_param) + len(payload)

	packaged := make([]byte, 5+size)
	packaged[0] = op
	binary.BigEndian.PutUint32(packaged[1:5], uint32(size))
	copy(packaged[5:5+len(b_param)], b_param)
	copy(packaged[5+len(b_param):], payload)

	return packaged
}
