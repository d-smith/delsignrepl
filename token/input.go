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

// CheatCodeLabel labels the dev-only button that mints a token (see DevSecret).
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
// GET /authping), stores it in appToken, and calls onLogin with the token's email.
// When DevSecret() is set, an Email field comes first, and the cheat code button mints a
// token for that email into the Token field. Errors show over the form.
func GetTokenInputForm(sh *ui.Shell, appToken *string, onLogin func(email string)) *tview.Form {
	tokenInput := tview.NewInputField().
		SetLabel("Token: ").
		SetFieldWidth(80).
		SetPlaceholder("paste a JWT").
		SetAcceptanceFunc(tview.InputFieldMaxLength(maxTokenLength))

	showError := sh.Message
	form := tview.NewForm()
	secret := DevSecret()

	var emailInput *tview.InputField
	minted := "" // the token the cheat code put in the Token field, if any
	if secret != "" {
		emailInput = tview.NewInputField().
			SetLabel("Email: ").
			SetFieldWidth(40).
			SetPlaceholder("you@example.com, for the cheat code")
		// A token minted for one email mustn't be saved under another: editing the
		// email takes it back.
		emailInput.SetChangedFunc(func(string) {
			if minted != "" && tokenInput.GetText() == minted {
				tokenInput.SetText("")
			}
			minted = ""
		})
		form.AddFormItem(ui.TrimPaste(emailInput))
	}

	form.AddFormItem(ui.TrimPaste(tokenInput))
	if secret != "" {
		form.AddButton(CheatCodeLabel, func() {
			email := strings.TrimSpace(emailInput.GetText())
			if _, err := mail.ParseAddress(email); err != nil {
				showError("Enter an email address in the Email field to apply the cheat code.")
				return
			}
			jwtToken, err := GenerateToken(8*time.Hour, &TokenFields{Id: randomString(20), Email: email}, secret)
			if err != nil {
				showError(err.Error())
				return
			}
			tokenInput.SetText(jwtToken)
			minted = jwtToken
			sh.Status("Token made for " + email + ". Press Save to log in.")
		})
	}
	form.AddButton("Save", func() {
		tok := strings.TrimSpace(tokenInput.GetText())
		if tok == "" && secret != "" {
			showError("Paste a token, or enter your email and press " + CheatCodeLabel + ".")
			return
		}
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
