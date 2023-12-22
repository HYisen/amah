package auth

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"io"
	"os"
	"strings"
	"time"
)

type Account struct {
	Username          string
	EncryptedPassword string
}

type Token struct {
	ID       string
	ExpireAt time.Time
	Username string
}

func newAccount(shadowLine string) (Account, error) {
	parts := strings.Split(shadowLine, ":")
	if len(parts) < 2 {
		return Account{}, fmt.Errorf("bad shardow line %s", shadowLine)
	}
	return Account{
		Username:          parts[0],
		EncryptedPassword: parts[1],
	}, nil
}

type Client struct {
	usernameToEncryptedPassword map[string]string
	tokenIDToToken              map[string]Token
}

func LoadAccounts(shadowFilePath string) ([]Account, error) {
	data, err := os.ReadFile(shadowFilePath)
	if err != nil {
		return nil, fmt.Errorf("read shadow file %s: %v", shadowFilePath, err)
	}
	ret, err := ParseShadow(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse shadow file %s: %v", shadowFilePath, err)
	}
	return ret, nil
}

func NewClient(accounts []Account) (*Client, error) {
	usernameToEncryptedPassword := make(map[string]string)
	for _, account := range accounts {
		usernameToEncryptedPassword[account.Username] = account.EncryptedPassword
	}
	return &Client{
		usernameToEncryptedPassword: usernameToEncryptedPassword,
		tokenIDToToken:              make(map[string]Token),
	}, nil
}

func ParseShadow(reader io.Reader) ([]Account, error) {
	var ret []Account
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		one, err := newAccount(scanner.Text())
		if err != nil {
			return nil, err
		}
		ret = append(ret, one)
	}
	return ret, nil
}

var dummyEncryptedPasswordData, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

func (c *Client) Auth(username, password string) (ok bool, err error) {
	encryptedPassword, ok := c.usernameToEncryptedPassword[username]
	encryptedPasswordData := []byte(encryptedPassword)
	if !ok {
		// If 404, crypto/bcrypt: hashedSecret too short to be a bcrypted password.
		// So use a dummy one to pass time-attack(check username exists through auth time cost).
		encryptedPasswordData = dummyEncryptedPasswordData
	}
	e := bcrypt.CompareHashAndPassword(encryptedPasswordData, []byte(password))
	if !ok {
		return false, nil
	}
	if e != nil && !errors.Is(e, bcrypt.ErrMismatchedHashAndPassword) {
		return false, e
	}
	return e == nil, nil
}

func expireAt(now time.Time) time.Time {
	return now.Add(10 * time.Minute)
}

func (c *Client) CreateToken(Username string) Token {
	ret := Token{
		ID:       uuid.NewString(),
		ExpireAt: expireAt(time.Now()),
		Username: Username,
	}
	c.tokenIDToToken[ret.ID] = ret
	return ret
}

func (c *Client) FindValidToken(id string) (t Token, ok bool) {
	token, ok := c.tokenIDToToken[id]
	if !ok {
		return Token{}, false
	}
	if token.ExpireAt.Before(time.Now()) {
		delete(c.tokenIDToToken, id)
		return Token{}, false
	}
	return token, true
}

func Register(username, password string) (shadowLine string, err error) {
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", username, encryptedPassword), nil
}
