package login

import (
	"errors"
	"github.com/miprotek/grafana-de/pkg/bus"
	m "github.com/miprotek/grafana-de/pkg/models"
)

var (
	ErrEmailNotAllowed       = errors.New("Erforderliche E-Mail Domäne wurde nicht erfüllt")
	ErrInvalidCredentials    = errors.New("Ungültiger Benutzername oder Passwort")
	ErrNoEmail               = errors.New("Der Login-Anbieter hat keine E-Mail Adresse angegeben")
	ErrProviderDeniedRequest = errors.New("Login-Anbieter verweigert Login-Anfrage")
	ErrSignUpNotAllowed      = errors.New("Die Anmeldung ist für diesen Adapter nicht zulässig")
	ErrTooManyLoginAttempts  = errors.New("Zuviele aufeinanderfolgende falsche Anmeldeversuche für den Benutzer. Login für Benutzer vorübergehend gesperrt")
	ErrPasswordEmpty         = errors.New("Kein Passwort angegeben.")
	ErrUsersQuotaReached     = errors.New("Benutzerkontingent erreicht")
	ErrGettingUserQuota      = errors.New("Fehler beim Abrufen des Benutzerkontingents")
)

func Init() {
	bus.AddHandler("auth", AuthenticateUser)
	loadLdapConfig()
}

func AuthenticateUser(query *m.LoginUserQuery) error {
	if err := validateLoginAttempts(query.Username); err != nil {
		return err
	}

	if err := validatePasswordSet(query.Password); err != nil {
		return err
	}

	err := loginUsingGrafanaDB(query)
	if err == nil || (err != m.ErrUserNotFound && err != ErrInvalidCredentials) {
		return err
	}

	ldapEnabled, ldapErr := loginUsingLdap(query)
	if ldapEnabled {
		if ldapErr == nil || ldapErr != ErrInvalidCredentials {
			return ldapErr
		}

		err = ldapErr
	}

	if err == ErrInvalidCredentials {
		saveInvalidLoginAttempt(query)
	}

	if err == m.ErrUserNotFound {
		return ErrInvalidCredentials
	}

	return err
}
func validatePasswordSet(password string) error {
	if len(password) == 0 {
		return ErrPasswordEmpty
	}

	return nil
}
