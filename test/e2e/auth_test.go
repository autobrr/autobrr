// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"testing"

	"github.com/autobrr/autobrr/test/e2e/harness"
)

// A fresh instance has no user, so the first page load lands on onboarding.
// Creating the account there must leave the browser at a login form that then
// accepts those same credentials.
func TestOnboardingAndLogin(t *testing.T) {
	app := harness.Start(t)
	page := app.NewPage()

	page.Open("/")

	page.Fill("username", harness.Username)
	page.Fill("password1", harness.Password)
	page.Fill("password2", harness.Password)
	page.ClickButton("Create account")

	// Onboarding hands off to the login form rather than signing the new user
	// straight in.
	page.ExpectText("Sign in")

	page.Fill("username", harness.Username)
	page.Fill("password", harness.Password)
	page.ClickButton("Sign in")

	// The dashboard is the first authenticated screen.
	page.ExpectText("Stats")
}

// An instance that already has a user must not offer onboarding again.
func TestOnboardedInstanceGoesToLogin(t *testing.T) {
	app := harness.Start(t)
	app.Onboard()

	page := app.NewPage()
	page.Open("/")

	page.ExpectText("Sign in")
}
