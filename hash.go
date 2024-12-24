package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

func calculateNodeId() string {
	data := ip + strconv.Itoa(port)
	sha := sha256.New()
	sha.Write([]byte(data))
	encrypted := sha.Sum(nil)
	encryptedString := fmt.Sprintf("%x", encrypted)

	return encryptedString
}

func calculateDistance(hash1, hash2 string) (uint8, error) {
	if len(hash1) != len(hash2) {
		return 0, errors.New("hashes do not match in size")
	}

	if len(hash1)%2 == 1 || len(hash2)%2 == 1 {
		return 0, errors.New("hashes uneven")
	}

	result := big.NewInt(0)
	for i := 0; i < len(hash1); i += 2 {
		hex1val1, err := getHexStrDecValue(hash1[i])
		if err != nil {
			return 0, err
		}
		hex1val2, err := getHexStrDecValue(hash1[i+1])
		if err != nil {
			return 0, err
		}
		hex2val1, err := getHexStrDecValue(hash2[i])
		if err != nil {
			return 0, err
		}
		hex2val2, err := getHexStrDecValue(hash2[i+1])
		if err != nil {
			return 0, err
		}
		hex1 := int64(hex1val1*16 + hex1val2)
		hex2 := int64(hex2val1*16 + hex2val2)
		xorResult := hex1 ^ hex2
		result = result.Lsh(result, 8).Or(result, big.NewInt(xorResult))
	}

	return uint8(result.BitLen() - 1), nil
}

func getHexStrDecValue(character uint8) (uint8, error) {
	if character >= 48 && character <= 57 {
		return character - 48, nil
	} else if character >= 97 && character <= 102 {
		return character - 87, nil
	}
	return 0, errors.New("invalid character in hash")
}
