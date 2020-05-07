package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

func createTempFile() (string, error) {
	tmpFile, err := ioutil.TempFile("/tmp", "tmp")
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	return tmpFile.Name(), nil
}

func TestGeneratePrivKey(test *testing.T) {
	testCases := []struct {
		name      string
		algo      string
		bits      int
		shouldErr bool
	}{
		// Negative test cases
		{"RSA-small-bits", "rsa", 1024, true},
		{"Wrong-algo", "rsaa", 2048, true},

		// Positive test cases
		{"RSA-basic", "rsa", 2048, false},
		{"ECDSA-basic", "ecdsa", 2048, false},
		{"Ed25519-basic", "Ed25519", 0, false},
		{"Secp256k1-basic", "Secp256k1", 0, false},
	}

	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			_, err := generatePrivKey(testCase.algo, testCase.bits)
			if testCase.shouldErr {
				if err == nil {
					test.Errorf("Passed case (%s); Expected it to fail.", testCase.name)
				} else {
					test.Log(err)
				}
			} else if !testCase.shouldErr && err != nil {
				test.Log(err)
				test.Errorf("Failed case (%s); Expected it to pass.\n%v", testCase.name, err)
			}
		})
	}
}

func TestStoreKey(test *testing.T) {
	// Setup for case of existing key file
	existingFile, err := createTempFile()
	if err != nil {
		panic(err)
	}

	testCases := []struct {
		name      string
		algo      int
		bits      int
		shouldErr bool
	}{
		// Negative test case
		{"ExistingFile", crypto.RSA, 2048, true},

		// Positive test cases
		{"RSA", crypto.RSA, 2048, false},
		{"Ed25519", crypto.Ed25519, 0, false},
		{"Secp256k1", crypto.Secp256k1, 0, false},
		{"ECDSA", crypto.ECDSA, 0, false},
		//{"TildeExpansion", "RSA", 2048, false},
	}

	var tmpFile string
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			// Create a dummy private key to be used for tests
			// Assume libp2p's crypto was properly tested by their devs
			priv, _, err := crypto.GenerateKeyPair(testCase.algo, testCase.bits)
			if err != nil {
				test.Fatalf("Unable to generate test key: libp2p's crypto pkg returned an error")
			}

			// This is a shitty hack... breaks generalization
			if testCase.name == "ExistingFile" {
				tmpFile = existingFile
			} else {
				tmpFile = "/tmp/tmp" + string(rand.Int())
			}

			err = storePrivKeyToFile(priv, tmpFile)
			if testCase.shouldErr {
				if err == nil {
					test.Errorf("Passed case (%s); Expected it to fail.", testCase.name)
				} else {
					test.Log(err)
				}
			} else if !testCase.shouldErr && err != nil {
				test.Log(err)
				test.Errorf("Failed case (%s); Expected it to pass.\n%v", testCase.name, err)
			}

			if !testCase.shouldErr {
				// Check that the key exists
				_, err = os.Stat(tmpFile)
				if os.IsNotExist(err) {
					test.Errorf("Expected key file (%s) does not exist.", tmpFile)
				}
			}

			os.Remove(tmpFile)
		})
	}
}

func TestLoadKey(test *testing.T) {
	// Create an existing key to load from
	keyType := pb.KeyType(3)
	keyB64 := "MHcCAQEEIHp/bhcT3Jge9ykOMjk+AgCi6qqM8it01IRoRbXphHXaoAoGCCqGSM49AwEHoUQDQgAEhN7JYn9DN9POlfbkDwR1T74gxPpUx90cWxbuyuvOL10DsQe1UD/IVBxdQ1nZPaYC/m+nSaUdZ53gFBaHLQg+QQ=="

	tmpFile, err := ioutil.TempFile("/tmp", "tmp")
	if err != nil {
		panic(err)
	}
	tmpFile.WriteString(fmt.Sprintf("%d %s\n", keyType, keyB64))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	priv, err := loadPrivKeyFromFile(tmpFile.Name())
	if err != nil {
		test.Fatalf("loadPrivKeyFromFile() failed with error:\n%v", err)
	}

	if priv.Type() != keyType {
		test.Fatalf("Incorrect key type loaded (%d), was expecting %d", priv.Type(), keyType)
	}

	rawBytes, err := priv.Raw()
	if err != nil {
		test.Fatalf("Could not load raw bytes from loaded key")
	}

	loadedKeyB64 := crypto.ConfigEncodeKey(rawBytes)
	if loadedKeyB64 != keyB64 {
		test.Fatalf("Loaded key is not identical to test key.\n"+
			"Loaded: %s\nExpect: %s\n", loadedKeyB64, keyB64)
	}
}