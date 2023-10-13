package token

import (
	"delsignrepl/state"
	"fmt"
	"math/rand"
	"net/mail"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rivo/tview"
)

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

func GetTokenInputForm(pages *tview.Pages, app *tview.Application) *tview.Form {
	tokenInput := tview.NewInputField().SetLabel("Token value: ").SetFieldWidth(256).SetAcceptanceFunc(tview.InputFieldMaxLength(256))

	cheatCode := tview.NewCheckbox().SetLabel("Apply Cheat Code").SetChecked(false)
	cheatCode.SetChangedFunc(func(checked bool) {
		if checked {
			email := tokenInput.GetText()
			_, err := mail.ParseAddress(email)
			if err != nil {
				modal := tview.NewModal().
					SetText("Invalid email address. Enter an email address to apply cheat.").
					AddButtons([]string{"Ok"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						cheatCode.SetChecked(false)
						pages.SwitchToPage("Add Token")
					})
				pages.AddPage("InvalidEmail", modal, true, false)
				pages.SwitchToPage("InvalidEmail")
				return
			}

			randomStr := randomString(20)
			var tokenFields TokenFields = TokenFields{
				Id:    randomStr,
				Email: tokenInput.GetText(),
			}
			jwtToken, _ := GenerateToken(8*time.Hour, &tokenFields, "secret")
			tokenInput.SetText(jwtToken)

		} else {
			tokenInput.SetText("")
		}
	})

	form := tview.NewForm().
		AddFormItem(tokenInput).
		AddFormItem(cheatCode).
		AddButton("Save", func() {
			state.Token = tokenInput.GetText()
			pages.SwitchToPage("Menu")
		}).
		AddButton("Quit", func() {
			app.Stop()
		})
	form.SetBorder(true).SetTitle("Initialize application with JWT token").SetTitleAlign(tview.AlignLeft)
	return form
}
