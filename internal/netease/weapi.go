package netease

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
)

// The keys of the web player's weapi, which every web client of NetEase
// Cloud Music sends its requests with.
const (
	presetKey = "0CoJUm6Qyw8W8jud"
	weapiIV   = "0102030405060708"
	pubModHex = "00e0b509f6259df8642dbc35662901477df22677ec152b5ff68ace615bb7b725152b3ab17a876aea8a5aa76d2e417629ec4ee341f56135fccf695280104e0312ecbda92557c93870114af6c9d05c4f7f0c3685b7a46bee255932575cce10b424d813cfe4875d3e82047b97ddef52741d546b8e289dc6935b3ece0462db0a22b8e7"
	base62    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var pubMod, _ = new(big.Int).SetString(pubModHex, 16)

func aesCBC(text, key []byte) []byte {
	block, _ := aes.NewCipher(key) // keys are 16 bytes
	pad := aes.BlockSize - len(text)%aes.BlockSize
	text = append(slices.Clone(text), bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(text))
	cipher.NewCBCEncrypter(block, []byte(weapiIV)).CryptBlocks(out, text)
	return []byte(base64.StdEncoding.EncodeToString(out))
}

// weapi seals a request's JSON as the web player does: the form of params
// and encSecKey it posts.
func weapi(body []byte) (url.Values, error) {
	secKey := make([]byte, 16)
	if _, err := rand.Read(secKey); err != nil {
		return nil, err
	}
	for i, b := range secKey {
		secKey[i] = base62[int(b)%len(base62)]
	}
	params := aesCBC(aesCBC(body, []byte(presetKey)), secKey)
	reversed := slices.Clone(secKey)
	slices.Reverse(reversed)
	// Textbook RSA, no padding, as the web player's own.
	enc := new(big.Int).Exp(new(big.Int).SetBytes(reversed), big.NewInt(65537), pubMod)
	return url.Values{"params": {string(params)}, "encSecKey": {fmt.Sprintf("%0256s", hex.EncodeToString(enc.Bytes()))}}, nil
}

// eapiKey is the key of the desktop app's eapi.
const eapiKey = "e82ckenh8dichen8"

// eapi seals a request's JSON for the eapi path as the desktop app does:
// with its digest, in AES-ECB, as the form's params.
func eapi(path string, body []byte) url.Values {
	text := string(body)
	sum := md5.Sum([]byte("nobody" + path + "use" + text + "md5forencrypt"))
	data := []byte(path + "-36cd479b6b5-" + text + "-36cd479b6b5-" + hex.EncodeToString(sum[:]))
	pad := aes.BlockSize - len(data)%aes.BlockSize
	data = append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
	block, _ := aes.NewCipher([]byte(eapiKey))
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Encrypt(out[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
	return url.Values{"params": {strings.ToUpper(hex.EncodeToString(out))}}
}
