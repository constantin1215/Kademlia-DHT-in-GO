package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
)

func calculateNodeId() string {
	data := getIP() + strconv.Itoa(*port)
	sha := sha256.New()
	sha.Write([]byte(data))
	encrypted := sha.Sum(nil)
	encryptedString := fmt.Sprintf("%x", encrypted)

	return encryptedString
}

func calculateDistance(hash1, hash2 string) (int64, error) {
	if len(hash1) != len(hash2) {
		return 0, errors.New("hashes do not match in size")
	}

	if len(hash1)%2 == 1 || len(hash2)%2 == 1 {
		return 0, errors.New("hashes uneven")
	}

	result := int64(0)
	for i := 0; i < len(hash1); i += 2 {
		hex1val1, err := getHexStrDecValue(hash1[i])
		if err != nil {
			return -1, err
		}
		hex1val2, err := getHexStrDecValue(hash1[i+1])
		if err != nil {
			return -1, err
		}
		hex2val1, err := getHexStrDecValue(hash2[i])
		if err != nil {
			return -1, err
		}
		hex2val2, err := getHexStrDecValue(hash2[i+1])
		if err != nil {
			return -1, err
		}
		hex1 := int64(hex1val1*16 + hex1val2)
		hex2 := int64(hex2val1*16 + hex2val2)
		xorResult := hex1 ^ hex2
		result = (result << 8) | xorResult
	}

	return result, nil
}

func getHexStrDecValue(character uint8) (uint8, error) {
	if character >= 48 && character <= 57 {
		return character - 48, nil
	} else if character >= 97 && character <= 102 {
		return character - 87, nil
	}
	return 0, errors.New("invalid character in hash")
}
