package data

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"

	"step/internal/biz"
	entStep "step/internal/data/ent/step"
	"step/internal/utils"

	"github.com/go-kratos/kratos/v2/log"
)

// PKCS7 padding implementation
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

// PKCS7 unpadding implementation
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("invalid padding size")
	}

	padding := int(data[length-1])
	if padding > blockSize || padding == 0 {
		return nil, errors.New("invalid padding size")
	}

	for i := length - padding; i < length; i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:length-padding], nil
}

type encryptRepo struct {
	data *Data
	log  *log.Helper
}

// NewEncryptRepo .
func NewEncryptRepo(data *Data, logger log.Logger) biz.EncryptRepo {
	return &encryptRepo{
		data: data,
		log:  log.NewHelper(logger, log.WithMessageKey("encryptRepo")),
	}
}

func (r *encryptRepo) Encrypt(ctx context.Context, stepId uint64, data string) (string, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return "", errors.New("uid is empty")
	}

	s, err := r.data.ent_client.Step.Query().Where(entStep.ID(stepId)).WithTarget().First(ctx)
	if err != nil {
		return "", err
	}

	if s.Edges.Target.UserID != uid {
		return "", errors.New("step not found")
	}

	// 使用aes加密
	secretKey := r.data.secret
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}

	plaintext := []byte(data)
	plaintext = pkcs7Pad(plaintext, block.BlockSize())

	// 使用stepId作为IV的基础
	iv := make([]byte, block.BlockSize())
	for i := 0; i < 8 && i < block.BlockSize(); i++ {
		iv[i] = byte(stepId >> uint(i*8))
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(plaintext, plaintext)

	encrypted := base64.StdEncoding.EncodeToString(plaintext)

	return encrypted, nil
}

func (r *encryptRepo) Decrypt(ctx context.Context, stepId uint64, data string) (string, error) {
	secretKey := r.data.secret
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}

	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	// 使用相同的stepId重建IV
	iv := make([]byte, block.BlockSize())
	for i := 0; i < 8 && i < block.BlockSize(); i++ {
		iv[i] = byte(stepId >> uint(i*8))
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(encrypted, encrypted)

	plaintext, err := pkcs7Unpad(encrypted, block.BlockSize())
	if err != nil {
		return "", err
	}

	decrypted := string(plaintext)

	return decrypted, nil
}
