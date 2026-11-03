package token

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"
	"math/rand"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rivo/tview"
)

// CheatCodeLabel labels the dev-only checkbox that mints a token (see DevSecret).
const CheatCodeLabel = "Apply Cheat Code"

// maxTokenLength caps the token input. Identity-provider tokens can be long.
const maxTokenLength = 4096

type TokenFields struct {
	Id    string
	Email string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func GenerateToken(ttl time.Duration, payload interface{}, secretJWTKey string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	now := time.Now().UTC()
	claims := token.Claims.(jwt.MapClaims)

	tokenFields := payload.(*TokenFields)

	claims["sub"] = tokenFields.Id
	claims["email"] = tokenFields.Email
	claims["exp"] = now.Add(ttl).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()

	tokenString, err := token.SignedString([]byte(secretJWTKey))

	if err != nil {
		return "", fmt.Errorf("generating JWT Token failed: %w", err)
	}

	return tokenString, nil
}

// GetTokenInputForm is the login page. On Save it checks the token (locally, then with
// GET /authping), stores it in appToken, and calls onLogin with the token's email. The
// cheat code checkbox appears only when DevSecret() is set. Errors show over the form.
func GetTokenInputForm(sh *ui.Shell, appToken *string, onLogin func(email string)) *tview.Form {
	tokenInput := tview.NewInputField().
		SetLabel("Token value: ").
		SetFieldWidth(80).
		SetAcceptanceFunc(tview.InputFieldMaxLength(maxTokenLength))

	showError := sh.Message

	form := tview.NewForm().AddFormItem(tokenInput)

	if secret := DevSecret(); secret != "" {
		cheatCode := tview.NewCheckbox().SetLabel(CheatCodeLabel).SetChecked(false)
		cheatCode.SetChangedFunc(func(checked bool) {
			if !checked {
				tokenInput.SetText("")
				return
			}
			email := strings.TrimSpace(tokenInput.GetText())
			if _, err := mail.ParseAddress(email); err != nil {
				cheatCode.SetChecked(false)
				showError("Invalid email address. Enter an email address to apply the cheat code.")
				return
			}
			jwtToken, err := GenerateToken(8*time.Hour, &TokenFields{Id: randomString(20), Email: email}, secret)
			if err != nil {
				cheatCode.SetChecked(false)
				showError(err.Error())
				return
			}
			tokenInput.SetText(jwtToken)
		})
		form.AddFormItem(cheatCode)
	}

	form.
		AddButton("Save", func() {
			tok := strings.TrimSpace(tokenInput.GetText())
			claims, err := Login(tok, api.New(tok), time.Now())
			if err != nil {
				showError("Login failed: " + ui.Describe(err, tok))
				return
			}
			*appToken = tok
			onLogin(claims.Email)
		}).
		AddButton("Quit", sh.App.Stop)
	form.SetBorder(true).SetTitle(" Log in with a JWT ").SetTitleAlign(tview.AlignLeft)
	return form
}
