// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func CommandUser() *cobra.Command {
	var command = &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	command.AddCommand(CommandUserCreate())
	command.AddCommand(CommandUserChangePassword())
	command.AddCommand(CommandUserGenPass())

	return command
}

func CommandUserCreate() *cobra.Command {
	var command = &cobra.Command{
		Use:     "create <username>",
		Short:   "Create a new user",
		Example: "  autobrr user create john --config /path/to/config/dir",
		Args:    cobra.ExactArgs(1),
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		username := args[0]

		db, l, err := openDatabase(cmd)
		if err != nil {
			return err
		}
		defer db.Close()

		userSvc := user.NewService(database.NewUserRepo(l, db))
		authSvc := auth.NewService(l, userSvc)

		password, err := readPassword()
		if err != nil {
			return err
		}

		hashed, err := authSvc.CreateHash(string(password))
		if err != nil {
			return errors.Wrap(err, "could not hash password")
		}

		req := domain.CreateUserRequest{
			Username: username,
			Password: hashed,
		}

		if err := userSvc.CreateUser(cmd.Context(), req); err != nil {
			return errors.Wrap(err, "could not create user: %s", username)
		}

		return nil
	}

	return command
}

func CommandUserChangePassword() *cobra.Command {
	var command = &cobra.Command{
		Use:     "change-password <username>",
		Short:   "Change the password of a user",
		Example: "  autobrr user change-password john --config /path/to/config/dir",
		Args:    cobra.ExactArgs(1),
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		username := args[0]

		db, l, err := openDatabase(cmd)
		if err != nil {
			return err
		}
		defer db.Close()

		userSvc := user.NewService(database.NewUserRepo(l, db))
		authSvc := auth.NewService(l, userSvc)

		ctx := cmd.Context()

		usr, err := userSvc.FindByUsername(ctx, username)
		if err != nil {
			return errors.Wrap(err, "could not find user: %s", username)
		}

		if usr == nil {
			return errors.New("could not find user: %s", username)
		}

		password, err := readPassword()
		if err != nil {
			return err
		}

		hashed, err := authSvc.CreateHash(string(password))
		if err != nil {
			return errors.Wrap(err, "could not hash password")
		}

		req := domain.UpdateUserRequest{
			UsernameCurrent: username,
			PasswordNew:     string(password),
			PasswordNewHash: hashed,
		}

		if err := userSvc.Update(ctx, req); err != nil {
			return errors.Wrap(err, "could not update password for user: %s", username)
		}

		fmt.Printf("successfully updated password for user %q\n", username)

		return nil
	}

	return command
}

func CommandUserGenPass() *cobra.Command {
	var command = &cobra.Command{
		Use:   "htpasswd",
		Short: "Generate a bcrypt password hash for basic auth",
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		password, err := readPassword()
		if err != nil {
			return err
		}

		hash, err := CreateHtpasswdHash(string(password))
		if err != nil {
			return err
		}

		fmt.Println(hash)

		return nil
	}

	return command
}

func readPassword() (password []byte, err error) {
	fd := int(os.Stdin.Fd())

	if term.IsTerminal(fd) {
		fmt.Printf("Password: ")
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Printf("\n")
		if err != nil {
			return nil, errors.Wrap(err, "failed to read password from terminal")
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, errors.Wrap(err, "failed to read password from stdin")
			}

			return nil, errors.New("password input is empty")
		}

		password = scanner.Bytes()
	}

	// make sure the password is not empty
	if len(password) == 0 {
		return nil, errors.New("zero length password")
	}

	return password, nil
}

// CreateHtpasswdHash generates a bcrypt hash of the password for use in basic auth
func CreateHtpasswdHash(password string) (string, error) {
	// Generate a bcrypt hash from the input password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.Wrap(err, "could not generate hash")
	}

	// Return the formatted bcrypt hash (with the bcrypt marker "$2y$")
	return string(hash), nil
}
